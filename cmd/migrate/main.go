// Command migrate applies the registered Goravel migrations and exits unsuccessfully
// if either a migration or the post-migration readiness check fails.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/platform/database"
)

func main() {
	bootstrap.Boot()
	catalog, err := database.OpenConfigured()
	if err == nil {
		err = edumigrations.Run(context.Background(), catalog)
	}
	if err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("all configured EduPilot databases are ready")
}
