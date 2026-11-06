package migrations

func OwnershipDDL() []string {
	return []string{
		`CREATE TABLE students_records_v2 (
 tenant_id TEXT NOT NULL,school_id TEXT NOT NULL,id TEXT NOT NULL CHECK(length(id)>0),
 given_name VARCHAR(100) NOT NULL,family_name VARCHAR(100) NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('active','archived')),version BIGINT NOT NULL CHECK(version>0 AND version=CAST(version AS BIGINT)),
 PRIMARY KEY(tenant_id,school_id,id), FOREIGN KEY(tenant_id,school_id) REFERENCES tenancy_schools(tenant_id,id)
 )`,
		`CREATE INDEX idx_students_roster_v2 ON students_records_v2(tenant_id,school_id,status,family_name,given_name,id)`,
	}
}
