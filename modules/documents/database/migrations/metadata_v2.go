package migrations

func MetadataDDL() []string {
	return []string{
		`CREATE TABLE documents_metadata_v2 (
 tenant_id TEXT NOT NULL,school_id TEXT NOT NULL,id TEXT NOT NULL CHECK(length(id)>0),
 storage_ref TEXT NOT NULL CHECK(length(trim(storage_ref))>0),
 object_key TEXT NOT NULL CHECK(length(trim(object_key))>0),
 format_version INTEGER NOT NULL CHECK(format_version>0 AND format_version=CAST(format_version AS BIGINT)),
 content_type TEXT NOT NULL CHECK(length(trim(content_type))>0),
 content_sha256 TEXT NOT NULL CHECK(length(content_sha256)=64),
 size_bytes BIGINT NOT NULL CHECK(size_bytes>=0 AND size_bytes=CAST(size_bytes AS BIGINT)),
 state TEXT NOT NULL CHECK(state IN ('staged','available','quarantined','deleted')),
 version BIGINT NOT NULL CHECK(version>0 AND version=CAST(version AS BIGINT)),
 PRIMARY KEY(tenant_id,school_id,id),UNIQUE(storage_ref,object_key),
 FOREIGN KEY(tenant_id,school_id) REFERENCES tenancy_schools(tenant_id,id)
 )`,
	}
}
