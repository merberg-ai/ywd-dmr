package config

import "testing"

func TestValidateRadioIdentityNormalizesCallsign(t *testing.T) {
	result := ValidateRadioIdentity(RadioIdentityInput{
		Callsign: "  n0call  ",
		DMRID:    1234567,
		ESSID:    1,
	})

	if !result.Valid {
		t.Fatalf("expected valid identity, got errors: %+v", result.Errors)
	}
	if result.Normalized.Callsign != "N0CALL" {
		t.Fatalf("expected normalized callsign N0CALL, got %q", result.Normalized.Callsign)
	}
	if result.Normalized.DMRID != 1234567 {
		t.Fatalf("expected DMR ID 1234567, got %d", result.Normalized.DMRID)
	}
	if result.Normalized.ESSID != 1 {
		t.Fatalf("expected ESSID 1, got %d", result.Normalized.ESSID)
	}
}

func TestValidateRadioIdentityRejectsInvalidFields(t *testing.T) {
	result := ValidateRadioIdentity(RadioIdentityInput{
		Callsign: "BAD CALL",
		DMRID:    10_000_000,
		ESSID:    100,
	})

	if result.Valid {
		t.Fatal("expected invalid identity")
	}
	if len(result.Errors) != 3 {
		t.Fatalf("expected 3 field errors, got %d: %+v", len(result.Errors), result.Errors)
	}
}

func TestValidateRadioIdentityRequiresLettersAndDigitsInCallsign(t *testing.T) {
	for _, callsign := range []string{"123456", "CALLSIGN"} {
		result := ValidateRadioIdentity(RadioIdentityInput{
			Callsign: callsign,
			DMRID:    1,
			ESSID:    0,
		})
		if result.Valid {
			t.Fatalf("expected callsign %q to be rejected", callsign)
		}
	}
}

func TestValidateRadioIdentityAcceptsESSIDBounds(t *testing.T) {
	for _, essid := range []int{0, 99} {
		result := ValidateRadioIdentity(RadioIdentityInput{
			Callsign: "N0CALL",
			DMRID:    1234567,
			ESSID:    essid,
		})
		if !result.Valid {
			t.Fatalf("expected ESSID %d to be valid, got %+v", essid, result.Errors)
		}
	}
}
