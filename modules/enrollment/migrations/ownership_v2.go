package migrations

func OwnershipDDL() []string {
	return []string{
		`CREATE TABLE enrollment_placements_v2 (
 tenant_id TEXT NOT NULL,school_id TEXT NOT NULL,id TEXT NOT NULL CHECK(length(id)>0),
 student_id TEXT NOT NULL CHECK(length(student_id)>0),academic_year VARCHAR(9) NOT NULL,
 grade VARCHAR(48) NOT NULL,course VARCHAR(120) NOT NULL DEFAULT '',
 status TEXT NOT NULL CHECK(status IN ('active','withdrawn')),version BIGINT NOT NULL CHECK(version>0 AND version=CAST(version AS BIGINT)),
 PRIMARY KEY(tenant_id,school_id,id),UNIQUE(tenant_id,school_id,student_id,academic_year),
 FOREIGN KEY(tenant_id,school_id) REFERENCES tenancy_schools(tenant_id,id)
 )`,
		`CREATE INDEX idx_enrollment_year_v2 ON enrollment_placements_v2(tenant_id,school_id,academic_year DESC,id)`,
	}
}
