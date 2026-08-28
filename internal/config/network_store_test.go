package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func validNetworkCandidate(password string) NetworkCandidate {
	return NetworkCandidate{
		Backend:                 NetworkBackendBrandMeister,
		MasterAddress:           "3103.master.brandmeister.network",
		MasterPort:              BrandMeisterDefaultPort,
		RegistrationFrequencyHz: 446_525_000,
		Password:                password,
	}
}

func TestFileStoreNetworkCommitCreatesSchema2WithoutSecretInKnownGood(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	if _, err := store.Commit(validCandidate("N0CALL", 1234567, 3)); err != nil {
		t.Fatal(err)
	}

	secret := "hotspot-secret"
	committed, err := store.CommitNetwork(validNetworkCandidate(secret))
	if err != nil {
		t.Fatalf("network commit failed: %v", err)
	}
	if committed.Schema != KnownGoodNetworkSchema || committed.Revision != 2 {
		t.Fatalf("unexpected committed metadata: %+v", committed)
	}
	if committed.Network == nil || !committed.Network.PasswordSet {
		t.Fatalf("expected non-secret network summary: %+v", committed.Network)
	}

	knownGoodData, err := os.ReadFile(filepath.Join(dir, knownGoodFilename))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(knownGoodData), secret) {
		t.Fatal("known-good configuration leaked the network password")
	}
	if !strings.Contains(string(knownGoodData), `"schema": 2`) {
		t.Fatalf("expected schema 2 known-good document: %s", knownGoodData)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.NetworkPassword != secret {
		t.Fatal("loaded network secret did not match committed secret")
	}
	if !reflect.DeepEqual(loaded.Config, committed) {
		t.Fatalf("loaded config mismatch: got %+v want %+v", loaded.Config, committed)
	}

	secretPath := store.networkSecretPath(committed.Revision)
	secretInfo, err := os.Stat(secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := secretInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected secret file mode 0600, got %04o", got)
	}
	secretDirInfo, err := os.Stat(store.secretDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := secretDirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected secret directory mode 0700, got %04o", got)
	}
}

func TestFileStoreNetworkCommitRotatesAndRecoversMatchingSecret(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	if _, err := store.Commit(validCandidate("N0CALL", 1234567, 3)); err != nil {
		t.Fatal(err)
	}

	firstPassword := "first-secret"
	first, err := store.CommitNetwork(validNetworkCandidate(firstPassword))
	if err != nil {
		t.Fatal(err)
	}
	secondCandidate := validNetworkCandidate("second-secret")
	secondCandidate.MasterAddress = "3102.master.brandmeister.network"
	second, err := store.CommitNetwork(secondCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != 2 || second.Revision != 3 {
		t.Fatalf("unexpected revisions: first=%d second=%d", first.Revision, second.Revision)
	}

	previous, err := readKnownGood(filepath.Join(dir, previousGoodFilename))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(previous, first) {
		t.Fatalf("previous snapshot mismatch: got %+v want %+v", previous, first)
	}

	if err := os.WriteFile(filepath.Join(dir, knownGoodFilename), []byte("{broken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("recovery load failed: %v", err)
	}
	if !loaded.RecoveredFromPrevious {
		t.Fatal("expected recovery from previous network snapshot")
	}
	if !reflect.DeepEqual(loaded.Config, first) {
		t.Fatalf("recovered wrong network snapshot: got %+v want %+v", loaded.Config, first)
	}
	if loaded.NetworkPassword != firstPassword {
		t.Fatal("recovery did not load the secret bound to the previous revision")
	}
}

func TestIdentityCommitAfterNetworkPreservesNetworkAndRebindsSecret(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	if _, err := store.Commit(validCandidate("N0CALL", 1234567, 3)); err != nil {
		t.Fatal(err)
	}
	secret := "preserve-me"
	before, err := store.CommitNetwork(validNetworkCandidate(secret))
	if err != nil {
		t.Fatal(err)
	}

	after, err := store.Commit(validCandidate("N1TEST", 7654321, 4))
	if err != nil {
		t.Fatal(err)
	}
	if after.Schema != KnownGoodNetworkSchema || after.Revision != before.Revision+1 {
		t.Fatalf("unexpected identity-after-network metadata: %+v", after)
	}
	if !reflect.DeepEqual(after.Network, before.Network) {
		t.Fatalf("identity commit changed tested network: before=%+v after=%+v", before.Network, after.Network)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.NetworkPassword != secret {
		t.Fatal("identity commit did not preserve/rebind the network secret")
	}
}

func TestSchema1CannotContainNetworkFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, knownGoodFilename)
	data := []byte(`{
  "schema": 1,
  "revision": 1,
  "identity": {"callsign":"N0CALL","dmr_id":1234567,"essid":1},
  "network": {"backend":"brandmeister","master_address":"3103.master.brandmeister.network","master_port":62031,"registration_frequency_hz":446525000,"password_set":true}
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileStore(dir).Load(); err == nil || !strings.Contains(err.Error(), "schema 1 must remain identity-only") {
		t.Fatalf("expected schema-1 network rejection, got %v", err)
	}
}
