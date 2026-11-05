package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elcomware/edupilot-os/app/facades"
)

var ErrPlacement = errors.New("tenant placement mismatch")

type Pool struct {
	DB     *sql.DB
	Driver string
}

// Bind converts question-mark placeholders in repository SQL to pgx's $1, $2, ...
// SQL is always application-owned. Quoted strings/identifiers and comments are left alone.
func (p Pool) Bind(query string) string {
	if p.Driver != "postgres" {
		return query
	}
	var result strings.Builder
	count := 0
	for i := 0; i < len(query); {
		start := i
		switch query[i] {
		case '\'', '"':
			quote := query[i]
			i++
			for i < len(query) {
				i++
				if query[i-1] != quote {
					continue
				}
				if i == len(query) || query[i] != quote {
					break
				}
				i++
			}
		case '?':
			count++
			result.WriteByte('$')
			result.WriteString(strconv.Itoa(count))
			i++
			continue
		default:
			if strings.HasPrefix(query[i:], "--") {
				for i < len(query) && query[i] != '\n' {
					i++
				}
			} else if strings.HasPrefix(query[i:], "/*") {
				end := strings.Index(query[i+2:], "*/")
				if end < 0 {
					i = len(query)
				} else {
					i += end + 4
				}
			} else {
				i++
			}
		}
		result.WriteString(query[start:i])
	}
	return result.String()
}

type target struct {
	pool  Pool
	owner string
}

type Catalog struct {
	Control Pool
	nodes   []target
	routes  map[string]target
}

type Resolver func(Connection) (*sql.DB, error)

// Open does not run migrations or close handles; their lifecycle belongs to the resolver.
func Open(settings Settings, resolve Resolver) (*Catalog, error) {
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	if resolve == nil {
		return nil, errors.New("missing database resolver")
	}
	catalog := &Catalog{routes: make(map[string]target)}
	seen := make(map[*sql.DB]bool)
	for _, node := range settings.Nodes() {
		handle, err := resolve(node)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", node.Name, err)
		}
		if handle == nil || seen[handle] {
			return nil, fmt.Errorf("connection %s has an invalid or reused pool", node.Name)
		}
		seen[handle] = true
		t := target{Pool{handle, node.Driver}, node.Owner()}
		catalog.nodes = append(catalog.nodes, t)
		if node.Tenant == "" {
			catalog.Control = t.pool
		} else {
			catalog.routes[node.Tenant] = t
		}
	}
	return catalog, nil
}

func OpenConfigured() (*Catalog, error) {
	settings, err := ConfiguredSettings()
	if err != nil {
		return nil, err
	}
	return Open(settings, func(node Connection) (*sql.DB, error) {
		orm := facades.Orm()
		if orm == nil || orm.Query() == nil {
			return nil, errors.New("Goravel ORM is not available")
		}
		connection := orm.Connection(node.Name)
		if connection == nil || connection.Query() == nil {
			return nil, errors.New("Goravel connection is not available")
		}
		return connection.DB()
	})
}

// All returns control/shared first and then each distinct dedicated pool.
func (c *Catalog) All() []Pool {
	result := make([]Pool, 0, len(c.nodes))
	for _, node := range c.nodes {
		result = append(result, node.pool)
	}
	return result
}

func (t target) identity(ctx context.Context) (string, error) {
	var owner, identity string
	if err := t.pool.DB.QueryRowContext(ctx, "SELECT owner, pool_id FROM edu_pool_marker WHERE singleton = 1").Scan(&owner, &identity); err != nil {
		return "", fmt.Errorf("read pool marker for %s: %w", t.owner, err)
	}
	if owner != t.owner || identity == "" {
		return "", fmt.Errorf("%w: expected %q", ErrPlacement, t.owner)
	}
	return identity, nil
}

// CheckReady validates migrations, connectivity and ownership on every physical database.
func (c *Catalog) CheckReady(ctx context.Context) error {
	for _, t := range c.nodes {
		if _, err := t.identity(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ForTenant is for code without a request context. Use ForTenantContext in request handlers.
func (c *Catalog) ForTenant(id string) (Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.ForTenantContext(ctx, id)
}

// ForTenantContext requires an already authenticated and authorized school ID.
// First use pins a school to the physical pool's immutable ID. Removing an override,
// changing the shared DB target, or silently remapping a school cannot bypass this pin
// while the control database is intact. Moving data requires explicit operator cutover.
func (c *Catalog) ForTenantContext(ctx context.Context, id string) (Pool, error) {
	if !ValidTenant(id) {
		return Pool{}, fmt.Errorf("%w: invalid school ID", ErrPlacement)
	}
	if c == nil || len(c.nodes) == 0 {
		return Pool{}, errors.New("database catalog is not configured")
	}
	t, ok := c.routes[id]
	if !ok {
		t = c.nodes[0]
	}
	identity, err := t.identity(ctx)
	if err != nil {
		return Pool{}, err
	}
	read := func() (string, error) {
		var pinned string
		err := c.Control.DB.QueryRowContext(ctx, c.Control.Bind("SELECT pool_id FROM edu_tenant_placement WHERE tenant_id = ?"), id).Scan(&pinned)
		return pinned, err
	}
	pinned, err := read()
	if errors.Is(err, sql.ErrNoRows) {
		q := c.Control.Bind("INSERT INTO edu_tenant_placement (tenant_id, pool_id) VALUES (?, ?) ON CONFLICT (tenant_id) DO NOTHING")
		if _, err = c.Control.DB.ExecContext(ctx, q, id, identity); err == nil {
			pinned, err = read()
		}
	}
	if err != nil {
		return Pool{}, fmt.Errorf("read placement: %w", err)
	}
	if pinned != identity {
		return Pool{}, fmt.Errorf("%w: school %q requires an explicit database cutover", ErrPlacement, id)
	}
	return t.pool, nil
}
