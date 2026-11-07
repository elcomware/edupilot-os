// Command check-dedicated-tenant inspects a proposed dedicated tenant placement.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func main() {
	var tenant scope.TenantID
	flag.TextVar(&tenant, "tenant", scope.TenantID{}, "fresh tenant UUID")
	pool := flag.String("pool-id", "", "expected physical ID of the dedicated database")
	flag.Parse()
	if tenant.IsZero() || *pool == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(tenant, *pool); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(tenant scope.TenantID, pool string) error {
	app := bootstrap.Boot()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	value, err := app.Make(database.CatalogBinding)
	if err != nil {
		return err
	}
	catalog := value.(*database.Catalog)
	report, err := provisioning.CheckDedicatedTenant(ctx, catalog, tenant, pool)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(report)
}
