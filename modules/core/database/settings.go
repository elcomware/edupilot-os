// Package database binds verified school IDs to Goravel-managed SQL pools.
package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/goravel/framework/contracts/database/driver"
	postgresfacades "github.com/goravel/postgres/facades"
	sqlitefacades "github.com/goravel/sqlite/facades"
)

type Connection struct {
	Name     string `json:"-"`
	Tenant   string `json:"tenant"`
	Driver   string `json:"driver"`
	Database string `json:"database"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	SSLMode  string `json:"sslmode,omitempty"`
}

type Settings struct {
	Control   Connection
	Dedicated []Connection
}

var tenantPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

func ValidTenant(id string) bool { return tenantPattern.MatchString(id) }

func (c Connection) Owner() string {
	if c.Tenant == "" {
		return "shared"
	}
	return "tenant:" + c.Tenant
}

// ConfiguredSettings uses the same environment reader as the application's config files.
func ConfiguredSettings() (Settings, error) {
	return ReadSettings(func(k, fallback string) string {
		return fmt.Sprint(facades.Config().Env(k, fallback))
	})
}

func ReadSettings(get func(key, fallback string) string) (Settings, error) {
	driverName := get("DB_CONNECTION", "sqlite")
	c := Connection{Name: driverName, Driver: driverName}
	switch driverName {
	case "postgres":
		port, err := strconv.Atoi(get("DB_PORT", "5432"))
		if err != nil {
			return Settings{}, errors.New("invalid DB_PORT")
		}
		c.Database = get("DB_DATABASE", "edupilot")
		c.Host, c.Port = get("DB_HOST", "127.0.0.1"), port
		c.Username, c.Password = get("DB_USERNAME", "edupilot"), get("DB_PASSWORD", "")
		c.SSLMode = get("DB_SSLMODE", "require")
	case "sqlite":
		c.Database = get("DB_DATABASE", "database/edupilot.sqlite")
	default:
		return Settings{}, errors.New("DB_CONNECTION must be postgres or sqlite")
	}
	settings := Settings{Control: c}
	raw := get("EDUPILOT_DEDICATED_DATABASES", "[]")
	if strings.TrimSpace(raw) == "" {
		raw = "[]"
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&settings.Dedicated); err != nil || settings.Dedicated == nil {
		return Settings{}, errors.New("EDUPILOT_DEDICATED_DATABASES must be a JSON array of database configurations")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return Settings{}, errors.New("unexpected data after EDUPILOT_DEDICATED_DATABASES")
	}
	for i := range settings.Dedicated {
		node := &settings.Dedicated[i]
		node.Name = "tenant_" + node.Tenant
		if node.Driver == "postgres" {
			if node.Port == 0 {
				node.Port = 5432
			}
			if node.SSLMode == "" {
				node.SSLMode = "require"
			}
		}
	}
	slices.SortFunc(settings.Dedicated, func(a, b Connection) int { return strings.Compare(a.Tenant, b.Tenant) })
	if err := settings.Validate(); err != nil {
		return Settings{}, err
	}
	return settings, nil
}

func (s Settings) Nodes() []Connection {
	return append([]Connection{s.Control}, s.Dedicated...)
}

func (s Settings) Validate() error {
	if s.Control.Tenant != "" || s.Control.Name != s.Control.Driver {
		return errors.New("control connection must be named for its driver")
	}
	seenName, seenDB := map[string]bool{}, map[string]bool{}
	for i, node := range s.Nodes() {
		if i > 0 && (!ValidTenant(node.Tenant) || node.Name != "tenant_"+node.Tenant) {
			return errors.New("invalid dedicated tenant or connection name")
		}
		key, err := node.targetKey()
		if err != nil {
			return fmt.Errorf("connection %q: %w", node.Name, err)
		}
		if seenName[node.Name] || seenDB[key] {
			return fmt.Errorf("connection %q repeats a tenant or database target", node.Name)
		}
		seenName[node.Name], seenDB[key] = true, true
	}
	return nil
}

func (c Connection) targetKey() (string, error) {
	if strings.TrimSpace(c.Database) == "" {
		return "", errors.New("database is required")
	}
	switch c.Driver {
	case "sqlite":
		if c.Database == ":memory:" || strings.HasPrefix(c.Database, "file:") || strings.ContainsAny(c.Database, "?#\x00") {
			return "", errors.New("use a SQLite file path, not a URI or in-memory database")
		}
		abs, err := filepath.Abs(c.Database)
		if err != nil {
			return "", err
		}
		// Resolve the existing parent even when the database has not been created yet.
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		} else if dir, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
			abs = filepath.Join(dir, filepath.Base(abs))
		}
		return "sqlite:" + abs, nil
	case "postgres":
		if c.Host == "" || c.Username == "" || c.Port < 1 || c.Port > 65535 {
			return "", errors.New("postgres requires host, username and a valid port")
		}
		switch c.SSLMode {
		case "disable", "require", "verify-ca", "verify-full":
		default:
			return "", errors.New("invalid postgres sslmode")
		}
		return "postgres:" + net.JoinHostPort(strings.ToLower(c.Host), strconv.Itoa(c.Port)) + "/" + c.Database, nil
	default:
		return "", errors.New("driver must be postgres or sqlite")
	}
}

// SQLiteFileDSN passes an absolute filesystem path unchanged to SQLite's VFS.
// Escaping the filename separately keeps Windows drive letters out of a URL
// authority and literal percent signs out of SQLite's query parameters.
func SQLiteFileDSN(absolutePath string, options url.Values) string {
	return "file:" + url.PathEscape(filepath.ToSlash(absolutePath)) + "?" + options.Encode()
}

func (s Settings) Connections() map[string]any {
	all := make(map[string]any)
	for _, node := range s.Nodes() {
		config := map[string]any{"database": node.Database, "singular": false, "prefix": ""}
		if node.Driver == "sqlite" {
			// The driver's URI options apply to every connection in the pool.
			path, _ := filepath.Abs(node.Database)
			config["dsn"] = SQLiteFileDSN(path, url.Values{"_pragma": {"foreign_keys(ON)", "busy_timeout(5000)"}})
			config["via"] = func() (driver.Driver, error) { return sqlitefacades.Sqlite(node.Name) }
		} else {
			uri := url.URL{Scheme: "postgres", User: url.UserPassword(node.Username, node.Password),
				Host: net.JoinHostPort(node.Host, strconv.Itoa(node.Port)), Path: "/" + node.Database}
			params := url.Values{"sslmode": {node.SSLMode}, "search_path": {"public"}, "timezone": {"UTC"}}
			uri.RawQuery = params.Encode()
			config["host"], config["port"], config["database"] = node.Host, node.Port, node.Database
			config["username"], config["password"], config["sslmode"] = node.Username, node.Password, node.SSLMode
			config["schema"], config["dsn"] = "public", uri.String()
			config["via"] = func() (driver.Driver, error) { return postgresfacades.Postgres(node.Name) }
		}
		all[node.Name] = config
	}
	return all
}
