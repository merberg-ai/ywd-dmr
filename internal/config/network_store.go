package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	networkSecretSchema  = 1
	networkSecretDirName = "secrets"
)

var ErrIdentityRequired = errors.New("station identity must be committed before network configuration")

type networkSecretRecord struct {
	Schema   int    `json:"schema"`
	Revision uint64 `json:"revision"`
	Password string `json:"password"`
}

func (s *FileStore) secretDir() string {
	return filepath.Join(s.dir, networkSecretDirName)
}

func (s *FileStore) networkSecretPath(revision uint64) string {
	return filepath.Join(s.secretDir(), fmt.Sprintf("network-%d.json", revision))
}

func (s *FileStore) readSnapshot(path string) (LoadResult, error) {
	cfg, err := readKnownGood(path)
	if err != nil {
		return LoadResult{}, err
	}

	result := LoadResult{Config: cfg}
	if cfg.Network == nil {
		return result, nil
	}

	password, err := s.readNetworkSecret(cfg.Revision)
	if err != nil {
		return LoadResult{}, fmt.Errorf("load network secret for revision %d: %w", cfg.Revision, err)
	}
	candidate, validation := ValidateNetworkCandidate(NetworkInput{
		Backend:                 cfg.Network.Backend,
		MasterAddress:           cfg.Network.MasterAddress,
		MasterPort:              cfg.Network.MasterPort,
		RegistrationFrequencyHz: cfg.Network.RegistrationFrequencyHz,
		Password:                password,
	})
	if !validation.Valid {
		return LoadResult{}, fmt.Errorf("network secret for revision %d does not produce a valid stored candidate", cfg.Revision)
	}

	// readKnownGood already normalized the non-secret fields. Keep only the
	// validated secret in daemon memory; it is never part of a response type.
	result.NetworkPassword = candidate.Password
	return result, nil
}

func (s *FileStore) commitIdentity(input RadioIdentityInput) (KnownGoodConfig, error) {
	identity := ValidateRadioIdentity(input)
	if !identity.Valid {
		return KnownGoodConfig{}, &CandidateError{Errors: identity.Errors}
	}

	var existing *LoadResult
	loaded, err := s.Load()
	switch {
	case err == nil:
		existing = &loaded
	case errors.Is(err, ErrNoKnownGoodConfig):
		// First identity commit.
	default:
		return KnownGoodConfig{}, err
	}

	revision := uint64(1)
	next := KnownGoodConfig{
		Schema:   KnownGoodSchema,
		Revision: revision,
		Identity: identity.Normalized,
	}
	password := ""

	if existing != nil {
		next.Revision = existing.Config.Revision + 1
		if existing.Config.Network != nil {
			// Once a tested network exists, changing callsign/DMR identity must not
			// silently erase it. Rebind the same secret to the new config revision.
			networkCopy := *existing.Config.Network
			next.Schema = KnownGoodNetworkSchema
			next.Network = &networkCopy
			password = existing.NetworkPassword
		}
	}

	return s.commitPrepared(existing, next, password)
}

// CommitNetwork persists a network candidate only after the caller has tested
// this exact normalized candidate successfully. The store revalidates it as a
// final defense-in-depth check, but it never performs network I/O itself.
func (s *FileStore) CommitNetwork(candidate NetworkCandidate) (KnownGoodConfig, error) {
	normalized, validation := ValidateNetworkCandidate(NetworkInput{
		Backend:                 candidate.Backend,
		MasterAddress:           candidate.MasterAddress,
		MasterPort:              candidate.MasterPort,
		RegistrationFrequencyHz: candidate.RegistrationFrequencyHz,
		Password:                candidate.Password,
	})
	if !validation.Valid {
		return KnownGoodConfig{}, &CandidateError{Errors: validation.Errors}
	}

	loaded, err := s.Load()
	if err != nil {
		if errors.Is(err, ErrNoKnownGoodConfig) {
			return KnownGoodConfig{}, ErrIdentityRequired
		}
		return KnownGoodConfig{}, err
	}

	stored := storedNetworkFromCandidate(normalized)
	stored.PasswordSet = true
	next := KnownGoodConfig{
		Schema:   KnownGoodNetworkSchema,
		Revision: loaded.Config.Revision + 1,
		Identity: loaded.Config.Identity,
		Network:  &stored,
	}
	return s.commitPrepared(&loaded, next, normalized.Password)
}

func (s *FileStore) commitPrepared(existing *LoadResult, next KnownGoodConfig, networkPassword string) (KnownGoodConfig, error) {
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return KnownGoodConfig{}, fmt.Errorf("create configuration directory: %w", err)
	}
	if err := os.Chmod(s.dir, 0o750); err != nil {
		return KnownGoodConfig{}, fmt.Errorf("set configuration directory mode: %w", err)
	}

	// Write a revision-bound secret before publishing schema 2. If a later
	// config write fails this file is merely orphaned; no active snapshot refers
	// to it. This avoids ever publishing a network config whose secret is absent.
	if next.Network != nil {
		if networkPassword == "" {
			return KnownGoodConfig{}, errors.New("network configuration requires a stored secret")
		}
		if err := s.writeNetworkSecret(next.Revision, networkPassword); err != nil {
			return KnownGoodConfig{}, fmt.Errorf("write network secret: %w", err)
		}
	}

	if existing != nil && !existing.RecoveredFromPrevious {
		if err := writeKnownGoodAtomic(s.previousPath(), existing.Config); err != nil {
			return KnownGoodConfig{}, fmt.Errorf("write rollback configuration: %w", err)
		}
	}

	if err := writeKnownGoodAtomic(s.currentPath(), next); err != nil {
		return KnownGoodConfig{}, fmt.Errorf("write known-good configuration: %w", err)
	}

	keep := map[uint64]bool{}
	if next.Network != nil {
		keep[next.Revision] = true
	}
	if existing != nil && existing.Config.Network != nil {
		// The existing revision is either the newly rotated rollback snapshot or,
		// during recovery, the valid previous snapshot that we deliberately keep.
		keep[existing.Config.Revision] = true
	}
	// Cleanup is best-effort after the durable transaction. Stale revision-bound
	// secrets are never selected by Load because active snapshots name revisions.
	_ = s.cleanupNetworkSecrets(keep)

	return next, nil
}

func (s *FileStore) readNetworkSecret(revision uint64) (string, error) {
	data, err := os.ReadFile(s.networkSecretPath(revision))
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record networkSecretRecord
	if err := decoder.Decode(&record); err != nil {
		return "", err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return "", errors.New("network secret contains multiple JSON values")
		}
		return "", err
	}
	if record.Schema != networkSecretSchema {
		return "", fmt.Errorf("unsupported network secret schema %d", record.Schema)
	}
	if record.Revision != revision {
		return "", errors.New("network secret revision does not match configuration revision")
	}
	if record.Password == "" {
		return "", errors.New("network secret password is empty")
	}
	return record.Password, nil
}

func (s *FileStore) writeNetworkSecret(revision uint64, password string) error {
	secretDir := s.secretDir()
	if err := os.MkdirAll(secretDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(secretDir, 0o700); err != nil {
		return err
	}

	record := networkSecretRecord{
		Schema:   networkSecretSchema,
		Revision: revision,
		Password: password,
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(secretDir, ".ywd-dmr-network-secret-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.networkSecretPath(revision)); err != nil {
		return err
	}
	keep = true

	d, err := os.Open(secretDir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (s *FileStore) cleanupNetworkSecrets(keep map[uint64]bool) error {
	entries, err := os.ReadDir(s.secretDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "network-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		revisionText := strings.TrimSuffix(strings.TrimPrefix(name, "network-"), ".json")
		revision, err := strconv.ParseUint(revisionText, 10, 64)
		if err != nil || keep[revision] {
			continue
		}
		_ = os.Remove(filepath.Join(s.secretDir(), name))
	}
	return nil
}
