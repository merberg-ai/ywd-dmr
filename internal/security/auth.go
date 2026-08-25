package security

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotClaimed         = errors.New("installation is not claimed")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login verifies the durable administrator password record and creates a new
// opaque in-memory session. The password KDF is evaluated even when the
// username is wrong so the normal invalid-credential path does not become a
// cheap username oracle.
func (m *Manager) Login(req LoginRequest) (Session, error) {
	username := strings.ToLower(strings.TrimSpace(req.Username))

	m.mu.Lock()
	if !m.claimed {
		m.mu.Unlock()
		return Session{}, ErrNotClaimed
	}
	admin := m.admin
	m.mu.Unlock()

	passwordOK, err := verifyPassword(admin.Password, req.Password)
	if err != nil {
		return Session{}, fmt.Errorf("verify administrator password: %w", err)
	}
	usernameOK := secureStringEqual(username, admin.Username)
	if !usernameOK || !passwordOK {
		return Session{}, ErrInvalidCredentials
	}

	token, tokenHash, err := newSessionToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(SessionLifetime)

	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.claimed || m.admin.Username != admin.Username || m.admin.Password != admin.Password {
		return Session{}, ErrInvalidCredentials
	}
	m.sessions[tokenHash] = sessionRecord{
		Username: admin.Username,
		Role:     admin.Role,
		Expires:  expires,
	}

	return Session{
		Token: token,
		Principal: Principal{
			Username: admin.Username,
			Role:     admin.Role,
			Expires:  expires,
		},
	}, nil
}

func verifyPassword(record passwordRecord, password string) (bool, error) {
	if record.Algorithm != "pbkdf2-sha256" || record.Iterations <= 0 {
		return false, errors.New("unsupported password verifier")
	}
	salt, err := base64.RawStdEncoding.DecodeString(record.Salt)
	if err != nil {
		return false, errors.New("invalid password salt")
	}
	expected, err := base64.RawStdEncoding.DecodeString(record.Hash)
	if err != nil || len(expected) != passwordKeyLength {
		return false, errors.New("invalid password hash")
	}
	actual, err := pbkdf2.Key(sha256.New, password, salt, record.Iterations, len(expected))
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
