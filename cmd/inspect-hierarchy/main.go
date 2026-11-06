// Command inspect-hierarchy resolves an existing tenant path for a local operator.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/elcomware/edupilot/bootstrap"
	contracts "github.com/elcomware/edupilot/modules/core/contracts/tenancy"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func main() {
	var sel scope.Selection
	flag.TextVar(&sel.Tenant, "tenant", scope.TenantID{}, "required tenant UUID")
	flag.TextVar(&sel.Organization, "organization", scope.OrganizationID{}, "organization UUID")
	flag.TextVar(&sel.School, "school", scope.SchoolID{}, "school UUID")
	flag.TextVar(&sel.Campus, "campus", scope.CampusID{}, "campus UUID")
	flag.Parse()
	if sel.Tenant.IsZero() || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := inspect(sel); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func inspect(sel scope.Selection) error {
	app := bootstrap.Boot()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	value, err := app.Make(database.CatalogBinding)
	if err != nil {
		return err
	}
	if err = value.(*database.Catalog).CheckReady(ctx); err != nil {
		return err
	}
	directory, err := app.Make(contracts.DirectoryBinding)
	if err != nil {
		return err
	}
	chain, err := directory.(contracts.Directory).Resolve(ctx, sel)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(chain)
}
