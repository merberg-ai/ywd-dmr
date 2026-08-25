package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

func claimedAdminForAuthorizationTest(t *testing.T) (*security.Manager, security.Session) {
	t.Helper()
	manager := security.NewManager(t.TempDir())
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize security manager: %v", err)
	}
	code, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	session, err := manager.Claim(security.ClaimRequest{
		ClaimCode: string(code),
		Username:  "sysop",
		Password:  "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	return manager, session
}

func TestRequireRoleRejectsMissingSession(t *testing.T) {
	manager, _ := claimedAdminForAuthorizationTest(t)
	s := &Server{state: core.NewState("test", "abc", "test"), security: manager, mux: http.NewServeMux()}

	h := s.requireRole(security.RoleObserver, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireRoleRejectsInvalidSession(t *testing.T) {
	manager, _ := claimedAdminForAuthorizationTest(t)
	s := &Server{state: core.NewState("test", "abc", "test"), security: manager, mux: http.NewServeMux()}

	h := s.requireRole(security.RoleObserver, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: "not-a-real-session"})
	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRequireRoleAllowsAdminAndPublishesPrincipal(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	s := &Server{state: core.NewState("test", "abc", "test"), security: manager, mux: http.NewServeMux()}

	h := s.requireRole(security.RoleAdmin, func(w http.ResponseWriter, r *http.Request) {
		principal, ok := authenticatedPrincipal(r)
		if !ok {
			t.Fatal("expected authenticated principal in request context")
		}
		if principal.Username != "sysop" || principal.Role != "admin" {
			t.Fatalf("unexpected principal: %+v", principal)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}
