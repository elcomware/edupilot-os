package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/schema"
)

// Accounts creates school-scoped users, memberships and revocable bearer sessions.
func Accounts(db migration.Schema) error {
	if err := db.Create("auth_realms", func(t schema.Blueprint) { t.String("school_id", 64); t.Primary("school_id") }); err != nil {
		return err
	}
	if err := db.Create("auth_roles", func(t schema.Blueprint) {
		t.String("school_id", 64)
		t.String("name", 64)
		t.Primary("school_id", "name")
		t.Foreign("school_id").References("school_id").On("auth_realms")
	}); err != nil {
		return err
	}
	if err := db.Create("auth_permissions", func(t schema.Blueprint) {
		t.String("school_id", 64)
		t.String("name", 100)
		t.Primary("school_id", "name")
		t.Foreign("school_id").References("school_id").On("auth_realms")
	}); err != nil {
		return err
	}
	if err := db.Create("auth_role_permissions", func(t schema.Blueprint) {
		t.String("school_id", 64)
		t.String("role", 64)
		t.String("permission", 100)
		t.Primary("school_id", "role", "permission")
		t.Foreign("school_id", "role").References("school_id", "name").On("auth_roles")
		t.Foreign("school_id", "permission").References("school_id", "name").On("auth_permissions")
	}); err != nil {
		return err
	}
	if err := db.Create("auth_users", func(t schema.Blueprint) {
		t.String("school_id", 64)
		t.String("id", 32)
		t.String("email", 254)
		t.Text("password_hash")
		t.Primary("school_id", "id")
		t.Unique("school_id", "email")
		t.Foreign("school_id").References("school_id").On("auth_realms")
	}); err != nil {
		return err
	}
	if err := db.Create("auth_memberships", func(t schema.Blueprint) {
		t.String("school_id", 64)
		t.String("user_id", 32)
		t.String("role", 64)
		t.Foreign("school_id", "role").References("school_id", "name").On("auth_roles")
		t.Primary("school_id", "user_id")
		t.Foreign("school_id", "user_id").References("school_id", "id").On("auth_users").CascadeOnDelete()
	}); err != nil {
		return err
	}
	return db.Create("auth_sessions", func(t schema.Blueprint) {
		t.String("token_hash", 64)
		t.String("school_id", 64)
		t.String("user_id", 32)
		t.BigInteger("expires_at")
		t.Primary("token_hash")
		t.Index("expires_at")
		t.Index("school_id", "user_id")
		t.Foreign("school_id", "user_id").References("school_id", "user_id").On("auth_memberships").CascadeOnDelete()
	})
}
