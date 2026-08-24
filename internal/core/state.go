package core

import (
	"runtime"
	"sync"
	"time"
)

type SetupConfigStatus struct {
	State              string `json:"state"`
	Revision           uint64 `json:"revision,omitempty"`
	IdentityConfigured bool   `json:"identity_configured"`
	Recovered          bool   `json:"recovered"`
}

type SetupStatus struct {
	Claimed       bool              `json:"claimed"`
	Stage         string            `json:"stage"`
	NextStep      string            `json:"next_step"`
	Configuration SetupConfigStatus `json:"configuration"`
}

type State struct {
	mu        sync.RWMutex
	startedAt time.Time
	version   string
	commit    string
	branch    string
	setup     SetupStatus
}

func NewState(version, commit, branch string) *State {
	return &State{
		startedAt: time.Now().UTC(),
		version:   version,
		commit:    commit,
		branch:    branch,
		setup: SetupStatus{
			Claimed:  false,
			Stage:    "unclaimed",
			NextStep: "claim",
			Configuration: SetupConfigStatus{
				State:              "missing",
				IdentityConfigured: false,
				Recovered:          false,
			},
		},
	}
}

func (s *State) System() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"name":       "YWD-DMR",
		"version":    s.version,
		"commit":     s.commit,
		"branch":     s.branch,
		"started_at": s.startedAt,
		"go_version": runtime.Version(),
		"goos":       runtime.GOOS,
		"goarch":     runtime.GOARCH,
		"cpus":       runtime.NumCPU(),
	}
}

func (s *State) Status() map[string]any {
	return map[string]any{
		"state":       "idle",
		"network":     map[string]any{"backend": "none", "connected": false},
		"vocoder":     map[string]any{"backend": "none", "available": false},
		"audio":       map[string]any{"input": "none", "output": "none"},
		"destination": nil,
		"rx":          false,
		"tx":          false,
	}
}

func (s *State) SetupStatus() SetupStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.setup
}

func (s *State) SetKnownGoodConfiguration(revision uint64, recovered bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setup.Configuration = SetupConfigStatus{
		State:              map[bool]string{true: "recovered", false: "loaded"}[recovered],
		Revision:           revision,
		IdentityConfigured: true,
		Recovered:          recovered,
	}
}

func (s *State) SetConfigurationLoadError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setup.Configuration = SetupConfigStatus{
		State:              "error",
		IdentityConfigured: false,
		Recovered:          false,
	}
}

func (s *State) Capabilities() map[string]any {
	return map[string]any{
		"control_api":        1,
		"event_api":          1,
		"audio_stream":       1,
		"vocoder_api":        1,
		"tx":                 false,
		"rx_voice":           false,
		"no_vocoder_mode":    true,
		"browser_audio":      false,
		"local_audio":        false,
		"network_management": false,
		"development_stub":   true,
	}
}
