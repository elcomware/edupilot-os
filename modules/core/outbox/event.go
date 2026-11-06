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

	"github.com/elcomware/edupilot/modules/core/database"
)

type Event struct {
	ID          string
	TenantID    string
	Type        string
	AggregateID string
	Payload     json.RawMessage
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
