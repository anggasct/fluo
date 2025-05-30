// Package fluo provides a comprehensive finite state machine library for Go
// that implements UML state machine concepts including hierarchical states,
// parallel regions, choice points, and event deferring.
package fluo

import (
	"time"

	"github.com/anggasct/fluo/pkg/builders"
	"github.com/anggasct/fluo/pkg/core"
	"github.com/anggasct/fluo/pkg/observers"
	"github.com/anggasct/fluo/pkg/states"
	"github.com/anggasct/fluo/pkg/utils"
)

// Core types
type (
	// StateMachine is the main state machine engine
	StateMachine = core.StateMachine

	// State represents a state in the state machine
	State = core.State

	// Event represents a state machine event with optional data and metadata
	Event = core.Event

	// EventPriority defines the priority level of an event
	EventPriority = core.EventPriority

	// EventFilter is a function that evaluates whether an event should be processed
	EventFilter = core.EventFilter

	// EventHandler defines a function that handles events
	EventHandler = core.EventHandler

	// Context holds the execution context for state machine operations
	Context = core.Context

	// GuardCondition evaluates whether a transition should be taken
	GuardCondition = core.GuardCondition

	// Action performs an operation during state transitions or state activities
	Action = core.Action

	// Transition represents a transition between states
	Transition = core.Transition

	// DeferredEvent represents an event that has been deferred
	DeferredEvent = core.DeferredEvent
)

// Re-export state types
type (
	// BaseState provides common functionality for all state types
	BaseState = states.BaseState

	// SimpleState represents a basic state with no internal structure
	SimpleState = states.SimpleState

	// CompositeState represents a hierarchical state that contains child states
	CompositeState = states.CompositeState

	// FinalState represents a terminal state
	FinalState = states.FinalState

	// ChoiceState represents a state that dynamically chooses between outgoing transitions
	ChoiceState = states.ChoiceState

	// HistoryState represents a pseudo-state that remembers the previously active state
	HistoryState = states.HistoryState

	// HistoryType specifies the type of history (shallow or deep)
	HistoryType = states.HistoryType

	// EntryPointState represents an entry point to a composite state
	EntryPointState = states.EntryPointState

	// ExitPointState represents an exit point from a composite state
	ExitPointState = states.ExitPointState

	// ParallelState manages multiple independent state machine regions
	ParallelState = states.ParallelState

	// ParallelRegion represents an independent region within a parallel state
	ParallelRegion = states.ParallelRegion

	// SubmachineState represents a state that encapsulates another state machine
	SubmachineState = states.SubmachineState

	// SubmachineConnector defines connection points between parent and submachine
	SubmachineConnector = states.SubmachineConnector

	// DeferState is a state that can defer specific events
	DeferState = states.DeferState

	// TimeoutState represents a state that transitions after a timeout
	TimeoutState = states.TimeoutState
)

// Re-export builder types
type (
	// StateMachineBuilder provides a fluent interface for building state machines
	StateMachineBuilder = builders.StateMachineBuilder

	// WorkflowBuilder provides specialized builder for workflow patterns
	WorkflowBuilder = builders.WorkflowBuilder

	// ValidationBuilder helps build validation rules for state machines
	ValidationBuilder = builders.ValidationBuilder

	// ConditionalActions provides helper functions for common conditional actions
	ConditionalActions = builders.ConditionalActions

	// SubmachineBuilder helps build submachine states with fluent API
	SubmachineBuilder = builders.SubmachineBuilder
)

// Re-export observer types
type (
	// LoggingObserver logs state machine events
	LoggingObserver = observers.LoggingObserver

	// LogLevel represents the logging level
	LogLevel = observers.LogLevel

	// LogFormatter formats log messages
	LogFormatter = observers.LogFormatter

	// ValidationObserver validates state machine behavior
	ValidationObserver = observers.ValidationObserver

	// MetricsObserver collects metrics about state machine execution
	MetricsObserver = observers.MetricsObserver
)

// Re-export error types
type (
	// StateMachineError represents a state machine specific error
	StateMachineError = utils.StateMachineError

	// ErrorCollector collects multiple errors during validation or processing
	ErrorCollector = utils.ErrorCollector
)

// Re-export constants
const (
	// ShallowHistory remembers only the direct substate that was active
	ShallowHistory = states.ShallowHistory

	// DeepHistory remembers the full path to the deepest active substate
	DeepHistory = states.DeepHistory

	// Event priority levels
	LowPriority      = core.LowPriority
	NormalPriority   = core.NormalPriority
	HighPriority     = core.HighPriority
	CriticalPriority = core.CriticalPriority

	// LogError logs only errors
	LogError = observers.LogError

	// LogWarning logs errors and warnings
	LogWarning = observers.LogWarning

	// LogInfo logs errors, warnings, and info
	LogInfo = observers.LogInfo

	// LogDebug logs errors, warnings, info, and debug
	LogDebug = observers.LogDebug
)

// Re-export core functions
var (
	// NewStateMachine creates a new state machine with the given name
	NewStateMachine = core.NewStateMachine

	// NewEvent creates a new event with the given name
	NewEvent = core.NewEvent

	// NewEventWithData creates a new event with name and data
	NewEventWithData = core.NewEventWithData

	// NewContext creates a new context for state machine operations
	NewContext = core.NewContext

	// NewTransition creates a new transition
	NewTransition = core.NewTransition

	// NewEventDeferrer creates a new event deferrer
	NewEventDeferrer = core.NewEventDeferrer
)

// Re-export state constructors
var (
	// NewBaseState creates a new base state
	NewBaseState = states.NewBaseState

	// NewSimpleState creates a new simple state
	NewSimpleState = states.NewSimpleState

	// NewCompositeState creates a new hierarchical composite state
	NewCompositeState = states.NewCompositeState

	// NewFinalState creates a new final state
	NewFinalState = states.NewFinalState

	// NewChoiceState creates a new choice state
	NewChoiceState = states.NewChoiceState

	// NewHistoryState creates a new history state with specified type
	NewHistoryState = states.NewHistoryState

	// NewEntryPointState creates a new entry point state
	NewEntryPointState = states.NewEntryPointState

	// NewExitPointState creates a new exit point state
	NewExitPointState = states.NewExitPointState

	// NewParallelState creates a new parallel state
	NewParallelState = states.NewParallelState

	// NewParallelRegion creates a new parallel region
	NewParallelRegion = states.NewParallelRegion

	// NewSubmachineState creates a new submachine state
	NewSubmachineState = states.NewSubmachineState

	// NewSubmachineConnector creates a new submachine connector
	NewSubmachineConnector = states.NewSubmachineConnector

	// NewDeferState creates a new defer state
	NewDeferState = states.NewDeferState

	// NewTimeoutState creates a new timeout state
	NewTimeoutState = states.NewTimeoutState
)

// Re-export builder constructors
var (
	// NewStateMachineBuilder creates a new state machine builder
	NewStateMachineBuilder = builders.NewStateMachineBuilder

	// NewWorkflowBuilder creates a new workflow builder
	NewWorkflowBuilder = builders.NewWorkflowBuilder

	// NewValidationBuilder creates a new validation builder
	NewValidationBuilder = builders.NewValidationBuilder

	// NewSubmachineBuilder creates a new submachine builder
	NewSubmachineBuilder = builders.NewSubmachineBuilder

	// Conditions provides a singleton instance of ConditionalActions
	Conditions = builders.Conditions
)

// Re-export observer constructors
var (
	// NewLoggingObserver creates a new logging observer with default settings
	NewLoggingObserver = observers.NewDefaultLoggingObserver

	// NewCustomLoggingObserver creates a new logging observer with custom settings
	NewCustomLoggingObserver = observers.NewLoggingObserver

	// DefaultLogFormatter provides default log formatting
	DefaultLogFormatter = observers.DefaultLogFormatter

	// NewValidationObserver creates a new validation observer
	NewValidationObserver = observers.NewValidationObserver

	// NewMetricsObserver creates a new metrics observer
	NewMetricsObserver = observers.NewMetricsObserver
)

// Re-export error constructors and variables
var (
	// ErrInvalidTransition is returned when a transition is not valid from the current state
	ErrInvalidTransition = utils.ErrInvalidTransition

	// ErrStateNotFound is returned when a referenced state is not found
	ErrStateNotFound = utils.ErrStateNotFound

	// ErrTransitionNotFound is returned when a referenced transition is not found
	ErrTransitionNotFound = utils.ErrTransitionNotFound

	// ErrGuardConditionFailed is returned when a guard condition prevents a transition
	ErrGuardConditionFailed = utils.ErrGuardConditionFailed

	// ErrActionExecutionFailed is returned when an action fails during execution
	ErrActionExecutionFailed = utils.ErrActionExecutionFailed

	// ErrNoInitialState is returned when the state machine has no initial state
	ErrNoInitialState = utils.ErrNoInitialState

	// ErrAlreadyStarted is returned when the state machine is already started
	ErrAlreadyStarted = utils.ErrAlreadyStarted

	// ErrNotStarted is returned when the state machine has not been started
	ErrNotStarted = utils.ErrNotStarted

	// ErrMachineStopped is returned when trying to use a stopped state machine
	ErrMachineStopped = utils.ErrMachineStopped

	// ErrEventTimeout is returned when event processing times out
	ErrEventTimeout = utils.ErrEventTimeout

	// ErrParallelRegionSync is returned when parallel region synchronization fails
	ErrParallelRegionSync = utils.ErrParallelRegionSync

	// ErrHistoryStateInvalid is returned when history state restoration fails
	ErrHistoryStateInvalid = utils.ErrHistoryStateInvalid

	// ErrSubmachineError is returned when submachine operations fail
	ErrSubmachineError = utils.ErrSubmachineError

	// ErrValidationFailed is returned when state machine validation fails
	ErrValidationFailed = utils.ErrValidationFailed

	// NewTransitionError creates an error for transition issues
	NewTransitionError = utils.NewTransitionError

	// NewStateError creates an error for state issues
	NewStateError = utils.NewStateError

	// NewEventError creates an error for event issues
	NewEventError = utils.NewEventError

	// NewActionError creates an error for action issues
	NewActionError = utils.NewActionError

	// NewGuardError creates an error for guard issues
	NewGuardError = utils.NewGuardError

	// NewConfigurationError creates an error for configuration issues
	NewConfigurationError = utils.NewConfigurationError

	// NewTimeoutError creates an error for timeout situations
	NewTimeoutError = utils.NewTimeoutError

	// NewParallelError creates an error for parallel region issues
	NewParallelError = utils.NewParallelError

	// NewHistoryError creates an error for history state issues
	NewHistoryError = utils.NewHistoryError

	// NewSubmachineError creates an error for submachine issues
	NewSubmachineError = utils.NewSubmachineError

	// NewErrorCollector creates a new error collector
	NewErrorCollector = utils.NewErrorCollector
)

// Duration converts an integer to a time.Duration
func Duration(milliseconds int) time.Duration {
	return time.Duration(milliseconds) * time.Millisecond
}
