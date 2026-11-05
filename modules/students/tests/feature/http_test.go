package feature_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	studenthttp "github.com/elcomware/edupilot-os/modules/students/adapters/inbound/http"
	"github.com/elcomware/edupilot-os/modules/students/application"
	"github.com/elcomware/edupilot-os/modules/students/contracts"
	"github.com/elcomware/edupilot-os/modules/students/tests/testkit"
	"github.com/elcomware/edupilot-os/platform/identity"
)

func TestSchoolScopedRosterAndOptimisticUpdates(t *testing.T) {
	h := studenthttp.New(application.New(testkit.NewMemory()))
	base := "/api/v1/students"
	north := identity.Principal{ID: "registrar", School: "north", Permissions: []string{identity.StudentsWrite, identity.StudentsRead}}
	viewer := identity.Principal{ID: "viewer", School: "north", Permissions: []string{identity.StudentsRead}}
	south := identity.Principal{ID: "other", School: "south", Permissions: north.Permissions}
	call := func(p identity.Principal, method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if p.ID != "" {
			r = r.WithContext(identity.WithPrincipal(r.Context(), p))
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, w.Code, want, w.Body.String())
		}
		return w
	}
	names := `{"givenName":"Sam","familyName":"Li"}`
	call(identity.Principal{}, http.MethodGet, base, "", 401)
	call(viewer, http.MethodPost, base, names, 403)
	call(north, http.MethodPost, base, `{"givenName":"Sam","familyName":"Li","school":"south"}`, 400)
	w := call(north, http.MethodPost, base, names, 201)
	var student contracts.Student
	if err := json.Unmarshal(w.Body.Bytes(), &student); err != nil {
		t.Fatal(err)
	}
	item := base + "/" + student.ID
	if w.Header().Get("Location") != item || student.Version != 1 {
		t.Fatal("bad creation response")
	}
	call(south, http.MethodGet, item, "", 404)
	update := `{"givenName":"Samuel","familyName":"Li","version":1}`
	call(south, http.MethodPatch, item, update, 404)
	call(north, http.MethodPatch, item, update, 200)
	stale := call(north, http.MethodPatch, item, update, 409)
	if !strings.Contains(stale.Body.String(), "version_conflict") {
		t.Fatal("missing conflict code")
	}
	call(north, http.MethodDelete, item+"?version=2", "", 204)
	w = call(viewer, http.MethodGet, item, "", 200)
	if err := json.Unmarshal(w.Body.Bytes(), &student); err != nil || student.Status != "archived" || student.Version != 3 {
		t.Fatalf("archived record: %+v %v", student, err)
	}
	w = call(viewer, http.MethodGet, base, "", 200)
	var page contracts.Page
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || page.Total != 0 || page.Items == nil || len(page.Items) != 0 {
		t.Fatalf("active roster: %+v %v", page, err)
	}
}
