package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
)

// HierarchyReader enumerates existing topology for trusted local operators.
// A result or continuation cursor grants no authorization.
type HierarchyReader struct{ Pools *database.Catalog }

// HierarchyQuery selects a kind and optional immediate owner. Organizations
// accept Organization as their parent, schools accept their owning Organization,
// and campuses accept School. Version zero starts at the current snapshot.
type HierarchyQuery struct {
	Tenant       scope.TenantID
	Kind         string
	Organization scope.OrganizationID
	School       scope.SchoolID
	Limit        int
	Cursor       string
	Version      uint64
}

// HierarchyEntry separates the node's own Active state from Path.Active,
// which includes every ancestor and the tenant gate.
type HierarchyEntry struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Code     string      `json:"code,omitempty"`
	Timezone string      `json:"timezone,omitempty"`
	Active   bool        `json:"active"`
	Path     scope.Chain `json:"path"`
}

type HierarchyPage struct {
	Entries          []HierarchyEntry `json:"entries"`
	HierarchyVersion uint64           `json:"hierarchyVersion"`
	NextCursor       string           `json:"nextCursor,omitempty"`
}

// directoryCursor is unsigned continuation metadata. Every page independently
// validates placement, the gate and ancestry. Never use it as an access grant.
type directoryCursor struct {
	Format  int    `json:"format"`
	Tenant  string `json:"tenant"`
	Kind    string `json:"kind"`
	Owner   string `json:"owner"`
	Version uint64 `json:"version"`
	Epoch   uint64 `json:"epoch"`
	Pool    string `json:"pool"`
	After   string `json:"after"`
}

func (q HierarchyQuery) continuation() (directoryCursor, error) {
	c := directoryCursor{Format: 1, Tenant: q.Tenant.String(), Kind: q.Kind, Version: q.Version}
	if q.Tenant.IsZero() || q.Limit < 1 || q.Limit > 100 || q.Version > math.MaxInt64 {
		return c, models.ErrInvalid
	}
	switch q.Kind {
	case "organizations", "schools":
		if !q.School.IsZero() {
			return c, models.ErrInvalid
		}
		if !q.Organization.IsZero() {
			c.Owner = q.Organization.String()
		}
	case "campuses":
		if !q.Organization.IsZero() {
			return c, models.ErrInvalid
		}
		if !q.School.IsZero() {
			c.Owner = q.School.String()
		}
	default:
		return c, models.ErrInvalid
	}
	if q.Cursor == "" {
		return c, nil
	}
	if len(q.Cursor) > 2048 {
		return c, models.ErrInvalid
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(q.Cursor)
	if err != nil {
		return c, models.ErrInvalid
	}
	var saved directoryCursor
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&saved); err != nil {
		return c, models.ErrInvalid
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return c, models.ErrInvalid
	}
	canonical, err := json.Marshal(saved)
	if err != nil || !bytes.Equal(data, canonical) {
		return c, models.ErrInvalid
	}
	if saved.Epoch == 0 || saved.Epoch > math.MaxInt64 || saved.Pool == "" {
		return c, models.ErrInvalid
	}
	if saved.Format != c.Format || saved.Tenant != c.Tenant || saved.Kind != c.Kind || saved.Owner != c.Owner || saved.Version == 0 || saved.Version > math.MaxInt64 || (q.Version != 0 && saved.Version != q.Version) {
		return c, models.ErrInvalid
	}
	if _, err := scope.Parse[scope.Organization](saved.After); err != nil {
		return c, models.ErrInvalid
	}
	return saved, nil
}

func (r HierarchyReader) List(ctx context.Context, q HierarchyQuery) (HierarchyPage, error) {
	cursor, err := q.continuation()
	if err != nil {
		return HierarchyPage{}, err
	}
	p, err := r.Pools.PinnedPool(ctx, q.Tenant.String())
	if err != nil {
		return HierarchyPage{}, err
	}
	opts := &sql.TxOptions{ReadOnly: true}
	if p.Driver == "postgres" {
		opts.Isolation = sql.LevelRepeatableRead
	}
	tx, err := p.DB.BeginTx(ctx, opts)
	if err != nil {
		return HierarchyPage{}, err
	}
	defer tx.Rollback()
	// Resolve the tenant gate separately so inactive owners can be reported while
	// a closed tenant gate always refuses enumeration, including empty pages.
	gate, err := directoryPath(ctx, tx, p, scope.Selection{Tenant: q.Tenant})
	if err != nil {
		return HierarchyPage{}, err
	}
	if !gate.Snapshot().Active {
		return HierarchyPage{}, models.ErrScope
	}
	var epoch uint64
	var poolID string
	if err := tx.QueryRowContext(ctx, p.Bind("SELECT placement_epoch,pool_id FROM tenant_gates WHERE tenant_id=?"), q.Tenant.String()).Scan(&epoch, &poolID); err != nil {
		return HierarchyPage{}, err
	}
	if q.Cursor != "" && (cursor.Epoch != epoch || cursor.Pool != poolID) {
		return HierarchyPage{}, models.ErrConflict
	}
	cursor.Epoch, cursor.Pool = epoch, poolID
	if cursor.Version != 0 && cursor.Version != gate.Version() {
		return HierarchyPage{}, models.ErrConflict
	}
	cursor.Version = gate.Version()
	if cursor.Owner != "" {
		_, err = directoryPath(ctx, tx, p, scope.Selection{Tenant: q.Tenant, Organization: q.Organization, School: q.School})
		if err != nil {
			return HierarchyPage{}, err
		}
	}
	ids, err := directoryIDs(ctx, tx, p, q, cursor)
	if err != nil {
		return HierarchyPage{}, err
	}
	page := HierarchyPage{Entries: make([]HierarchyEntry, 0, q.Limit), HierarchyVersion: cursor.Version}
	if len(ids) > q.Limit {
		ids = ids[:q.Limit]
		cursor.After = ids[len(ids)-1]
		data, err := json.Marshal(cursor)
		if err != nil {
			return HierarchyPage{}, err
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	for _, id := range ids {
		entry, err := directoryEntry(ctx, tx, p, q, id)
		if err != nil {
			return HierarchyPage{}, err
		}
		page.Entries = append(page.Entries, entry)
	}
	if err := tx.Commit(); err != nil {
		return HierarchyPage{}, err
	}
	return page, nil
}

func directoryPath(ctx context.Context, tx *sql.Tx, p database.Pool, sel scope.Selection) (models.Graph, error) {
	graph, err := loadPath(ctx, tx, p, sel)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Graph{}, models.ErrScope
	}
	if err != nil {
		return models.Graph{}, err
	}
	if _, err := graph.Resolve(sel); err != nil {
		return models.Graph{}, err
	}
	return graph, nil
}

func directoryIDs(ctx context.Context, tx *sql.Tx, p database.Pool, q HierarchyQuery, c directoryCursor) ([]string, error) {
	var statement string
	switch q.Kind {
	case "organizations":
		statement = "SELECT id FROM tenancy_organizations WHERE tenant_id=? AND id>? AND (?='' OR parent_id=?) ORDER BY id LIMIT ?"
	case "schools":
		statement = "SELECT id FROM tenancy_schools WHERE tenant_id=? AND id>? AND (?='' OR organization_id=?) ORDER BY id LIMIT ?"
	case "campuses":
		statement = "SELECT id FROM tenancy_campuses WHERE tenant_id=? AND id>? AND (?='' OR school_id=?) ORDER BY id LIMIT ?"
	}
	rows, err := tx.QueryContext(ctx, p.Bind(statement), c.Tenant, c.After, c.Owner, c.Owner, q.Limit+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0, q.Limit+1)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func directoryEntry(ctx context.Context, tx *sql.Tx, p database.Pool, q HierarchyQuery, id string) (HierarchyEntry, error) {
	sel := scope.Selection{Tenant: q.Tenant}
	var err error
	switch q.Kind {
	case "organizations":
		sel.Organization, err = scope.Parse[scope.Organization](id)
	case "schools":
		sel.School, err = scope.Parse[scope.School](id)
		sel.Organization = q.Organization
	case "campuses":
		sel.Campus, err = scope.Parse[scope.Campus](id)
		sel.School = q.School
	}
	if err != nil {
		return HierarchyEntry{}, err
	}
	graph, err := directoryPath(ctx, tx, p, sel)
	if err != nil {
		return HierarchyEntry{}, err
	}
	path, err := graph.Resolve(sel)
	if err != nil {
		return HierarchyEntry{}, err
	}
	entry := HierarchyEntry{ID: id, Path: path}
	snapshot := graph.Snapshot()
	switch q.Kind {
	case "organizations":
		node := snapshot.Organizations[len(snapshot.Organizations)-1]
		if !q.Organization.IsZero() && node.Parent() != q.Organization {
			return HierarchyEntry{}, models.ErrScope
		}
		entry.Name, entry.Active = node.Name(), node.Active()
	case "schools":
		node := snapshot.Schools[0]
		entry.Name, entry.Code, entry.Active = node.Name(), node.Code(), node.Active()
	case "campuses":
		node := snapshot.Campuses[0]
		entry.Name, entry.Code, entry.Timezone, entry.Active = node.Name(), node.Code(), node.Timezone(), node.Active()
	}
	return entry, nil
}
