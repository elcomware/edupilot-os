package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/schema"
)

// Topology keeps organization, school and campus relationships within one tenant.
func Topology(db migration.Schema) error {
	if err := db.Create("tenant_gates", func(t schema.Blueprint) {
		t.String("tenant_id", 36)
		t.Primary("tenant_id")
		t.String("pool_id", 36)
		t.UnsignedBigInteger("placement_epoch")
		t.UnsignedBigInteger("hierarchy_version")
		t.Enum("state", []any{"provisioning", "active", "maintenance", "suspended", "retired"})
	}); err != nil {
		return err
	}
	if err := db.Create("tenancy_organizations", func(t schema.Blueprint) {
		t.String("tenant_id", 36)
		t.String("id", 36)
		t.String("parent_id", 36).Nullable()
		t.Text("name")
		t.Boolean("active")
		t.Primary("tenant_id", "id")
		t.Foreign("tenant_id").References("tenant_id").On("tenant_gates")
		t.Foreign("tenant_id", "parent_id").References("tenant_id", "id").On("tenancy_organizations")
	}); err != nil {
		return err
	}
	if err := db.Create("tenancy_schools", func(t schema.Blueprint) {
		t.String("tenant_id", 36)
		t.String("id", 36)
		t.String("organization_id", 36)
		t.String("code", 64)
		t.Text("name")
		t.Boolean("active")
		t.Primary("tenant_id", "id")
		t.Unique("tenant_id", "organization_id", "code")
		t.Foreign("tenant_id", "organization_id").References("tenant_id", "id").On("tenancy_organizations")
	}); err != nil {
		return err
	}
	return db.Create("tenancy_campuses", func(t schema.Blueprint) {
		t.String("tenant_id", 36)
		t.String("id", 36)
		t.String("school_id", 36)
		t.String("code", 64)
		t.Text("name")
		t.String("timezone")
		t.Boolean("active")
		t.Primary("tenant_id", "id")
		t.Unique("tenant_id", "school_id", "code")
		t.Foreign("tenant_id", "school_id").References("tenant_id", "id").On("tenancy_schools")
	})
}
