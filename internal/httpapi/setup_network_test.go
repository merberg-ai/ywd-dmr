package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

func TestNetworkValidateRequiresAdminSession(t *testing.T) {
	manager, _ := claimedAdminForAuthorizationTest(t)
	h := newServer(core.NewState("test", "abc", "test"), manager, nil, "", "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/network/validate", strings.NewReader(`{
		"backend":"brandmeister",
		"master_address":"master.example.net",
		"master_port":62031,
		"registration_frequency_hz":446525000,
		"password":"secret"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestNetworkValidateNormalizesWithoutEchoingPassword(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	h := newServer(core.NewState("test", "abc", "test"), manager, nil, "", "")
	secret := "do-not-return-me"

	req := httptest.NewRequest(http.MethodPost, "http://ywd-dmr.local/api/v1/setup/network/validate", strings.NewReader(`{
		"backend":" BrandMeister ",
		"master_address":" BM3103.EXAMPLE.NET. ",
		"master_port":0,
		"registration_frequency_hz":446525000,
		"password":"`+secret+`"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, secret) {
		t.Fatal("response leaked BrandMeister password")
	}
	for _, want := range []string{
		`"valid":true`,
		`"backend":"brandmeister"`,
		`"master_address":"bm3103.example.net"`,
		`"master_port":62031`,
		`"registration_frequency_hz":446525000`,
		`"password_set":true`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
}

func TestNetworkValidateReturnsFieldErrors(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	h := newServer(core.NewState("test", "abc", "test"), manager, nil, "", "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/network/validate", strings.NewReader(`{
		"backend":"other",
		"master_address":"https://bad.example:62031/path",
		"master_port":70000,
		"registration_frequency_hz":0,
		"password":""
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 validation result, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"valid":false`) {
		t.Fatalf("expected invalid result: %s", body)
	}
	for _, field := range []string{"backend", "master_address", "master_port", "registration_frequency_hz", "password"} {
		if !strings.Contains(body, `"field":"`+field+`"`) {
			t.Fatalf("expected field error for %s: %s", field, body)
		}
	}
}

func TestNetworkValidateRejectsCrossOriginBrowser(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	h := newServer(core.NewState("test", "abc", "test"), manager, nil, "", "")

	req := httptest.NewRequest(http.MethodPost, "http://ywd-dmr.local/api/v1/setup/network/validate", strings.NewReader(`{
		"backend":"brandmeister",
		"master_address":"master.example.net",
		"master_port":62031,
		"registration_frequency_hz":446525000,
		"password":"secret"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "{\"error\":\"same-origin request required\"}\n" {
		t.Fatalf("unexpected response: %s", rr.Body.String())
	}
}
