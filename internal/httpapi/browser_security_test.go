package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/core"
)

func TestBrowserMutationProtectionAllowsDirectClientWithoutBrowserHeaders(t *testing.T) {
	h := browserMutationProtection(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "http://ywd-dmr.local/api/v1/test", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBrowserMutationProtectionAllowsSameOrigin(t *testing.T) {
	h := browserMutationProtection(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "http://192.168.1.11:8990/api/v1/test", nil)
	req.Header.Set("Origin", "http://192.168.1.11:8990")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBrowserMutationProtectionRejectsCrossOrigin(t *testing.T) {
	h := browserMutationProtection(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "http://192.168.1.11:8990/api/v1/test", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBrowserMutationProtectionRejectsSameSiteDifferentOrigin(t *testing.T) {
	h := browserMutationProtection(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "http://radio.example.test/api/v1/test", nil)
	req.Header.Set("Origin", "http://other.example.test")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBrowserMutationProtectionDoesNotBlockReadOnlyGet(t *testing.T) {
	h := browserMutationProtection(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "http://192.168.1.11:8990/api/v1/test", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestNewServerRejectsCrossOriginMutationBeforeEndpoint(t *testing.T) {
	h := New(core.NewState("test", "abc", "test"), "does-not-exist", "does-not-exist")
	body := `{"callsign":"N0CALL","dmr_id":1234567,"essid":1}`
	req := httptest.NewRequest(http.MethodPost, "http://ywd-dmr.local/api/v1/setup/identity/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}
