package migrations

// TopologyDDL runs on each data pool. A gate row is the local tenant root
// against which a scoped transaction checks placement epoch and write state.
func TopologyDDL() []string {
	return []string{
		`CREATE TABLE tenant_gates (
   tenant_id TEXT PRIMARY KEY NOT NULL CHECK(length(tenant_id)=36),
   pool_id TEXT NOT NULL CHECK(length(pool_id)=36),
   placement_epoch BIGINT NOT NULL CHECK(placement_epoch>0 AND placement_epoch=CAST(placement_epoch AS BIGINT)),
   hierarchy_version BIGINT NOT NULL CHECK(hierarchy_version>0 AND hierarchy_version=CAST(hierarchy_version AS BIGINT)),
   state TEXT NOT NULL CHECK(state IN ('provisioning','active','maintenance','suspended','retired'))
 )`,
		`CREATE TABLE tenant_upgrade_receipts (
   tenant_id TEXT NOT NULL REFERENCES tenant_gates(tenant_id),
   run_id TEXT NOT NULL CHECK(length(run_id)=36),
   stage TEXT NOT NULL CHECK(length(trim(stage))>0),
   batch_id TEXT NOT NULL CHECK(length(trim(batch_id))>0),
   row_count BIGINT NOT NULL CHECK(row_count>=0 AND row_count=CAST(row_count AS BIGINT)),
   content_sha256 TEXT NOT NULL CHECK(length(content_sha256)=64),
   completed_at BIGINT NOT NULL CHECK(completed_at>=0 AND completed_at=CAST(completed_at AS BIGINT)), PRIMARY KEY(tenant_id,run_id,stage,batch_id)
 )`,
		`CREATE TABLE tenancy_organizations (
   tenant_id TEXT NOT NULL REFERENCES tenant_gates(tenant_id),
   id TEXT NOT NULL CHECK(length(id)=36),
   parent_id TEXT, name TEXT NOT NULL CHECK(length(trim(name))>0), active BOOLEAN NOT NULL CHECK(active IN (TRUE,FALSE)),
   PRIMARY KEY(tenant_id,id), CHECK(parent_id IS NULL OR parent_id<>id),
   FOREIGN KEY(tenant_id,parent_id) REFERENCES tenancy_organizations(tenant_id,id)
 )`,
		`CREATE TABLE tenancy_schools (
   tenant_id TEXT NOT NULL, id TEXT NOT NULL CHECK(length(id)=36), organization_id TEXT NOT NULL,
   code TEXT NOT NULL CHECK(length(code) BETWEEN 1 AND 64 AND code=upper(trim(code))),
   name TEXT NOT NULL CHECK(length(trim(name))>0),active BOOLEAN NOT NULL CHECK(active IN (TRUE,FALSE)),
   PRIMARY KEY(tenant_id,id), UNIQUE(tenant_id,organization_id,code),
   FOREIGN KEY(tenant_id,organization_id) REFERENCES tenancy_organizations(tenant_id,id)
 )`,
		`CREATE TABLE tenancy_campuses (
   tenant_id TEXT NOT NULL,id TEXT NOT NULL CHECK(length(id)=36),school_id TEXT NOT NULL,
   code TEXT NOT NULL CHECK(length(code) BETWEEN 1 AND 64 AND code=upper(trim(code))),
   name TEXT NOT NULL CHECK(length(trim(name))>0),timezone TEXT NOT NULL CHECK(length(trim(timezone))>0),
   active BOOLEAN NOT NULL CHECK(active IN (TRUE,FALSE)),PRIMARY KEY(tenant_id,id),UNIQUE(tenant_id,school_id,code),
   FOREIGN KEY(tenant_id,school_id) REFERENCES tenancy_schools(tenant_id,id)
 )`,
	}
}
