package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/core"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

type Server struct {
	state    *core.State
	security *security.Manager
	mux      *http.ServeMux
}

func New(state *core.State, webRoot, docsRoot string) http.Handler {
	return newServer(state, nil, webRoot, docsRoot)
}

func NewWithSecurity(state *core.State, securityManager *security.Manager, webRoot, docsRoot string) http.Handler {
	return newServer(state, securityManager, webRoot, docsRoot)
}

func newServer(state *core.State, securityManager *security.Manager, webRoot, docsRoot string) http.Handler {
	s := &Server{state: state, security: securityManager, mux: http.NewServeMux()}
	s.routes(webRoot, docsRoot)
	return securityHeaders(s.mux)
}

func (s *Server) routes(webRoot, docsRoot string) {
	s.mux.HandleFunc("/api/v1/health", s.getOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UTC()})
	}))
	s.mux.HandleFunc("/api/v1/system", s.getOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.state.System())
	}))
	s.mux.HandleFunc("/api/v1/status", s.getOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.state.Status())
	}))
	s.mux.HandleFunc("/api/v1/capabilities", s.getOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.state.Capabilities())
	}))
	s.mux.HandleFunc("/api/v1/setup/status", s.getOnly(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.state.SetupStatus())
	}))

	// Phase 2 starts with validation before persistence. This endpoint does not
	// mutate daemon state, create credentials, or commit configuration.
	s.mux.HandleFunc("/api/v1/setup/identity/validate", s.postOnly(func(w http.ResponseWriter, r *http.Request) {
		var input config.RadioIdentityInput
		if err := readJSON(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
			return
		}
		writeJSON(w, http.StatusOK, config.ValidateRadioIdentity(input))
	}))

	// Claim is the one deliberate unauthenticated mutation: a fresh appliance
	// can be claimed only with the high-entropy bootstrap secret available on
	// the local machine. Success creates the first administrator and consumes
	// that secret permanently.
	s.mux.HandleFunc("/api/v1/setup/claim", s.postOnly(func(w http.ResponseWriter, r *http.Request) {
		if s.security == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "security service unavailable"})
			return
		}
		var input security.ClaimRequest
		if err := readJSON(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
			return
		}
		session, err := s.security.Claim(input)
		if err != nil {
			var validation *security.ClaimValidationError
			switch {
			case errors.As(err, &validation):
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid claim request", "fields": validation.Errors})
			case errors.Is(err, security.ErrInvalidClaimCode):
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "claim failed"})
			case errors.Is(err, security.ErrAlreadyClaimed):
				writeJSON(w, http.StatusConflict, map[string]string{"error": "installation is already claimed"})
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "claim could not be completed"})
			}
			return
		}
		setSessionCookie(w, r, session.Token, session.Principal.Expires)
		s.state.SetClaimed(true)
		writeJSON(w, http.StatusCreated, map[string]any{
			"claimed":    true,
			"username":   session.Principal.Username,
			"role":       session.Principal.Role,
			"expires_at": session.Principal.Expires,
		})
	}))

	// Session inspection is read-only and safe for the first-run WebUI. The
	// opaque token remains only in the HttpOnly cookie and is never echoed back.
	s.mux.HandleFunc("/api/v1/auth/session", s.getOnly(func(w http.ResponseWriter, r *http.Request) {
		if s.security == nil {
			writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
			return
		}
		cookie, err := r.Cookie(security.SessionCookieName)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
			return
		}
		principal, err := s.security.Authenticate(cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": true,
			"username":      principal.Username,
			"role":          principal.Role,
			"expires_at":    principal.Expires,
		})
	}))

	s.registerAuthRoutes()

	if dirExists(docsRoot) {
		s.mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir(docsRoot))))
	}
	if dirExists(webRoot) {
		files := http.FileServer(http.Dir(webRoot))
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
			if clean == "." {
				clean = "index.html"
			}
			if _, err := os.Stat(filepath.Join(webRoot, clean)); err != nil {
				r.URL.Path = "/"
			}
			files.ServeHTTP(w, r)
		})
	}
}

func (s *Server) getOnly(fn http.HandlerFunc) http.HandlerFunc {
	return methodOnly(http.MethodGet, fn)
}

func (s *Server) postOnly(fn http.HandlerFunc) http.HandlerFunc {
	return methodOnly(http.MethodPost, fn)
}

func methodOnly(method string, fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		fn(w, r)
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request must contain exactly one JSON value")
		}
		return err
	}
	return nil
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	http.SetCookie(w, &http.Cookie{
		Name:     security.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  expires,
		MaxAge:   maxAge,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:")
		next.ServeHTTP(w, r)
	})
}
