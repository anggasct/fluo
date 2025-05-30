package observers

import (
	"fmt"
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// ValidationObserver validates state machine behavior
type ValidationObserver struct {
	expectedStates     map[string]bool
	visitedStates      map[string]bool
	allowedTransitions map[string]map[string]bool
	violations         []string
	mutex              sync.RWMutex
}

// NewValidationObserver creates a new validation observer
func NewValidationObserver() *ValidationObserver {
	return &ValidationObserver{
		expectedStates:     make(map[string]bool),
		visitedStates:      make(map[string]bool),
		allowedTransitions: make(map[string]map[string]bool),
		violations:         make([]string, 0),
	}
}

// AddExpectedState adds an expected state
func (o *ValidationObserver) AddExpectedState(stateName string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.expectedStates[stateName] = true
}

// AddAllowedTransition adds an allowed transition
func (o *ValidationObserver) AddAllowedTransition(from, to string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if _, exists := o.allowedTransitions[from]; !exists {
		o.allowedTransitions[from] = make(map[string]bool)
	}

	o.allowedTransitions[from][to] = true
}

// addViolation adds a violation
func (o *ValidationObserver) addViolation(message string) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.violations = append(o.violations, message)
}

// OnStateEnter validates state entry
func (o *ValidationObserver) OnStateEnter(sm *core.StateMachine, state core.State) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Mark state as visited
	o.visitedStates[state.Name()] = true
}

// OnStateExit validates state exit
func (o *ValidationObserver) OnStateExit(sm *core.StateMachine, state core.State) {
	// No validation needed for exit
}

// OnTransition validates transitions
func (o *ValidationObserver) OnTransition(sm *core.StateMachine, from, to core.State, event *core.Event) {
	// Skip if either state is nil
	if from == nil || to == nil {
		return
	}

	fromName := from.Name()
	toName := to.Name()

	// Check if this transition is allowed
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if allowed, exists := o.allowedTransitions[fromName]; exists {
		if !allowed[toName] {
			o.violations = append(o.violations, fmt.Sprintf(
				"Invalid transition from '%s' to '%s' on event '%s'",
				fromName, toName, event.Name))
		}
	}
}

// OnEventProcessed validates event processing
func (o *ValidationObserver) OnEventProcessed(sm *core.StateMachine, event *core.Event) {
	// No validation needed for events
}

// OnError validates error handling
func (o *ValidationObserver) OnError(sm *core.StateMachine, err error) {
	o.addViolation(fmt.Sprintf("Error occurred: %v", err))
}

// GetViolations returns all validation violations
func (o *ValidationObserver) GetViolations() []string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	result := make([]string, len(o.violations))
	copy(result, o.violations)
	return result
}

// GetUnvisitedStates returns states that were expected but not visited
func (o *ValidationObserver) GetUnvisitedStates() []string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	var unvisited []string
	for state := range o.expectedStates {
		if !o.visitedStates[state] {
			unvisited = append(unvisited, state)
		}
	}

	return unvisited
}

// HasViolations returns whether any violations occurred
func (o *ValidationObserver) HasViolations() bool {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return len(o.violations) > 0
}

// Reset resets the validation state
func (o *ValidationObserver) Reset() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.visitedStates = make(map[string]bool)
	o.violations = make([]string, 0)
}
