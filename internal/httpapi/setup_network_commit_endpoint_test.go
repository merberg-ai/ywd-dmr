package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/dmrnet"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

func TestNetworkTestAndCommitPersistsOnlyAfterSuccessfulTest(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	if _, err := store.Commit(config.Candidate{Identity: config.RadioIdentityInput{
		Callsign: "N0CALL",
		DMRID:    1234567,
		ESSID:    3,
	}}); err != nil {
		t.Fatal(err)
	}

	secret := "commit-secret"
	tester := &fakeNetworkTester{result: dmrnet.TestResult{
		OK:      true,
		Backend: config.NetworkBackendBrandMeister,
		Reason:  dmrnet.TestReasonOK,
		Message: "accepted",
	}}
	state := core.NewState("test", "abc", "test")
	state.SetClaimed(true)
	state.SetKnownGoodConfiguration(1, false)
	h := newServerWithTester(state, manager, store, tester, "", "")

	req := networkCommitRequest(t, session.Token, `{
		"backend":"brandmeister",
		"master_address":"3103.master.brandmeister.network",
		"master_port":62031,
		"registration_frequency_hz":446525000,
		"password":"`+secret+`"
	}`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !tester.called {
		t.Fatal("expected live tester to run before commit")
	}
	if strings.Contains(rr.Body.String(), secret) {
		t.Fatal("test-and-commit response leaked network password")
	}
	if !strings.Contains(rr.Body.String(), `"committed":true`) {
		t.Fatalf("expected committed response: %s", rr.Body.String())
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Config.Schema != config.KnownGoodNetworkSchema || loaded.Config.Revision != 2 || loaded.Config.Network == nil {
		t.Fatalf("unexpected committed config: %+v", loaded.Config)
	}
	if loaded.NetworkPassword != secret {
		t.Fatal("durable store did not retain the tested password")
	}
	setup := state.SetupStatus()
	if setup.Stage != "network_complete" || setup.NextStep != "audio" || !setup.Configuration.NetworkConfigured {
		t.Fatalf("setup state did not advance after durable network commit: %+v", setup)
	}
}

func TestNetworkTestAndCommitFailureLeavesKnownGoodUntouched(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	first, err := store.Commit(config.Candidate{Identity: config.RadioIdentityInput{
		Callsign: "N0CALL",
		DMRID:    1234567,
		ESSID:    3,
	}})
	if err != nil {
		t.Fatal(err)
	}

	tester := &fakeNetworkTester{result: dmrnet.TestResult{
		OK:      false,
		Backend: config.NetworkBackendBrandMeister,
		Reason:  dmrnet.TestReasonAuth,
		Message: "rejected",
	}}
	state := core.NewState("test", "abc", "test")
	state.SetClaimed(true)
	state.SetKnownGoodConfiguration(first.Revision, false)
	h := newServerWithTester(state, manager, store, tester, "", "")

	req := networkCommitRequest(t, session.Token, `{
		"backend":"brandmeister",
		"master_address":"3103.master.brandmeister.network",
		"master_port":62031,
		"registration_frequency_hz":446525000,
		"password":"wrong-secret"
	}`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 structured test failure, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"committed":false`) {
		t.Fatalf("expected non-commit response: %s", rr.Body.String())
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Config.Revision != first.Revision || loaded.Config.Schema != config.KnownGoodSchema || loaded.Config.Network != nil {
		t.Fatalf("failed live test changed known-good state: %+v", loaded.Config)
	}
	setup := state.SetupStatus()
	if setup.Stage != "identity_complete" || setup.Configuration.NetworkConfigured {
		t.Fatalf("failed test advanced setup state: %+v", setup)
	}
}

func TestNetworkTestAndCommitCrossOriginBlockedBeforeTest(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	if _, err := store.Commit(config.Candidate{Identity: config.RadioIdentityInput{Callsign: "N0CALL", DMRID: 1234567}}); err != nil {
		t.Fatal(err)
	}
	tester := &fakeNetworkTester{}
	h := newServerWithTester(core.NewState("test", "abc", "test"), manager, store, tester, "", "")

	req := networkCommitRequest(t, session.Token, `{
		"backend":"brandmeister",
		"master_address":"3103.master.brandmeister.network",
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
		t.Fatal("cross-origin request reached live network tester")
	}
}

func networkCommitRequest(t *testing.T, token, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://ywd-dmr.local/api/v1/setup/network/test-and-commit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: token})
	return req
}
