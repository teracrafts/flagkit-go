package types

import (
	"encoding/json"
	"time"
)

// FlagType represents the type of a flag value.
type FlagType string

const (
	FlagTypeBoolean FlagType = "boolean"
	FlagTypeString  FlagType = "string"
	FlagTypeNumber  FlagType = "number"
	FlagTypeJSON    FlagType = "json"
)

// EvaluationReason represents the reason for an evaluation result.
type EvaluationReason string

const (
	ReasonCached       EvaluationReason = "CACHED"
	ReasonFallthrough  EvaluationReason = "FALLTHROUGH"
	ReasonTargeted     EvaluationReason = "TARGETED"
	ReasonDefault      EvaluationReason = "DEFAULT"
	ReasonDisabled     EvaluationReason = "DISABLED"
	ReasonFlagNotFound EvaluationReason = "FLAG_NOT_FOUND"
	ReasonError        EvaluationReason = "ERROR"
	ReasonStaleCache   EvaluationReason = "STALE_CACHE"
	ReasonBootstrap    EvaluationReason = "BOOTSTRAP"
)

// FlagState represents the state of a feature flag.
type FlagState struct {
	Key          string      `json:"key"`
	Value        any `json:"value"`
	Enabled      bool        `json:"enabled"`
	Version      int         `json:"version"`
	FlagType     FlagType    `json:"flagType"`
	LastModified string      `json:"lastModified"`
}

// EvaluationResult represents the result of evaluating a flag.
type EvaluationResult struct {
	FlagKey   string           `json:"flagKey"`
	Value     any      `json:"value"`
	Enabled   bool             `json:"enabled"`
	Reason    EvaluationReason `json:"reason"`
	Version   int              `json:"version"`
	Timestamp time.Time        `json:"timestamp"`
	Error     error            `json:"-"`
}

// BoolValue returns the value as a boolean.
func (r *EvaluationResult) BoolValue() bool {
	if v, ok := r.Value.(bool); ok {
		return v
	}
	return false
}

// StringValue returns the value as a string.
func (r *EvaluationResult) StringValue() string {
	if v, ok := r.Value.(string); ok {
		return v
	}
	return ""
}

// Float64Value returns the value as a float64.
func (r *EvaluationResult) Float64Value() float64 {
	switch v := r.Value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// IntValue returns the value as an int.
func (r *EvaluationResult) IntValue() int {
	switch v := r.Value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

// JSONValue returns the value as a map.
func (r *EvaluationResult) JSONValue() map[string]any {
	if v, ok := r.Value.(map[string]any); ok {
		return v
	}
	return nil
}

// InitResponse represents the response from the init endpoint.
type InitResponse struct {
	Flags                  []FlagState `json:"flags"`
	Environment            string      `json:"environment"`
	EnvironmentID          string      `json:"environmentId"`
	ProjectID              string      `json:"projectId"`
	OrganizationID         string      `json:"organizationId"`
	ServerTime             string      `json:"serverTime"`
	PollingIntervalSeconds int         `json:"pollingIntervalSeconds"`
}

// UpdatesResponse represents the response from the updates endpoint.
type UpdatesResponse struct {
	Flags     []FlagState `json:"flags"`
	CheckedAt string      `json:"checkedAt"`
	Since     string      `json:"since"`
}

// EventsBatchResponse represents the response from the events batch endpoint.
type EventsBatchResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Recorded int    `json:"recorded"`
	Errors   int    `json:"errors"`
}

// ParseInitResponse parses JSON data into an InitResponse.
func ParseInitResponse(data []byte) (*InitResponse, error) {
	var resp InitResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ParseUpdatesResponse parses JSON data into an UpdatesResponse.
func ParseUpdatesResponse(data []byte) (*UpdatesResponse, error) {
	var resp UpdatesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// InferFlagType infers the flag type from a value.
func InferFlagType(value any) FlagType {
	switch value.(type) {
	case bool:
		return FlagTypeBoolean
	case string:
		return FlagTypeString
	case int, int32, int64, float32, float64:
		return FlagTypeNumber
	default:
		return FlagTypeJSON
	}
}
