// Command worker drains local transactional outboxes across the entire configured catalog.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/bootstrap"
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/outbox"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("outbox worker stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	bootstrap.Boot()
	catalog, err := database.OpenConfigured()
	if err != nil {
		return err
	}
	start, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := edumigrations.CheckReady(start, catalog, facades.Schema().Migrations()); err != nil {
		return err
	}
	pools := catalog.All()
	dispatcher := outbox.Dispatcher{Consumer: outbox.AuditInbox{}}
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	slog.Info("outbox worker ready", "pools", len(pools))
	for ctx.Err() == nil {
		worked := false
		// Round robin: a busy or failing school must not starve the others.
		for i, pool := range pools {
			attempt, stop := context.WithTimeout(ctx, 20*time.Second)
			found, err := dispatcher.Step(attempt, pool)
			stop()
			if err != nil && ctx.Err() == nil {
				slog.Error("outbox delivery", "pool", i, "error", err)
			}
			worked = worked || found
		}
		if worked {
			continue
		}
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}
	return ctx.Err()
}
