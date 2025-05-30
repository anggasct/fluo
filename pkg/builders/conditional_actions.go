package builders

import (
	"fmt"

	"github.com/anggasct/fluo/pkg/core"
)

// ConditionalActions provides helper functions for common conditional actions
type ConditionalActions struct{}

// IfDataEquals creates a guard condition that checks if context data equals a value
func (ConditionalActions) IfDataEquals(key string, value interface{}) core.GuardCondition {
	return func(ctx *core.Context) bool {
		if val, exists := ctx.Get(key); exists {
			return val == value
		}
		return false
	}
}

// IfDataExists creates a guard condition that checks if context data exists
func (ConditionalActions) IfDataExists(key string) core.GuardCondition {
	return func(ctx *core.Context) bool {
		_, exists := ctx.Get(key)
		return exists
	}
}

// IfEventDataEquals creates a guard condition that checks if event data equals a value
func (ConditionalActions) IfEventDataEquals(value interface{}) core.GuardCondition {
	return func(ctx *core.Context) bool {
		if ctx.GetEvent() != nil {
			return ctx.GetEvent().Data == value
		}
		return false
	}
}

// SetData creates an action that sets context data
func (ConditionalActions) SetData(key string, value interface{}) core.Action {
	return func(ctx *core.Context) error {
		ctx.Set(key, value)
		return nil
	}
}

// LogMessage creates an action that logs a message
func (ConditionalActions) LogMessage(message string) core.Action {
	return func(ctx *core.Context) error {
		fmt.Printf("[State Machine] %s\n", message)
		return nil
	}
}

// Conditions provides a singleton instance of ConditionalActions
var Conditions = ConditionalActions{}
