package flux

import (
	"fmt"
	"strings"
)

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

	// ErrCyclicTransition is returned when a cyclic transition is detected in validation
	ErrCyclicTransition = &StateMachineError{
		Code:    "CYCLIC_TRANSITION",
		Message: "cyclic transition detected",
	}

	// ErrInvalidConfiguration is returned when state machine configuration is invalid
	ErrInvalidConfiguration = &StateMachineError{
		Code:    "INVALID_CONFIGURATION",
		Message: "invalid state machine configuration",
	}

	// ErrMachineNotStarted is returned when operations are attempted on non-started machine
	ErrMachineNotStarted = &StateMachineError{
		Code:    "MACHINE_NOT_STARTED",
		Message: "state machine has not been started",
	}

	// ErrMachineAlreadyStarted is returned when trying to start an already started machine
	ErrMachineAlreadyStarted = &StateMachineError{
		Code:    "MACHINE_ALREADY_STARTED",
		Message: "state machine has already been started",
	}

	// ErrMachineStopped is returned when operations are attempted on stopped machine
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

// StateMachineError represents an error specific to state machine operations
type StateMachineError struct {
	Code      string
	Message   string
	Details   map[string]interface{}
	Cause     error
	StateID   string
	EventType string
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

// Unwrap returns the underlying cause error
func (e *StateMachineError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error
func (e *StateMachineError) Is(target error) bool {
	if t, ok := target.(*StateMachineError); ok {
		return e.Code == t.Code
	}
	return false
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

// WithDetails adds additional details to the error
func (e *StateMachineError) WithDetails(details map[string]interface{}) *StateMachineError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithDetail adds a single detail to the error
func (e *StateMachineError) WithDetail(key string, value interface{}) *StateMachineError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause wraps another error as the cause
func (e *StateMachineError) WithCause(cause error) *StateMachineError {
	e.Cause = cause
	return e
}

// NewStateMachineError creates a new state machine error
func NewStateMachineError(code, message string) *StateMachineError {
	return &StateMachineError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// NewTransitionError creates an error for invalid transitions
func NewTransitionError(fromState, event, reason string) *StateMachineError {
	return NewStateMachineError("INVALID_TRANSITION", "transition not allowed").
		WithState(fromState).
		WithEvent(event).
		WithDetail("reason", reason)
}

// NewGuardError creates an error for failed guard conditions
func NewGuardError(stateID, event, condition string) *StateMachineError {
	return NewStateMachineError("GUARD_CONDITION_FAILED", "guard condition failed").
		WithState(stateID).
		WithEvent(event).
		WithDetail("condition", condition)
}

// NewActionError creates an error for failed actions
func NewActionError(stateID, event, action string, cause error) *StateMachineError {
	return NewStateMachineError("ACTION_EXECUTION_FAILED", "action execution failed").
		WithState(stateID).
		WithEvent(event).
		WithDetail("action", action).
		WithCause(cause)
}

// NewValidationError creates an error for validation failures
func NewValidationError(message string, details map[string]interface{}) *StateMachineError {
	return NewStateMachineError("VALIDATION_FAILED", message).
		WithDetails(details)
}

// NewConfigurationError creates an error for configuration issues
func NewConfigurationError(message string) *StateMachineError {
	return NewStateMachineError("INVALID_CONFIGURATION", message)
}

// NewTimeoutError creates an error for timeout situations
func NewTimeoutError(operation string, duration interface{}) *StateMachineError {
	return NewStateMachineError("EVENT_TIMEOUT", "operation timed out").
		WithDetail("operation", operation).
		WithDetail("timeout", duration)
}

// NewParallelError creates an error for parallel region issues
func NewParallelError(message string, regionID string) *StateMachineError {
	return NewStateMachineError("PARALLEL_REGION_SYNC", message).
		WithDetail("regionID", regionID)
}

// NewHistoryError creates an error for history state issues
func NewHistoryError(message string, historyType string) *StateMachineError {
	return NewStateMachineError("HISTORY_STATE_INVALID", message).
		WithDetail("historyType", historyType)
}

// NewSubmachineError creates an error for submachine issues
func NewSubmachineError(message string, submachineID string, cause error) *StateMachineError {
	return NewStateMachineError("SUBMACHINE_ERROR", message).
		WithDetail("submachineID", submachineID).
		WithCause(cause)
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

// AddWithContext adds an error with additional context
func (ec *ErrorCollector) AddWithContext(err error, context string) {
	if err != nil {
		contextErr := fmt.Errorf("%s: %w", context, err)
		ec.errors = append(ec.errors, contextErr)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Error returns a combined error message
func (ec *ErrorCollector) Error() error {
	if len(ec.errors) == 0 {
		return nil
	}

	if len(ec.errors) == 1 {
		return ec.errors[0]
	}

	var messages []string
	for i, err := range ec.errors {
		messages = append(messages, fmt.Sprintf("%d. %v", i+1, err))
	}

	return fmt.Errorf("multiple errors occurred:\n%s", strings.Join(messages, "\n"))
}

// Clear removes all errors from the collector
func (ec *ErrorCollector) Clear() {
	ec.errors = ec.errors[:0]
}
