// Command add-campus adds one campus to an initialized tenant.
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
	var id scope.CampusID
	var parent scope.SchoolID
	flag.TextVar(&tenant, "tenant", scope.TenantID{}, "existing tenant UUID")
	flag.TextVar(&id, "campus", scope.CampusID{}, "new campus UUID")
	flag.TextVar(&parent, "school", scope.SchoolID{}, "existing parent school UUID")
	code := flag.String("code", "", "campus code (1-64 letters, digits, periods, underscores or hyphens, starting with a letter or digit)")
	name := flag.String("name", "", "campus name (1-200 characters, without control characters)")
	timezone := flag.String("timezone", "", "IANA timezone, for example Africa/Nairobi")
	epoch := flag.Uint64("epoch", 0, "expected placement epoch")
	version := flag.Uint64("version", 0, "expected hierarchy version")
	flag.Parse()
	campus, err := models.NewCampus(id, tenant, parent, *code, *name, *timezone)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid campus: supply tenant, school and campus UUIDs, a valid code and a name", "and a valid IANA timezone")
		flag.Usage()
		os.Exit(2)
	}
	if *epoch == 0 || *epoch > math.MaxInt64 || *version == 0 || *version >= math.MaxInt64 || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "supply -epoch between 1 and 9223372036854775807 and -version between 1 and 9223372036854775806, with no positional arguments")
		flag.Usage()
		os.Exit(2)
	}
	if err = run(campus, *epoch, *version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(campus models.Campus, epoch, version uint64) error {
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
	writer, err := providers.CampusWriter(app)
	if err != nil {
		return err
	}
	chain, err := writer.Create(ctx, campus, epoch, version)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(chain)
}
