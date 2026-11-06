package database

// FoundationControlDDL creates the control-plane records for the group model.
// It stores configuration names and physical markers. DSNs and credentials
// remain in operator configuration. The original placement tables stay intact.
func FoundationControlDDL(driver string) []string {
	// SQLite gives ordinary tables an implicit unique rowid. A registry with an
	// immutable named primary key must have no alternate replacement identity.
	poolSuffix := ""
	if driver == "sqlite" {
		poolSuffix = " WITHOUT ROWID"
	}
	ddl := []string{
		`CREATE TABLE tenants (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36),
    slug TEXT NOT NULL UNIQUE CHECK(length(trim(slug)) BETWEEN 1 AND 63),
    state TEXT NOT NULL CHECK(state IN ('provisioning','active','suspended','archived')),
    version BIGINT NOT NULL CHECK(version>0 AND version=CAST(version AS BIGINT))
 )`,
		`CREATE TABLE database_pools (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36),
    connection_name TEXT NOT NULL UNIQUE CHECK(length(trim(connection_name))>0),
    pool_marker TEXT NOT NULL UNIQUE CHECK(length(trim(pool_marker))>0),
    driver TEXT NOT NULL CHECK(driver IN ('sqlite','postgres')),
    kind TEXT NOT NULL CHECK(kind IN ('shared','dedicated')),
    dedicated_tenant_id TEXT REFERENCES tenants(id),
    CHECK((kind='shared' AND dedicated_tenant_id IS NULL) OR
          (kind='dedicated' AND dedicated_tenant_id IS NOT NULL)),
    UNIQUE(id,kind), UNIQUE(id,dedicated_tenant_id), UNIQUE(id,pool_marker)
 )` + poolSuffix,
		// The kind FK prevents treating a dedicated pool as shared. The second
		// composite FK and explicit non-null check bind a dedicated pool to its owner.
		`CREATE TABLE tenant_placements (
    tenant_id TEXT PRIMARY KEY NOT NULL REFERENCES tenants(id),
    pool_id TEXT NOT NULL, pool_kind TEXT NOT NULL,
    dedicated_owner TEXT,
    epoch BIGINT NOT NULL CHECK(epoch>0 AND epoch=CAST(epoch AS BIGINT)),
    state TEXT NOT NULL CHECK(state IN ('provisioning','active','maintenance','suspended')),
    FOREIGN KEY(pool_id,pool_kind) REFERENCES database_pools(id,kind),
    FOREIGN KEY(pool_id,dedicated_owner) REFERENCES database_pools(id,dedicated_tenant_id),
    CHECK((pool_kind='shared' AND dedicated_owner IS NULL) OR
          (pool_kind='dedicated' AND dedicated_owner IS NOT NULL AND dedicated_owner=tenant_id))
 )`,
		`CREATE INDEX idx_tenant_placements_pool ON tenant_placements(pool_id,state,tenant_id)`,
		`CREATE TABLE tenant_upgrade_runs (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36),
    manifest_version INTEGER NOT NULL CHECK(manifest_version>0 AND manifest_version=CAST(manifest_version AS BIGINT)),
    manifest_sha256 TEXT NOT NULL UNIQUE CHECK(length(manifest_sha256)=64),
    source_sha256 TEXT NOT NULL CHECK(length(source_sha256)=64),
    stage TEXT NOT NULL CHECK(stage IN ('prepared','copying','verifying','verified','activated','failed')),
    revision BIGINT NOT NULL CHECK(revision>0 AND revision=CAST(revision AS BIGINT)),
    created_at BIGINT NOT NULL CHECK(created_at>=0 AND created_at=CAST(created_at AS BIGINT)), updated_at BIGINT NOT NULL CHECK(updated_at>=0 AND updated_at=CAST(updated_at AS BIGINT)),
    verification_sha256 TEXT CHECK(length(verification_sha256)=64),
    CHECK(stage NOT IN ('verified','activated') OR verification_sha256 IS NOT NULL)
 )`,
		`CREATE TABLE tenant_upgrade_checkpoints (
    run_id TEXT NOT NULL REFERENCES tenant_upgrade_runs(id),
    pool_id TEXT NOT NULL REFERENCES database_pools(id),
    step TEXT NOT NULL CHECK(length(trim(step))>0),
    fingerprint TEXT NOT NULL CHECK(length(fingerprint)=64),
    completed_at BIGINT NOT NULL CHECK(completed_at>=0 AND completed_at=CAST(completed_at AS BIGINT)), PRIMARY KEY(run_id,pool_id,step)
 )`,
		`CREATE TABLE legacy_scope_mappings (
    legacy_school_id TEXT PRIMARY KEY NOT NULL CHECK(length(trim(legacy_school_id))>0),
    run_id TEXT NOT NULL REFERENCES tenant_upgrade_runs(id),
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    source_pool_id TEXT NOT NULL, source_marker TEXT NOT NULL,
    organization_id TEXT NOT NULL CHECK(length(organization_id)=36),
    school_id TEXT NOT NULL CHECK(length(school_id)=36),
    campus_id TEXT NOT NULL CHECK(length(campus_id)=36),
    FOREIGN KEY(source_pool_id,source_marker) REFERENCES database_pools(id,pool_marker),
    UNIQUE(tenant_id), UNIQUE(tenant_id,school_id), UNIQUE(legacy_school_id,tenant_id)
 )`,
		`CREATE TABLE operator_audit_v2 (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36),
    operator_id TEXT NOT NULL CHECK(length(trim(operator_id))>0),
    action TEXT NOT NULL CHECK(length(trim(action))>0),
    tenant_id TEXT REFERENCES tenants(id), run_id TEXT REFERENCES tenant_upgrade_runs(id),
    reason TEXT NOT NULL CHECK(length(trim(reason))>0), occurred_at BIGINT NOT NULL CHECK(occurred_at>=0 AND occurred_at=CAST(occurred_at AS BIGINT))
 )`,
	}
	return append(ddl, poolIdentityGuards()[driver]...)
}

// poolIdentityGuards make a registered physical pool identity permanent.
// Routing, suspension and retirement change placement records. Reprovisioning
// uses an empty physical destination and a new marker and registration.
func poolIdentityGuards() map[string][]string {
	return map[string][]string{
		"sqlite": {
			`CREATE TRIGGER immutable_database_pools_insert BEFORE INSERT ON database_pools
 WHEN EXISTS(SELECT 1 FROM database_pools WHERE id=NEW.id OR pool_marker=NEW.pool_marker OR connection_name=NEW.connection_name)
 BEGIN SELECT RAISE(ABORT,'registered pool identity already exists'); END`,
			`CREATE TRIGGER immutable_database_pools_update BEFORE UPDATE ON database_pools
 BEGIN SELECT RAISE(ABORT,'registered pool identity is immutable'); END`,
			`CREATE TRIGGER immutable_database_pools_delete BEFORE DELETE ON database_pools
 BEGIN SELECT RAISE(ABORT,'registered pool identity is immutable'); END`,
		},
		"postgres": {
			`CREATE FUNCTION edupilot_refuse_pool_change() RETURNS trigger LANGUAGE plpgsql AS
 'BEGIN RAISE EXCEPTION ''registered pool identity is immutable''; END;'`,
			`CREATE TRIGGER immutable_database_pools BEFORE UPDATE OR DELETE ON database_pools
 FOR EACH ROW EXECUTE FUNCTION edupilot_refuse_pool_change()`,
		},
	}
}
