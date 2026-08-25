package config

import (
	"strings"
	"testing"
)

func TestValidateNetworkCandidateNormalizesBrandMeister(t *testing.T) {
	candidate, result := ValidateNetworkCandidate(NetworkInput{
		Backend:                 " BrandMeister ",
		MasterAddress:           " BM3103.EXAMPLE.NET. ",
		MasterPort:              0,
		RegistrationFrequencyHz: 446_525_000,
		Password:                "radio-secret",
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
	if candidate.RegistrationFrequencyHz != 446_525_000 {
		t.Fatalf("unexpected registration frequency %d", candidate.RegistrationFrequencyHz)
	}
	if candidate.Password != "radio-secret" {
		t.Fatal("password must be preserved internally for the connection test")
	}
	if result.Normalized.PasswordSet != true {
		t.Fatal("expected password_set=true")
	}
}

func TestNetworkValidationDoesNotEchoPassword(t *testing.T) {
	secret := "no-echo-secret"
	_, result := ValidateNetworkCandidate(NetworkInput{
		Backend:                 "brandmeister",
		MasterAddress:           "127.0.0.1",
		MasterPort:              62031,
		RegistrationFrequencyHz: 446_525_000,
		Password:                secret,
	})

	if !result.Valid {
		t.Fatalf("expected valid result: %+v", result.Errors)
	}
	serialized := result.Normalized.Backend + result.Normalized.MasterAddress
	if strings.Contains(serialized, secret) {
		t.Fatal("network validation summary leaked password")
	}
}

func TestValidateNetworkCandidateRejectsInvalidFields(t *testing.T) {
	_, result := ValidateNetworkCandidate(NetworkInput{
		Backend:                 "other-network",
		MasterAddress:           "https://master.example:62031/path",
		MasterPort:              70000,
		RegistrationFrequencyHz: 0,
		Password:                "bad\npassword",
	})

	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if len(result.Errors) != 5 {
		t.Fatalf("expected five field errors, got %d: %+v", len(result.Errors), result.Errors)
	}
}

func TestValidateNetworkCandidateAcceptsIPv6(t *testing.T) {
	candidate, result := ValidateNetworkCandidate(NetworkInput{
		Backend:                 "brandmeister",
		MasterAddress:           "2001:db8::1",
		MasterPort:              62031,
		RegistrationFrequencyHz: 446_525_000,
		Password:                "secret",
	})
	if !result.Valid {
		t.Fatalf("expected IPv6 to be valid: %+v", result.Errors)
	}
	if candidate.MasterAddress != "2001:db8::1" {
		t.Fatalf("unexpected normalized IPv6 %q", candidate.MasterAddress)
	}
}

func TestValidateNetworkCandidateAcceptsTwentyCharacterHotspotPassword(t *testing.T) {
	password := strings.Repeat("a", BrandMeisterMaxHotspotPassword)
	_, result := ValidateNetworkCandidate(NetworkInput{
		Backend:                 "brandmeister",
		MasterAddress:           "bm.example.net",
		MasterPort:              62031,
		RegistrationFrequencyHz: 446_525_000,
		Password:                password,
	})
	if !result.Valid {
		t.Fatalf("expected %d-character password to be valid: %+v", BrandMeisterMaxHotspotPassword, result.Errors)
	}
}

func TestValidateNetworkCandidateRejectsLongHotspotPassword(t *testing.T) {
	password := strings.Repeat("a", BrandMeisterMaxHotspotPassword+1)
	_, result := ValidateNetworkCandidate(NetworkInput{
		Backend:                 "brandmeister",
		MasterAddress:           "bm.example.net",
		MasterPort:              62031,
		RegistrationFrequencyHz: 446_525_000,
		Password:                password,
	})
	if result.Valid {
		t.Fatal("expected over-length BrandMeister password to be rejected")
	}
	for _, fieldErr := range result.Errors {
		if fieldErr.Field == "password" && fieldErr.Code == "invalid_password" {
			return
		}
	}
	t.Fatalf("expected password field error: %+v", result.Errors)
}

func TestValidateNetworkCandidateRejectsMissingRegistrationFrequency(t *testing.T) {
	_, result := ValidateNetworkCandidate(NetworkInput{
		Backend:       "brandmeister",
		MasterAddress: "bm.example.net",
		MasterPort:    62031,
		Password:      "secret",
	})
	if result.Valid {
		t.Fatal("expected missing registration frequency to be rejected")
	}
	for _, fieldErr := range result.Errors {
		if fieldErr.Field == "registration_frequency_hz" && fieldErr.Code == "invalid_registration_frequency" {
			return
		}
	}
	t.Fatalf("expected registration frequency field error: %+v", result.Errors)
}
