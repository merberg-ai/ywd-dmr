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

func TestIdentityValidationEndpoint(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	body := `{"callsign":"  kj6ywd  ","dmr_id":3196104,"essid":1}`
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
	if result.Normalized.Callsign != "KJ6YWD" {
		t.Fatalf("expected normalized callsign KJ6YWD, got %q", result.Normalized.Callsign)
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
	body := `{"callsign":"KJ6YWD","dmr_id":3196104,"essid":1,"surprise":true}`
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
