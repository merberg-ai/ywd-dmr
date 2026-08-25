package security

import (
	"errors"
	"os"
	"testing"
)

func claimedManagerForAuthTest(t *testing.T) (*Manager, Session, string) {
	t.Helper()
	dir := t.TempDir()
	manager := NewManager(dir)
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	code, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	session, err := manager.Claim(ClaimRequest{
		ClaimCode: string(code),
		Username:  "SysOp",
		Password:  "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	return manager, session, dir
}

func TestLoginAfterRestartUsesPersistedPasswordAndCreatesFreshSession(t *testing.T) {
	_, claimSession, dir := claimedManagerForAuthTest(t)

	restarted := NewManager(dir)
	result, err := restarted.Initialize()
	if err != nil {
		t.Fatalf("restart initialize: %v", err)
	}
	if !result.Claimed {
		t.Fatal("expected durable claimed state after restart")
	}
	if _, err := restarted.Authenticate(claimSession.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("claim session unexpectedly survived restart: %v", err)
	}

	loginSession, err := restarted.Login(LoginRequest{
		Username: "  SYSOP  ",
		Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loginSession.Token == "" || loginSession.Token == claimSession.Token {
		t.Fatal("expected a fresh opaque login session token")
	}
	principal, err := restarted.Authenticate(loginSession.Token)
	if err != nil {
		t.Fatalf("authenticate login session: %v", err)
	}
	if principal.Username != "sysop" || principal.Role != "admin" {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestLoginRejectsWrongUsernameAndPasswordWithSameError(t *testing.T) {
	_, _, dir := claimedManagerForAuthTest(t)
	restarted := NewManager(dir)
	if _, err := restarted.Initialize(); err != nil {
		t.Fatalf("restart initialize: %v", err)
	}

	_, wrongUserErr := restarted.Login(LoginRequest{
		Username: "someone-else",
		Password: "correct horse battery staple",
	})
	_, wrongPasswordErr := restarted.Login(LoginRequest{
		Username: "sysop",
		Password: "this is definitely the wrong password",
	})
	if !errors.Is(wrongUserErr, ErrInvalidCredentials) {
		t.Fatalf("wrong username expected ErrInvalidCredentials, got %v", wrongUserErr)
	}
	if !errors.Is(wrongPasswordErr, ErrInvalidCredentials) {
		t.Fatalf("wrong password expected ErrInvalidCredentials, got %v", wrongPasswordErr)
	}
}

func TestLoginRequiresClaimedInstallation(t *testing.T) {
	manager := NewManager(t.TempDir())
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	_, err := manager.Login(LoginRequest{Username: "sysop", Password: "anything at all"})
	if !errors.Is(err, ErrNotClaimed) {
		t.Fatalf("expected ErrNotClaimed, got %v", err)
	}
}
