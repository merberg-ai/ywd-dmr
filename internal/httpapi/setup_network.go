package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/config"
	"github.com/merberg-ai/ywd-dmr/internal/dmrnet"
	"github.com/merberg-ai/ywd-dmr/internal/security"
)

// registerNetworkRoutes exposes protected network candidate validation, a real
// non-persisting diagnostic test, and the durable test-and-commit transaction.
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

		candidate, identity, ok := s.readNetworkCandidateForTest(w, r)
		if !ok {
			return
		}
		result, ok := s.runNetworkTest(w, r, identity, candidate)
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, result)
	})))

	// test-and-commit deliberately performs both operations in one request. The
	// normalized candidate remains in daemon memory from local validation through
	// the real BrandMeister handshake and durable commit, avoiding a reusable
	// browser proof token and a time-of-check/time-of-use mismatch.
	s.mux.HandleFunc("/api/v1/setup/network/test-and-commit", s.postOnly(s.requireRole(security.RoleAdmin, func(w http.ResponseWriter, r *http.Request) {
		if s.configStore == nil || s.networkTester == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "network configuration service unavailable"})
			return
		}

		candidate, identity, ok := s.readNetworkCandidateForTest(w, r)
		if !ok {
			return
		}
		result, ok := s.runNetworkTest(w, r, identity, candidate)
		if !ok {
			return
		}
		if !result.OK {
			writeJSON(w, http.StatusOK, map[string]any{
				"committed": false,
				"test":      result,
			})
			return
		}

		committed, err := s.configStore.CommitNetwork(candidate)
		if err != nil {
			var candidateErr *config.CandidateError
			switch {
			case errors.As(err, &candidateErr):
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"error":  "invalid network candidate",
					"fields": candidateErr.Errors,
				})
			case errors.Is(err, config.ErrIdentityRequired):
				writeJSON(w, http.StatusConflict, map[string]string{"error": "station identity must be committed before network configuration"})
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error": "network test passed but durable commit failed",
					"test":  result,
				})
			}
			return
		}

		s.state.SetKnownGoodConfigurationDetails(committed.Revision, false, true)
		writeJSON(w, http.StatusOK, map[string]any{
			"committed": true,
			"revision":  committed.Revision,
			"network":   committed.Network,
			"test":      result,
		})
	})))
}

func (s *Server) readNetworkCandidateForTest(w http.ResponseWriter, r *http.Request) (config.NetworkCandidate, config.RadioIdentity, bool) {
	var input config.NetworkInput
	if err := readJSON(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
		return config.NetworkCandidate{}, config.RadioIdentity{}, false
	}
	candidate, validation := config.ValidateNetworkCandidate(input)
	if !validation.Valid {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "invalid network candidate",
			"fields": validation.Errors,
		})
		return config.NetworkCandidate{}, config.RadioIdentity{}, false
	}

	loaded, err := s.configStore.Load()
	if err != nil {
		if errors.Is(err, config.ErrNoKnownGoodConfig) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "station identity must be committed before testing a DMR network"})
			return config.NetworkCandidate{}, config.RadioIdentity{}, false
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "configuration service unavailable"})
		return config.NetworkCandidate{}, config.RadioIdentity{}, false
	}
	return candidate, loaded.Config.Identity, true
}

func (s *Server) runNetworkTest(w http.ResponseWriter, r *http.Request, identity config.RadioIdentity, candidate config.NetworkCandidate) (dmrnet.TestResult, bool) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	result, err := s.networkTester.Test(ctx, identity, candidate)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "network test could not be completed"})
		return dmrnet.TestResult{}, false
	}
	return result, true
}
