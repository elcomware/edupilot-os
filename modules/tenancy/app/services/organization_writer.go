package services

import (
	"context"
	"encoding/json"
	"math"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
)

// OrganizationWriter is a trusted operator service. The directory contract
// exposed to other modules grants no mutation capability.
type OrganizationWriter struct{ Pools *database.Catalog }

func (w OrganizationWriter) Create(ctx context.Context, org models.Organization, epoch, version uint64) (scope.Chain, error) {
	if org.ID().IsZero() || !org.Active() || epoch == 0 || epoch > math.MaxInt64 || version == 0 || version >= math.MaxInt64 {
		return scope.Chain{}, models.ErrInvalid
	}
	p, err := w.Pools.PinnedPool(ctx, org.Tenant().String())
	if err != nil {
		return scope.Chain{}, err
	}
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return scope.Chain{}, err
	}
	defer tx.Rollback()
	// Lock the active gate before reading ancestry. The conditional write
	// serializes concurrent edits and is rolled back with any failed insert.
	lock, err := tx.ExecContext(ctx, p.Bind(`UPDATE tenant_gates SET hierarchy_version=hierarchy_version
	WHERE tenant_id=? AND state='active' AND placement_epoch=? AND hierarchy_version=?
	AND pool_id=(SELECT pool_id FROM edu_pool_marker WHERE singleton=1)`), org.Tenant().String(), epoch, version)
	if err != nil {
		return scope.Chain{}, err
	}
	n, err := lock.RowsAffected()
	if err != nil {
		return scope.Chain{}, err
	}
	if n != 1 {
		return scope.Chain{}, models.ErrConflict
	}
	sel := scope.Selection{Tenant: org.Tenant(), Organization: org.Parent()}
	graph, err := loadPath(ctx, tx, p, sel)
	if err != nil {
		return scope.Chain{}, err
	}
	parent, err := graph.Resolve(sel)
	if err != nil {
		return scope.Chain{}, err
	}
	if !parent.Active {
		return scope.Chain{}, models.ErrScope
	}
	graph, err = graph.AddOrganization(org)
	if err != nil {
		return scope.Chain{}, err
	}
	var parentID any
	if !org.Parent().IsZero() {
		parentID = org.Parent().String()
	}
	_, err = tx.ExecContext(ctx, p.Bind(`INSERT INTO tenancy_organizations (tenant_id,id,parent_id,name,active) VALUES (?,?,?,?,true)`), org.Tenant().String(), org.ID().String(), parentID, org.Name())
	if err != nil {
		return scope.Chain{}, err
	}
	_, err = tx.ExecContext(ctx, p.Bind("UPDATE tenant_gates SET hierarchy_version=? WHERE tenant_id=?"), graph.Version(), org.Tenant().String())
	if err != nil {
		return scope.Chain{}, err
	}
	sel.Organization = org.ID()
	chain, err := graph.Resolve(sel)
	if err != nil {
		return scope.Chain{}, err
	}
	payload, err := json.Marshal(chain)
	if err != nil {
		return scope.Chain{}, err
	}
	if err = outbox.Append(ctx, tx, p, outbox.Event{TenantID: org.Tenant().String(), Type: "tenancy.organization.created.v1", AggregateID: org.ID().String(), Payload: payload}); err != nil {
		return scope.Chain{}, err
	}
	if err = tx.Commit(); err != nil {
		return scope.Chain{}, err
	}
	return chain, nil
}
