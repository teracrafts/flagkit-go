package flagkit

import (
	"fmt"
	"os"
	"strings"
)

// PII field patterns (case-insensitive matching)
var piiPatterns = []string{
	"email",
	"phone",
	"telephone",
	"mobile",
	"ssn",
	"social_security",
	"socialsecurity",
	"credit_card",
	"creditcard",
	"card_number",
	"cardnumber",
	"cvv",
	"password",
	"passwd",
	"secret",
	"token",
	"api_key",
	"apikey",
	"private_key",
	"privatekey",
	"access_token",
	"accesstoken",
	"refresh_token",
	"refreshtoken",
	"auth_token",
	"authtoken",
	"address",
	"street",
	"zip_code",
	"zipcode",
	"postal_code",
	"postalcode",
	"date_of_birth",
	"dateofbirth",
	"dob",
	"birth_date",
	"birthdate",
	"passport",
	"driver_license",
	"driverlicense",
	"national_id",
	"nationalid",
	"bank_account",
	"bankaccount",
	"routing_number",
	"routingnumber",
	"iban",
	"swift",
}

// IsPotentialPIIField checks if a field name potentially contains PII.
func IsPotentialPIIField(fieldName string) bool {
	lowerName := strings.ToLower(fieldName)
	lowerName = strings.ReplaceAll(lowerName, "-", "")
	lowerName = strings.ReplaceAll(lowerName, "_", "")

	for _, pattern := range piiPatterns {
		normalizedPattern := strings.ReplaceAll(pattern, "_", "")
		if strings.Contains(lowerName, normalizedPattern) {
			return true
		}
	}
	return false
}

// DetectPotentialPII detects potential PII fields in a map and returns field paths.
func DetectPotentialPII(data map[string]interface{}, prefix string) []string {
	var piiFields []string

	for key, value := range data {
		fullPath := key
		if prefix != "" {
			fullPath = prefix + "." + key
		}

		if IsPotentialPIIField(key) {
			piiFields = append(piiFields, fullPath)
		}

		// Recursively check nested maps
		if nestedMap, ok := value.(map[string]interface{}); ok {
			nestedPII := DetectPotentialPII(nestedMap, fullPath)
			piiFields = append(piiFields, nestedPII...)
		}
	}

	return piiFields
}

// WarnIfPotentialPII logs a warning if potential PII is detected in data.
func WarnIfPotentialPII(data map[string]interface{}, dataType string, logger Logger) {
	if data == nil || logger == nil {
		return
	}

	piiFields := DetectPotentialPII(data, "")

	if len(piiFields) > 0 {
		advice := "Consider removing sensitive data from events."
		if dataType == "context" {
			advice = "Consider adding these to private attributes."
		}

		logger.Warn(fmt.Sprintf(
			"[FlagKit Security] Potential PII detected in %s data: %s. %s",
			dataType,
			strings.Join(piiFields, ", "),
			advice,
		))
	}
}

// IsServerKey checks if an API key is a server key.
func IsServerKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, "srv_")
}

// IsClientKey checks if an API key is a client/SDK key.
func IsClientKey(apiKey string) bool {
	return strings.HasPrefix(apiKey, "sdk_") || strings.HasPrefix(apiKey, "cli_")
}

// IsBrowserEnvironment checks if running in a browser-like environment.
// Go typically runs server-side, so this returns false by default.
// This is here for API consistency with other SDKs.
func IsBrowserEnvironment() bool {
	// Check for WebAssembly environment
	// In GOOS=js GOARCH=wasm builds, this would be true
	return false
}

// WarnIfServerKeyInBrowser warns if a server key is used in a browser environment.
func WarnIfServerKeyInBrowser(apiKey string, logger Logger) {
	if IsBrowserEnvironment() && IsServerKey(apiKey) {
		message := "[FlagKit Security] WARNING: Server keys (srv_) should not be used " +
			"in browser environments. This exposes your server key in client-side " +
			"code, which is a security risk. Use SDK keys (sdk_) for client-side " +
			"applications instead. See: https://docs.flagkit.dev/sdk/security#api-keys"

		// Print to stderr for visibility
		fmt.Fprintln(os.Stderr, message)

		// Also log through the SDK logger if available
		if logger != nil {
			logger.Warn(message)
		}
	}
}

// SecurityConfig holds security configuration options.
type SecurityConfig struct {
	// WarnOnPotentialPII enables warnings when potential PII is detected.
	// Defaults to true in non-production environments.
	WarnOnPotentialPII bool

	// WarnOnServerKeyInBrowser enables warnings when server keys are used
	// in browser environments. Default: true.
	WarnOnServerKeyInBrowser bool

	// AdditionalPIIPatterns allows adding custom PII patterns to detect.
	AdditionalPIIPatterns []string
}

// DefaultSecurityConfig returns the default security configuration.
func DefaultSecurityConfig() SecurityConfig {
	// Check environment for production
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	isProduction := strings.EqualFold(env, "production")

	return SecurityConfig{
		WarnOnPotentialPII:       !isProduction,
		WarnOnServerKeyInBrowser: true,
		AdditionalPIIPatterns:    nil,
	}
}

// AddPIIPatterns adds custom PII patterns to the detection list.
func AddPIIPatterns(patterns []string) {
	for _, p := range patterns {
		piiPatterns = append(piiPatterns, strings.ToLower(p))
	}
}
