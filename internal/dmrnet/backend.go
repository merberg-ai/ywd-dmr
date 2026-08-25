package dmrnet

import (
	"context"
	"time"

	"github.com/merberg-ai/ywd-dmr/internal/config"
)

// TestReason is a stable machine-readable reason for a network configuration
// test result. The first BrandMeister backend will map the Homebrew login
// handshake into these stages so clients can give useful troubleshooting help.
type TestReason string

const (
	TestReasonOK          TestReason = "ok"
	TestReasonLogin       TestReason = "login"
	TestReasonAuth        TestReason = "auth"
	TestReasonConfig      TestReason = "config"
	TestReasonTimeout     TestReason = "timeout"
	TestReasonNetwork     TestReason = "network"
	TestReasonUnavailable TestReason = "unavailable"
)

// TestResult is safe to expose to clients. It must never contain the submitted
// network password or protocol challenge/response material.
type TestResult struct {
	OK         bool       `json:"ok"`
	Backend    string     `json:"backend"`
	Reason     TestReason `json:"reason"`
	Message    string     `json:"message"`
	DurationMS int64      `json:"duration_ms"`
}

// Tester proves that a normalized network candidate can complete the backend's
// real setup/connectivity check using the already-known station identity. A
// successful result is a prerequisite for the future durable network commit.
type Tester interface {
	Test(ctx context.Context, identity config.RadioIdentity, candidate config.NetworkCandidate) (TestResult, error)
}

func DurationMilliseconds(start time.Time) int64 {
	ms := time.Since(start).Milliseconds()
	if ms < 0 {
		return 0
	}
	return ms
}
