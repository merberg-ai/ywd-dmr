package httpapi

import (
	"net/http"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

// registerNetworkRoutes starts the BrandMeister setup surface with protected,
// non-mutating candidate validation. A live connectivity-test endpoint is not
// exposed until a real backend probe exists; validation alone must never be
// presented as proof that credentials or a master server work.
func (s *Server) registerNetworkRoutes() {
	s.mux.HandleFunc("/api/v1/setup/network/validate", s.postOnly(s.requireRole(security.RoleAdmin, func(w http.ResponseWriter, r *http.Request) {
		var input config.NetworkInput
		if err := readJSON(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
			return
		}

		_, result := config.ValidateNetworkCandidate(input)
		writeJSON(w, http.StatusOK, result)
	})))
}
