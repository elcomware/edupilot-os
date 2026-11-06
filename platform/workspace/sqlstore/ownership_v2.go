package sqlstore

import "github.com/elcomware/edupilot-os/database/edumigrations"

// OwnershipPlan appends a new schema beside the module's starter work list.
func OwnershipPlan(module string) edumigrations.Plan {
	return edumigrations.Plan{ID: "20261106010008_workspace_" + module, ForwardOnly: true, Up: []string{
		"CREATE TABLE " + tableName(module) + "_v2" + ` (
 tenant_id TEXT NOT NULL,school_id TEXT NOT NULL,id TEXT NOT NULL CHECK(length(id)>0),
 title TEXT NOT NULL,status TEXT NOT NULL CHECK(status IN ('open','closed')),version BIGINT NOT NULL CHECK(version>0 AND version=CAST(version AS BIGINT)),
 PRIMARY KEY(tenant_id,school_id,id),FOREIGN KEY(tenant_id,school_id) REFERENCES tenancy_schools(tenant_id,id)
 )`,
	}}
}
