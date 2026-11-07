package providers

import (
	"github.com/elcomware/edupilot/modules/core/contracts/modulekit"
	contracts "github.com/elcomware/edupilot/modules/core/contracts/tenancy"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/workspace"
	"github.com/elcomware/edupilot/modules/core/workspace/sqlstore"
	"github.com/elcomware/edupilot/modules/tenancy"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	"github.com/goravel/framework/contracts/foundation"
)

type TenancyServiceProvider struct{}
type organizationWriterBinding struct{}

func OrganizationWriter(app foundation.Application) (services.OrganizationWriter, error) {
	value, err := app.Make(organizationWriterBinding{})
	if err != nil {
		return services.OrganizationWriter{}, err
	}
	return value.(services.OrganizationWriter), nil
}

type schoolWriterBinding struct{}

func SchoolWriter(app foundation.Application) (services.SchoolWriter, error) {
	value, err := app.Make(schoolWriterBinding{})
	if err != nil {
		return services.SchoolWriter{}, err
	}
	return value.(services.SchoolWriter), nil
}

type campusWriterBinding struct{}

func CampusWriter(app foundation.Application) (services.CampusWriter, error) {
	value, err := app.Make(campusWriterBinding{})
	if err != nil {
		return services.CampusWriter{}, err
	}
	return value.(services.CampusWriter), nil
}

type hierarchyReaderBinding struct{}

func HierarchyReader(app foundation.Application) (services.HierarchyReader, error) {
	value, err := app.Make(hierarchyReaderBinding{})
	if err != nil {
		return services.HierarchyReader{}, err
	}
	return value.(services.HierarchyReader), nil
}

func (*TenancyServiceProvider) Boot(foundation.Application) {}
func (*TenancyServiceProvider) Register(app foundation.Application) {
	app.Singleton(hierarchyReaderBinding{}, func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return services.HierarchyReader{Pools: pools.(*database.Catalog)}, nil
	})
	app.Singleton(organizationWriterBinding{}, func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return services.OrganizationWriter{Pools: pools.(*database.Catalog)}, nil
	})
	app.Singleton(schoolWriterBinding{}, func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return services.SchoolWriter{Pools: pools.(*database.Catalog)}, nil
	})
	app.Singleton(campusWriterBinding{}, func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return services.CampusWriter{Pools: pools.(*database.Catalog)}, nil
	})
	app.Singleton(contracts.DirectoryBinding, func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return services.PersistedDirectory{Pools: pools.(*database.Catalog)}, nil
	})
	app.Singleton(modulekit.WorkspaceBinding(tenancy.Key), func(app foundation.Application) (any, error) {
		pools, err := app.Make(database.CatalogBinding)
		if err != nil {
			return nil, err
		}
		return workspace.Service{Store: sqlstore.New(pools.(*database.Catalog), tenancy.Key)}, nil
	})
}
