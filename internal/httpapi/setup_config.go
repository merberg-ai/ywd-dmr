package httpapi

import (
	"errors"
	"net/http"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

// registerConfigurationRoutes exposes mutations only through the daemon-owned
// known-good configuration transaction path. The public identity-validation
// endpoint remains separate; protected network setup routes are registered from
// the same setup/configuration surface.
func (s *Server) registerConfigurationRoutes() {
	s.mux.HandleFunc("/api/v1/setup/identity/commit", s.postOnly(s.requireRole(security.RoleAdmin, func(w http.ResponseWriter, r *http.Request) {
		if s.configStore == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "configuration service unavailable"})
			return
		}

		var input config.RadioIdentityInput
		if err := readJSON(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
			return
		}

		committed, err := s.configStore.Commit(config.Candidate{Identity: input})
		if err != nil {
			var candidateErr *config.CandidateError
			switch {
			case errors.As(err, &candidateErr):
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"error":  "invalid configuration candidate",
					"fields": candidateErr.Errors,
				})
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "configuration could not be committed"})
			}
			return
		}

		// Durable commit is authoritative. Only after it succeeds may the
		// in-memory setup state advance. If a tested schema-2 network already
		// exists, the store preserves it across this identity revision.
		s.state.SetKnownGoodConfigurationDetails(committed.Revision, false, committed.Network != nil)
		response := map[string]any{
			"committed": true,
			"revision":  committed.Revision,
			"identity":  committed.Identity,
		}
		if committed.Network != nil {
			response["network_preserved"] = true
		}
		writeJSON(w, http.StatusOK, response)
	})))

	s.registerNetworkRoutes()
}
