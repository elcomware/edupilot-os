// Command activate-dedicated-tenant activates a prepared dedicated placement.
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
	operation := flag.String("operation", "", "caller-generated UUID retained for retry")
	flag.Parse()
	if tenant.IsZero() || *pool == "" || *operation == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(tenant, *pool, *operation); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(tenant scope.TenantID, pool, operation string) error {
	app := bootstrap.Boot()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	value, err := app.Make(database.CatalogBinding)
	if err != nil {
		return err
	}
	catalog := value.(*database.Catalog)
	report, err := provisioning.ActivateDedicatedTenant(ctx, catalog, tenant, pool, operation)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(report)
}
