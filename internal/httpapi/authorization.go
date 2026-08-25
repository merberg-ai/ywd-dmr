package httpapi

import (
	"context"
	"net/http"

	"github.com/merberg-ai/ywd-dmr/internal/security"
)

type principalContextKey struct{}

func (s *Server) requireRole(minimum security.Role, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.security == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "security service unavailable"})
			return
		}

		cookie, err := r.Cookie(security.SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		principal, err := s.security.Authenticate(cookie.Value)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		if !security.RoleAllows(principal.Role, minimum) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
		next(w, r.WithContext(ctx))
	}
}

func authenticatedPrincipal(r *http.Request) (security.Principal, bool) {
	principal, ok := r.Context().Value(principalContextKey{}).(security.Principal)
	return principal, ok
}
