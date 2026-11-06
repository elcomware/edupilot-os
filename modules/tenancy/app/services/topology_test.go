package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/elcomware/edupilot/modules/tenancy/app/models"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	"github.com/elcomware/edupilot/platform/scope"
)

type reader struct {
	graph models.Graph
	loads int
	err   error
}

func (r *reader) Load(ctx context.Context, tenant scope.TenantID) (models.Graph, error) {
	r.loads++
	if err := ctx.Err(); err != nil {
		return models.Graph{}, err
	}
	return r.graph, r.err
}
func TestDirectoryRejectsAbsentOrMismatchedTenant(t *testing.T) {
	tenant, err := scope.NewID[scope.Tenant]()
	if err != nil {
		t.Fatal(err)
	}
	other, err := scope.NewID[scope.Tenant]()
	if err != nil {
		t.Fatal(err)
	}
	graph, err := models.NewGraph(tenant)
	if err != nil {
		t.Fatal(err)
	}
	repository := &reader{graph: graph}
	service := services.Topology{Store: repository}
	if _, err = service.Resolve(t.Context(), scope.Selection{}); !errors.Is(err, scope.ErrUnknown) || repository.loads != 0 {
		t.Fatal(err, repository.loads)
	}
	if _, err = service.Resolve(t.Context(), scope.Selection{Tenant: other}); !errors.Is(err, scope.ErrUnknown) {
		t.Fatal("mismatched storage result accepted", err)
	}
	selected, err := service.Resolve(t.Context(), scope.Selection{Tenant: tenant})
	if err != nil || selected.Tenant != tenant || !selected.Active {
		t.Fatal(selected, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err = service.Resolve(ctx, scope.Selection{Tenant: tenant}); !errors.Is(err, context.Canceled) {
		t.Fatal("lost request cancellation", err)
	}
}
