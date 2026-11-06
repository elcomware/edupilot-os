package outbox

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/elcomware/edupilot/platform/database"
)

// Consumer applies a local durable effect using the supplied transaction. This
// commit also acknowledges the event. External effects require their own durable
// inbox and idempotency keyed by Event.ID; they cannot be rolled back by this tx.
type Consumer interface {
	Consume(context.Context, *sql.Tx, database.Pool, Event) error
}

type AuditInbox struct{}

func (AuditInbox) Consume(ctx context.Context, tx *sql.Tx, pool database.Pool, event Event) error {
	_, err := tx.ExecContext(ctx, pool.Bind(`INSERT INTO edu_audit_inbox
		(event_id, tenant_id, event_type, aggregate_id, received_at)
		VALUES (?, ?, ?, ?, ?) ON CONFLICT (event_id) DO NOTHING`),
		event.ID, event.TenantID, event.Type, event.AggregateID, time.Now().UTC().UnixMilli())
	return err
}

type Dispatcher struct {
	Consumer Consumer
	Now      func() time.Time
}

const (
	leaseDuration = 30 * time.Second
	maxAttempts   = 8
)

var errLeaseLost = errors.New("outbox lease no longer belongs to this worker")

type delivery struct {
	event    Event
	lease    string
	attempts int
}

func (d Dispatcher) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d Dispatcher) claim(ctx context.Context, pool database.Pool) (delivery, error) {
	now := d.now()
	lock := ""
	if pool.Driver == "postgres" {
		lock = " FOR UPDATE SKIP LOCKED"
	}
	eligible := "published_at IS NULL AND dead_at IS NULL AND available_at <= ? AND (lease_until IS NULL OR lease_until <= ?)"
	query := `UPDATE edu_outbox SET lease_id = ?, lease_until = ?, attempts = attempts + 1
		WHERE event_id = (SELECT event_id FROM edu_outbox WHERE ` + eligible +
		" ORDER BY available_at, created_at, event_id LIMIT 1" + lock + ") AND " + eligible +
		" RETURNING event_id, tenant_id, event_type, aggregate_id, payload, attempts"
	item := delivery{lease: rand.Text()}
	var payload string
	err := pool.DB.QueryRowContext(ctx, pool.Bind(query), item.lease, now.Add(leaseDuration).UnixMilli(),
		now.UnixMilli(), now.UnixMilli(), now.UnixMilli(), now.UnixMilli()).Scan(
		&item.event.ID, &item.event.TenantID, &item.event.Type, &item.event.AggregateID, &payload, &item.attempts)
	item.event.Payload = json.RawMessage(payload)
	return item, err
}

func owned(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errLeaseLost
	}
	return nil
}

func (d Dispatcher) deliver(ctx context.Context, pool database.Pool, item delivery) error {
	tx, err := pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	handler := d.Consumer
	if handler == nil {
		handler = AuditInbox{}
	}
	if err := handler.Consume(ctx, tx, pool, item.event); err != nil {
		return err
	}
	err = owned(tx.ExecContext(ctx, pool.Bind(`UPDATE edu_outbox SET published_at = ?, lease_id = NULL,
		lease_until = NULL, last_error = NULL WHERE event_id = ? AND lease_id = ? AND published_at IS NULL`),
		d.now().UnixMilli(), item.event.ID, item.lease))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (d Dispatcher) reject(ctx context.Context, pool database.Pool, item delivery) error {
	now := d.now()
	var deadAt any
	if item.attempts >= maxAttempts {
		deadAt = now.UnixMilli()
	}
	delay := time.Second << min(max(item.attempts-1, 0), 10)
	// Persist no raw exception (it may contain SQL values, customer data or secrets).
	return owned(pool.DB.ExecContext(ctx, pool.Bind(`UPDATE edu_outbox SET
		available_at = ?, dead_at = ?, lease_id = NULL, lease_until = NULL, last_error = ?
		WHERE event_id = ? AND lease_id = ? AND published_at IS NULL`),
		now.Add(delay).UnixMilli(), deadAt, "delivery failed", item.event.ID, item.lease))
}

// Step consumes at most one event. False,nil means this pool has no eligible work.
// If the process stops after claiming or before commit, the lease expires and an
// independent worker retries. Failed deliveries back off, then remain inspectable.
func (d Dispatcher) Step(ctx context.Context, pool database.Pool) (bool, error) {
	item, err := d.claim(ctx, pool)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := d.deliver(ctx, pool, item); err != nil {
		if ctx.Err() != nil || errors.Is(err, errLeaseLost) {
			return true, err
		}
		retryErr := d.reject(ctx, pool, item)
		return true, errors.Join(fmt.Errorf("deliver event %s (attempt %d): %w", item.event.ID, item.attempts, err), retryErr)
	}
	return true, nil
}
