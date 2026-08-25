package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

// registerNetworkRoutes exposes protected network candidate validation and a
// real, non-persisting connectivity/authentication test. Durable network commit
// remains closed until this live probe is validated on the appliance.
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

	s.mux.HandleFunc("/api/v1/setup/network/test", s.postOnly(s.requireRole(security.RoleAdmin, func(w http.ResponseWriter, r *http.Request) {
		if s.configStore == nil || s.networkTester == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "network test service unavailable"})
			return
		}

		var input config.NetworkInput
		if err := readJSON(w, r, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
			return
		}
		candidate, validation := config.ValidateNetworkCandidate(input)
		if !validation.Valid {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error":  "invalid network candidate",
				"fields": validation.Errors,
			})
			return
		}

		loaded, err := s.configStore.Load()
		if err != nil {
			if errors.Is(err, config.ErrNoKnownGoodConfig) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "station identity must be committed before testing a DMR network"})
				return
			}
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "configuration service unavailable"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		result, err := s.networkTester.Test(ctx, loaded.Config.Identity, candidate)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "network test could not be completed"})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})))
}
