package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	KnownGoodSchema      = 1
	knownGoodFilename    = "known-good.json"
	previousGoodFilename = "known-good.previous.json"
)

var ErrNoKnownGoodConfig = errors.New("no known-good configuration")

// Candidate is untrusted configuration proposed by a setup/control client.
// More sections will be added as network, audio, and vocoder configuration land.
type Candidate struct {
	Identity RadioIdentityInput `json:"identity"`
}

// KnownGoodConfig is the normalized configuration format persisted by the
// daemon. Schema and Revision are daemon-owned metadata and are never supplied
// by a client candidate.
type KnownGoodConfig struct {
	Schema   int           `json:"schema"`
	Revision uint64        `json:"revision"`
	Identity RadioIdentity `json:"identity"`
}

// LoadResult tells callers whether the normal current snapshot or the previous
// rollback snapshot supplied the usable known-good configuration.
type LoadResult struct {
	Config                KnownGoodConfig
	RecoveredFromPrevious bool
}

// CandidateError keeps field-level validation failures available to future API
// handlers without making validation failures look like disk/storage failures.
type CandidateError struct {
	Errors []FieldError
}

func (e *CandidateError) Error() string {
	return "candidate configuration is invalid"
}

// FileStore owns the small durable known-good configuration snapshot. Event
// history may use a database later; this file stays intentionally simple and
// recoverable with ordinary Linux tools.
type FileStore struct {
	dir string
}

func NewFileStore(dir string) *FileStore {
	return &FileStore{dir: dir}
}

func (s *FileStore) currentPath() string {
	return filepath.Join(s.dir, knownGoodFilename)
}

func (s *FileStore) previousPath() string {
	return filepath.Join(s.dir, previousGoodFilename)
}

// Load returns the current known-good snapshot. If the current snapshot cannot
// be read or validated, a valid previous snapshot is returned as a recovery
// source. The caller can then surface that degraded/recovered state explicitly.
func (s *FileStore) Load() (LoadResult, error) {
	current, currentErr := readKnownGood(s.currentPath())
	if currentErr == nil {
		return LoadResult{Config: current}, nil
	}

	previous, previousErr := readKnownGood(s.previousPath())
	if previousErr == nil {
		return LoadResult{Config: previous, RecoveredFromPrevious: true}, nil
	}

	if errors.Is(currentErr, os.ErrNotExist) && errors.Is(previousErr, os.ErrNotExist) {
		return LoadResult{}, ErrNoKnownGoodConfig
	}

	return LoadResult{}, fmt.Errorf("no readable known-good configuration: current: %v; previous: %v", currentErr, previousErr)
}

// Commit validates and normalizes a candidate before changing durable state.
// A successful second/subsequent commit first writes the old known-good value
// to the rollback snapshot, then atomically replaces the current snapshot.
func (s *FileStore) Commit(candidate Candidate) (KnownGoodConfig, error) {
	identity := ValidateRadioIdentity(candidate.Identity)
	if !identity.Valid {
		return KnownGoodConfig{}, &CandidateError{Errors: identity.Errors}
	}

	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return KnownGoodConfig{}, fmt.Errorf("create configuration directory: %w", err)
	}
	if err := os.Chmod(s.dir, 0o750); err != nil {
		return KnownGoodConfig{}, fmt.Errorf("set configuration directory mode: %w", err)
	}

	var revision uint64 = 1
	if existing, err := s.Load(); err == nil {
		revision = existing.Config.Revision + 1
		// Do not overwrite a known-good previous snapshot with a corrupt current
		// file during recovery. Only rotate when the current snapshot itself was
		// the successfully loaded source.
		if !existing.RecoveredFromPrevious {
			if err := writeKnownGoodAtomic(s.previousPath(), existing.Config); err != nil {
				return KnownGoodConfig{}, fmt.Errorf("write rollback configuration: %w", err)
			}
		}
	} else if !errors.Is(err, ErrNoKnownGoodConfig) {
		return KnownGoodConfig{}, err
	}

	committed := KnownGoodConfig{
		Schema:   KnownGoodSchema,
		Revision: revision,
		Identity: identity.Normalized,
	}
	if err := writeKnownGoodAtomic(s.currentPath(), committed); err != nil {
		return KnownGoodConfig{}, fmt.Errorf("write known-good configuration: %w", err)
	}
	return committed, nil
}

func readKnownGood(path string) (KnownGoodConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return KnownGoodConfig{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var cfg KnownGoodConfig
	if err := decoder.Decode(&cfg); err != nil {
		return KnownGoodConfig{}, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return KnownGoodConfig{}, fmt.Errorf("decode %s: multiple JSON values", filepath.Base(path))
		}
		return KnownGoodConfig{}, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}

	if cfg.Schema != KnownGoodSchema {
		return KnownGoodConfig{}, fmt.Errorf("unsupported configuration schema %d", cfg.Schema)
	}
	if cfg.Revision == 0 {
		return KnownGoodConfig{}, errors.New("configuration revision must be greater than zero")
	}
	identity := ValidateRadioIdentity(RadioIdentityInput{
		Callsign: cfg.Identity.Callsign,
		DMRID:    cfg.Identity.DMRID,
		ESSID:    cfg.Identity.ESSID,
	})
	if !identity.Valid {
		return KnownGoodConfig{}, errors.New("stored radio identity is invalid")
	}
	cfg.Identity = identity.Normalized
	return cfg, nil
}

func writeKnownGoodAtomic(path string, cfg KnownGoodConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".ywd-dmr-config-*")
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
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	keep = true

	// Sync the containing directory so the rename itself reaches stable storage
	// on filesystems that support directory fsync.
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return err
	}
	return nil
}
