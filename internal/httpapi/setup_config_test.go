package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

func TestIdentityCommitRequiresAdminSession(t *testing.T) {
	manager, _ := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	state := core.NewState("test", "abc", "test")
	state.SetClaimed(true)
	h := NewWithSecurityAndConfig(state, manager, store, "does-not-exist", "does-not-exist")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/identity/commit", strings.NewReader(`{"callsign":"N0CALL","dmr_id":1234567,"essid":1}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := store.Load(); !errors.Is(err, config.ErrNoKnownGoodConfig) {
		t.Fatalf("unauthenticated request changed durable configuration: %v", err)
	}
}

func TestIdentityCommitNormalizesPersistsAndAdvancesSetup(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	state := core.NewState("test", "abc", "test")
	state.SetClaimed(true)
	h := NewWithSecurityAndConfig(state, manager, store, "does-not-exist", "does-not-exist")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/identity/commit", strings.NewReader(`{"callsign":"  n0call  ","dmr_id":1234567,"essid":1}`))
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var response struct {
		Committed bool                 `json:"committed"`
		Revision  uint64               `json:"revision"`
		Identity  config.RadioIdentity `json:"identity"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Committed || response.Revision != 1 || response.Identity.Callsign != "N0CALL" {
		t.Fatalf("unexpected commit response: %+v", response)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load committed config: %v", err)
	}
	if loaded.Config.Revision != 1 || loaded.Config.Identity.Callsign != "N0CALL" {
		t.Fatalf("unexpected durable config: %+v", loaded.Config)
	}

	setup := state.SetupStatus()
	if setup.Stage != "identity_complete" || setup.NextStep != "network" || setup.Configuration.Revision != 1 {
		t.Fatalf("setup state did not advance after durable commit: %+v", setup)
	}
}

func TestIdentityCommitRejectsInvalidCandidateWithoutReplacingKnownGood(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	first, err := store.Commit(config.Candidate{Identity: config.RadioIdentityInput{Callsign: "N0CALL", DMRID: 1234567, ESSID: 1}})
	if err != nil {
		t.Fatalf("seed known-good config: %v", err)
	}
	state := core.NewState("test", "abc", "test")
	state.SetClaimed(true)
	state.SetKnownGoodConfiguration(first.Revision, false)
	h := NewWithSecurityAndConfig(state, manager, store, "does-not-exist", "does-not-exist")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/identity/commit", strings.NewReader(`{"callsign":"bad call","dmr_id":0,"essid":100}`))
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load known-good after rejection: %v", err)
	}
	if loaded.Config.Revision != 1 || loaded.Config.Identity.Callsign != "N0CALL" {
		t.Fatalf("invalid candidate replaced known-good config: %+v", loaded.Config)
	}
	if state.SetupStatus().Configuration.Revision != 1 {
		t.Fatalf("invalid candidate changed runtime revision: %+v", state.SetupStatus())
	}
}

func TestIdentityCommitSecondCommitAdvancesRevision(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	state := core.NewState("test", "abc", "test")
	state.SetClaimed(true)
	h := NewWithSecurityAndConfig(state, manager, store, "does-not-exist", "does-not-exist")

	for _, body := range []string{
		`{"callsign":"N0CALL","dmr_id":1234567,"essid":1}`,
		`{"callsign":"K1ABC","dmr_id":7654321,"essid":2}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/identity/commit", strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("commit failed: %d: %s", rr.Code, rr.Body.String())
		}
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load second commit: %v", err)
	}
	if loaded.Config.Revision != 2 || loaded.Config.Identity.Callsign != "K1ABC" {
		t.Fatalf("expected revision 2 K1ABC, got %+v", loaded.Config)
	}
	if state.SetupStatus().Configuration.Revision != 2 {
		t.Fatalf("runtime state did not advance to revision 2: %+v", state.SetupStatus())
	}
}

func TestIdentityCommitRejectsCrossOriginBeforeMutation(t *testing.T) {
	manager, session := claimedAdminForAuthorizationTest(t)
	store := config.NewFileStore(t.TempDir())
	state := core.NewState("test", "abc", "test")
	state.SetClaimed(true)
	h := NewWithSecurityAndConfig(state, manager, store, "does-not-exist", "does-not-exist")

	req := httptest.NewRequest(http.MethodPost, "http://ywd-dmr.local/api/v1/setup/identity/commit", strings.NewReader(`{"callsign":"N0CALL","dmr_id":1234567,"essid":1}`))
	req.Header.Set("Origin", "http://evil.example")
	req.AddCookie(&http.Cookie{Name: security.SessionCookieName, Value: session.Token})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := store.Load(); !errors.Is(err, config.ErrNoKnownGoodConfig) {
		t.Fatalf("cross-origin request changed durable configuration: %v", err)
	}
}
