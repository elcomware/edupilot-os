package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/schema"
)

// Activations retains the immutable completion receipt after reservation removal.
func Activations(db migration.Schema) error {
	return db.Create("edu_tenant_activations", func(t schema.Blueprint) {
		t.String("tenant_id", 64)
		t.Primary("tenant_id")
		t.String("operation_id", 36)
		t.Unique("operation_id")
		t.String("pool_id", 64)
		t.Unique("pool_id")
		t.Text("payload")
		t.Foreign("tenant_id").References("tenant_id").On("edu_tenant_placement")
	})
}
