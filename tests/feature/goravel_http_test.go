package feature

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/tests"
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
