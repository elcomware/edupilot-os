package feature

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/google/uuid"
)

// Pause the real dispatcher transaction after its audit INSERT and before its
// outbox UPDATE. The channel controls the interleaving, and pg_locks confirms
// enrollment actually reached the conflicting table fence before release.
type pausedEnrollmentAudit struct {
	inserted chan outbox.Event
	release  chan struct{}
}

func (p pausedEnrollmentAudit) Consume(ctx context.Context, tx *sql.Tx, pool database.Pool, event outbox.Event) error {
	if err := (outbox.AuditInbox{}).Consume(ctx, tx, pool, event); err != nil {
		return err
	}
	p.inserted <- event
	select {
	case <-p.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// The returned function must run after enrollment starts. SQLite exercises the
// same enrollment callers without the PostgreSQL-specific table lock fixture.
func pauseEnrollmentWorker(t *testing.T, pool database.Pool, owner string) func() {
	t.Helper()
	if pool.Driver != "postgres" {
		return func() {}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)
	eventID := uuid.NewString()
	if _, err := pool.DB.ExecContext(ctx, pool.Bind(`INSERT INTO edu_outbox (event_id,tenant_id,event_type,aggregate_id,payload,created_at,available_at)
 VALUES (?,?,'fixture.enrollment-lock.v1',?,'{}',0,-1)`), eventID, owner, owner); err != nil {
		t.Fatal(err)
	}
	paused := pausedEnrollmentAudit{inserted: make(chan outbox.Event, 1), release: make(chan struct{})}
	// Cancellation on any early failure rolls back the worker transaction.
	delivery := make(chan error, 1)
	go func() {
		done, err := (outbox.Dispatcher{Consumer: paused}).Step(ctx, pool)
		if err == nil && !done {
			err = fmt.Errorf("worker did not claim the fence fixture event")
		}
		delivery <- err
	}()
	select {
	case event := <-paused.inserted:
		if event.ID != eventID {
			t.Fatal("worker selected an unexpected fixture event")
		}
	case err := <-delivery:
		t.Fatal("worker failed before the audit barrier", err)
	case <-ctx.Done():
		t.Fatal("worker did not reach the audit barrier", ctx.Err())
	}
	return func() {
		t.Helper()
		var fencePID int
		deadline := time.Now().Add(5 * time.Second)
		for {
			err := pool.DB.QueryRowContext(ctx, `SELECT locks.pid FROM pg_locks locks JOIN pg_stat_activity activity ON activity.pid=locks.pid
 WHERE activity.datname=current_database() AND locks.relation='edu_audit_inbox'::regclass
 AND locks.mode='ShareRowExclusiveLock' AND NOT locks.granted
 AND activity.query LIKE 'LOCK TABLE %' LIMIT 1`).Scan(&fencePID)
			if err == nil {
				break
			}
			if err != sql.ErrNoRows {
				t.Fatal("inspect enrollment audit wait", err)
			}
			if time.Now().After(deadline) {
				t.Fatal("enrollment never waited for the paused worker audit lock")
			}
			time.Sleep(10 * time.Millisecond)
		}
		var ownsOutbox bool
		if err := pool.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_locks WHERE pid=$1
 AND relation='edu_outbox'::regclass AND mode='ShareRowExclusiveLock' AND granted)`, fencePID).Scan(&ownsOutbox); err != nil {
			t.Fatal(err)
		}
		if ownsOutbox {
			t.Error("enrollment holds outbox while waiting for audit, reversing worker lock order")
		}
		close(paused.release)
		select {
		case err := <-delivery:
			if err != nil {
				t.Fatal("paused worker delivery failed", err)
			}
		case <-ctx.Done():
			t.Fatal("paused worker delivery did not finish", ctx.Err())
		}
		var published bool
		var attempts int
		if err := pool.DB.QueryRowContext(ctx, pool.Bind("SELECT published_at IS NOT NULL,attempts FROM edu_outbox WHERE event_id=?"), eventID).Scan(&published, &attempts); err != nil {
			t.Fatal(err)
		}
		if !published || attempts != 1 {
			t.Fatalf("worker publication=%v attempts=%d, want a single successful attempt", published, attempts)
		}
		cancel()
	}
}
