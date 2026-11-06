package edumigrations_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/bootstrap"
	"github.com/elcomware/edupilot/modules"
	"github.com/elcomware/edupilot/platform/database"
)

const (
	groupA  = "ab31f5e3-77af-408b-ae67-82a2b72244ee"
	groupB  = "12455e04-9b86-4ad7-99be-0c8eacc70c2c"
	orgA    = "d40b0f38-a5cc-40bd-b092-d112ebf6fe7c"
	orgB    = "d387360a-551d-4446-86c6-ca119797d5b3"
	schoolA = "7ed14ea1-495a-40a9-be01-9c41ca2b14d0"
	schoolB = "08261ea5-1624-46cc-8474-58930342a4c9"
)

func foundationSQL(t *testing.T, driver string) database.Pool {
	t.Helper()
	var db *sql.DB
	if driver == "sqlite" {
		db = sqlitePool(t, filepath.Join(t.TempDir(), "pool.sqlite"))
	} else {
		dsn := os.Getenv("EDUPILOT_TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("requires disposable EDUPILOT_TEST_POSTGRES_DSN")
		}
		db = postgresPool(t, dsn)
	}
	return database.Pool{Driver: driver, DB: db}
}

func TestFoundationCrossTenantFK(t *testing.T) {
	for _, driver := range []string{"sqlite", "postgres"} {
		t.Run(driver, func(t *testing.T) {
			pool := foundationSQL(t, driver)
			for _, plan := range bootstrap.FoundationPlans() {
				for _, query := range plan.Statements(database.Connection{Driver: driver}, false) {
					if _, err := pool.DB.Exec(query); err != nil {
						t.Fatalf("%s: %v", plan.ID, err)
					}
				}
			}
			write := func(wantOK bool, q string, args ...any) {
				t.Helper()
				_, err := pool.DB.Exec(pool.Bind(q), args...)
				if (err == nil) != wantOK {
					t.Fatalf("wanted success=%v for %s: %v", wantOK, q, err)
				}
			}
			for _, pair := range [][2]string{{groupA, orgA}, {groupB, orgB}} {
				write(true, "INSERT INTO tenant_gates (tenant_id,pool_id,placement_epoch,hierarchy_version,state) VALUES (?,?,1,1,'active')", pair[0], groupB)
				write(true, "INSERT INTO tenancy_organizations (tenant_id,id,name,active) VALUES (?,?,'Group',TRUE)", pair[0], pair[1])
			}
			school := "INSERT INTO tenancy_schools (tenant_id,id,organization_id,code,name,active) VALUES (?,?,?,?,'School',TRUE)"
			write(false, school, groupA, schoolA, orgB, "NORTH")
			write(true, school, groupA, schoolA, orgA, "NORTH")
			write(true, school, groupA, schoolB, orgA, "SOUTH")
			student := "INSERT INTO students_records_v2 (tenant_id,school_id,id,given_name,family_name,status,version) VALUES (?,?,'legacy-1','Mary','Anne','active',1)"
			write(true, student, groupA, schoolA)
			write(true, student, groupA, schoolB)
			write(false, student, groupB, schoolA)
			for _, module := range modules.Definitions() {
				if module.Migration != nil {
					q := "INSERT INTO ws_" + module.Key + "_v2 (tenant_id,school_id,id,title,status,version) VALUES (?,?,'legacy-1','Follow up','open',1)"
					write(true, q, groupA, schoolA)
					write(false, q, groupB, schoolA)
				}
			}
			write(true, "INSERT INTO tenants VALUES (?,'group-a','active',1)", groupA)
			write(true, "INSERT INTO tenants VALUES (?,'group-b','active',1)", groupB)
			for _, sub := range []string{orgA, orgB} {
				write(true, "INSERT INTO identity_subjects_v2 VALUES (?,'active')", sub)
			}
			member := "INSERT INTO identity_memberships_v2 VALUES (?,?,?,'active',1)"
			write(true, member, schoolA, groupA, orgA)
			write(true, member, schoolB, groupB, orgB)
			credential := "INSERT INTO identity_credentials_v2 VALUES (?,?,'staff@example.test','fixture-hash')"
			write(true, credential, groupA, schoolA)
			write(true, credential, groupB, schoolB)
			write(false, credential, groupA, schoolB)
			alias := "INSERT INTO identity_login_aliases_v2 VALUES (?,'school-login','alias@example.test',?)"
			write(true, alias, groupA, schoolA)
			write(false, alias, groupB, schoolA)
			write(true, alias, groupB, schoolB)
			event := "INSERT INTO edu_outbox_v2(tenant_id,school_id,event_id,event_type,aggregate_id,payload,payload_version,created_at,available_at) VALUES (?,?,'event-1','student.registered','legacy-1','{}',1,0,0)"
			write(true, event, groupA, schoolA)
			write(false, event, groupA, schoolB)

			grant := "INSERT INTO identity_grants_v2(id,tenant_id,membership_id,permission,owner_kind,owner_id) VALUES (?,?,?,'students.read','tenant',?)"
			write(false, grant, orgA, groupA, schoolB, groupA)
			write(false, grant, orgA, groupA, schoolA, groupB)
			write(true, grant, orgA, groupA, schoolA, groupA)
			projection := "INSERT INTO identity_context_index_v2(tenant_id,school_id,organization_id,label,hierarchy_version,active) VALUES (?,?,?,'School',1,TRUE)"
			write(true, projection, groupA, schoolA, orgA)
			write(true, projection, groupB, schoolB, orgB)
			schoolGrant := "INSERT INTO identity_grants_v2(id,tenant_id,membership_id,permission,owner_kind,owner_id,school_id) VALUES (?,?,?,'students.read','school',?,?)"
			write(false, schoolGrant, orgB, groupA, schoolA, schoolB, schoolB)
			write(false, schoolGrant, orgB, groupA, schoolA, schoolA, nil)
			write(true, schoolGrant, orgB, groupA, schoolA, schoolA, schoolA)
			session := "INSERT INTO identity_sessions_v2(id,token_hash,tenant_id,membership_id,subject_id,organization_id,school_id,grant_version,hierarchy_version,placement_epoch,locale,expires_at) VALUES (?,?,?,?,?,?,?,1,1,1,'fr',90)"
			write(false, session, schoolB, strings.Repeat("a", 64), groupA, schoolA, orgA, orgB, schoolA)
			write(true, session, schoolB, strings.Repeat("a", 64), groupA, schoolA, orgA, orgA, schoolA)
			write(false, "UPDATE identity_sessions_v2 SET expires_at=? WHERE id=?", "bogus", schoolB)
			write(false, "UPDATE identity_grants_v2 SET expires_at=? WHERE id=?", "bogus", orgB)

		})
	}
}
