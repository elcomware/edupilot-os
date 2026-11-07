package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/schema"
)

// TenantGrants is control-only. Terminal rows retain assignment identity.
func TenantGrants(db migration.Schema) error {
	return db.Create("auth_tenant_grants", func(t schema.Blueprint) {
		t.String("id", 36)
		t.Primary("id")
		t.String("realm_id", 64)
		t.String("user_id", 32)
		t.String("tenant_id", 36)
		t.String("permission", 100)
		t.String("state", 16)
		t.BigInteger("version")
		t.BigInteger("expires_at")
		t.BigInteger("created_at")
		t.BigInteger("revoked_at").Nullable()
		t.Unique("realm_id", "user_id", "tenant_id", "permission")
		t.Foreign("realm_id", "user_id").References("school_id", "user_id").On("auth_memberships").CascadeOnDelete()
		t.Foreign("tenant_id").References("tenant_id").On("edu_tenant_placement")
	})
}
