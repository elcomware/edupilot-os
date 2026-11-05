// Package outbox provides atomic event recording and a durable local subscriber.
// Payloads must be narrow integration contracts; avoid secrets and unnecessary PII.
package outbox

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/elcomware/edupilot-os/platform/database"
)

type Event struct {
	ID          string
	TenantID    string
	Type        string
	AggregateID string
	Payload     json.RawMessage
}

var SchemaUp = []string{
	`CREATE TABLE IF NOT EXISTS edu_outbox (
		event_id TEXT PRIMARY KEY NOT NULL, tenant_id TEXT NOT NULL, event_type TEXT NOT NULL,
		aggregate_id TEXT NOT NULL, payload TEXT NOT NULL, created_at BIGINT NOT NULL,
		available_at BIGINT NOT NULL, attempts INTEGER NOT NULL DEFAULT 0,
		lease_id TEXT, lease_until BIGINT, published_at BIGINT, dead_at BIGINT, last_error TEXT
	)`,
	`CREATE INDEX IF NOT EXISTS idx_edu_outbox_ready ON edu_outbox(available_at, created_at, event_id)
		WHERE published_at IS NULL AND dead_at IS NULL`,
	`CREATE TABLE IF NOT EXISTS edu_audit_inbox (
		event_id TEXT PRIMARY KEY NOT NULL, tenant_id TEXT NOT NULL, event_type TEXT NOT NULL,
		aggregate_id TEXT NOT NULL, received_at BIGINT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_edu_audit_tenant ON edu_audit_inbox(tenant_id, received_at)`,
}
var SchemaDown = []string{
	"DROP TABLE IF EXISTS edu_audit_inbox", "DROP TABLE IF EXISTS edu_outbox",
}

// Append must be called inside the very same transaction that mutates the aggregate.
// A supplied ID enables stable event identity across retries; duplicate IDs fail loudly.
func Append(ctx context.Context, tx *sql.Tx, pool database.Pool, e Event) error {
	if tx == nil || !database.ValidTenant(e.TenantID) || strings.TrimSpace(e.Type) == "" || strings.TrimSpace(e.AggregateID) == "" {
		return errors.New("outbox event requires a transaction, tenant, type and aggregate")
	}
	if e.ID == "" {
		e.ID = rand.Text()
	}
	if !json.Valid(e.Payload) {
		return errors.New("outbox payload must be valid JSON")
	}
	now := time.Now().UTC().UnixMilli()
	_, err := tx.ExecContext(ctx, pool.Bind(`INSERT INTO edu_outbox
		(event_id, tenant_id, event_type, aggregate_id, payload, created_at, available_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`), e.ID, e.TenantID, e.Type, e.AggregateID, string(e.Payload), now, now)
	return err
}
