// Package migrations owns Students' portable schema. The platform migration
// registry runs these statements inside a versioned transaction on every pool.
package migrations

var SchemaUp = []string{
	`CREATE TABLE IF NOT EXISTS students_records (
		school_id VARCHAR(63) NOT NULL,
		id VARCHAR(64) NOT NULL,
		given_name VARCHAR(100) NOT NULL,
		family_name VARCHAR(100) NOT NULL,
		status VARCHAR(16) NOT NULL CHECK (status IN ('active','archived')),
		version BIGINT NOT NULL CHECK (version > 0),
		PRIMARY KEY (school_id, id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_students_roster ON students_records (school_id, status, family_name, given_name, id)`,
}

var SchemaDown = []string{`DROP TABLE IF EXISTS students_records`}
