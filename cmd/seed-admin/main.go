// Command seed-admin bootstraps a school or administers its non-admin memberships.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/auth/database/seeders"
	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
)

func main() {
	mode := flag.String("mode", "seed", "seed | create-member | set-role")
	flag.Parse()
	if *mode != "seed" && *mode != "create-member" && *mode != "set-role" {
		fmt.Fprintln(os.Stderr, "Invalid -mode: use seed, create-member or set-role.")
		os.Exit(2)
	}
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Supply credentials in environment variables, not arguments.")
		os.Exit(2)
	}
	bootstrap.Boot()
	if err := run(*mode); err != nil {
		fmt.Fprintln(os.Stderr, "seed-admin:", safeMessage(err))
		os.Exit(1)
	}
}

var errMissing = errors.New("set EDUPILOT_SEED_SCHOOL, EDUPILOT_SEED_EMAIL and EDUPILOT_SEED_PASSWORD")
var errMissingMember = errors.New("set EDUPILOT_MEMBER_EMAIL, EDUPILOT_MEMBER_PASSWORD and EDUPILOT_MEMBER_ROLE")
var errMissingRole = errors.New("set EDUPILOT_MEMBER_ID and EDUPILOT_MEMBER_ROLE")

func safeMessage(err error) string {
	for _, known := range []error{errMissing, errMissingMember, errMissingRole, services.ErrInvalidInput, services.ErrConflict, services.ErrInvalidCredentials, access.ErrForbidden, services.ErrNotFound, access.ErrUnauthenticated, context.DeadlineExceeded} {
		if errors.Is(err, known) {
			return known.Error()
		}
	}
	// Database errors may include a DSN; never interpolate raw errors here.
	return "database operation failed; check connection settings and run the migrations for all configured pools"
}

func env(key string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fmt.Sprint(facades.Config().Env(key, ""))
}

func run(mode string) error {
	school, email, password := env("EDUPILOT_SEED_SCHOOL"), env("EDUPILOT_SEED_EMAIL"), env("EDUPILOT_SEED_PASSWORD")
	if school == "" || email == "" || password == "" {
		return errMissing
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := database.OpenConfigured()
	if err != nil {
		return err
	}
	if err = edumigrations.CheckReady(ctx, catalog, facades.Schema().Migrations()); err != nil {
		return err
	}
	service := services.New(&catalog.Control)
	if mode == "seed" {
		p, err := seeders.Provision(ctx, catalog, school, email, password)
		if err != nil {
			return err
		}
		fmt.Printf("School %s: administrator %s is provisioned.\n", p.School, p.ID)
		return nil
	}
	// No seed fallback: member administration always requires an active school admin.
	session, err := service.Login(ctx, school, email, password)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		_ = service.Logout(cleanup, session.Token)
	}()
	if err = session.Principal.Require(school, access.ManageMembers); err != nil {
		return err
	}
	role := env("EDUPILOT_MEMBER_ROLE")
	switch mode {
	case "create-member":
		memberEmail, memberPassword := env("EDUPILOT_MEMBER_EMAIL"), env("EDUPILOT_MEMBER_PASSWORD")
		if memberEmail == "" || memberPassword == "" || role == "" {
			return errMissingMember
		}
		p, err := service.CreateMember(ctx, session.Token, school, memberEmail, memberPassword, role)
		if err != nil {
			return err
		}
		fmt.Printf("School %s: member %s created as %s.\n", school, p.ID, role)
	case "set-role":
		id := env("EDUPILOT_MEMBER_ID")
		if id == "" || role == "" {
			return errMissingRole
		}
		if err := service.SetMemberRole(ctx, session.Token, school, id, role); err != nil {
			return err
		}
		fmt.Printf("School %s: member %s now has role %s.\n", school, id, role)
	}
	return nil
}
