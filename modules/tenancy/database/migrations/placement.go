package migrations

import (
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/goravel/framework/contracts/database/schema"
)

func Placement(db migration.Schema, control bool) error {
	if err := db.Create("edu_pool_marker", func(t schema.Blueprint) {
		t.Integer("singleton")
		t.Primary("singleton")
		t.String("owner", 64)
		t.String("pool_id", 64)
		t.Unique("pool_id")
	}); err != nil {
		return err
	}
	if control {
		return db.Create("edu_tenant_placement", func(t schema.Blueprint) { t.String("tenant_id", 64); t.Primary("tenant_id"); t.String("pool_id", 64) })
	}
	return nil
}
