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
	// Schema 1 is deliberately frozen as identity-only. Network persistence uses
	// schema 2 rather than silently extending an already validated schema.
	KnownGoodSchema        = 1
	KnownGoodNetworkSchema = 2

	knownGoodFilename    = "known-good.json"
	previousGoodFilename = "known-good.previous.json"
)

var ErrNoKnownGoodConfig = errors.New("no known-good configuration")

// Candidate is untrusted configuration proposed by a setup/control client.
// The legacy Commit entrypoint still accepts identity only; network commit uses
// its own tested transaction so a syntactically valid but unreachable network
// cannot be promoted to known-good state.
type Candidate struct {
	Identity RadioIdentityInput `json:"identity"`
}

// KnownGoodConfig is the normalized non-secret configuration format persisted
// by the daemon. Schema and Revision are daemon-owned metadata. Schema 1 is
// identity-only; schema 2 adds the non-secret tested network shape. Network
// credentials are stored separately and are never serialized here.
type KnownGoodConfig struct {
	Schema   int                  `json:"schema"`
	Revision uint64               `json:"revision"`
	Identity RadioIdentity        `json:"identity"`
	Network  *StoredNetworkConfig `json:"network,omitempty"`
}

// LoadResult tells callers whether the normal current snapshot or the previous
// rollback snapshot supplied the usable known-good configuration. A network
// password is available only to daemon internals and is explicitly excluded
// from JSON serialization.
type LoadResult struct {
	Config                KnownGoodConfig
	NetworkPassword       string `json:"-"`
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
// be read or validated, including its revision-bound network secret when schema
// 2 is active, a valid previous snapshot is returned as a recovery source.
func (s *FileStore) Load() (LoadResult, error) {
	current, currentErr := s.readSnapshot(s.currentPath())
	if currentErr == nil {
		return current, nil
	}

	previous, previousErr := s.readSnapshot(s.previousPath())
	if previousErr == nil {
		previous.RecoveredFromPrevious = true
		return previous, nil
	}

	// A schema-2 snapshot whose secret is missing also wraps os.ErrNotExist, but
	// that is a configuration error, not an empty appliance. Report "missing"
	// only when the snapshot files themselves truly do not exist.
	if fileMissing(s.currentPath()) && fileMissing(s.previousPath()) {
		return LoadResult{}, ErrNoKnownGoodConfig
	}

	return LoadResult{}, fmt.Errorf("no readable known-good configuration: current: %v; previous: %v", currentErr, previousErr)
}

func fileMissing(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist)
}

// Commit is retained as the protected identity-commit entrypoint. Once schema 2
// exists, an identity change preserves the already-tested network configuration
// and its secret instead of accidentally dropping it.
func (s *FileStore) Commit(candidate Candidate) (KnownGoodConfig, error) {
	return s.commitIdentity(candidate.Identity)
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

	switch cfg.Schema {
	case KnownGoodSchema:
		if cfg.Network != nil {
			return KnownGoodConfig{}, errors.New("configuration schema 1 must remain identity-only")
		}
	case KnownGoodNetworkSchema:
		if cfg.Network == nil {
			return KnownGoodConfig{}, errors.New("configuration schema 2 requires network configuration")
		}
	default:
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

	if cfg.Network != nil {
		if !cfg.Network.PasswordSet {
			return KnownGoodConfig{}, errors.New("stored network configuration is missing its secret marker")
		}
		candidate, validation := ValidateNetworkCandidate(NetworkInput{
			Backend:                 cfg.Network.Backend,
			MasterAddress:           cfg.Network.MasterAddress,
			MasterPort:              cfg.Network.MasterPort,
			RegistrationFrequencyHz: cfg.Network.RegistrationFrequencyHz,
			// The real revision-bound secret is validated by readSnapshot. This
			// placeholder lets the shared non-secret field validator normalize the
			// stored network shape without putting a password in known-good.json.
			Password: "stored-secret",
		})
		if !validation.Valid {
			return KnownGoodConfig{}, errors.New("stored network configuration is invalid")
		}
		normalized := storedNetworkFromCandidate(candidate)
		normalized.PasswordSet = true
		cfg.Network = &normalized
	}
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
