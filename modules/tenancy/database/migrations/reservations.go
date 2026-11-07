package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/schema"
)

// Reservations is an additive control-only migration. Empty placement pins
// keep pending tenants unroutable even by binaries predating this journal.
func Reservations(db migration.Schema) error {
	return db.Create("edu_tenant_reservations", func(t schema.Blueprint) {
		t.String("tenant_id", 64)
		t.Primary("tenant_id")
		t.String("operation_id", 36)
		t.Unique("operation_id")
		t.String("pool_id", 64)
		t.Unique("pool_id")
		t.String("created_at", 64)
		t.Foreign("tenant_id").References("tenant_id").On("edu_tenant_placement")
	})
}
