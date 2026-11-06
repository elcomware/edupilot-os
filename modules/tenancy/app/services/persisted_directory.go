package services

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
)

// PersistedDirectory resolves a selected path from one consistent tenant-pool
// snapshot. A lookup never provisions a placement or grants access to records.
type PersistedDirectory struct{ Pools *database.Catalog }

func (d PersistedDirectory) Resolve(ctx context.Context, sel scope.Selection) (scope.Chain, error) {
	if sel.Tenant.IsZero() {
		return scope.Chain{}, models.ErrScope
	}
	if !sel.Team.IsZero() {
		return scope.Chain{}, models.ErrTeam
	}
	p, err := d.Pools.PinnedPool(ctx, sel.Tenant.String())
	if err != nil {
		return scope.Chain{}, err
	}
	opts := &sql.TxOptions{ReadOnly: true}
	if p.Driver == "postgres" {
		opts.Isolation = sql.LevelRepeatableRead
	}
	tx, err := p.DB.BeginTx(ctx, opts)
	if err != nil {
		return scope.Chain{}, err
	}
	defer tx.Rollback()
	graph, err := loadPath(ctx, tx, p, sel)
	if errors.Is(err, sql.ErrNoRows) {
		return scope.Chain{}, models.ErrScope
	}
	if err != nil {
		return scope.Chain{}, err
	}
	chain, err := graph.Resolve(sel)
	if err != nil {
		return scope.Chain{}, err
	}
	if err := tx.Commit(); err != nil {
		return scope.Chain{}, err
	}
	return chain, nil
}

func loadPath(ctx context.Context, tx *sql.Tx, p database.Pool, sel scope.Selection) (models.Graph, error) {
	tenant := sel.Tenant
	row := models.Snapshot{Tenant: tenant}
	var epoch int64
	var state string
	err := tx.QueryRowContext(ctx, p.Bind(`SELECT g.hierarchy_version,g.placement_epoch,g.state
		FROM tenant_gates g JOIN edu_pool_marker m ON m.singleton=1 AND m.pool_id=g.pool_id
		WHERE g.tenant_id=?`), tenant.String()).Scan(&row.Version, &epoch, &state)
	if err != nil {
		return models.Graph{}, err
	}
	if epoch < 1 {
		return models.Graph{}, database.ErrPlacement
	}
	row.Active = state == "active"
	query := func(statement, id string) *sql.Row {
		return tx.QueryRowContext(ctx, p.Bind(statement), tenant.String(), id)
	}
	school, org := sel.School, sel.Organization
	if !sel.Campus.IsZero() {
		var parent, code, name, zone string
		var active bool
		err := query("SELECT school_id,code,name,timezone,active FROM tenancy_campuses WHERE tenant_id=? AND id=?", sel.Campus.String()).Scan(&parent, &code, &name, &zone, &active)
		if err != nil {
			return models.Graph{}, err
		}
		school, err = scope.Parse[scope.School](parent)
		if err != nil {
			return models.Graph{}, err
		}
		campus, err := models.NewCampus(sel.Campus, tenant, school, code, name, zone)
		if err != nil {
			return models.Graph{}, err
		}
		row.Campuses = append(row.Campuses, campus.WithActive(active))
	}
	if !school.IsZero() {
		var parent, code, name string
		var active bool
		err := query("SELECT organization_id,code,name,active FROM tenancy_schools WHERE tenant_id=? AND id=?", school.String()).Scan(&parent, &code, &name, &active)
		if err != nil {
			return models.Graph{}, err
		}
		org, err = scope.Parse[scope.Organization](parent)
		if err != nil {
			return models.Graph{}, err
		}
		node, err := models.NewSchool(school, tenant, org, code, name)
		if err != nil {
			return models.Graph{}, err
		}
		row.Schools = append(row.Schools, node.WithActive(active))
	}
	seen := make(map[scope.OrganizationID]bool)
	for !org.IsZero() {
		if seen[org] {
			return models.Graph{}, models.ErrScope
		}
		seen[org] = true
		var parent sql.NullString
		var name string
		var active bool
		if err := query("SELECT parent_id,name,active FROM tenancy_organizations WHERE tenant_id=? AND id=?", org.String()).Scan(&parent, &name, &active); err != nil {
			return models.Graph{}, err
		}
		var next scope.OrganizationID
		if parent.Valid {
			next, err = scope.Parse[scope.Organization](parent.String)
			if err != nil {
				return models.Graph{}, err
			}
		}
		node, err := models.NewOrganization(org, tenant, next, name)
		if err != nil {
			return models.Graph{}, err
		}
		row.Organizations = append(row.Organizations, node.WithActive(active))
		org = next
	}
	slices.Reverse(row.Organizations)
	return models.Restore(row)
}
