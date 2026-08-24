package security

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestInitializeCreatesStablePrivateClaimCode(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(dir)
	result, err := manager.Initialize()
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if result.Claimed {
		t.Fatal("fresh manager should be unclaimed")
	}

	data, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	first := strings.TrimSpace(string(data))
	if len(normalizeClaimCode(first)) != 24 {
		t.Fatalf("expected 24 base32 claim characters, got %q", first)
	}
	info, err := os.Stat(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("stat claim code: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected claim code mode 0600, got %04o", got)
	}

	manager2 := NewManager(dir)
	if _, err := manager2.Initialize(); err != nil {
		t.Fatalf("reinitialize: %v", err)
	}
	data2, err := os.ReadFile(manager2.ClaimCodePath())
	if err != nil {
		t.Fatalf("read second claim code: %v", err)
	}
	if strings.TrimSpace(string(data2)) != first {
		t.Fatal("claim code changed across unclaimed restart")
	}
}

func TestClaimCreatesAdminDeletesCodeAndCreatesSession(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(dir)
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	codeBytes, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}

	session, err := manager.Claim(ClaimRequest{
		ClaimCode: string(codeBytes),
		Username:  "  SysOp  ",
		Password:  "correct horse battery staple",
	})
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if session.Token == "" {
		t.Fatal("expected opaque session token")
	}
	if session.Principal.Username != "sysop" || session.Principal.Role != "admin" {
		t.Fatalf("unexpected principal: %+v", session.Principal)
	}
	if _, err := os.Stat(manager.ClaimCodePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected claim code to be deleted, got %v", err)
	}

	info, err := os.Stat(manager.securityPath())
	if err != nil {
		t.Fatalf("stat security state: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected security state mode 0600, got %04o", got)
	}

	principal, err := manager.Authenticate(session.Token)
	if err != nil {
		t.Fatalf("authenticate session: %v", err)
	}
	if principal.Role != "admin" || principal.Username != "sysop" {
		t.Fatalf("unexpected authenticated principal: %+v", principal)
	}
}

func TestClaimRejectsWrongCodeWithoutChangingState(t *testing.T) {
	manager := NewManager(t.TempDir())
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	_, err := manager.Claim(ClaimRequest{
		ClaimCode: "AAAAAA-BBBBBB-CCCCCC-DDDDDD",
		Username:  "sysop",
		Password:  "correct horse battery staple",
	})
	if !errors.Is(err, ErrInvalidClaimCode) {
		t.Fatalf("expected ErrInvalidClaimCode, got %v", err)
	}
	if manager.Claimed() {
		t.Fatal("wrong code must not claim installation")
	}
	if _, err := os.Stat(manager.ClaimCodePath()); err != nil {
		t.Fatalf("claim code should remain after failed attempt: %v", err)
	}
}

func TestClaimValidatesAdminFields(t *testing.T) {
	manager := NewManager(t.TempDir())
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	code, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	_, err = manager.Claim(ClaimRequest{
		ClaimCode: string(code),
		Username:  "x!",
		Password:  "short",
	})
	var validation *ClaimValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected ClaimValidationError, got %v", err)
	}
	if len(validation.Errors) != 2 {
		t.Fatalf("expected username and password errors, got %+v", validation.Errors)
	}
	if manager.Claimed() {
		t.Fatal("invalid admin fields must not claim installation")
	}
}

func TestClaimCannotBeReusedAndPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(dir)
	if _, err := manager.Initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	code, err := os.ReadFile(manager.ClaimCodePath())
	if err != nil {
		t.Fatalf("read claim code: %v", err)
	}
	request := ClaimRequest{
		ClaimCode: string(code),
		Username:  "sysop",
		Password:  "correct horse battery staple",
	}
	if _, err := manager.Claim(request); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if _, err := manager.Claim(request); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("expected ErrAlreadyClaimed, got %v", err)
	}

	restarted := NewManager(dir)
	result, err := restarted.Initialize()
	if err != nil {
		t.Fatalf("restart initialize: %v", err)
	}
	if !result.Claimed || !restarted.Claimed() {
		t.Fatal("claimed state did not persist across restart")
	}
	if _, err := os.Stat(restarted.ClaimCodePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("claim code reappeared after claimed restart: %v", err)
	}
}

func TestCorruptSecurityStateFailsClosed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepathJoin(dir, securityFilename), []byte("{broken\n"), 0o600); err != nil {
		t.Fatalf("write corrupt state: %v", err)
	}
	manager := NewManager(dir)
	if _, err := manager.Initialize(); err == nil {
		t.Fatal("expected corrupt security state to fail initialization")
	}
	if _, err := os.Stat(manager.ClaimCodePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupt security state must not generate a new claim code: %v", err)
	}
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}
