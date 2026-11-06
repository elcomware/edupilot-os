package controllers

import (
	"encoding/json"
	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/platform/access"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestCurrentUserFollowsSchoolMembershipAndLiveRole(t *testing.T) {
	s := fixture(t)
	ctx := t.Context()
	const password = "unique-fixture-passphrase"
	for _, school := range []string{"north", "south"} {
		if _, err := s.Seed(ctx, school, "admin@example.test", password); err != nil {
			t.Fatal(err)
		}
	}
	admin, err := s.Login(ctx, "north", "admin@example.test", password)
	if err != nil {
		t.Fatal(err)
	}
	member, err := s.CreateMember(ctx, admin.Token, "north", "teacher@example.test", password, models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.Login(ctx, "north", "teacher@example.test", password)
	if err != nil {
		t.Fatal(err)
	}
	get := func(token, school string, want int) models.User {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if school != "" {
			r.Header.Set("X-School-ID", school)
		}
		w := httptest.NewRecorder()
		testController(s).CurrentUser(w, r)
		if w.Code != want || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("status %d; cache %q", w.Code, w.Header().Get("Cache-Control"))
		}
		var u models.User
		if want == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), session.Token) {
				t.Fatal("private field exposed")
			}
		}
		return u
	}
	get("", "", 401)
	get(session.Token, "south", 403)
	viewer := get(session.Token, "north", 200)
	if viewer.ID != member.ID || viewer.Email != "teacher@example.test" || viewer.Membership != (models.Membership{School: "north", UserID: member.ID, Role: models.RoleViewer}) || slices.Contains(viewer.Permissions, access.StudentsWrite) {
		t.Fatal("wrong viewer relationships")
	}
	if err = s.SetMemberRole(ctx, admin.Token, "north", member.ID, models.RoleRegistrar); err != nil {
		t.Fatal(err)
	}
	registrar := get(session.Token, "", 200)
	if registrar.Membership.Role != models.RoleRegistrar || !slices.Contains(registrar.Permissions, access.StudentsWrite) {
		t.Fatal("stale role or permissions")
	}
	if err = s.Logout(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	get(session.Token, "north", 401)
}
