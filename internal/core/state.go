package core

import (
	"runtime"
	"sync"
	"time"
)

type State struct {
	mu        sync.RWMutex
	startedAt time.Time
	version   string
	commit    string
	branch    string
}

func NewState(version, commit, branch string) *State {
	return &State{startedAt: time.Now().UTC(), version: version, commit: commit, branch: branch}
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
