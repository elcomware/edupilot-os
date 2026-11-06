// Command migrate applies the registered Goravel migrations and exits unsuccessfully
// if either a migration or the post-migration readiness check fails.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/core/database"
)

func main() {
	bootstrap.Boot()
	catalog, err := database.OpenConfigured()
	if err == nil {
		err = edumigrations.Run(context.Background(), catalog)
	}
	if err != nil {
		facades.Log().Errorf("migration failed: %v", err)
		fmt.Fprintln(os.Stderr, "migration failed:", err)
		os.Exit(1)
	}
	facades.Log().Info("all configured EduPilot databases are ready")
}
