package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/teracrafts/flagkit-go/config"
	"github.com/teracrafts/flagkit-go/errors"
	"github.com/teracrafts/flagkit-go/types"
)

// Type aliases for convenience
type Logger = types.Logger
type BootstrapConfig = config.BootstrapConfig
type BootstrapVerificationConfig = config.BootstrapVerificationConfig

// Error function aliases
var (
	SecurityError     = errors.SecurityError
	NewError          = errors.NewError
	NewErrorWithCause = errors.NewErrorWithCause
)

// Error code aliases
const (
	ErrSecurityPIIDetected      = errors.ErrSecurityPIIDetected
	ErrSecuritySignatureInvalid = errors.ErrSecuritySignatureInvalid
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
func DetectPotentialPII(data map[string]any, prefix string) []string {
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
		if nestedMap, ok := value.(map[string]any); ok {
			nestedPII := DetectPotentialPII(nestedMap, fullPath)
			piiFields = append(piiFields, nestedPII...)
		}
	}

	return piiFields
}

// WarnIfPotentialPII logs a warning if potential PII is detected in data.
func WarnIfPotentialPII(data map[string]any, dataType string, logger Logger) {
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

// PIIDetectionResult contains the result of PII detection.
type PIIDetectionResult struct {
	HasPII  bool
	Fields  []string
	Message string
}

// CheckForPotentialPII checks for potential PII in data and returns detailed result.
func CheckForPotentialPII(data map[string]any, dataType string) PIIDetectionResult {
	if data == nil {
		return PIIDetectionResult{HasPII: false, Fields: nil, Message: ""}
	}

	piiFields := DetectPotentialPII(data, "")

	if len(piiFields) == 0 {
		return PIIDetectionResult{HasPII: false, Fields: nil, Message: ""}
	}

	advice := "Consider removing sensitive data from events."
	if dataType == "context" {
		advice = "Consider adding these to privateAttributes."
	}

	message := fmt.Sprintf(
		"[FlagKit Security] Potential PII detected in %s data: %s. %s",
		dataType,
		strings.Join(piiFields, ", "),
		advice,
	)

	return PIIDetectionResult{
		HasPII:  true,
		Fields:  piiFields,
		Message: message,
	}
}

// CheckPIIWithStrictMode checks for PII and returns error if strict mode is enabled.
func CheckPIIWithStrictMode(data map[string]any, dataType string, strictMode bool, logger Logger) error {
	result := CheckForPotentialPII(data, dataType)

	if !result.HasPII {
		return nil
	}

	if strictMode {
		return SecurityError(ErrSecurityPIIDetected, result.Message)
	}

	if logger != nil {
		logger.Warn(result.Message)
	}

	return nil
}

// IsProductionEnvironment checks if the current environment is production.
// Checks GO_ENV and APP_ENV environment variables.
func IsProductionEnvironment() bool {
	goEnv := os.Getenv("GO_ENV")
	appEnv := os.Getenv("APP_ENV")

	return strings.EqualFold(goEnv, "production") || strings.EqualFold(appEnv, "production")
}

// GetKeyID returns the first 8 characters of an API key for identification.
// This is safe to expose as it doesn't reveal the full key.
func GetKeyID(apiKey string) string {
	if len(apiKey) < 8 {
		return apiKey
	}
	return apiKey[:8]
}

// SignedPayload represents a payload with HMAC-SHA256 signature.
type SignedPayload struct {
	Data      any `json:"data"`
	Signature string      `json:"signature"`
	Timestamp int64       `json:"timestamp"`
	KeyID     string      `json:"keyId"`
}

// GenerateHMACSHA256 generates an HMAC-SHA256 signature for a message.
func GenerateHMACSHA256(message, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// RequestSignature contains signature information for request headers.
type RequestSignature struct {
	Signature string
	Timestamp int64
	KeyID     string
}

// CreateRequestSignature creates a signature for request body.
// The message format is: timestamp.body
func CreateRequestSignature(body, apiKey string) RequestSignature {
	timestamp := time.Now().UnixMilli()
	message := strconv.FormatInt(timestamp, 10) + "." + body
	signature := GenerateHMACSHA256(message, apiKey)

	return RequestSignature{
		Signature: signature,
		Timestamp: timestamp,
		KeyID:     GetKeyID(apiKey),
	}
}

// VerifyRequestSignature verifies a request signature.
// maxAgeMs is the maximum age of the signature in milliseconds (default 5 minutes).
func VerifyRequestSignature(body, signature string, timestamp int64, apiKey string, maxAgeMs int64) bool {
	if maxAgeMs == 0 {
		maxAgeMs = 300000 // 5 minutes default
	}

	// Check timestamp age
	now := time.Now().UnixMilli()
	age := now - timestamp
	if age > maxAgeMs || age < 0 {
		return false
	}

	// Verify signature
	message := strconv.FormatInt(timestamp, 10) + "." + body
	expectedSignature := GenerateHMACSHA256(message, apiKey)

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// SignPayload signs a payload with HMAC-SHA256.
func SignPayload(data any, apiKey string, timestamp int64) SignedPayload {
	if timestamp == 0 {
		timestamp = time.Now().UnixMilli()
	}

	// Convert data to JSON string for signing
	var payload string
	switch v := data.(type) {
	case string:
		payload = v
	case []byte:
		payload = string(v)
	default:
		// For other types, we'd need to marshal to JSON
		// This is handled in the HTTP client
		payload = fmt.Sprintf("%v", v)
	}

	message := strconv.FormatInt(timestamp, 10) + "." + payload
	signature := GenerateHMACSHA256(message, apiKey)

	return SignedPayload{
		Data:      data,
		Signature: signature,
		Timestamp: timestamp,
		KeyID:     GetKeyID(apiKey),
	}
}

// VerifySignedPayload verifies a signed payload.
func VerifySignedPayload(payload SignedPayload, apiKey string, maxAgeMs int64) bool {
	if maxAgeMs == 0 {
		maxAgeMs = 300000 // 5 minutes default
	}

	// Check timestamp age
	now := time.Now().UnixMilli()
	age := now - payload.Timestamp
	if age > maxAgeMs || age < 0 {
		return false
	}

	// Verify key ID matches
	if payload.KeyID != GetKeyID(apiKey) {
		return false
	}

	// Verify signature
	var dataStr string
	switch v := payload.Data.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		dataStr = fmt.Sprintf("%v", v)
	}

	message := strconv.FormatInt(payload.Timestamp, 10) + "." + dataStr
	expectedSignature := GenerateHMACSHA256(message, apiKey)

	return hmac.Equal([]byte(payload.Signature), []byte(expectedSignature))
}

// CanonicalizeObject converts a map to a canonical JSON string for signature verification.
// Keys are sorted alphabetically, and the output is deterministic.
func CanonicalizeObject(obj map[string]any) (string, error) {
	if obj == nil {
		return "{}", nil
	}

	// Use a custom encoder to ensure consistent output
	canonical, err := canonicalizeValue(obj)
	if err != nil {
		return "", err
	}

	return canonical, nil
}

// canonicalizeValue recursively canonicalizes a value.
func canonicalizeValue(v any) (string, error) {
	switch val := v.(type) {
	case nil:
		return "null", nil
	case bool:
		if val {
			return "true", nil
		}
		return "false", nil
	case float64:
		// Handle integers stored as float64
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10), nil
		}
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case float32:
		f64 := float64(val)
		if f64 == float64(int64(f64)) {
			return strconv.FormatInt(int64(f64), 10), nil
		}
		return strconv.FormatFloat(f64, 'f', -1, 32), nil
	case int:
		return strconv.Itoa(val), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case string:
		// Use JSON encoding for proper escaping
		bytes, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	case []any:
		return canonicalizeArray(val)
	case map[string]any:
		return canonicalizeMap(val)
	default:
		// For unknown types, use JSON marshaling
		bytes, err := json.Marshal(val)
		if err != nil {
			return "", fmt.Errorf("cannot canonicalize value of type %T: %w", v, err)
		}
		return string(bytes), nil
	}
}

// canonicalizeArray canonicalizes an array.
func canonicalizeArray(arr []any) (string, error) {
	if len(arr) == 0 {
		return "[]", nil
	}

	parts := make([]string, len(arr))
	for i, v := range arr {
		canonical, err := canonicalizeValue(v)
		if err != nil {
			return "", err
		}
		parts[i] = canonical
	}

	return "[" + strings.Join(parts, ",") + "]", nil
}

// canonicalizeMap canonicalizes a map with sorted keys.
func canonicalizeMap(m map[string]any) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, len(keys))
	for i, k := range keys {
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return "", err
		}

		valueCanonical, err := canonicalizeValue(m[k])
		if err != nil {
			return "", err
		}

		parts[i] = string(keyJSON) + ":" + valueCanonical
	}

	return "{" + strings.Join(parts, ",") + "}", nil
}

// BootstrapVerificationResult contains the result of bootstrap verification.
type BootstrapVerificationResult struct {
	Valid   bool
	Error   error
	Message string
}

// VerifyBootstrapSignature verifies the HMAC-SHA256 signature of bootstrap data.
// The signature is computed over: timestamp.canonicalized_flags_json
// Returns (valid, error) where error contains details about any verification failure.
func VerifyBootstrapSignature(bootstrap BootstrapConfig, apiKey string, config BootstrapVerificationConfig) (bool, error) {
	// If verification is disabled, always return valid
	if !config.Enabled {
		return true, nil
	}

	// If no signature provided, skip verification (legacy format)
	if bootstrap.Signature == "" {
		return true, nil
	}

	// Check timestamp age if MaxAge is set
	if config.MaxAge > 0 && bootstrap.Timestamp > 0 {
		now := time.Now().UnixMilli()
		age := now - bootstrap.Timestamp
		maxAgeMs := config.MaxAge.Milliseconds()

		if age > maxAgeMs {
			return false, NewError(ErrSecuritySignatureInvalid,
				fmt.Sprintf("bootstrap data is expired: age %dms exceeds max age %dms", age, maxAgeMs))
		}

		// Check for future timestamp (clock skew protection)
		if age < -300000 { // Allow 5 minutes of clock skew
			return false, NewError(ErrSecuritySignatureInvalid,
				"bootstrap timestamp is in the future")
		}
	}

	// Canonicalize the flags
	canonical, err := CanonicalizeObject(bootstrap.Flags)
	if err != nil {
		return false, NewErrorWithCause(ErrSecuritySignatureInvalid,
			"failed to canonicalize bootstrap flags", err)
	}

	// Build the message: timestamp.canonical_json
	message := strconv.FormatInt(bootstrap.Timestamp, 10) + "." + canonical

	// Generate expected signature
	expectedSignature := GenerateHMACSHA256(message, apiKey)

	// Use constant-time comparison
	if !hmac.Equal([]byte(bootstrap.Signature), []byte(expectedSignature)) {
		return false, NewError(ErrSecuritySignatureInvalid,
			"bootstrap signature verification failed: signature mismatch")
	}

	return true, nil
}

// CreateBootstrapSignature creates a signed bootstrap configuration.
// This is a helper function for generating signed bootstrap data.
func CreateBootstrapSignature(flags map[string]any, apiKey string) (*BootstrapConfig, error) {
	timestamp := time.Now().UnixMilli()

	canonical, err := CanonicalizeObject(flags)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize flags: %w", err)
	}

	message := strconv.FormatInt(timestamp, 10) + "." + canonical
	signature := GenerateHMACSHA256(message, apiKey)

	return &BootstrapConfig{
		Flags:     flags,
		Signature: signature,
		Timestamp: timestamp,
	}, nil
}
