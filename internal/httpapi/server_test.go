package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
