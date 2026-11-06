// Command worker drains local transactional outboxes across the entire configured catalog.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/database/edumigrations"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		facades.Log().Errorf("outbox worker stopped: %v", err)
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
	facades.Log().Infof("outbox worker ready with %d pools", len(pools))
	for ctx.Err() == nil {
		worked := false
		// Round robin: a busy or failing school must not starve the others.
		for i, pool := range pools {
			attempt, stop := context.WithTimeout(ctx, 20*time.Second)
			found, err := dispatcher.Step(attempt, pool)
			stop()
			if err != nil && ctx.Err() == nil {
				facades.Log().WithContext(ctx).Errorf("outbox delivery failed on pool %d: %v", i, err)
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
