package services

import (
	"context"

	"github.com/elcomware/edupilot/modules/core/contracts/tenancy"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
)

// TopologyStore loads one tenant snapshot. Implementations must reject a stale
// expected version atomically when persisting its successor.
type TopologyStore interface {
	TopologyReader
	Create(ctx context.Context, graph models.Graph, expectedVersion uint64) error
}
type TopologyReader interface {
	Load(ctx context.Context, tenant scope.TenantID) (models.Graph, error)
}
type Topology struct{ Store TopologyReader }

var _ contracts.Directory = Topology{}

func (service Topology) Resolve(ctx context.Context, sel scope.Selection) (scope.Chain, error) {
	if sel.Tenant.IsZero() {
		return scope.Chain{}, models.ErrScope
	}
	graph, err := service.Store.Load(ctx, sel.Tenant)
	if err != nil {
		return scope.Chain{}, err
	}
	return graph.Resolve(sel)
}
