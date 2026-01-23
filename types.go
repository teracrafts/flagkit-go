package flagkit

import (
	"github.com/flagkit/flagkit-go/internal/types"
)

// Re-export type definitions for public API

// FlagType represents the type of a flag value.
type FlagType = types.FlagType

const (
	FlagTypeBoolean = types.FlagTypeBoolean
	FlagTypeString  = types.FlagTypeString
	FlagTypeNumber  = types.FlagTypeNumber
	FlagTypeJSON    = types.FlagTypeJSON
)

// EvaluationReason represents the reason for an evaluation result.
type EvaluationReason = types.EvaluationReason

const (
	ReasonCached       = types.ReasonCached
	ReasonFallthrough  = types.ReasonFallthrough
	ReasonTargeted     = types.ReasonTargeted
	ReasonDefault      = types.ReasonDefault
	ReasonDisabled     = types.ReasonDisabled
	ReasonFlagNotFound = types.ReasonFlagNotFound
	ReasonError        = types.ReasonError
	ReasonStaleCache   = types.ReasonStaleCache
	ReasonBootstrap    = types.ReasonBootstrap
)

// FlagState represents the state of a feature flag.
type FlagState = types.FlagState

// EvaluationResult represents the result of evaluating a flag.
type EvaluationResult = types.EvaluationResult

// InitResponse represents the response from the init endpoint.
type InitResponse = types.InitResponse

// UpdatesResponse represents the response from the updates endpoint.
type UpdatesResponse = types.UpdatesResponse

// EventsBatchResponse represents the response from the events batch endpoint.
type EventsBatchResponse = types.EventsBatchResponse

// ParseInitResponse parses JSON data into an InitResponse.
func ParseInitResponse(data []byte) (*InitResponse, error) {
	return types.ParseInitResponse(data)
}

// ParseUpdatesResponse parses JSON data into an UpdatesResponse.
func ParseUpdatesResponse(data []byte) (*UpdatesResponse, error) {
	return types.ParseUpdatesResponse(data)
}

// createDefaultResult creates a default evaluation result.
func createDefaultResult(key string, defaultValue interface{}, reason EvaluationReason) *EvaluationResult {
	return types.CreateDefaultResult(key, defaultValue, reason)
}

// inferFlagType infers the flag type from a value.
func inferFlagType(value interface{}) FlagType {
	return types.InferFlagType(value)
}
