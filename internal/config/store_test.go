package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validCandidate(callsign string, dmrID, essid int) Candidate {
	return Candidate{Identity: RadioIdentityInput{
		Callsign: callsign,
		DMRID:    dmrID,
		ESSID:    essid,
	}}
}

func TestFileStoreStartsEmpty(t *testing.T) {
	store := NewFileStore(t.TempDir())
	_, err := store.Load()
	if !errors.Is(err, ErrNoKnownGoodConfig) {
		t.Fatalf("expected ErrNoKnownGoodConfig, got %v", err)
	}
}

func TestFileStoreCommitNormalizesAndPersists(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	committed, err := store.Commit(validCandidate("  n0call  ", 1234567, 1))
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}
	if committed.Schema != KnownGoodSchema || committed.Revision != 1 {
		t.Fatalf("unexpected metadata: %+v", committed)
	}
	if committed.Identity.Callsign != "N0CALL" {
		t.Fatalf("expected normalized callsign, got %q", committed.Identity.Callsign)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.RecoveredFromPrevious {
		t.Fatal("did not expect recovery from previous snapshot")
	}
	if loaded.Config != committed {
		t.Fatalf("loaded config mismatch: got %+v want %+v", loaded.Config, committed)
	}

	info, err := os.Stat(filepath.Join(dir, knownGoodFilename))
	if err != nil {
		t.Fatalf("stat current config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected current config mode 0600, got %04o", got)
	}
}

func TestFileStoreRotatesPreviousSnapshot(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	first, err := store.Commit(validCandidate("N0CALL", 1234567, 1))
	if err != nil {
		t.Fatalf("first commit failed: %v", err)
	}
	second, err := store.Commit(validCandidate("N1TEST", 7654321, 2))
	if err != nil {
		t.Fatalf("second commit failed: %v", err)
	}
	if second.Revision != 2 {
		t.Fatalf("expected revision 2, got %d", second.Revision)
	}

	previous, err := readKnownGood(filepath.Join(dir, previousGoodFilename))
	if err != nil {
		t.Fatalf("read previous config: %v", err)
	}
	if previous != first {
		t.Fatalf("previous snapshot mismatch: got %+v want %+v", previous, first)
	}
}

func TestFileStoreRejectsInvalidCandidateWithoutChangingCurrent(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	first, err := store.Commit(validCandidate("N0CALL", 1234567, 1))
	if err != nil {
		t.Fatalf("first commit failed: %v", err)
	}

	_, err = store.Commit(validCandidate("BAD CALL", 0, 100))
	var candidateErr *CandidateError
	if !errors.As(err, &candidateErr) {
		t.Fatalf("expected CandidateError, got %v", err)
	}
	if len(candidateErr.Errors) != 3 {
		t.Fatalf("expected three field errors, got %+v", candidateErr.Errors)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load after rejected commit: %v", err)
	}
	if loaded.Config != first {
		t.Fatalf("invalid commit changed known-good state: got %+v want %+v", loaded.Config, first)
	}
}

func TestFileStoreRecoversFromCorruptCurrent(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	first, err := store.Commit(validCandidate("N0CALL", 1234567, 1))
	if err != nil {
		t.Fatalf("first commit failed: %v", err)
	}
	if _, err := store.Commit(validCandidate("N1TEST", 7654321, 2)); err != nil {
		t.Fatalf("second commit failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, knownGoodFilename), []byte("{broken\n"), 0o600); err != nil {
		t.Fatalf("corrupt current config: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("recovery load failed: %v", err)
	}
	if !loaded.RecoveredFromPrevious {
		t.Fatal("expected recovery from previous snapshot")
	}
	if loaded.Config != first {
		t.Fatalf("recovered wrong snapshot: got %+v want %+v", loaded.Config, first)
	}
}

func TestFileStoreRejectsUnsupportedStoredSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, knownGoodFilename)
	data := []byte(`{"schema":99}`)
	data = []byte(strings.ReplaceAll(string(data), `\"`, `"`))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	store := NewFileStore(dir)
	_, err := store.Load()
	if err == nil || !strings.Contains(err.Error(), "unsupported configuration schema 99") {
		t.Fatalf("expected explicit schema error, got %v", err)
	}
}
