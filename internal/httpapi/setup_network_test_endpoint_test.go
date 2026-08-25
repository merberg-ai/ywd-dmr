package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/dmrnet"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

type fakeNetworkTester struct {
	called    bool
	identity  config.RadioIdentity
	candidate config.NetworkCandidate
	result    dmrnet.TestResult
	err       error
}

func (f *fakeNetworkTester) Test(_ context.Context, identity config.RadioIdentity, candidate config.NetworkCandidate) (dmrnet.TestResult, error) {
	f.called = true
	f.identity = identity
	f.candidate = candidate
	return f.result, f.err
}

func TestNetworkTestRequiresCommittedIdentity(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	tester := &fakeNetworkTester{}
	h := newServerWithTester(core.NewState("test", "abc", "test"), manager, store, tester, "", "")

	req := networkTestRequest(t, session.Token, `{
		"backend":"brandmeister",
		"master_address":"bm.example.test",
		"master_port":62031,
		"registration_frequency_hz":446525000,
		"password":"secret"
	}`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
	if tester.called {
		t.Fatal("tester ran without a committed station identity")
	}
}

func TestNetworkTestUsesKnownGoodIdentityWithoutEchoingPassword(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	if _, err := store.Commit(config.Candidate{Identity: config.RadioIdentityInput{
		Callsign: "N0CALL",
		DMRID:    1234567,
		ESSID:    1,
	}}); err != nil {
		t.Fatal(err)
	}

	secret := "no-echo-secret"
	tester := &fakeNetworkTester{result: dmrnet.TestResult{
		OK:      true,
		Backend: config.NetworkBackendBrandMeister,
		Reason:  dmrnet.TestReasonOK,
		Message: "BrandMeister accepted the test.",
	}}
	h := newServerWithTester(core.NewState("test", "abc", "test"), manager, store, tester, "", "")

	req := networkTestRequest(t, session.Token, `{
		"backend":"brandmeister",
		"master_address":"BM.EXAMPLE.TEST.",
		"master_port":0,
		"registration_frequency_hz":446525000,
		"password":"`+secret+`"
	}`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), secret) {
		t.Fatal("test response leaked network password")
	}
	if !tester.called {
		t.Fatal("expected tester to run")
	}
	if tester.identity.Callsign != "N0CALL" || tester.identity.DMRID != 1234567 || tester.identity.ESSID != 1 {
		t.Fatalf("unexpected identity passed to tester: %+v", tester.identity)
	}
	if tester.candidate.MasterAddress != "bm.example.test" || tester.candidate.MasterPort != config.BrandMeisterDefaultPort || tester.candidate.RegistrationFrequencyHz != 446_525_000 || tester.candidate.Password != secret {
		t.Fatalf("unexpected candidate passed to tester: %+v", tester.candidate)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Config.Revision != 1 {
		t.Fatalf("network test changed known-good revision to %d", loaded.Config.Revision)
	}
}

func TestNetworkTestRejectsInvalidCandidateBeforeTester(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	if _, err := store.Commit(config.Candidate{Identity: config.RadioIdentityInput{Callsign: "N0CALL", DMRID: 1234567}}); err != nil {
		t.Fatal(err)
	}
	tester := &fakeNetworkTester{}
	h := newServerWithTester(core.NewState("test", "abc", "test"), manager, store, tester, "", "")

	req := networkTestRequest(t, session.Token, `{
		"backend":"other",
		"master_address":"https://bad.example/path",
		"master_port":70000,
		"registration_frequency_hz":0,
		"password":""
	}`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	if tester.called {
		t.Fatal("tester ran for an invalid local candidate")
	}
}

func TestNetworkTestRejectsCrossOriginBeforeTester(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	if _, err := store.Commit(config.Candidate{Identity: config.RadioIdentityInput{Callsign: "N0CALL", DMRID: 1234567}}); err != nil {
		t.Fatal(err)
	}
	tester := &fakeNetworkTester{}
	h := newServerWithTester(core.NewState("test", "abc", "test"), manager, store, tester, "", "")

	req := networkTestRequest(t, session.Token, `{
		"backend":"brandmeister",
		"master_address":"bm.example.test",
		"master_port":62031,
		"registration_frequency_hz":446525000,
		"password":"secret"
	}`)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if tester.called {
		t.Fatal("tester ran for a cross-origin request")
	}
}

func networkTestRequest(t *testing.T, token, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://ywd-dmr.local/api/v1/setup/network/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: token})
	return req
}
