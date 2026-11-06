// Package feature exercises the real session, router and SQL adapter together.
package feature

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/elcomware/edupilot/app/schoolapi"
	modules "github.com/elcomware/edupilot/bootstrap/modulecatalog"
	"github.com/elcomware/edupilot/modules/auth/app/http/controllers"
	"github.com/elcomware/edupilot/modules/auth/app/http/middleware"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	"github.com/elcomware/edupilot/modules/auth/database/migrations"
	authroutes "github.com/elcomware/edupilot/modules/auth/routes"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/migration"
	"github.com/elcomware/edupilot/modules/core/workspace"
	"github.com/elcomware/edupilot/modules/core/workspace/sqlstore"
	tenancy "github.com/elcomware/edupilot/modules/tenancy/database/migrations"
	"github.com/elcomware/edupilot/tests/schemafixture"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestSchoolWorkListsUseTheSameVersionedAPIForEveryModule(t *testing.T) {
	ctx := context.Background()
	file := filepath.Join(t.TempDir(), "school.sqlite")
	raw, err := sql.Open("sqlite3", file)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	conn := database.Connection{Name: "sqlite", Driver: "sqlite", Database: file}
	db, err := database.Open(database.Settings{Control: conn}, func(database.Connection) (*sql.DB, error) { return raw, nil })
	if err != nil {
		t.Fatal(err)
	}
	tables := schemafixture.Builder{DB: raw}
	if err := tenancy.Placement(tables, true); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec("INSERT INTO edu_pool_marker VALUES(1,'shared','test-pool')"); err != nil {
		t.Fatal(err)
	}
	for _, create := range []func(migration.Schema) error{migrations.Accounts, migration.Outbox, tenancy.Workspace} {
		if err := create(tables); err != nil {
			t.Fatal(err)
		}
	}
	auth := services.New(&db.Control)
	handler, err := schoolapi.New(db, "", func(key string) (workspace.Service, error) {
		return workspace.Service{Store: sqlstore.New(db, key)}, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolve := func() (*services.Service, error) { return auth, nil }
	schoolHandler := handler
	handler = authroutes.HTTP(controllers.New(resolve, nil), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middleware.ServeAuthenticated(w, r, schoolHandler, &services.Guard{Request: r, Resolve: resolve})
	}))

	request := func(method, path, token string, body any, status int) []byte {
		t.Helper()
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, status, w.Body.String())
		}
		return w.Body.Bytes()
	}
	login := func(school string) string {
		t.Helper()
		password := "fixture-password-please-change"
		if _, err := auth.Seed(ctx, school, "admin@example.test", password); err != nil {
			t.Fatal(err)
		}
		response := request(http.MethodPost, "/api/v1/session", "", map[string]string{"school": school, "email": "admin@example.test", "password": password}, 200)
		var s services.Session
		if err := json.Unmarshal(response, &s); err != nil || s.Token == "" {
			t.Fatal("missing session", err)
		}
		return s.Token
	}
	a, b := login("school-a"), login("school-b")
	request("GET", "/api/v1/modules", "", nil, 401)
	var catalog struct{ Items []struct{ Key string } }
	if err := json.Unmarshal(request("GET", "/api/v1/modules", a, nil, 200), &catalog); err != nil || len(catalog.Items) != 2 || catalog.Items[0].Key != "tenancy" || catalog.Items[1].Key != "auth" {
		t.Fatal("unexpected active catalog", catalog, err)
	}
	for _, path := range []string{"/api/v1/students", "/api/v1/modules/students", "/api/v1/modules/documents/workspace"} {
		request("GET", path, a, nil, 404)
	}
	for _, m := range modules.Definitions() {

		if m.WorkspaceSchema == nil {
			request("GET", "/api/v1/modules/"+m.Key+"/workspace", a, nil, 404)
			continue
		}
		group := "/api/v1/modules/" + m.Key + "/workspace"
		var entry workspace.Entry
		payload := request("POST", group, a, map[string]string{"title": "Review the school calendar"}, 201)
		if err := json.Unmarshal(payload, &entry); err != nil || entry.Version != 1 {
			t.Fatalf("%s creation: %s", m.Key, payload)
		}
		item := group + "/" + entry.ID
		request("GET", item, a, nil, 200)
		request("GET", item, b, nil, 404)
		edit := map[string]any{"title": "Calendar reviewed", "status": "closed", "version": 1}
		request("PATCH", item, b, edit, 404)
		request("PATCH", item, a, edit, 200)
		request("PATCH", item, a, edit, 409)
		var page workspace.Page
		if err := json.Unmarshal(request("GET", group, b, nil, 200), &page); err != nil || page.Total != 0 || len(page.Items) != 0 {
			t.Fatalf("%s cross-school disclosure", m.Key)
		}
	}
	request("DELETE", "/api/v1/session", a, nil, 204)
	request("GET", "/api/v1/modules", a, nil, 401)
}
