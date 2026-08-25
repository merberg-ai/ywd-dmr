package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

func claimedSecurityManagerForHTTPTest(t *testing.T) *security.Manager {
	t.Helper()
	dir := t.TempDir()
	manager := security.NewManager(dir)
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize security manager: %v", err)
	}
	code, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	if _, err := manager.Claim(security.ClaimRequest{
		ClaimCode: string(code),
		Username:  "sysop",
		Password:  "correct horse battery staple",
	}); err != nil {
		t.Fatalf("claim: %v", err)
	}

	restarted := security.NewManager(dir)
	if _, err := restarted.Initialize(); err != nil {
		t.Fatalf("restart security manager: %v", err)
	}
	return restarted
}

func TestLoginSessionAndLogoutEndpoints(t *testing.T) {
	manager := claimedSecurityManagerForHTTPTest(t)
	h := NewWithSecurity(core.NewState("test", "abc", "test"), manager, "does-not-exist", "does-not-exist")

	body := `{"username":"sysop","password":"correct horse battery staple"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.RemoteAddr = "192.0.2.10:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one login cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != security.SessionCookieName || cookie.Value == "" {
		t.Fatalf("unexpected login cookie: %+v", cookie)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("login cookie missing security attributes: %+v", cookie)
	}
	if strings.Contains(rr.Body.String(), cookie.Value) {
		t.Fatal("opaque session token must not appear in login JSON")
	}

	var loginResponse struct {
		Authenticated bool   `json:"authenticated"`
		Username      string `json:"username"`
		Role          string `json:"role"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &loginResponse); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if !loginResponse.Authenticated || loginResponse.Username != "sysop" || loginResponse.Role != "admin" {
		t.Fatalf("unexpected login response: %+v", loginResponse)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionReq.AddCookie(cookie)
	sessionRR := httptest.NewRecorder()
	h.ServeHTTP(sessionRR, sessionReq)
	if sessionRR.Code != http.StatusOK || !strings.Contains(sessionRR.Body.String(), `"authenticated":true`) {
		t.Fatalf("expected authenticated session, got %d: %s", sessionRR.Code, sessionRR.Body.String())
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRR := httptest.NewRecorder()
	h.ServeHTTP(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusNoContent {
		t.Fatalf("logout expected 204, got %d: %s", logoutRR.Code, logoutRR.Body.String())
	}
	logoutCookies := logoutRR.Result().Cookies()
	if len(logoutCookies) != 1 || logoutCookies[0].Name != security.SessionCookieName || logoutCookies[0].MaxAge >= 0 {
		t.Fatalf("expected expired session cookie on logout, got %+v", logoutCookies)
	}

	sessionReq = httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionReq.AddCookie(cookie)
	sessionRR = httptest.NewRecorder()
	h.ServeHTTP(sessionRR, sessionReq)
	if sessionRR.Code != http.StatusOK || !strings.Contains(sessionRR.Body.String(), `"authenticated":false`) {
		t.Fatalf("expected logged-out session, got %d: %s", sessionRR.Code, sessionRR.Body.String())
	}
}

func TestLoginFailureIsGeneric(t *testing.T) {
	manager := claimedSecurityManagerForHTTPTest(t)
	h := NewWithSecurity(core.NewState("test", "abc", "test"), manager, "does-not-exist", "does-not-exist")

	body := `{"username":"sysop","password":"wrong password but long enough"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.RemoteAddr = "192.0.2.11:12345"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong credentials expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.TrimSpace(rr.Body.String()) != `{"error":"authentication failed"}` {
		t.Fatalf("expected generic authentication failure, got %s", rr.Body.String())
	}
}

func TestLoginLimiterBlocksAfterFailureLimitAndRecovers(t *testing.T) {
	limiter := newLoginLimiter()
	now := time.Date(2026, time.August, 24, 18, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	key := "192.0.2.20"

	for i := 0; i < loginFailureLimit; i++ {
		if allowed, _ := limiter.Allow(key); !allowed {
			t.Fatalf("attempt %d was blocked before the failure limit", i+1)
		}
		limiter.Failure(key)
	}
	if allowed, retryAfter := limiter.Allow(key); allowed || retryAfter <= 0 {
		t.Fatalf("expected limiter block after %d failures, allowed=%v retry=%s", loginFailureLimit, allowed, retryAfter)
	}

	now = now.Add(loginBlockDuration + time.Second)
	if allowed, _ := limiter.Allow(key); !allowed {
		t.Fatal("expected login attempts to resume after block duration")
	}
	limiter.Success(key)
	if allowed, _ := limiter.Allow(key); !allowed {
		t.Fatal("successful login should clear limiter state")
	}
}

func TestLoginAndLogoutRequirePost(t *testing.T) {
	manager := claimedSecurityManagerForHTTPTest(t)
	h := NewWithSecurity(core.NewState("test", "abc", "test"), manager, "does-not-exist", "does-not-exist")

	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/logout"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s expected 405, got %d", path, rr.Code)
		}
		if got := rr.Header().Get("Allow"); got != http.MethodPost {
			t.Fatalf("%s expected Allow: POST, got %q", path, got)
		}
	}
}
