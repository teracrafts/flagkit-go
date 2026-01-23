package flagkit

import (
	"github.com/flagkit/flagkit-go/internal/types"
)

// EvaluationContext contains user and environment information for flag evaluation.
type EvaluationContext = types.EvaluationContext

// NewContext creates a new EvaluationContext with the given user ID.
func NewContext(userID string) *EvaluationContext {
	return types.NewContext(userID)
}

// NewAnonymousContext creates a new anonymous EvaluationContext.
func NewAnonymousContext() *EvaluationContext {
	return types.NewAnonymousContext()
}
