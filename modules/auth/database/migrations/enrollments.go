package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/schema"
)

// AdminEnrollments retains the original result even if the principal is deleted.
func AdminEnrollments(db migration.Schema) error {
	return db.Create("auth_admin_enrollments", func(t schema.Blueprint) {
		t.String("school_id", 64)
		t.Primary("school_id")
		t.String("operation_id", 36)
		t.Unique("operation_id")
		t.Text("payload")
		t.String("state_digest", 64)
		t.Foreign("school_id").References("school_id").On("auth_realms")
	})
}
