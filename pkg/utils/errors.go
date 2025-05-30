// Package utils provides utility functions for the state machine library
package utils

import (
	"fmt"
	"strings"
)

// StateMachineError represents a state machine specific error
type StateMachineError struct {
	Code      string
	Message   string
	StateID   string
	EventType string
	Cause     error
	Details   map[string]interface{}
}

// Error implements the error interface
func (e *StateMachineError) Error() string {
	var parts []string

	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	if e.StateID != "" {
		parts = append(parts, fmt.Sprintf("state: %s", e.StateID))
	}

	if e.EventType != "" {
		parts = append(parts, fmt.Sprintf("event: %s", e.EventType))
	}

	if len(e.Details) > 0 {
		var details []string
		for k, v := range e.Details {
			details = append(details, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("details: {%s}", strings.Join(details, ", ")))
	}

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Cause))
	}

	return strings.Join(parts, " - ")
}

// WithState adds state information to the error
func (e *StateMachineError) WithState(stateID string) *StateMachineError {
	e.StateID = stateID
	return e
}

// WithEvent adds event information to the error
func (e *StateMachineError) WithEvent(eventType string) *StateMachineError {
	e.EventType = eventType
	return e
}

// WithCause adds cause information to the error
func (e *StateMachineError) WithCause(err error) *StateMachineError {
	e.Cause = err
	return e
}

// WithDetail adds a detail to the error
func (e *StateMachineError) WithDetail(key string, value interface{}) *StateMachineError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Core error types for the state machine library
var (
	// ErrInvalidTransition is returned when a transition is not valid from the current state
	ErrInvalidTransition = &StateMachineError{
		Code:    "INVALID_TRANSITION",
		Message: "invalid transition from current state",
	}

	// ErrStateNotFound is returned when a referenced state is not found
	ErrStateNotFound = &StateMachineError{
		Code:    "STATE_NOT_FOUND",
		Message: "state not found in state machine",
	}

	// ErrTransitionNotFound is returned when a referenced transition is not found
	ErrTransitionNotFound = &StateMachineError{
		Code:    "TRANSITION_NOT_FOUND",
		Message: "transition not found",
	}

	// ErrGuardConditionFailed is returned when a guard condition prevents a transition
	ErrGuardConditionFailed = &StateMachineError{
		Code:    "GUARD_CONDITION_FAILED",
		Message: "guard condition failed",
	}

	// ErrActionExecutionFailed is returned when an action fails during execution
	ErrActionExecutionFailed = &StateMachineError{
		Code:    "ACTION_EXECUTION_FAILED",
		Message: "action execution failed",
	}

	// ErrNoInitialState is returned when the state machine has no initial state
	ErrNoInitialState = &StateMachineError{
		Code:    "NO_INITIAL_STATE",
		Message: "state machine has no initial state",
	}

	// ErrAlreadyStarted is returned when the state machine is already started
	ErrAlreadyStarted = &StateMachineError{
		Code:    "ALREADY_STARTED",
		Message: "state machine has already been started",
	}

	// ErrNotStarted is returned when the state machine has not been started
	ErrNotStarted = &StateMachineError{
		Code:    "NOT_STARTED",
		Message: "state machine has not been started",
	}

	// ErrMachineStopped is returned when trying to use a stopped state machine
	ErrMachineStopped = &StateMachineError{
		Code:    "MACHINE_STOPPED",
		Message: "state machine has been stopped",
	}

	// ErrEventTimeout is returned when event processing times out
	ErrEventTimeout = &StateMachineError{
		Code:    "EVENT_TIMEOUT",
		Message: "event processing timed out",
	}

	// ErrParallelRegionSync is returned when parallel region synchronization fails
	ErrParallelRegionSync = &StateMachineError{
		Code:    "PARALLEL_REGION_SYNC",
		Message: "parallel region synchronization failed",
	}

	// ErrHistoryStateInvalid is returned when history state restoration fails
	ErrHistoryStateInvalid = &StateMachineError{
		Code:    "HISTORY_STATE_INVALID",
		Message: "invalid history state restoration",
	}

	// ErrSubmachineError is returned when submachine operations fail
	ErrSubmachineError = &StateMachineError{
		Code:    "SUBMACHINE_ERROR",
		Message: "submachine operation failed",
	}

	// ErrValidationFailed is returned when state machine validation fails
	ErrValidationFailed = &StateMachineError{
		Code:    "VALIDATION_FAILED",
		Message: "state machine validation failed",
	}
)

// NewTransitionError creates an error for transition issues
func NewTransitionError(message string, fromState, toState, event string) *StateMachineError {
	return &StateMachineError{
		Code:      "TRANSITION_ERROR",
		Message:   message,
		StateID:   fromState,
		EventType: event,
		Details: map[string]interface{}{
			"target_state": toState,
		},
	}
}

// NewStateError creates an error for state issues
func NewStateError(message string, stateID string) *StateMachineError {
	return &StateMachineError{
		Code:    "STATE_ERROR",
		Message: message,
		StateID: stateID,
	}
}

// NewEventError creates an error for event issues
func NewEventError(message string, eventType string) *StateMachineError {
	return &StateMachineError{
		Code:      "EVENT_ERROR",
		Message:   message,
		EventType: eventType,
	}
}

// NewActionError creates an error for action issues
func NewActionError(message string, stateID string, cause error) *StateMachineError {
	return &StateMachineError{
		Code:    "ACTION_ERROR",
		Message: message,
		StateID: stateID,
		Cause:   cause,
	}
}

// NewGuardError creates an error for guard issues
func NewGuardError(message string, stateID string, eventType string) *StateMachineError {
	return &StateMachineError{
		Code:      "GUARD_ERROR",
		Message:   message,
		StateID:   stateID,
		EventType: eventType,
	}
}

// NewConfigurationError creates an error for configuration issues
func NewConfigurationError(message string) *StateMachineError {
	return &StateMachineError{
		Code:    "CONFIGURATION_ERROR",
		Message: message,
	}
}

// NewTimeoutError creates an error for timeout situations
func NewTimeoutError(operation string, duration interface{}) *StateMachineError {
	return &StateMachineError{
		Code:    "TIMEOUT_ERROR",
		Message: fmt.Sprintf("operation %s timed out after %v", operation, duration),
		Details: map[string]interface{}{
			"operation": operation,
			"duration":  duration,
		},
	}
}

// NewParallelError creates an error for parallel region issues
func NewParallelError(message string, regionID string) *StateMachineError {
	return &StateMachineError{
		Code:    "PARALLEL_ERROR",
		Message: message,
		Details: map[string]interface{}{
			"region_id": regionID,
		},
	}
}

// NewHistoryError creates an error for history state issues
func NewHistoryError(message string, historyType string) *StateMachineError {
	return &StateMachineError{
		Code:    "HISTORY_ERROR",
		Message: message,
		Details: map[string]interface{}{
			"history_type": historyType,
		},
	}
}

// NewSubmachineError creates an error for submachine issues
func NewSubmachineError(message string, submachineID string, cause error) *StateMachineError {
	return &StateMachineError{
		Code:    "SUBMACHINE_ERROR",
		Message: message,
		Details: map[string]interface{}{
			"submachine_id": submachineID,
		},
		Cause: cause,
	}
}

// ErrorCollector collects multiple errors during validation or processing
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns whether any errors were collected
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// GetErrors returns all collected errors
func (ec *ErrorCollector) GetErrors() []error {
	return ec.errors
}

// Error returns a string representation of all errors
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return "no errors"
	}

	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d errors occurred:\n", len(ec.errors)))

	for i, err := range ec.errors {
		sb.WriteString(fmt.Sprintf("  %d: %v\n", i+1, err))
	}

	return sb.String()
}
