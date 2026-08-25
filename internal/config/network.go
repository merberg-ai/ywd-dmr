package config

import (
	"net"
	"strings"
	"unicode"
)

const (
	NetworkBackendBrandMeister = "brandmeister"
	BrandMeisterDefaultPort    = 62031
)

// NetworkInput is the untrusted network configuration submitted by a setup
// client. Password is accepted here because the future connectivity test needs
// it, but validation responses never echo the secret back to the client.
type NetworkInput struct {
	Backend       string `json:"backend"`
	MasterAddress string `json:"master_address"`
	MasterPort    int    `json:"master_port"`
	Password      string `json:"password"`
}

// NetworkCandidate is the normalized internal form passed to a network backend
// test. It is not itself a response type because Password must never be echoed.
type NetworkCandidate struct {
	Backend       string
	MasterAddress string
	MasterPort    int
	Password      string
}

// NetworkSummary is the non-secret normalized shape safe to return to clients.
type NetworkSummary struct {
	Backend       string `json:"backend"`
	MasterAddress string `json:"master_address"`
	MasterPort    int    `json:"master_port"`
	PasswordSet   bool   `json:"password_set"`
}

// NetworkValidation is the protected, non-mutating validation result used by
// the setup UI before any live network test or durable commit exists.
type NetworkValidation struct {
	Valid      bool           `json:"valid"`
	Normalized NetworkSummary `json:"normalized"`
	Errors     []FieldError   `json:"errors"`
}

// ValidateNetworkCandidate normalizes and validates the first backend-neutral
// network candidate. Only BrandMeister is accepted in Alpha1, but the model
// keeps the backend explicit so later networks do not require a parallel API.
func ValidateNetworkCandidate(input NetworkInput) (NetworkCandidate, NetworkValidation) {
	candidate := NetworkCandidate{
		Backend:       strings.ToLower(strings.TrimSpace(input.Backend)),
		MasterAddress: normalizeMasterAddress(input.MasterAddress),
		MasterPort:    input.MasterPort,
		Password:      input.Password,
	}
	if candidate.MasterPort == 0 {
		candidate.MasterPort = BrandMeisterDefaultPort
	}

	errs := make([]FieldError, 0, 4)
	if candidate.Backend != NetworkBackendBrandMeister {
		errs = append(errs, FieldError{
			Field:   "backend",
			Code:    "unsupported_backend",
			Message: "Select BrandMeister as the DMR network for this Alpha1 build.",
		})
	}
	if !validMasterAddress(candidate.MasterAddress) {
		errs = append(errs, FieldError{
			Field:   "master_address",
			Code:    "invalid_master_address",
			Message: "Enter a BrandMeister master hostname or IP address without http://, https://, or a port number.",
		})
	}
	if candidate.MasterPort < 1 || candidate.MasterPort > 65535 {
		errs = append(errs, FieldError{
			Field:   "master_port",
			Code:    "invalid_master_port",
			Message: "Master port must be a number from 1 through 65535. Leave it at 0 to use the BrandMeister default 62031.",
		})
	}
	if strings.TrimSpace(candidate.Password) == "" || len(candidate.Password) > 256 || containsControl(candidate.Password) {
		errs = append(errs, FieldError{
			Field:   "password",
			Code:    "invalid_password",
			Message: "Enter the BrandMeister hotspot password. It may be up to 256 characters and must not contain control characters.",
		})
	}

	result := NetworkValidation{
		Valid: len(errs) == 0,
		Normalized: NetworkSummary{
			Backend:       candidate.Backend,
			MasterAddress: candidate.MasterAddress,
			MasterPort:    candidate.MasterPort,
			PasswordSet:   candidate.Password != "",
		},
		Errors: errs,
	}
	return candidate, result
}

func normalizeMasterAddress(value string) string {
	value = strings.TrimSpace(value)
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return strings.ToLower(strings.TrimSuffix(value, "."))
}

func validMasterAddress(value string) bool {
	if value == "" || len(value) > 253 {
		return false
	}
	if net.ParseIP(value) != nil {
		return true
	}
	if strings.ContainsAny(value, "/:@[] ") || strings.ContainsRune(value, '\\') || containsControl(value) {
		return false
	}
	labels := strings.Split(value, ".")
	for _, label := range labels {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
	}
	return true
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
