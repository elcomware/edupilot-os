package feature

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/tests"
)

func TestLiveEndpointThroughGoravel(t *testing.T) {
	tests.Boot()
	response := httptest.NewRecorder()
	facades.Route().ServeHTTP(response, httptest.NewRequest("GET", "/api/v1/health", nil))
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || response.Code != 200 || result.Status != "ok" {
		t.Fatalf("Goravel health: %d %s", response.Code, response.Body)
	}
}

func TestAuthRoutesAndSeederAreDiscoverableFromGoravel(t *testing.T) {
	tests.Boot()
	required := map[string]bool{
		"GET /login":             false,
		"GET /api/v1/session":    false,
		"POST /api/v1/session":   false,
		"DELETE /api/v1/session": false,
		"GET /api/v1/auth/me":    false,
	}
	for _, route := range facades.Route().GetRoutes() {
		for method := range strings.SplitSeq(route.Method, "|") {
			key := method + " " + route.Path
			if _, ok := required[key]; ok {
				required[key] = true
			}
		}
	}
	for route, found := range required {
		if !found {
			t.Errorf("missing Auth route %s", route)
		}
	}
	if rpc, ok := facades.Grpc().Server().GetServiceInfo()["edupilot.auth.v1.Auth"]; !ok || len(rpc.Methods) != 1 || rpc.Methods[0].Name != "GetCurrentUser" {
		t.Fatal("Goravel gRPC registration is missing")
	}
	if facades.Seeder().GetSeeder("AuthSchoolAdminSeeder") == nil {
		t.Fatal("AuthSchoolAdminSeeder is missing from Artisan")
	}
}
