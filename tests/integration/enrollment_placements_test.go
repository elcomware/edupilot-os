package integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/modules/enrollment/placements"
	studentmigrations "github.com/elcomware/edupilot-os/modules/students/migrations"
	studentproviders "github.com/elcomware/edupilot-os/modules/students/providers"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/identity"
	"github.com/elcomware/edupilot-os/platform/outbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// Exercise the public handlers with both modules' real providers. The pool is
// deliberately restricted to one connection so a cross-module query must
// borrow the current transaction instead of trying to acquire a second handle.
func TestEnrollmentAcrossModuleBoundaries(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		node := database.Connection{Name: "sqlite", Driver: "sqlite", Database: filepath.Join(t.TempDir(), "edu.sqlite")}
		db, err := sql.Open("sqlite3", node.Database)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		exercisePlacements(t, node, db)
	})
	t.Run("postgres", func(t *testing.T) {
		dsn := os.Getenv("EDUPILOT_TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("set EDUPILOT_TEST_POSTGRES_DSN to a disposable test database")
		}
		config, err := pgx.ParseConfig(dsn)
		if err != nil {
			t.Fatal(err)
		}
		var token [12]byte
		if _, err = rand.Read(token[:]); err != nil {
			t.Fatal(err)
		}
		schema := "placements_test_" + hex.EncodeToString(token[:])
		config.RuntimeParams["search_path"] = schema
		db := stdlib.OpenDB(*config)
		t.Cleanup(func() { _ = db.Close() })
		if _, err = db.Exec("CREATE SCHEMA " + schema); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := db.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
				t.Error(err)
			}
		})
		node := database.Connection{Name: "postgres", Driver: "postgres", Database: config.Database, Host: config.Host, Port: int(config.Port), Username: config.User, SSLMode: "disable"}
		exercisePlacements(t, node, db)
	})
}

func exercisePlacements(t *testing.T, node database.Connection, db *sql.DB) {
	t.Helper()
	db.SetMaxOpenConns(1)
	catalog, err := database.Open(database.Settings{Control: node}, func(database.Connection) (*sql.DB, error) { return db, nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range [][]string{edumigrations.PlacementPlan().Statements(node, false), outbox.SchemaUp, studentmigrations.SchemaUp, placements.SchemaUp} {
		for _, ddl := range group {
			if _, err = db.Exec(ddl); err != nil {
				t.Fatal(err)
			}
		}
	}
	students := studentproviders.New(catalog)
	enrollment := placements.New(catalog, students.Directory)
	north := identity.Principal{ID: "registrar", School: "north", Permissions: []string{identity.StudentsWrite, identity.StudentsRead, identity.WorkspaceWrite, identity.WorkspaceRead}}
	south := north
	south.School = "south"
	call := func(handler http.Handler, p identity.Principal, method, path string, body any, want int) *httptest.ResponseRecorder {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req := httptest.NewRequest(method, path, bytes.NewReader(data)).WithContext(identity.WithPrincipal(ctx, p))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: expected %d got %d (%s)", method, path, want, w.Code, w.Body.String())
		}
		return w
	}
	const base = "/api/v1/enrollment/placements"
	record := call(students.HTTP, north, "POST", "/api/v1/students", map[string]string{"givenName": "Ana", "familyName": "Liu"}, 201)
	var student struct{ ID string }
	if err = json.Unmarshal(record.Body.Bytes(), &student); err != nil {
		t.Fatal(err)
	}
	body := map[string]string{"studentId": student.ID, "academicYear": "2026-2027", "grade": "Grade 8"}
	restricted := north
	restricted.Permissions = []string{identity.StudentsWrite}
	call(enrollment, restricted, "POST", base, body, 403)
	call(enrollment, south, "POST", base, body, 404)
	record = call(enrollment, north, "POST", base, body, 201)
	var first struct {
		ID      string
		Version int
	}
	if err = json.Unmarshal(record.Body.Bytes(), &first); err != nil || first.ID == "" || first.Version != 1 {
		t.Fatalf("initial placement: %v %v", first, err)
	}
	retry := call(enrollment, north, "POST", base, body, 200)
	if !bytes.Equal(record.Body.Bytes(), retry.Body.Bytes()) {
		t.Fatal("identical retry did not return original placement")
	}
	body["grade"] = "Grade 9"
	call(enrollment, north, "POST", base, body, 409)
	body["grade"] = "Grade 8"
	call(students.HTTP, north, "DELETE", "/api/v1/students/"+student.ID+"?version=1", nil, 204)
	call(enrollment, north, "POST", base, body, 200)
	body["academicYear"] = "2027-2028"
	call(enrollment, north, "POST", base, body, 409)
	body["academicYear"] = "2026-2027"
	end := map[string]any{"status": "withdrawn", "version": 1}
	call(enrollment, south, "PATCH", base+"/"+first.ID, end, 404)
	call(enrollment, north, "PATCH", base+"/"+first.ID, end, 200)
	call(enrollment, north, "PATCH", base+"/"+first.ID, end, 409)
	call(enrollment, north, "POST", base, body, 409)
	var roster struct {
		Total int
		Items []struct {
			Status  string
			Version int
		}
	}
	w := call(enrollment, north, "GET", base+"?studentId="+student.ID, nil, 200)
	if err = json.Unmarshal(w.Body.Bytes(), &roster); err != nil || roster.Total != 1 || len(roster.Items) != 1 || roster.Items[0].Status != "withdrawn" || roster.Items[0].Version != 2 {
		t.Fatalf("roster: %+v %v", roster, err)
	}
	w = call(enrollment, south, "GET", base, nil, 200)
	if err = json.Unmarshal(w.Body.Bytes(), &roster); err != nil || roster.Total != 0 || len(roster.Items) != 0 {
		t.Fatal("placement leaked to another school")
	}
	var eventCount int
	if err = db.QueryRow("SELECT COUNT(*) FROM edu_outbox WHERE event_type IN ('enrollment.placed.v1','enrollment.withdrawn.v1')").Scan(&eventCount); err != nil || eventCount != 2 {
		t.Fatalf("expected one placement and one withdrawal event, got %d (%v)", eventCount, err)
	}
}
