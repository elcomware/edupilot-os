// Command add-organization adds one organization to an initialized tenant.
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
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
	"github.com/elcomware/edupilot/modules/tenancy/providers"
)

func main() {
	var tenant scope.TenantID
	var id, parent scope.OrganizationID
	flag.TextVar(&tenant, "tenant", scope.TenantID{}, "existing tenant UUID")
	flag.TextVar(&id, "organization", scope.OrganizationID{}, "new organization UUID")
	flag.TextVar(&parent, "parent", scope.OrganizationID{}, "optional parent organization UUID")
	name := flag.String("name", "", "organization name")
	epoch := flag.Uint64("epoch", 0, "expected placement epoch")
	version := flag.Uint64("version", 0, "expected hierarchy version")
	flag.Parse()
	org, err := models.NewOrganization(id, tenant, parent, *name)
	if err != nil || *epoch == 0 || *version == 0 || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err = run(org, *epoch, *version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(org models.Organization, epoch, version uint64) error {
	app := bootstrap.Boot()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	catalog, err := app.Make(database.CatalogBinding)
	if err != nil {
		return err
	}
	if err = catalog.(*database.Catalog).CheckReady(ctx); err != nil {
		return err
	}
	writer, err := providers.OrganizationWriter(app)
	if err != nil {
		return err
	}
	chain, err := writer.Create(ctx, org, epoch, version)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(chain)
}
