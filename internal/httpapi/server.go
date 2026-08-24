package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/core"
)

type Server struct {
	state *core.State
	mux   *http.ServeMux
}

func New(state *core.State, webRoot, docsRoot string) http.Handler {
	s := &Server{state: state, mux: http.NewServeMux()}
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
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		fn(w, r)
	}
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
