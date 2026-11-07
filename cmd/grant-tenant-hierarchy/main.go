// Command grant-tenant-hierarchy explicitly authorizes one realm-local subject.
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

	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/bootstrap"
	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func main() {
	// Goravel ORM initialization can emit raw connection diagnostics. These
	// machine-readable commands expose only their sanitized errors and receipts.
	pterm.DisableOutput()
	flags := flag.NewFlagSet("grant-tenant-hierarchy", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	in := auth.TenantGrantInput{Permission: access.HierarchyRead}
	flags.StringVar(&in.ID, "grant", "", "new canonical grant UUID retained for retry")
	flags.StringVar(&in.Subject.Realm, "realm", "", "existing credential realm")
	flags.StringVar(&in.Subject.ID, "user", "", "existing user ID in that realm")
	flags.TextVar(&in.Tenant, "tenant", scope.TenantID{}, "target tenant UUID")
	flags.Int64Var(&in.ExpiresAt, "expires-at", 0, "required UTC epoch-seconds expiry")
	pool := flags.String("pool-id", "", "expected physical pool ID")
	epoch := flags.Uint64("epoch", 0, "expected placement epoch")
	if err := flags.Parse(os.Args[1:]); err != nil || flags.NArg() != 0 || in.Validate() != nil || *pool == "" || *epoch == 0 {
		fmt.Fprintln(os.Stderr, "Usage: grant-tenant-hierarchy -realm REALM -user ID -tenant UUID -grant UUID -expires-at SECONDS -pool-id ID -epoch N")
		os.Exit(2)
	}
	if err := run(in, *pool, *epoch); err != nil {
		fmt.Fprintln(os.Stderr, "grant-tenant-hierarchy: grant failed; check current subject, assignment, expiry and physical placement")
		os.Exit(1)
	}
}
func run(in auth.TenantGrantInput, pool string, epoch uint64) error {
	bootstrap.Boot()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := database.OpenConfigured()
	if err != nil {
		return err
	}
	receipt, err := provisioning.GrantTenantHierarchy(ctx, catalog, in, pool, epoch)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(receipt)
}
