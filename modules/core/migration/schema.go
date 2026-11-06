// Package migration defines module schema callbacks using Goravel's Blueprint.
package migration

import "github.com/goravel/framework/contracts/database/schema"

type Schema interface {
	Create(string, func(schema.Blueprint)) error
}

func Workspace(db Schema, table string) error {
	return db.Create(table, func(t schema.Blueprint) {
		t.String("school_id", 64)
		t.String("id", 64)
		t.Text("title")
		t.Enum("status", []any{"open", "closed"})
		t.UnsignedBigInteger("version")
		t.Primary("school_id", "id")
	})
}
func Outbox(db Schema) error {
	if err := db.Create("edu_outbox", func(t schema.Blueprint) {
		t.String("event_id")
		t.Primary("event_id")
		t.String("tenant_id")
		t.String("event_type")
		t.String("aggregate_id")
		t.Text("payload")
		t.BigInteger("created_at")
		t.BigInteger("available_at")
		t.Integer("attempts").Default(0)
		t.String("lease_id").Nullable()
		t.BigInteger("lease_until").Nullable()
		t.BigInteger("published_at").Nullable()
		t.BigInteger("dead_at").Nullable()
		t.Text("last_error").Nullable()
		t.Index("available_at", "created_at", "event_id")
	}); err != nil {
		return err
	}
	return db.Create("edu_audit_inbox", func(t schema.Blueprint) {
		t.String("event_id")
		t.Primary("event_id")
		t.String("tenant_id")
		t.String("event_type")
		t.String("aggregate_id")
		t.BigInteger("received_at")
		t.Index("tenant_id", "received_at")
	})
}
