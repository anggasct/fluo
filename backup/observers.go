package flux

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	// LogLevelError logs only errors
	LogLevelError LogLevel = iota
	// LogLevelWarn logs warnings and errors
	LogLevelWarn
	// LogLevelInfo logs info, warnings, and errors
	LogLevelInfo
	// LogLevelDebug logs everything
	LogLevelDebug
)

// LoggingObserver implements StateMachineObserver for logging
type LoggingObserver struct {
	level  LogLevel
	prefix string
}

// NewLoggingObserver creates a new logging observer
func NewLoggingObserver(level LogLevel, prefix string) *LoggingObserver {
	return &LoggingObserver{
		level:  level,
		prefix: prefix,
	}
}

// OnStateEnter logs state entry
func (o *LoggingObserver) OnStateEnter(sm *StateMachine, state State) {
	if o.level >= LogLevelInfo {
		log.Printf("[%s] State entered: %s", o.prefix, state.Name())
	}
}

// OnStateExit logs state exit
func (o *LoggingObserver) OnStateExit(sm *StateMachine, state State) {
	if o.level >= LogLevelInfo {
		log.Printf("[%s] State exited: %s", o.prefix, state.Name())
	}
}

// OnTransition logs state transitions
func (o *LoggingObserver) OnTransition(sm *StateMachine, from, to State, event *Event) {
	if o.level >= LogLevelInfo {
		log.Printf("[%s] Transition: %s -> %s (event: %s)", o.prefix, from.Name(), to.Name(), event.Name)
	}
}

// OnEventProcessed logs event processing
func (o *LoggingObserver) OnEventProcessed(sm *StateMachine, event *Event) {
	if o.level >= LogLevelDebug {
		log.Printf("[%s] Event processed: %s", o.prefix, event.Name)
	}
}

// OnError logs errors
func (o *LoggingObserver) OnError(sm *StateMachine, err error) {
	if o.level >= LogLevelError {
		log.Printf("[%s] Error: %v", o.prefix, err)
	}
}

// MetricsObserver collects metrics about state machine execution
type MetricsObserver struct {
	stateEnterCounts   map[string]int
	stateExitCounts    map[string]int
	transitionCounts   map[string]int
	eventCounts        map[string]int
	errorCount         int
	totalTransitions   int
	executionStartTime time.Time
	lastTransitionTime time.Time
	stateDurations     map[string]time.Duration
	stateEnterTimes    map[string]time.Time
}

// NewMetricsObserver creates a new metrics observer
func NewMetricsObserver() *MetricsObserver {
	return &MetricsObserver{
		stateEnterCounts: make(map[string]int),
		stateExitCounts:  make(map[string]int),
		transitionCounts: make(map[string]int),
		eventCounts:      make(map[string]int),
		stateDurations:   make(map[string]time.Duration),
		stateEnterTimes:  make(map[string]time.Time),
	}
}

// OnStateEnter records state entry metrics
func (o *MetricsObserver) OnStateEnter(sm *StateMachine, state State) {
	stateName := state.Name()
	o.stateEnterCounts[stateName]++
	o.stateEnterTimes[stateName] = time.Now()

	if o.executionStartTime.IsZero() {
		o.executionStartTime = time.Now()
	}
}

// OnStateExit records state exit metrics
func (o *MetricsObserver) OnStateExit(sm *StateMachine, state State) {
	stateName := state.Name()
	o.stateExitCounts[stateName]++

	if enterTime, exists := o.stateEnterTimes[stateName]; exists {
		duration := time.Since(enterTime)
		o.stateDurations[stateName] += duration
		delete(o.stateEnterTimes, stateName)
	}
}

// OnTransition records transition metrics
func (o *MetricsObserver) OnTransition(sm *StateMachine, from, to State, event *Event) {
	transitionKey := fmt.Sprintf("%s->%s", from.Name(), to.Name())
	o.transitionCounts[transitionKey]++
	o.totalTransitions++
	o.lastTransitionTime = time.Now()
}

// OnEventProcessed records event processing metrics
func (o *MetricsObserver) OnEventProcessed(sm *StateMachine, event *Event) {
	o.eventCounts[event.Name]++
}

// OnError records error metrics
func (o *MetricsObserver) OnError(sm *StateMachine, err error) {
	o.errorCount++
}

// GetStateEnterCount returns the enter count for a state
func (o *MetricsObserver) GetStateEnterCount(stateName string) int {
	return o.stateEnterCounts[stateName]
}

// GetTransitionCount returns the count for a specific transition
func (o *MetricsObserver) GetTransitionCount(from, to string) int {
	key := fmt.Sprintf("%s->%s", from, to)
	return o.transitionCounts[key]
}

// GetEventCount returns the count for a specific event
func (o *MetricsObserver) GetEventCount(eventName string) int {
	return o.eventCounts[eventName]
}

// GetTotalTransitions returns the total number of transitions
func (o *MetricsObserver) GetTotalTransitions() int {
	return o.totalTransitions
}

// GetErrorCount returns the total number of errors
func (o *MetricsObserver) GetErrorCount() int {
	return o.errorCount
}

// GetExecutionDuration returns the total execution duration
func (o *MetricsObserver) GetExecutionDuration() time.Duration {
	if o.executionStartTime.IsZero() {
		return 0
	}
	if o.lastTransitionTime.IsZero() {
		return time.Since(o.executionStartTime)
	}
	return o.lastTransitionTime.Sub(o.executionStartTime)
}

// GetStateDuration returns the total duration spent in a state
func (o *MetricsObserver) GetStateDuration(stateName string) time.Duration {
	return o.stateDurations[stateName]
}

// GetReport generates a metrics report
func (o *MetricsObserver) GetReport() string {
	var report strings.Builder

	report.WriteString("State Machine Metrics Report\n")
	report.WriteString("============================\n\n")

	report.WriteString("Execution Summary:\n")
	report.WriteString(fmt.Sprintf("  Total Transitions: %d\n", o.totalTransitions))
	report.WriteString(fmt.Sprintf("  Total Errors: %d\n", o.errorCount))
	report.WriteString(fmt.Sprintf("  Execution Duration: %v\n\n", o.GetExecutionDuration()))

	report.WriteString("State Statistics:\n")
	for stateName, count := range o.stateEnterCounts {
		duration := o.stateDurations[stateName]
		report.WriteString(fmt.Sprintf("  %s: entered %d times, total duration %v\n",
			stateName, count, duration))
	}

	report.WriteString("\nTransition Statistics:\n")
	for transition, count := range o.transitionCounts {
		report.WriteString(fmt.Sprintf("  %s: %d times\n", transition, count))
	}

	report.WriteString("\nEvent Statistics:\n")
	for eventName, count := range o.eventCounts {
		report.WriteString(fmt.Sprintf("  %s: %d times\n", eventName, count))
	}

	return report.String()
}

// ValidationObserver validates state machine behavior
type ValidationObserver struct {
	expectedStates     map[string]bool
	allowedTransitions map[string][]string
	violations         []string
}

// NewValidationObserver creates a new validation observer
func NewValidationObserver() *ValidationObserver {
	return &ValidationObserver{
		expectedStates:     make(map[string]bool),
		allowedTransitions: make(map[string][]string),
		violations:         make([]string, 0),
	}
}

// AddExpectedState adds a state that should be visited
func (o *ValidationObserver) AddExpectedState(stateName string) {
	o.expectedStates[stateName] = false
}

// AddAllowedTransition adds an allowed transition
func (o *ValidationObserver) AddAllowedTransition(from, to string) {
	if o.allowedTransitions[from] == nil {
		o.allowedTransitions[from] = make([]string, 0)
	}
	o.allowedTransitions[from] = append(o.allowedTransitions[from], to)
}

// OnStateEnter validates state entry
func (o *ValidationObserver) OnStateEnter(sm *StateMachine, state State) {
	if _, exists := o.expectedStates[state.Name()]; exists {
		o.expectedStates[state.Name()] = true
	}
}

// OnStateExit validates state exit
func (o *ValidationObserver) OnStateExit(sm *StateMachine, state State) {
}

// OnTransition validates transitions
func (o *ValidationObserver) OnTransition(sm *StateMachine, from, to State, event *Event) {
	if allowed, exists := o.allowedTransitions[from.Name()]; exists {
		isAllowed := false
		for _, allowedTo := range allowed {
			if allowedTo == to.Name() {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			violation := fmt.Sprintf("Invalid transition: %s -> %s", from.Name(), to.Name())
			o.violations = append(o.violations, violation)
		}
	}
}

// OnEventProcessed validates event processing
func (o *ValidationObserver) OnEventProcessed(sm *StateMachine, event *Event) {
}

// OnError validates error handling
func (o *ValidationObserver) OnError(sm *StateMachine, err error) {
}

// GetViolations returns all validation violations
func (o *ValidationObserver) GetViolations() []string {
	return o.violations
}

// GetUnvisitedStates returns states that were expected but not visited
func (o *ValidationObserver) GetUnvisitedStates() []string {
	unvisited := make([]string, 0)
	for stateName, visited := range o.expectedStates {
		if !visited {
			unvisited = append(unvisited, stateName)
		}
	}
	return unvisited
}

// IsValid returns whether the state machine execution was valid
func (o *ValidationObserver) IsValid() bool {
	return len(o.violations) == 0 && len(o.GetUnvisitedStates()) == 0
}
