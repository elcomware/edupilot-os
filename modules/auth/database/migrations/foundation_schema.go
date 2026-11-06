package migrations

// FoundationDDL puts identity and grants on the control database. A login realm
// disambiguates equal addresses at separate schools without linking subjects.
func FoundationDDL() []string {
	return []string{
		`CREATE TABLE identity_subjects_v2 (
   id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36),
   state TEXT NOT NULL CHECK(state IN ('active','suspended','archived'))
 )`,
		`CREATE TABLE identity_memberships_v2 (
   id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36),
   tenant_id TEXT NOT NULL REFERENCES tenants(id),
   subject_id TEXT NOT NULL REFERENCES identity_subjects_v2(id),
   state TEXT NOT NULL CHECK(state IN ('active','suspended','revoked')),
   grant_version BIGINT NOT NULL CHECK(grant_version>0 AND grant_version=CAST(grant_version AS BIGINT)),
   UNIQUE(tenant_id,subject_id), UNIQUE(tenant_id,id,subject_id), UNIQUE(tenant_id,id)
 )`,
		`CREATE TABLE identity_credentials_v2 (
   tenant_id TEXT NOT NULL, membership_id TEXT NOT NULL,
   normalized_email TEXT NOT NULL CHECK(length(normalized_email) BETWEEN 1 AND 254 AND normalized_email=lower(trim(normalized_email))),
   password_hash TEXT NOT NULL CHECK(length(password_hash)>0),
   PRIMARY KEY(tenant_id,membership_id), UNIQUE(tenant_id,normalized_email),
   FOREIGN KEY(tenant_id,membership_id) REFERENCES identity_memberships_v2(tenant_id,id)
 )`,
		`CREATE TABLE identity_login_aliases_v2 (
   tenant_id TEXT NOT NULL, login_realm TEXT NOT NULL CHECK(length(trim(login_realm))>0),
   normalized_email TEXT NOT NULL CHECK(length(normalized_email) BETWEEN 1 AND 254 AND normalized_email=lower(trim(normalized_email))),
   membership_id TEXT NOT NULL,
   PRIMARY KEY(tenant_id,login_realm,normalized_email),
   FOREIGN KEY(tenant_id,membership_id) REFERENCES identity_credentials_v2(tenant_id,membership_id)
 )`,
		`CREATE TABLE legacy_subject_mappings (
   legacy_school_id TEXT NOT NULL, legacy_user_id TEXT NOT NULL CHECK(length(legacy_user_id)>0),
   tenant_id TEXT NOT NULL, subject_id TEXT NOT NULL UNIQUE,
   membership_id TEXT NOT NULL, PRIMARY KEY(legacy_school_id,legacy_user_id),
   FOREIGN KEY(legacy_school_id,tenant_id) REFERENCES legacy_scope_mappings(legacy_school_id,tenant_id),
   FOREIGN KEY(tenant_id,membership_id,subject_id) REFERENCES identity_memberships_v2(tenant_id,id,subject_id)
 )`,
		`CREATE TABLE identity_context_index_v2 (
   tenant_id TEXT NOT NULL REFERENCES tenants(id),
   school_id TEXT NOT NULL CHECK(length(school_id)=36),
   organization_id TEXT NOT NULL CHECK(length(organization_id)=36),
   label TEXT NOT NULL CHECK(length(trim(label))>0),
   hierarchy_version BIGINT NOT NULL CHECK(hierarchy_version>0 AND hierarchy_version=CAST(hierarchy_version AS BIGINT)),
   active BOOLEAN NOT NULL CHECK(active IN (TRUE,FALSE)), PRIMARY KEY(tenant_id,school_id), UNIQUE(tenant_id,school_id,organization_id)
 )`,
		`CREATE TABLE identity_grants_v2 (
   id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36), tenant_id TEXT NOT NULL,
   membership_id TEXT NOT NULL, permission TEXT NOT NULL CHECK(length(trim(permission))>0),
   owner_kind TEXT NOT NULL CHECK(owner_kind IN ('tenant','school')),
   owner_id TEXT NOT NULL CHECK(length(owner_id)=36),
   expires_at BIGINT CHECK(expires_at>=0 AND expires_at=CAST(expires_at AS BIGINT)), revoked_at BIGINT CHECK(revoked_at>=0 AND revoked_at=CAST(revoked_at AS BIGINT)),
   school_id TEXT,
   CHECK((owner_kind='tenant' AND owner_id=tenant_id AND school_id IS NULL) OR
         (owner_kind='school' AND school_id IS NOT NULL AND owner_id=school_id)),
   FOREIGN KEY(tenant_id,school_id) REFERENCES identity_context_index_v2(tenant_id,school_id),
   FOREIGN KEY(tenant_id,membership_id) REFERENCES identity_memberships_v2(tenant_id,id)
 )`,
		`CREATE INDEX idx_identity_grants_member_v2 ON identity_grants_v2(tenant_id,membership_id,permission)`,
		`CREATE TABLE identity_sessions_v2 (
   id TEXT PRIMARY KEY NOT NULL CHECK(length(id)=36),
   token_hash TEXT NOT NULL UNIQUE CHECK(length(token_hash)=64),
   tenant_id TEXT NOT NULL, membership_id TEXT NOT NULL,subject_id TEXT NOT NULL,
   organization_id TEXT CHECK(length(organization_id)=36),
   school_id TEXT CHECK(length(school_id)=36),campus_id TEXT CHECK(length(campus_id)=36),
   grant_version BIGINT NOT NULL CHECK(grant_version>0 AND grant_version=CAST(grant_version AS BIGINT)),
   hierarchy_version BIGINT NOT NULL CHECK(hierarchy_version>0 AND hierarchy_version=CAST(hierarchy_version AS BIGINT)),
   placement_epoch BIGINT NOT NULL CHECK(placement_epoch>0 AND placement_epoch=CAST(placement_epoch AS BIGINT)),
   locale TEXT NOT NULL CHECK(length(trim(locale))>0),
   expires_at BIGINT NOT NULL CHECK(expires_at>=0 AND expires_at=CAST(expires_at AS BIGINT)), revoked_at BIGINT CHECK(revoked_at>=0 AND revoked_at=CAST(revoked_at AS BIGINT)),
   CHECK(campus_id IS NULL OR school_id IS NOT NULL),
   CHECK(school_id IS NULL OR organization_id IS NOT NULL),
   FOREIGN KEY(tenant_id,school_id,organization_id) REFERENCES identity_context_index_v2(tenant_id,school_id,organization_id),
   FOREIGN KEY(tenant_id,membership_id,subject_id) REFERENCES identity_memberships_v2(tenant_id,id,subject_id)
 )`,
		`CREATE INDEX idx_identity_sessions_expiry_v2 ON identity_sessions_v2(expires_at)`,
	}
}
