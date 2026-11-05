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

	"github.com/elcomware/edupilot-os/app/schoolapi"
	"github.com/elcomware/edupilot-os/database/edumigrations"
	"github.com/elcomware/edupilot-os/modules"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/identity"
	"github.com/elcomware/edupilot-os/platform/workspace"
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
	plans := []edumigrations.Plan{edumigrations.PlacementPlan(), edumigrations.OutboxPlan(), {Up: identity.SchemaUp}}
	for _, d := range modules.Definitions() {
		if d.Migration != nil {
			plans = append(plans, d.Migration())
		}
	}
	for _, plan := range plans {
		for _, stmt := range plan.Statements(conn, false) {
			if _, err := raw.Exec(stmt); err != nil {
				t.Fatal(err)
			}
		}
	}
	handler, err := schoolapi.New(db, "")
	if err != nil {
		t.Fatal(err)
	}
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
	auth := identity.New(&db.Control)
	login := func(school string) string {
		t.Helper()
		password := "fixture-password-please-change"
		if _, err := auth.Seed(ctx, school, "admin@example.test", password); err != nil {
			t.Fatal(err)
		}
		response := request(http.MethodPost, "/api/v1/session", "", map[string]string{"school": school, "email": "admin@example.test", "password": password}, 200)
		var s identity.Session
		if err := json.Unmarshal(response, &s); err != nil || s.Token == "" {
			t.Fatal("missing session", err)
		}
		return s.Token
	}
	a, b := login("school-a"), login("school-b")
	request("GET", "/api/v1/modules", "", nil, 401)
	request("GET", "/api/v1/modules", a, nil, 200)
	for _, m := range modules.Definitions() {
		if m.Key == "students" {
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
