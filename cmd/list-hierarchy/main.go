// Command list-hierarchy enumerates existing topology for a trusted local operator.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	"github.com/elcomware/edupilot/modules/tenancy/providers"
)

func main() {
	var query services.HierarchyQuery
	flag.TextVar(&query.Tenant, "tenant", scope.TenantID{}, "required existing tenant UUID")
	flag.StringVar(&query.Kind, "kind", "", "required organizations, schools or campuses")
	flag.TextVar(&query.Organization, "organization", scope.OrganizationID{}, "immediate parent organization for organizations or schools")
	flag.TextVar(&query.School, "school", scope.SchoolID{}, "immediate parent school for campuses")
	flag.IntVar(&query.Limit, "limit", 50, "page size from 1 through 100")
	flag.StringVar(&query.Cursor, "cursor", "", "continuation returned by the preceding page with the same tenant, kind and filter")
	flag.Uint64Var(&query.Version, "version", 0, "optional expected hierarchy version")
	flag.Parse()
	if query.Tenant.IsZero() || query.Kind == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := list(query); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func list(query services.HierarchyQuery) error {
	app := bootstrap.Boot()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	value, err := app.Make(database.CatalogBinding)
	if err != nil {
		return err
	}
	if err := value.(*database.Catalog).CheckReady(ctx); err != nil {
		return err
	}
	reader, err := providers.HierarchyReader(app)
	if err != nil {
		return err
	}
	page, err := reader.List(ctx, query)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(page)
}
