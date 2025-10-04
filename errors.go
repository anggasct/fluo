package fluo

import "fmt"

// ErrorCode represents specific error conditions in the state machine
type ErrorCode int

const (
	// No error occurred
	ErrCodeNone ErrorCode = iota
	// State was not found in the machine
	ErrCodeStateNotFound
	// Transition is not allowed from current state
	ErrCodeTransitionNotAllowed
	// Guard condition rejected the transition
	ErrCodeGuardRejected
	// Event is invalid for current context
	ErrCodeInvalidEvent
	// Machine is not in started state
	ErrCodeMachineNotStarted
	// Action execution failed
	ErrCodeActionFailed
	// Machine configuration is invalid
	ErrCodeInvalidConfiguration
	// State is in invalid condition
	ErrCodeInvalidState
	// Concurrent modification detected
	ErrCodeConcurrentModification
)

// StateError represents state-related errors
type StateError struct {
	Code    ErrorCode
	StateID string
	Message string
}

func (e *StateError) Error() string {
	return fmt.Sprintf("state error [%s]: %s", e.StateID, e.Message)
}

// NewStateNotFoundError creates a new state not found error
func NewStateNotFoundError(stateID string) *StateError {
	return &StateError{
		Code:    ErrCodeStateNotFound,
		StateID: stateID,
		Message: fmt.Sprintf("state '%s' not found", stateID),
	}
}

// NewStateError creates a new state error with custom values
func NewStateError(code ErrorCode, stateID string, message string) *StateError {
	return &StateError{
		Code:    code,
		StateID: stateID,
		Message: message,
	}
}

// NewInvalidStateError creates a new invalid state error
func NewInvalidStateError(stateID string, reason string) *StateError {
	return &StateError{
		Code:    ErrCodeInvalidState,
		StateID: stateID,
		Message: reason,
	}
}

// TransitionError represents transition-related errors
type TransitionError struct {
	Code   ErrorCode
	From   string
	To     string
	Event  string
	Reason string
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("transition error [%s->%s on %s]: %s", e.From, e.To, e.Event, e.Reason)
}

// NewTransitionNotAllowedError creates a new transition not allowed error
func NewTransitionNotAllowedError(from, to, event string) *TransitionError {
	return &TransitionError{
		Code:   ErrCodeTransitionNotAllowed,
		From:   from,
		To:     to,
		Event:  event,
		Reason: "transition not allowed",
	}
}

// NewTransitionError creates a new transition error with custom values
func NewTransitionError(code ErrorCode, from, to, event, reason string) *TransitionError {
	return &TransitionError{
		Code:   code,
		From:   from,
		To:     to,
		Event:  event,
		Reason: reason,
	}
}

// NewNoTransitionError creates a new no transition found error
func NewNoTransitionError(from, event string) *TransitionError {
	return &TransitionError{
		Code:   ErrCodeTransitionNotAllowed,
		From:   from,
		Event:  event,
		Reason: fmt.Sprintf("no transition found from state '%s' for event '%s'", from, event),
	}
}

// GuardError represents guard condition failures
type GuardError struct {
	From  string
	To    string
	Event string
	Guard string
}

func (e *GuardError) Error() string {
	if e.Guard != "" {
		return fmt.Sprintf("guard rejected transition [%s->%s on %s]: %s", e.From, e.To, e.Event, e.Guard)
	}
	return fmt.Sprintf("guard rejected transition [%s->%s on %s]", e.From, e.To, e.Event)
}

// NewGuardRejectedError creates a new guard rejected error
func NewGuardRejectedError(from, to, event string, guardName string) *GuardError {
	return &GuardError{
		From:  from,
		To:    to,
		Event: event,
		Guard: guardName,
	}
}

// ConfigurationError represents machine configuration issues
type ConfigurationError struct {
	Component string
	Issue     string
}

func (e *ConfigurationError) Error() string {
	return fmt.Sprintf("configuration error in %s: %s", e.Component, e.Issue)
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(component, issue string) *ConfigurationError {
	return &ConfigurationError{
		Component: component,
		Issue:     issue,
	}
}

// MachineError represents state machine operation errors
type MachineError struct {
	Code      ErrorCode
	Operation string
	Message   string
}

func (e *MachineError) Error() string {
	return fmt.Sprintf("machine error during %s: %s", e.Operation, e.Message)
}

// NewMachineNotStartedError creates a new machine not started error
func NewMachineNotStartedError(operation string) *MachineError {
	return &MachineError{
		Code:      ErrCodeMachineNotStarted,
		Operation: operation,
		Message:   "state machine is not started",
	}
}

// NewMachineError creates a new machine error
func NewMachineError(code ErrorCode, operation string, message string) *MachineError {
	return &MachineError{
		Code:      code,
		Operation: operation,
		Message:   message,
	}
}

// ActionError represents action execution errors
type ActionError struct {
	Action      string
	State       string
	OriginalErr error
}

func (e *ActionError) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("action '%s' failed in state '%s': %v", e.Action, e.State, e.OriginalErr)
	}
	return fmt.Sprintf("action '%s' failed in state '%s'", e.Action, e.State)
}

func (e *ActionError) Unwrap() error {
	return e.OriginalErr
}

// NewActionError creates a new action execution error
func NewActionError(action, state string, err error) *ActionError {
	return &ActionError{
		Action:      action,
		State:       state,
		OriginalErr: err,
	}
}

// IsStateError checks if an error is a StateError
func IsStateError(err error) bool {
	_, ok := err.(*StateError)
	return ok
}

// IsTransitionError checks if an error is a TransitionError
func IsTransitionError(err error) bool {
	_, ok := err.(*TransitionError)
	return ok
}

// IsGuardError checks if an error is a GuardError
func IsGuardError(err error) bool {
	_, ok := err.(*GuardError)
	return ok
}

// IsConfigurationError checks if an error is a ConfigurationError
func IsConfigurationError(err error) bool {
	_, ok := err.(*ConfigurationError)
	return ok
}

// IsMachineError checks if an error is a MachineError
func IsMachineError(err error) bool {
	_, ok := err.(*MachineError)
	return ok
}

// IsActionError checks if an error is an ActionError
func IsActionError(err error) bool {
	_, ok := err.(*ActionError)
	return ok
}

// GetErrorCode returns the error code for known error types
func GetErrorCode(err error) ErrorCode {
	switch e := err.(type) {
	case *StateError:
		return e.Code
	case *TransitionError:
		return e.Code
	case *MachineError:
		return e.Code
	case *GuardError:
		return ErrCodeGuardRejected
	case *ConfigurationError:
		return ErrCodeInvalidConfiguration
	case *ActionError:
		return ErrCodeActionFailed
	default:
		return ErrCodeNone
	}
}
