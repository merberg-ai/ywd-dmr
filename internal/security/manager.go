package security

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	SecuritySchema     = 1
	passwordIterations = 310_000
	passwordKeyLength  = 32
	claimRandomBytes   = 15 // 120 bits, rendered as 24 base32 characters.
	sessionRandomBytes = 32
	SessionCookieName  = "ywd_dmr_session"
	SessionLifetime    = 12 * time.Hour

	securityFilename  = "security.json"
	claimCodeFilename = "claim-code"
)

var (
	ErrAlreadyClaimed   = errors.New("installation is already claimed")
	ErrInvalidClaimCode = errors.New("invalid claim code")
	ErrInvalidSession   = errors.New("invalid session")
)

type InitResult struct {
	Claimed bool
}

type ClaimRequest struct {
	ClaimCode string `json:"claim_code"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type AdminValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ClaimValidationError struct {
	Errors []AdminValidationError
}

func (e *ClaimValidationError) Error() string {
	return "claim request is invalid"
}

type Principal struct {
	Username string    `json:"username"`
	Role     string    `json:"role"`
	Expires  time.Time `json:"expires_at"`
}

type Session struct {
	Token     string
	Principal Principal
}

type passwordRecord struct {
	Algorithm  string `json:"algorithm"`
	Iterations int    `json:"iterations"`
	Salt       string `json:"salt"`
	Hash       string `json:"hash"`
}

type adminRecord struct {
	Username string         `json:"username"`
	Role     string         `json:"role"`
	Password passwordRecord `json:"password"`
}

type securityState struct {
	Schema    int         `json:"schema"`
	ClaimedAt time.Time   `json:"claimed_at"`
	Admin     adminRecord `json:"admin"`
}

type sessionRecord struct {
	Username string
	Role     string
	Expires  time.Time
}

type Manager struct {
	dir      string
	mu       sync.Mutex
	claimed  bool
	admin    adminRecord
	sessions map[string]sessionRecord
}

func NewManager(dir string) *Manager {
	return &Manager{dir: dir, sessions: make(map[string]sessionRecord)}
}

func (m *Manager) securityPath() string {
	return filepath.Join(m.dir, securityFilename)
}

func (m *Manager) ClaimCodePath() string {
	return filepath.Join(m.dir, claimCodeFilename)
}

// Initialize establishes a safe bootstrap state. A malformed existing
// security file is an error: the daemon must never silently turn corruption
// into a new unclaimed installation that could be claimed by someone else.
func (m *Manager) Initialize() (InitResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.dir, 0o750); err != nil {
		return InitResult{}, fmt.Errorf("create security state directory: %w", err)
	}
	if err := os.Chmod(m.dir, 0o750); err != nil {
		return InitResult{}, fmt.Errorf("set security state directory mode: %w", err)
	}

	state, err := readSecurityState(m.securityPath())
	if err == nil {
		m.claimed = true
		m.admin = state.Admin
		_ = os.Remove(m.ClaimCodePath())
		return InitResult{Claimed: true}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return InitResult{}, fmt.Errorf("load security state: %w", err)
	}

	if _, err := m.ensureClaimCodeLocked(); err != nil {
		return InitResult{}, err
	}
	m.claimed = false
	return InitResult{Claimed: false}, nil
}

func (m *Manager) Claimed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.claimed
}

func (m *Manager) Claim(req ClaimRequest) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.claimed {
		return Session{}, ErrAlreadyClaimed
	}

	storedCode, err := os.ReadFile(m.ClaimCodePath())
	if err != nil {
		return Session{}, fmt.Errorf("read claim code: %w", err)
	}
	if !secureStringEqual(normalizeClaimCode(req.ClaimCode), normalizeClaimCode(string(storedCode))) {
		return Session{}, ErrInvalidClaimCode
	}

	username, validationErrors := validateAdmin(req.Username, req.Password)
	if len(validationErrors) != 0 {
		return Session{}, &ClaimValidationError{Errors: validationErrors}
	}

	password, err := newPasswordRecord(req.Password)
	if err != nil {
		return Session{}, err
	}
	token, tokenHash, err := newSessionToken()
	if err != nil {
		return Session{}, err
	}

	now := time.Now().UTC()
	state := securityState{
		Schema:    SecuritySchema,
		ClaimedAt: now,
		Admin: adminRecord{
			Username: username,
			Role:     "admin",
			Password: password,
		},
	}
	if err := writeSecurityStateAtomic(m.securityPath(), state); err != nil {
		return Session{}, fmt.Errorf("persist claimed security state: %w", err)
	}
	if err := os.Remove(m.ClaimCodePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Session{}, fmt.Errorf("remove used claim code: %w", err)
	}

	expires := now.Add(SessionLifetime)
	m.sessions[tokenHash] = sessionRecord{Username: username, Role: "admin", Expires: expires}
	m.claimed = true
	m.admin = state.Admin

	return Session{
		Token: token,
		Principal: Principal{
			Username: username,
			Role:     "admin",
			Expires:  expires,
		},
	}, nil
}

func (m *Manager) Authenticate(token string) (Principal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if token == "" {
		return Principal{}, ErrInvalidSession
	}
	hash := sessionHash(token)
	record, ok := m.sessions[hash]
	if !ok {
		return Principal{}, ErrInvalidSession
	}
	if !time.Now().UTC().Before(record.Expires) {
		delete(m.sessions, hash)
		return Principal{}, ErrInvalidSession
	}
	return Principal{Username: record.Username, Role: record.Role, Expires: record.Expires}, nil
}

func (m *Manager) InvalidateSession(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if token != "" {
		delete(m.sessions, sessionHash(token))
	}
}

func (m *Manager) ensureClaimCodeLocked() (string, error) {
	if data, err := os.ReadFile(m.ClaimCodePath()); err == nil {
		code := normalizeClaimCode(string(data))
		if len(code) == 24 {
			return formatClaimCode(code), nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read existing claim code: %w", err)
	}

	randomBytes := make([]byte, claimRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate claim code: %w", err)
	}
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	formatted := formatClaimCode(code)
	if err := writePrivateFileAtomic(m.ClaimCodePath(), []byte(formatted+"\n")); err != nil {
		return "", fmt.Errorf("write claim code: %w", err)
	}
	return formatted, nil
}

func validateAdmin(usernameInput, password string) (string, []AdminValidationError) {
	username := strings.ToLower(strings.TrimSpace(usernameInput))
	errs := make([]AdminValidationError, 0, 2)

	if len(username) < 3 || len(username) > 32 || !validUsername(username) {
		errs = append(errs, AdminValidationError{
			Field:   "username",
			Code:    "invalid_username",
			Message: "Username must be 3 to 32 letters, numbers, dots, dashes, or underscores.",
		})
	}
	if len(password) < 12 || len(password) > 256 {
		errs = append(errs, AdminValidationError{
			Field:   "password",
			Code:    "invalid_password",
			Message: "Administrator password must be 12 to 256 characters long.",
		})
	}
	return username, errs
}

func validUsername(username string) bool {
	for _, r := range username {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func newPasswordRecord(password string) (passwordRecord, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return passwordRecord{}, fmt.Errorf("generate password salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyLength)
	if err != nil {
		return passwordRecord{}, fmt.Errorf("derive password hash: %w", err)
	}
	return passwordRecord{
		Algorithm:  "pbkdf2-sha256",
		Iterations: passwordIterations,
		Salt:       base64.RawStdEncoding.EncodeToString(salt),
		Hash:       base64.RawStdEncoding.EncodeToString(key),
	}, nil
}

func readSecurityState(path string) (securityState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return securityState{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var state securityState
	if err := decoder.Decode(&state); err != nil {
		return securityState{}, fmt.Errorf("decode security state: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return securityState{}, errors.New("security state contains multiple JSON values")
		}
		return securityState{}, fmt.Errorf("decode security state: %w", err)
	}
	if state.Schema != SecuritySchema {
		return securityState{}, fmt.Errorf("unsupported security schema %d", state.Schema)
	}
	if state.Admin.Role != "admin" || state.Admin.Username == "" {
		return securityState{}, errors.New("security state does not contain a valid administrator")
	}
	if state.Admin.Password.Algorithm != "pbkdf2-sha256" || state.Admin.Password.Iterations <= 0 {
		return securityState{}, errors.New("security state contains unsupported password metadata")
	}
	if _, err := base64.RawStdEncoding.DecodeString(state.Admin.Password.Salt); err != nil {
		return securityState{}, errors.New("security state contains invalid password salt")
	}
	if hash, err := base64.RawStdEncoding.DecodeString(state.Admin.Password.Hash); err != nil || len(hash) != passwordKeyLength {
		return securityState{}, errors.New("security state contains invalid password hash")
	}
	return state, nil
}

func newSessionToken() (token string, hash string, err error) {
	buf := make([]byte, sessionRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, sessionHash(token), nil
}

func sessionHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func normalizeClaimCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}

func formatClaimCode(code string) string {
	code = normalizeClaimCode(code)
	if len(code) != 24 {
		return code
	}
	return code[0:6] + "-" + code[6:12] + "-" + code[12:18] + "-" + code[18:24]
}

func secureStringEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func writeSecurityStateAtomic(path string, state securityState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writePrivateFileAtomic(path, data)
}

func writePrivateFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ywd-dmr-security-*")
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
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
