package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
)

// CampusWriter is a trusted operator service. The directory contract
// exposed to other modules grants no mutation capability.
type CampusWriter struct{ Pools *database.Catalog }

func (w CampusWriter) Create(ctx context.Context, campus models.Campus, epoch, version uint64) (scope.Chain, error) {
	if campus.ID().IsZero() || !campus.Active() || epoch == 0 || epoch > math.MaxInt64 || version == 0 || version >= math.MaxInt64 {
		return scope.Chain{}, models.ErrInvalid
	}
	p, err := w.Pools.PinnedPool(ctx, campus.Tenant().String())
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
	AND pool_id=(SELECT pool_id FROM edu_pool_marker WHERE singleton=1)`), campus.Tenant().String(), epoch, version)
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
	sel := scope.Selection{Tenant: campus.Tenant(), School: campus.School()}
	graph, err := loadPath(ctx, tx, p, sel)
	if errors.Is(err, sql.ErrNoRows) {
		return scope.Chain{}, models.ErrScope
	}
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
	graph, err = graph.AddCampus(campus)
	if err != nil {
		return scope.Chain{}, err
	}
	_, err = tx.ExecContext(ctx, p.Bind(`INSERT INTO tenancy_campuses (tenant_id,id,school_id,code,name,timezone,active) VALUES (?,?,?,?,?,?,true)`), campus.Tenant().String(), campus.ID().String(), campus.School().String(), campus.Code(), campus.Name(), campus.Timezone())
	if err != nil {
		return scope.Chain{}, err
	}
	_, err = tx.ExecContext(ctx, p.Bind("UPDATE tenant_gates SET hierarchy_version=? WHERE tenant_id=?"), graph.Version(), campus.Tenant().String())
	if err != nil {
		return scope.Chain{}, err
	}
	sel.Campus = campus.ID()
	chain, err := graph.Resolve(sel)
	if err != nil {
		return scope.Chain{}, err
	}
	payload, err := json.Marshal(chain)
	if err != nil {
		return scope.Chain{}, err
	}
	if err = outbox.Append(ctx, tx, p, outbox.Event{TenantID: campus.Tenant().String(), Type: "tenancy.campus.created.v1", AggregateID: campus.ID().String(), Payload: payload}); err != nil {
		return scope.Chain{}, err
	}
	if err = tx.Commit(); err != nil {
		return scope.Chain{}, err
	}
	return chain, nil
}
