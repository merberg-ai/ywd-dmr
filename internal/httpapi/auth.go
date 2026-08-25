package httpapi

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/security"
)

const (
	loginFailureLimit  = 5
	loginFailureWindow = 5 * time.Minute
	loginBlockDuration = 1 * time.Minute
)

type loginAttemptState struct {
	Failures     int
	WindowStart  time.Time
	BlockedUntil time.Time
}

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttemptState
	now      func() time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{
		attempts: make(map[string]loginAttemptState),
		now:      time.Now,
	}
}

func (l *loginLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	state, ok := l.attempts[key]
	if !ok {
		return true, 0
	}
	if state.BlockedUntil.After(now) {
		return false, state.BlockedUntil.Sub(now)
	}
	if !state.WindowStart.IsZero() && now.Sub(state.WindowStart) >= loginFailureWindow {
		delete(l.attempts, key)
	}
	return true, 0
}

func (l *loginLimiter) Failure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	state := l.attempts[key]
	if state.WindowStart.IsZero() || now.Sub(state.WindowStart) >= loginFailureWindow {
		state = loginAttemptState{WindowStart: now}
	}
	state.Failures++
	if state.Failures >= loginFailureLimit {
		state.Failures = 0
		state.WindowStart = now
		state.BlockedUntil = now.Add(loginBlockDuration)
	}
	l.attempts[key] = state
}

func (l *loginLimiter) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func loginClientKey(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

func (s *Server) registerAuthRoutes() {
	limiter := newLoginLimiter()

	s.mux.HandleFunc("/api/v1/auth/login", s.postOnly(func(w http.ResponseWriter, r *http.Request) {
		if s.security == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "security service unavailable"})
			return
		}

		key := loginClientKey(r)
		if allowed, retryAfter := limiter.Allow(key); !allowed {
			seconds := int(retryAfter.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "login temporarily unavailable"})
			return
		}

		var input security.LoginRequest
		if err := readJSON(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
			return
		}

		session, err := s.security.Login(input)
		if err != nil {
			switch {
			case errors.Is(err, security.ErrNotClaimed):
				writeJSON(w, http.StatusConflict, map[string]string{"error": "installation is not claimed"})
			case errors.Is(err, security.ErrInvalidCredentials):
				limiter.Failure(key)
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication failed"})
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login could not be completed"})
			}
			return
		}

		limiter.Success(key)
		setSessionCookie(w, r, session.Token, session.Principal.Expires)
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"username":      session.Principal.Username,
			"role":          session.Principal.Role,
			"expires_at":    session.Principal.Expires,
		})
	}))

	s.mux.HandleFunc("/api/v1/auth/logout", s.postOnly(func(w http.ResponseWriter, r *http.Request) {
		if s.security != nil {
			if cookie, err := r.Cookie(security.SessionCookieName); err == nil {
				s.security.InvalidateSession(cookie.Value)
			}
		}
		clearSessionCookie(w, r)
		w.WriteHeader(http.StatusNoContent)
	}))
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     security.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
	})
}
