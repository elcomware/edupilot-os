// Package identity owns school memberships, password authentication and bearer sessions.
// These tables live in the control database even when a school uses a dedicated data pool.
package identity

// SchemaUp is applied by the project's central migration registry, in order.
var SchemaUp = []string{
	`CREATE TABLE identity_schools (
		school_id VARCHAR(64) PRIMARY KEY
	)`,
	`CREATE TABLE identity_accounts (
		school_id VARCHAR(64) NOT NULL REFERENCES identity_schools(school_id),
		id VARCHAR(32) NOT NULL,
		email VARCHAR(254) NOT NULL,
		password_hash TEXT NOT NULL,
		PRIMARY KEY (school_id,id),
		UNIQUE (school_id,email)
	)`,
	`CREATE TABLE identity_memberships (
		school_id VARCHAR(64) NOT NULL,
		user_id VARCHAR(32) NOT NULL,
		role VARCHAR(16) NOT NULL CHECK (role IN ('admin','registrar','viewer')),
		PRIMARY KEY (school_id,user_id),
		FOREIGN KEY (school_id,user_id) REFERENCES identity_accounts(school_id,id) ON DELETE CASCADE
	)`,
	`CREATE TABLE identity_sessions (
		token_hash VARCHAR(64) PRIMARY KEY,
		school_id VARCHAR(64) NOT NULL,
		user_id VARCHAR(32) NOT NULL,
		expires_at BIGINT NOT NULL,
		FOREIGN KEY (school_id,user_id) REFERENCES identity_memberships(school_id,user_id) ON DELETE CASCADE
	)`,
	`CREATE INDEX identity_sessions_expiry ON identity_sessions (expires_at)`,
	`CREATE INDEX identity_sessions_member ON identity_sessions (school_id,user_id)`,
}

var SchemaDown = []string{
	`DROP TABLE identity_sessions`, `DROP TABLE identity_memberships`,
	`DROP TABLE identity_accounts`, `DROP TABLE identity_schools`,
}
