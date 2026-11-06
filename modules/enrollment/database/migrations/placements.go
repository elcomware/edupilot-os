package migrations

var PlacementsUp = []string{
	`CREATE TABLE IF NOT EXISTS enrollment_placements (
		school_id VARCHAR(63) NOT NULL,
		id VARCHAR(64) NOT NULL,
		student_id VARCHAR(64) NOT NULL,
		academic_year VARCHAR(9) NOT NULL,
		grade VARCHAR(48) NOT NULL,
		course VARCHAR(120) NOT NULL DEFAULT '',
		status VARCHAR(16) NOT NULL CHECK (status IN ('active','withdrawn')),
		version BIGINT NOT NULL CHECK (version>0),
		PRIMARY KEY (school_id,id),
		UNIQUE (school_id,student_id,academic_year)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_enrollment_year ON enrollment_placements(school_id, academic_year DESC, id)`,
}
var PlacementsDown = []string{`DROP TABLE IF EXISTS enrollment_placements`}
