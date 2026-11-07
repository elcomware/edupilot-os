// Command enroll-dedicated-admin enrolls the initial admin after storage activation.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/app/provisioning"
	"github.com/elcomware/edupilot/bootstrap"
	auth "github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func main() {
	// Flag diagnostics echo unknown arguments. Suppress them because an operator
	// might mistakenly paste a credential into an unsupported flag or argument.
	flags := flag.NewFlagSet("enroll-dedicated-admin", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var tenant scope.TenantID
	flags.TextVar(&tenant, "tenant", scope.TenantID{}, "canonical tenant UUID")
	pool := flags.String("pool-id", "", "expected dedicated physical pool ID")
	operation := flags.String("operation", "", "original activation operation UUID")
	if err := flags.Parse(os.Args[1:]); err != nil || flags.NArg() != 0 || tenant.IsZero() || *pool == "" || *operation == "" {
		fmt.Fprintln(os.Stderr, "Usage: enroll-dedicated-admin -tenant UUID -pool-id ID -operation UUID. Supply EDUPILOT_SEED_EMAIL and EDUPILOT_SEED_PASSWORD in the environment.")
		os.Exit(2)
	}
	if err := run(tenant, *pool, *operation); err != nil {
		fmt.Fprintln(os.Stderr, "enroll-dedicated-admin:", safeMessage(err))
		os.Exit(1)
	}
}

func safeMessage(err error) string {
	for _, known := range []error{auth.ErrInvalidInput, auth.ErrConflict, scope.ErrInvalidID, database.ErrPlacement, context.DeadlineExceeded} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	return "enrollment failed; check activation, account state and connection settings"
}
func env(key string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return facades.Config().EnvString(key, "")
}
func run(tenant scope.TenantID, pool, operation string) error {
	bootstrap.Boot()
	email, password := env("EDUPILOT_SEED_EMAIL"), env("EDUPILOT_SEED_PASSWORD")
	if email == "" || password == "" {
		return auth.ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := database.OpenConfigured()
	if err != nil {
		return err
	}
	receipt, err := provisioning.EnrollDedicatedAdmin(ctx, catalog, tenant, pool, operation, email, password)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(receipt)
}
