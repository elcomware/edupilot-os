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

// HierarchyEditor edits metadata for trusted native operators. Directory
// consumers receive no mutation capability through their lookup contract.
type HierarchyEditor struct{ Pools *database.Catalog }

// HierarchyMetadata is a complete replacement of the selected kind's metadata.
// Organizations require Name, schools Name/Code, campuses Name/Code/Timezone.
// Fields belonging to another kind are rejected.
type HierarchyMetadata struct {
	Name     string `json:"name"`
	Code     string `json:"code,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// HierarchyEdit uses the plural directory kind and an existing record UUID.
// Identity, parentage and lifecycle are always loaded from the pinned pool.
type HierarchyEdit struct {
	Tenant   scope.TenantID
	Kind     string
	ID       string
	Metadata HierarchyMetadata
	Epoch    uint64
	Version  uint64
}

type HierarchyEditResult struct {
	Changed bool           `json:"changed"`
	Entry   HierarchyEntry `json:"entry"`
}

// HierarchyMetadataUpdated is the payload for the kind-specific v1 event.
// Path contains the committed version and complete resolved ownership chain.
type HierarchyMetadataUpdated struct {
	Before HierarchyMetadata `json:"before"`
	After  HierarchyMetadata `json:"after"`
	Path   scope.Chain       `json:"path"`
}

// selection validates the command shape before opening a transaction. Metadata
// values are validated with the existing model constructors and loaded parents.
func (cmd HierarchyEdit) selection() (scope.Selection, error) {
	sel := scope.Selection{Tenant: cmd.Tenant}
	if cmd.Tenant.IsZero() || cmd.Epoch == 0 || cmd.Epoch > math.MaxInt64 || cmd.Version == 0 || cmd.Version >= math.MaxInt64 {
		return sel, models.ErrInvalid
	}
	var err error
	switch cmd.Kind {
	case "organizations":
		if cmd.Metadata.Code != "" || cmd.Metadata.Timezone != "" {
			return sel, models.ErrInvalid
		}
		sel.Organization, err = scope.Parse[scope.Organization](cmd.ID)
	case "schools":
		if cmd.Metadata.Timezone != "" {
			return sel, models.ErrInvalid
		}
		sel.School, err = scope.Parse[scope.School](cmd.ID)
	case "campuses":
		sel.Campus, err = scope.Parse[scope.Campus](cmd.ID)
	default:
		return sel, models.ErrInvalid
	}
	if err != nil {
		return sel, models.ErrInvalid
	}
	return sel, nil
}

func (w HierarchyEditor) Edit(ctx context.Context, cmd HierarchyEdit) (HierarchyEditResult, error) {
	sel, err := cmd.selection()
	if err != nil {
		return HierarchyEditResult{}, err
	}
	p, err := w.Pools.PinnedPool(ctx, cmd.Tenant.String())
	if err != nil {
		return HierarchyEditResult{}, err
	}
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return HierarchyEditResult{}, err
	}
	defer tx.Rollback()
	// The same conditional write used by create serializes every hierarchy
	// mutation. Even an unchanged receipt must satisfy the caller's preconditions.
	lock, err := tx.ExecContext(ctx, p.Bind(`UPDATE tenant_gates SET hierarchy_version=hierarchy_version
 WHERE tenant_id=? AND state='active' AND placement_epoch=? AND hierarchy_version=?
 AND pool_id=(SELECT pool_id FROM edu_pool_marker WHERE singleton=1)`), cmd.Tenant.String(), cmd.Epoch, cmd.Version)
	if err != nil {
		return HierarchyEditResult{}, err
	}
	n, err := lock.RowsAffected()
	if err != nil {
		return HierarchyEditResult{}, err
	}
	if n != 1 {
		return HierarchyEditResult{}, models.ErrConflict
	}
	graph, err := directoryPath(ctx, tx, p, sel)
	if err != nil {
		return HierarchyEditResult{}, err
	}
	path, err := graph.Resolve(sel)
	if err != nil {
		return HierarchyEditResult{}, err
	}
	if !path.Active {
		return HierarchyEditResult{}, models.ErrScope
	}
	snapshot := graph.Snapshot()
	var before, after HierarchyMetadata
	var statement, eventType, id string
	var args []any
	// Construct the replacement from the persisted identity and owner. SQL changes
	// only metadata, preserving parentage and the node's own lifecycle state.
	switch cmd.Kind {
	case "organizations":
		current := snapshot.Organizations[len(snapshot.Organizations)-1]
		replacement, err := models.NewOrganization(current.ID(), current.Tenant(), current.Parent(), cmd.Metadata.Name)
		if err != nil {
			return HierarchyEditResult{}, err
		}
		before = HierarchyMetadata{Name: current.Name()}
		after = HierarchyMetadata{Name: replacement.Name()}
		id = current.ID().String()
		statement = "UPDATE tenancy_organizations SET name=? WHERE tenant_id=? AND id=?"
		args = []any{after.Name, cmd.Tenant.String(), id}
		eventType = "tenancy.organization.updated.v1"
	case "schools":
		current := snapshot.Schools[0]
		replacement, err := models.NewSchool(current.ID(), current.Tenant(), current.Organization(), cmd.Metadata.Code, cmd.Metadata.Name)
		if err != nil {
			return HierarchyEditResult{}, err
		}
		before = HierarchyMetadata{Name: current.Name(), Code: current.Code()}
		after = HierarchyMetadata{Name: replacement.Name(), Code: replacement.Code()}
		id = current.ID().String()
		statement = "UPDATE tenancy_schools SET name=?,code=? WHERE tenant_id=? AND id=?"
		args = []any{after.Name, after.Code, cmd.Tenant.String(), id}
		eventType = "tenancy.school.updated.v1"
	case "campuses":
		current := snapshot.Campuses[0]
		replacement, err := models.NewCampus(current.ID(), current.Tenant(), current.School(), cmd.Metadata.Code, cmd.Metadata.Name, cmd.Metadata.Timezone)
		if err != nil {
			return HierarchyEditResult{}, err
		}
		before = HierarchyMetadata{Name: current.Name(), Code: current.Code(), Timezone: current.Timezone()}
		after = HierarchyMetadata{Name: replacement.Name(), Code: replacement.Code(), Timezone: replacement.Timezone()}
		id = current.ID().String()
		statement = "UPDATE tenancy_campuses SET name=?,code=?,timezone=? WHERE tenant_id=? AND id=?"
		args = []any{after.Name, after.Code, after.Timezone, cmd.Tenant.String(), id}
		eventType = "tenancy.campus.updated.v1"
	}
	result := HierarchyEditResult{Changed: before != after, Entry: HierarchyEntry{ID: id, Name: after.Name, Code: after.Code, Timezone: after.Timezone, Active: true, Path: path}}
	if result.Changed {
		update, err := tx.ExecContext(ctx, p.Bind(statement), args...)
		if err != nil {
			return HierarchyEditResult{}, err
		}
		n, err := update.RowsAffected()
		if err != nil {
			return HierarchyEditResult{}, err
		}
		if n != 1 {
			return HierarchyEditResult{}, models.ErrConflict
		}
		result.Entry.Path.HierarchyVersion++
		if _, err := tx.ExecContext(ctx, p.Bind("UPDATE tenant_gates SET hierarchy_version=? WHERE tenant_id=?"), result.Entry.Path.HierarchyVersion, cmd.Tenant.String()); err != nil {
			return HierarchyEditResult{}, err
		}
		payload, err := json.Marshal(HierarchyMetadataUpdated{Before: before, After: after, Path: result.Entry.Path})
		if err != nil {
			return HierarchyEditResult{}, err
		}
		if err := outbox.Append(ctx, tx, p, outbox.Event{TenantID: cmd.Tenant.String(), Type: eventType, AggregateID: id, Payload: payload}); err != nil {
			return HierarchyEditResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return HierarchyEditResult{}, err
	}
	return result, nil
}
