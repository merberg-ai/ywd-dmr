package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/core"
)

func TestHealth(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestMutatingMethodRejected(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/status", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestSetupStatusStartsUnclaimedAndMissing(t *testing.T) {
	state := core.NewState("test", "abc", "test")
	h := New(state, "does-not-exist", "does-not-exist")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var status core.SetupStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if status.Claimed || status.Stage != "unclaimed" || status.NextStep != "claim" {
		t.Fatalf("unexpected initial setup state: %+v", status)
	}
	if status.Configuration.State != "missing" || status.Configuration.IdentityConfigured {
		t.Fatalf("unexpected initial configuration state: %+v", status.Configuration)
	}
}

func TestSetupStatusReportsLoadedConfiguration(t *testing.T) {
	state := core.NewState("test", "abc", "test")
	state.SetKnownGoodConfiguration(7, true)
	h := New(state, "does-not-exist", "does-not-exist")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var status core.SetupStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if status.Configuration.State != "recovered" || !status.Configuration.Recovered {
		t.Fatalf("expected recovered configuration status, got %+v", status.Configuration)
	}
	if status.Configuration.Revision != 7 || !status.Configuration.IdentityConfigured {
		t.Fatalf("unexpected configuration metadata: %+v", status.Configuration)
	}
}

func TestSetupStatusRequiresGet(t *testing.T) {
	state := core.NewState("test", "abc", "test")
	h := New(state, "does-not-exist", "does-not-exist")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/status", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("expected Allow: GET, got %q", got)
	}
}

func TestIdentityValidationEndpoint(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	body := `{"callsign":"  n0call  ","dmr_id":1234567,"essid":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/identity/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result config.IdentityValidation
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid identity, got %+v", result.Errors)
	}
	if result.Normalized.Callsign != "N0CALL" {
		t.Fatalf("expected normalized callsign N0CALL, got %q", result.Normalized.Callsign)
	}
}

func TestIdentityValidationReturnsFieldErrors(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	body := `{"callsign":"bad call","dmr_id":0,"essid":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/identity/validate", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 validation result, got %d: %s", rr.Code, rr.Body.String())
	}

	var result config.IdentityValidation
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid identity")
	}
	if len(result.Errors) != 3 {
		t.Fatalf("expected 3 field errors, got %d: %+v", len(result.Errors), result.Errors)
	}
}

func TestIdentityValidationRejectsUnknownJSONFields(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	body := `{"callsign":"N0CALL","dmr_id":1234567,"essid":1,"surprise":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/identity/validate", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestIdentityValidationRequiresPost(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/identity/validate", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("expected Allow: POST, got %q", got)
	}
}
