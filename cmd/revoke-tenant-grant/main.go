// Command revoke-tenant-grant revokes authority using only control storage.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pterm/pterm"
	"io"
	"os"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/database/edumigrations"
	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func main() {
	// Goravel ORM initialization can emit raw connection diagnostics. These
	// machine-readable commands expose only their sanitized errors and receipts.
	pterm.DisableOutput()
	flags := flag.NewFlagSet("revoke-tenant-grant", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	grant := flags.String("grant", "", "canonical grant UUID")
	version := flags.Uint64("version", 0, "expected active grant version")
	err := flags.Parse(os.Args[1:])
	_, idErr := scope.Parse[scope.Membership](*grant)
	if err != nil || idErr != nil || flags.NArg() != 0 || *version == 0 {
		fmt.Fprintln(os.Stderr, "Usage: revoke-tenant-grant -grant UUID -version N")
		os.Exit(2)
	}
	if err := run(*grant, *version); err != nil {
		fmt.Fprintln(os.Stderr, "revoke-tenant-grant: revocation failed; check grant version and control installation")
		os.Exit(1)
	}
}
func run(grant string, version uint64) error {
	bootstrap.Boot()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	settings, err := database.ConfiguredSettings()
	if err != nil {
		return err
	}
	// Resolve only the control handle. Opening the full catalog may connect to an
	// unavailable dedicated target before authority can be revoked.
	db, err := facades.Orm().Connection(settings.Control.Name).DB()
	if err != nil {
		return err
	}
	pool := database.Pool{DB: db, Driver: settings.Control.Driver}
	if err := edumigrations.CheckControlReady(ctx, pool, facades.Schema().Migrations()); err != nil {
		return err
	}
	receipt, err := auth.New(&pool).RevokeTenantGrant(ctx, grant, version)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(receipt)
}
