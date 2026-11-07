// Command add-school adds one school to an initialized tenant.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
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
	var id scope.SchoolID
	var parent scope.OrganizationID
	flag.TextVar(&tenant, "tenant", scope.TenantID{}, "existing tenant UUID")
	flag.TextVar(&id, "school", scope.SchoolID{}, "new school UUID")
	flag.TextVar(&parent, "organization", scope.OrganizationID{}, "existing parent organization UUID")
	code := flag.String("code", "", "school code (1-64 letters, digits, periods, underscores or hyphens, starting with a letter or digit)")
	name := flag.String("name", "", "school name (1-200 characters, without control characters)")
	epoch := flag.Uint64("epoch", 0, "expected placement epoch")
	version := flag.Uint64("version", 0, "expected hierarchy version")
	flag.Parse()
	school, err := models.NewSchool(id, tenant, parent, *code, *name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid school: supply tenant, organization and school UUIDs, a valid code and a name")
		flag.Usage()
		os.Exit(2)
	}
	if *epoch == 0 || *epoch > math.MaxInt64 || *version == 0 || *version >= math.MaxInt64 || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "supply -epoch between 1 and 9223372036854775807 and -version between 1 and 9223372036854775806, with no positional arguments")
		flag.Usage()
		os.Exit(2)
	}
	if err = run(school, *epoch, *version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(school models.School, epoch, version uint64) error {
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
	writer, err := providers.SchoolWriter(app)
	if err != nil {
		return err
	}
	chain, err := writer.Create(ctx, school, epoch, version)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(chain)
}
