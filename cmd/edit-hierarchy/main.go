// Command edit-hierarchy replaces metadata on one existing hierarchy record.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
	"github.com/elcomware/edupilot/modules/tenancy/providers"
)

func main() {
	cmd, err := parseEdit(os.Args[1:], os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := edit(cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseEdit(args []string, output io.Writer) (services.HierarchyEdit, error) {
	var cmd services.HierarchyEdit
	flags := flag.NewFlagSet("edit-hierarchy", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.TextVar(&cmd.Tenant, "tenant", scope.TenantID{}, "required existing tenant UUID")
	flags.StringVar(&cmd.Kind, "kind", "", "required organizations, schools or campuses")
	flags.StringVar(&cmd.ID, "id", "", "required existing record UUID")
	flags.StringVar(&cmd.Metadata.Name, "name", "", "required complete name")
	flags.StringVar(&cmd.Metadata.Code, "code", "", "required complete school or campus code")
	flags.StringVar(&cmd.Metadata.Timezone, "timezone", "", "required campus IANA timezone")
	flags.Uint64Var(&cmd.Epoch, "epoch", 0, "required expected placement epoch")
	flags.Uint64Var(&cmd.Version, "version", 0, "required expected hierarchy version")
	if err := flags.Parse(args); err != nil {
		return cmd, err
	}
	supplied := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) { supplied[f.Name] = true })
	for _, required := range []string{"tenant", "kind", "id", "name", "epoch", "version"} {
		if !supplied[required] {
			return cmd, fmt.Errorf("supply -%s", required)
		}
	}
	switch cmd.Kind {
	case "organizations":
		if supplied["code"] || supplied["timezone"] {
			return cmd, fmt.Errorf("organizations accept only -name metadata")
		}
	case "schools":
		if !supplied["code"] || supplied["timezone"] {
			return cmd, fmt.Errorf("schools require -name and -code metadata")
		}
	case "campuses":
		if !supplied["code"] || !supplied["timezone"] {
			return cmd, fmt.Errorf("campuses require -name, -code and -timezone metadata")
		}
	default:
		return cmd, fmt.Errorf("supply -kind organizations, schools or campuses")
	}
	if _, err := scope.Parse[scope.Organization](cmd.ID); err != nil {
		return cmd, fmt.Errorf("supply an existing record UUID with -id")
	}
	if cmd.Tenant.IsZero() || cmd.Epoch == 0 || cmd.Epoch > math.MaxInt64 || cmd.Version == 0 || cmd.Version >= math.MaxInt64 || flags.NArg() != 0 {
		return cmd, fmt.Errorf("supply a tenant UUID, -epoch between 1 and 9223372036854775807 and -version between 1 and 9223372036854775806, with no positional arguments")
	}
	return cmd, nil
}

func edit(cmd services.HierarchyEdit) error {
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
	editor, err := providers.HierarchyEditor(app)
	if err != nil {
		return err
	}
	result, err := editor.Edit(ctx, cmd)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
