package config

import "strings"

// RadioIdentityInput is the user-supplied station identity collected during
// first-run setup. ESSID is kept separate from the base DMR ID so network
// backends can derive their own hotspot/device identifiers without changing the
// operator's canonical identity.
type RadioIdentityInput struct {
	Callsign string `json:"callsign"`
	DMRID    int    `json:"dmr_id"`
	ESSID    int    `json:"essid"`
}

// RadioIdentity is the normalized form that may eventually be committed to the
// known-good configuration store after validation and authorization.
type RadioIdentity struct {
	Callsign string `json:"callsign"`
	DMRID    int    `json:"dmr_id"`
	ESSID    int    `json:"essid"`
}

// FieldError is intentionally small and stable enough for both the WebUI and
// future native clients to present useful field-level setup errors.
type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// IdentityValidation is a non-mutating validation result. Invalid input is a
// normal result, not an HTTP transport error.
type IdentityValidation struct {
	Valid      bool          `json:"valid"`
	Normalized RadioIdentity `json:"normalized"`
	Errors     []FieldError  `json:"errors"`
}

// ValidateRadioIdentity normalizes and validates the base station identity.
// The rules are intentionally conservative without encoding one country's
// callsign allocation scheme into the daemon.
func ValidateRadioIdentity(input RadioIdentityInput) IdentityValidation {
	normalized := RadioIdentity{
		Callsign: strings.ToUpper(strings.TrimSpace(input.Callsign)),
		DMRID:    input.DMRID,
		ESSID:    input.ESSID,
	}

	errors := make([]FieldError, 0, 3)

	if !validBaseCallsign(normalized.Callsign) {
		errors = append(errors, FieldError{
			Field:   "callsign",
			Code:    "invalid_callsign",
			Message: "Enter a base amateur-radio callsign using 3 to 12 letters and numbers.",
		})
	}

	if normalized.DMRID < 1 || normalized.DMRID > 9_999_999 {
		errors = append(errors, FieldError{
			Field:   "dmr_id",
			Code:    "invalid_dmr_id",
			Message: "Enter the base DMR ID as a number from 1 through 9999999.",
		})
	}

	if normalized.ESSID < 0 || normalized.ESSID > 99 {
		errors = append(errors, FieldError{
			Field:   "essid",
			Code:    "invalid_essid",
			Message: "ESSID must be a number from 0 through 99.",
		})
	}

	return IdentityValidation{
		Valid:      len(errors) == 0,
		Normalized: normalized,
		Errors:     errors,
	}
}

func validBaseCallsign(callsign string) bool {
	if len(callsign) < 3 || len(callsign) > 12 {
		return false
	}

	hasLetter := false
	hasDigit := false
	for _, r := range callsign {
		switch {
		case r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		default:
			return false
		}
	}

	return hasLetter && hasDigit
}
