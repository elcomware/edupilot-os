package outbox

func OwnershipDDL() []string {
	return []string{
		`CREATE TABLE edu_outbox_v2 (
 tenant_id TEXT NOT NULL, school_id TEXT NOT NULL, event_id TEXT NOT NULL CHECK(length(event_id)>0),
 event_type TEXT NOT NULL,aggregate_id TEXT NOT NULL,payload TEXT NOT NULL,
 payload_version INTEGER NOT NULL CHECK(payload_version>0 AND payload_version=CAST(payload_version AS BIGINT)),created_at BIGINT NOT NULL CHECK(created_at>=0 AND created_at=CAST(created_at AS BIGINT)),
 available_at BIGINT NOT NULL CHECK(available_at>=0 AND available_at=CAST(available_at AS BIGINT)), attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts>=0 AND attempts=CAST(attempts AS BIGINT)),
 lease_id TEXT,lease_until BIGINT CHECK(lease_until>=0 AND lease_until=CAST(lease_until AS BIGINT)),published_at BIGINT CHECK(published_at>=0 AND published_at=CAST(published_at AS BIGINT)),dead_at BIGINT CHECK(dead_at>=0 AND dead_at=CAST(dead_at AS BIGINT)),last_error TEXT,
 PRIMARY KEY(tenant_id,school_id,event_id), UNIQUE(tenant_id,event_id),
 FOREIGN KEY(tenant_id,school_id) REFERENCES tenancy_schools(tenant_id,id)
 )`,
		`CREATE INDEX idx_edu_outbox_ready_v2 ON edu_outbox_v2(tenant_id,available_at,created_at,event_id)
 WHERE published_at IS NULL AND dead_at IS NULL`,
		`CREATE TABLE edu_audit_inbox_v2 (
 tenant_id TEXT NOT NULL,school_id TEXT NOT NULL,event_id TEXT NOT NULL CHECK(length(event_id)>0),
 event_type TEXT NOT NULL,aggregate_id TEXT NOT NULL,received_at BIGINT NOT NULL CHECK(received_at>=0 AND received_at=CAST(received_at AS BIGINT)),
 PRIMARY KEY(tenant_id,school_id,event_id), UNIQUE(tenant_id,event_id),
 FOREIGN KEY(tenant_id,school_id) REFERENCES tenancy_schools(tenant_id,id)
 )`,
	}
}
