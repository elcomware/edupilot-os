package seeders

import (
	"context"
	"errors"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/platform/access"
	"github.com/elcomware/edupilot/platform/database"
)

// SchoolAdminSeeder is opt-in. It never generates a default credential.
type SchoolAdminSeeder struct{}

func (*SchoolAdminSeeder) Signature() string { return "AuthSchoolAdminSeeder" }
func (*SchoolAdminSeeder) Run() error {
	config := facades.Config()
	school, email, password := config.EnvString("EDUPILOT_SEED_SCHOOL", ""), config.EnvString("EDUPILOT_SEED_EMAIL", ""), config.EnvString("EDUPILOT_SEED_PASSWORD", "")
	if school == "" || email == "" || password == "" {
		return errors.New("school, administrator email and password are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := database.OpenConfigured()
	if err != nil {
		return errors.New("school database configuration unavailable")
	}
	if err = edumigrations.CheckReady(ctx, catalog, facades.Schema().Migrations()); err != nil {
		return errors.New("school migrations are incomplete")
	}
	if _, err = Provision(ctx, catalog, school, email, password); err != nil {
		return errors.New("school administrator provisioning failed")
	}
	return nil
}

func Provision(ctx context.Context, catalog *database.Catalog, school, email, password string) (access.Principal, error) {
	p, err := services.New(&catalog.Control).Seed(ctx, school, email, password)
	if err != nil {
		return access.Principal{}, err
	}
	if _, err = catalog.ForTenantContext(ctx, p.School); err != nil {
		return access.Principal{}, err
	}
	return p, nil
}
