package config

import (
	"strings"
	"testing"
)

func TestValidateNetworkCandidateNormalizesBrandMeister(t *testing.T) {
	candidate, result := ValidateNetworkCandidate(NetworkInput{
		Backend:       " BrandMeister ",
		MasterAddress: " BM3103.EXAMPLE.NET. ",
		MasterPort:    0,
		Password:      "radio-secret",
	})

	if !result.Valid {
		t.Fatalf("expected valid result: %+v", result.Errors)
	}
	if candidate.Backend != NetworkBackendBrandMeister {
		t.Fatalf("unexpected backend %q", candidate.Backend)
	}
	if candidate.MasterAddress != "bm3103.example.net" {
		t.Fatalf("unexpected master address %q", candidate.MasterAddress)
	}
	if candidate.MasterPort != BrandMeisterDefaultPort {
		t.Fatalf("unexpected master port %d", candidate.MasterPort)
	}
	if candidate.Password != "radio-secret" {
		t.Fatal("password must be preserved internally for the future connection test")
	}
	if result.Normalized.PasswordSet != true {
		t.Fatal("expected password_set=true")
	}
}

func TestNetworkValidationDoesNotEchoPassword(t *testing.T) {
	secret := "do-not-echo-this-password"
	_, result := ValidateNetworkCandidate(NetworkInput{
		Backend:       "brandmeister",
		MasterAddress: "127.0.0.1",
		MasterPort:    62031,
		Password:      secret,
	})

	serialized := result.Normalized.Backend + result.Normalized.MasterAddress
	if strings.Contains(serialized, secret) {
		t.Fatal("network validation summary leaked password")
	}
}

func TestValidateNetworkCandidateRejectsInvalidFields(t *testing.T) {
	_, result := ValidateNetworkCandidate(NetworkInput{
		Backend:       "other-network",
		MasterAddress: "https://master.example:62031/path",
		MasterPort:    70000,
		Password:      "bad\npassword",
	})

	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if len(result.Errors) != 4 {
		t.Fatalf("expected four field errors, got %d: %+v", len(result.Errors), result.Errors)
	}
}

func TestValidateNetworkCandidateAcceptsIPv6(t *testing.T) {
	candidate, result := ValidateNetworkCandidate(NetworkInput{
		Backend:       "brandmeister",
		MasterAddress: "2001:db8::1",
		MasterPort:    62031,
		Password:      "secret",
	})
	if !result.Valid {
		t.Fatalf("expected IPv6 to be valid: %+v", result.Errors)
	}
	if candidate.MasterAddress != "2001:db8::1" {
		t.Fatalf("unexpected normalized IPv6 %q", candidate.MasterAddress)
	}
}
