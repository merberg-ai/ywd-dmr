package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/security"
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

func TestClaimCreatesAdminSessionAndUpdatesSetupStatus(t *testing.T) {
	manager := security.NewManager(t.TempDir())
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize security manager: %v", err)
	}
	code, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	state := core.NewState("test", "abc", "test")
	h := NewWithSecurity(state, manager, "does-not-exist", "does-not-exist")

	body, err := json.Marshal(security.ClaimRequest{
		ClaimCode: string(code),
		Username:  "SysOp",
		Password:  "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("marshal claim: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/claim", strings.NewReader(string(body)))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one session cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != security.SessionCookieName || cookie.Value == "" {
		t.Fatalf("unexpected session cookie: %+v", cookie)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie missing security attributes: %+v", cookie)
	}
	if strings.Contains(rr.Body.String(), cookie.Value) {
		t.Fatal("opaque session token must not be returned in JSON")
	}

	status := state.SetupStatus()
	if !status.Claimed || status.Stage != "claimed" || status.NextStep != "identity" {
		t.Fatalf("unexpected setup state after claim: %+v", status)
	}

	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	sessionReq.AddCookie(cookie)
	sessionRR := httptest.NewRecorder()
	h.ServeHTTP(sessionRR, sessionReq)
	if sessionRR.Code != http.StatusOK {
		t.Fatalf("session status expected 200, got %d", sessionRR.Code)
	}
	var sessionStatus struct {
		Authenticated bool   `json:"authenticated"`
		Username      string `json:"username"`
		Role          string `json:"role"`
	}
	if err := json.Unmarshal(sessionRR.Body.Bytes(), &sessionStatus); err != nil {
		t.Fatalf("decode session status: %v", err)
	}
	if !sessionStatus.Authenticated || sessionStatus.Username != "sysop" || sessionStatus.Role != "admin" {
		t.Fatalf("unexpected session status: %+v", sessionStatus)
	}
}

func TestClaimRejectsWrongCodeAndReuse(t *testing.T) {
	manager := security.NewManager(t.TempDir())
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize security manager: %v", err)
	}
	state := core.NewState("test", "abc", "test")
	h := NewWithSecurity(state, manager, "does-not-exist", "does-not-exist")

	wrong := `{"claim_code":"AAAAAA-BBBBBB-CCCCCC-DDDDDD","username":"sysop","password":"correct horse battery staple"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/claim", strings.NewReader(wrong))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("wrong claim code expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if state.SetupStatus().Claimed {
		t.Fatal("wrong code changed setup state")
	}

	code, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	good, _ := json.Marshal(security.ClaimRequest{ClaimCode: string(code), Username: "sysop", Password: "correct horse battery staple"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/claim", strings.NewReader(string(good)))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("valid claim expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/claim", strings.NewReader(string(good)))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("reused claim expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}
