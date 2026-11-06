package application

import (
	"context"

	"github.com/elcomware/edupilot-os/modules/tenancy/contracts"
	"github.com/elcomware/edupilot-os/modules/tenancy/internal/domain"
	"github.com/elcomware/edupilot-os/platform/scope"
)

// TopologyStore loads one tenant snapshot. Implementations must reject a stale
// expected version atomically when persisting its successor.
type TopologyStore interface {
	TopologyReader
	Create(ctx context.Context, graph domain.Graph, expectedVersion uint64) error
}
type TopologyReader interface {
	Load(ctx context.Context, tenant scope.TenantID) (domain.Graph, error)
}
type Topology struct{ Store TopologyReader }

var _ contracts.Directory = Topology{}

func (service Topology) Resolve(ctx context.Context, sel scope.Selection) (scope.Chain, error) {
	if sel.Tenant.IsZero() {
		return scope.Chain{}, domain.ErrScope
	}
	graph, err := service.Store.Load(ctx, sel.Tenant)
	if err != nil {
		return scope.Chain{}, err
	}
	return graph.Resolve(sel)
}
