package flux

import (
	"fmt"
	"sync"
)

// SubmachineState represents a state that encapsulates another state machine
// This allows for hierarchical state machines where one state machine can be
// embedded within a state of another state machine
type SubmachineState struct {
	BaseState
	submachine      *StateMachine
	entryConnector  *SubmachineConnector
	exitConnector   *SubmachineConnector
	completionEvent string
	errorEvent      string
	stateMapping    map[string]string // Maps submachine states to parent states
	eventForwarding bool              // Whether to forward unhandled events to parent
	mutex           sync.RWMutex
}

// SubmachineConnector defines connection points between parent and submachine
type SubmachineConnector struct {
	ParentState     string            // State in parent machine
	SubmachineState string            // State in submachine
	EventMapping    map[string]string // Maps parent events to submachine events
	GuardCondition  GuardCondition    // Optional guard for the connection
}

// SubmachineBuilder helps build submachine states with fluent API
type SubmachineBuilder struct {
	state *SubmachineState
}

// NewSubmachineState creates a new submachine state
func NewSubmachineState(id string, submachine *StateMachine) *SubmachineState {
	return &SubmachineState{
		BaseState:       *NewBaseState(id),
		submachine:      submachine,
		completionEvent: "submachine.completed",
		errorEvent:      "submachine.error",
		stateMapping:    make(map[string]string),
		eventForwarding: true,
	}
}

// SetSubmachine sets the embedded state machine
func (s *SubmachineState) SetSubmachine(sm *StateMachine) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.submachine = sm
}

// GetSubmachine returns the embedded state machine
func (s *SubmachineState) GetSubmachine() *StateMachine {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.submachine
}

// SetEntryConnector sets the entry connection point
func (s *SubmachineState) SetEntryConnector(connector *SubmachineConnector) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.entryConnector = connector
}

// SetExitConnector sets the exit connection point
func (s *SubmachineState) SetExitConnector(connector *SubmachineConnector) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.exitConnector = connector
}

// SetCompletionEvent sets the event fired when submachine completes
func (s *SubmachineState) SetCompletionEvent(event string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.completionEvent = event
}

// SetErrorEvent sets the event fired when submachine encounters error
func (s *SubmachineState) SetErrorEvent(event string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.errorEvent = event
}

// AddStateMapping maps a submachine state to a parent state
func (s *SubmachineState) AddStateMapping(submachineState, parentState string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stateMapping[submachineState] = parentState
}

// SetEventForwarding enables/disables event forwarding to parent machine
func (s *SubmachineState) SetEventForwarding(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.eventForwarding = enabled
}

// Enter handles entry into the submachine state
func (s *SubmachineState) Enter(ctx *Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Execute base entry actions
	if err := s.BaseState.Enter(ctx); err != nil {
		return NewSubmachineError("entry action failed", s.Name(), err)
	}

	// Start the submachine if not already started
	if s.submachine != nil && !s.submachine.IsStarted() {
		// Set up submachine observers to handle completion and errors
		s.submachine.AddObserver(&submachineObserver{
			parent:          s,
			completionEvent: s.completionEvent,
			errorEvent:      s.errorEvent,
		})

		// Start the submachine
		if err := s.submachine.Start(); err != nil {
			return NewSubmachineError("failed to start submachine", s.Name(), err)
		}

		// Forward entry event if connector specifies event mapping
		if s.entryConnector != nil && len(s.entryConnector.EventMapping) > 0 {
			for parentEvent, submachineEvent := range s.entryConnector.EventMapping {
				if event := ctx.GetEvent(); event != nil && event.Name == parentEvent {
					submachineCtx := NewContext(ctx.Context, s.submachine)
					submachineCtx.SetEvent(NewEventWithData(submachineEvent, event.Data))

					if err := s.submachine.SendEvent(submachineCtx.GetEvent()); err != nil {
						return NewSubmachineError("failed to forward entry event", s.Name(), err)
					}
				}
			}
		}
	}

	return nil
}

// Exit handles exit from the submachine state
func (s *SubmachineState) Exit(ctx *Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Stop the submachine if it's running
	if s.submachine != nil && s.submachine.IsStarted() {
		if err := s.submachine.Stop(); err != nil {
			return NewSubmachineError("failed to stop submachine", s.Name(), err)
		}
	}

	// Execute base exit actions
	if err := s.BaseState.Exit(ctx); err != nil {
		return NewSubmachineError("exit action failed", s.Name(), err)
	}

	return nil
}

// HandleEvent processes events for the submachine state
func (s *SubmachineState) HandleEvent(event *Event, ctx *Context) (State, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check if submachine is available and started
	if s.submachine == nil || !s.submachine.IsStarted() {
		return nil, NewSubmachineError("submachine not available or not started", s.Name(), nil)
	}

	// Forward event to submachine if event forwarding is enabled
	if s.eventForwarding {
		if err := s.submachine.SendEvent(event); err != nil {
			return nil, NewSubmachineError("failed to process event in submachine", s.Name(), err)
		}
	}

	return nil, nil
}

// GetCurrentSubmachineState returns the current state of the submachine
func (s *SubmachineState) GetCurrentSubmachineState() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.submachine != nil {
		current := s.submachine.CurrentState()
		if current != nil {
			return current.Name()
		}
	}
	return ""
}

// GetSubmachineHistory returns the state history of the submachine
func (s *SubmachineState) GetSubmachineHistory() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.submachine != nil {
		// For now, return current state as history since GetStateHistory doesn't exist
		current := s.submachine.CurrentState()
		if current != nil {
			return []string{current.Name()}
		}
	}
	return []string{}
}

// IsSubmachineInFinalState checks if submachine is in a final state
func (s *SubmachineState) IsSubmachineInFinalState() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.submachine != nil {
		currentState := s.submachine.CurrentState()
		if finalState, ok := currentState.(*FinalState); ok {
			return finalState != nil
		}
	}
	return false
}

// submachineObserver observes submachine events and forwards them to parent
type submachineObserver struct {
	parent          *SubmachineState
	completionEvent string
	errorEvent      string
}

func (o *submachineObserver) OnStateEnter(sm *StateMachine, state State) {
	// Handle state entry - could forward to parent if needed
}

func (o *submachineObserver) OnStateExit(sm *StateMachine, state State) {
	// Handle state exit - could forward to parent if needed
}

func (o *submachineObserver) OnTransition(sm *StateMachine, from, to State, event *Event) {
	// Handle transition completion
	// Check if this is a transition to a final state
	if _, isFinal := to.(*FinalState); isFinal {
		// Create completion event - would need to forward to parent
		// This would typically trigger the completion event in the parent state machine
	}
}

func (o *submachineObserver) OnEventProcessed(sm *StateMachine, event *Event) {
	// Could forward certain events to parent machine if needed
}

func (o *submachineObserver) OnError(sm *StateMachine, err error) {
	// Create error event for parent machine
	// This would typically trigger the error event in the parent state machine
}

// NewSubmachineConnector creates a new submachine connector
func NewSubmachineConnector(parentState, submachineState string) *SubmachineConnector {
	return &SubmachineConnector{
		ParentState:     parentState,
		SubmachineState: submachineState,
		EventMapping:    make(map[string]string),
	}
}

// WithEventMapping adds event mapping to the connector
func (c *SubmachineConnector) WithEventMapping(parentEvent, submachineEvent string) *SubmachineConnector {
	c.EventMapping[parentEvent] = submachineEvent
	return c
}

// WithGuard adds a guard condition to the connector
func (c *SubmachineConnector) WithGuard(guard GuardCondition) *SubmachineConnector {
	c.GuardCondition = guard
	return c
}

// SubmachineManager manages multiple submachines within a state machine
type SubmachineManager struct {
	submachines map[string]*SubmachineState
	mutex       sync.RWMutex
}

// NewSubmachineManager creates a new submachine manager
func NewSubmachineManager() *SubmachineManager {
	return &SubmachineManager{
		submachines: make(map[string]*SubmachineState),
	}
}

// AddSubmachine adds a submachine to the manager
func (sm *SubmachineManager) AddSubmachine(id string, submachine *SubmachineState) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.submachines[id] = submachine
}

// GetSubmachine retrieves a submachine by ID
func (sm *SubmachineManager) GetSubmachine(id string) *SubmachineState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.submachines[id]
}

// RemoveSubmachine removes a submachine from the manager
func (sm *SubmachineManager) RemoveSubmachine(id string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	if submachine, exists := sm.submachines[id]; exists {
		// Stop the submachine if it's running
		if submachine.submachine != nil && submachine.submachine.IsStarted() {
			submachine.submachine.Stop()
		}
		delete(sm.submachines, id)
	}
}

// GetAllSubmachines returns all managed submachines
func (sm *SubmachineManager) GetAllSubmachines() map[string]*SubmachineState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result := make(map[string]*SubmachineState)
	for id, submachine := range sm.submachines {
		result[id] = submachine
	}
	return result
}

// StopAllSubmachines stops all managed submachines
func (sm *SubmachineManager) StopAllSubmachines() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var errors []error
	for id, submachine := range sm.submachines {
		if submachine.submachine != nil && submachine.submachine.IsStarted() {
			if err := submachine.submachine.Stop(); err != nil {
				errors = append(errors, fmt.Errorf("failed to stop submachine %s: %w", id, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping submachines: %v", errors)
	}
	return nil
}

// NewSubmachineBuilder creates a new submachine builder
func NewSubmachineBuilder(id string) *SubmachineBuilder {
	return &SubmachineBuilder{
		state: NewSubmachineState(id, nil),
	}
}

// WithSubmachine sets the embedded state machine
func (b *SubmachineBuilder) WithSubmachine(sm *StateMachine) *SubmachineBuilder {
	b.state.SetSubmachine(sm)
	return b
}

// WithEntryConnector sets the entry connector
func (b *SubmachineBuilder) WithEntryConnector(connector *SubmachineConnector) *SubmachineBuilder {
	b.state.SetEntryConnector(connector)
	return b
}

// WithExitConnector sets the exit connector
func (b *SubmachineBuilder) WithExitConnector(connector *SubmachineConnector) *SubmachineBuilder {
	b.state.SetExitConnector(connector)
	return b
}

// WithCompletionEvent sets the completion event
func (b *SubmachineBuilder) WithCompletionEvent(event string) *SubmachineBuilder {
	b.state.SetCompletionEvent(event)
	return b
}

// WithErrorEvent sets the error event
func (b *SubmachineBuilder) WithErrorEvent(event string) *SubmachineBuilder {
	b.state.SetErrorEvent(event)
	return b
}

// WithStateMapping adds a state mapping
func (b *SubmachineBuilder) WithStateMapping(submachineState, parentState string) *SubmachineBuilder {
	b.state.AddStateMapping(submachineState, parentState)
	return b
}

// WithEventForwarding enables/disables event forwarding
func (b *SubmachineBuilder) WithEventForwarding(enabled bool) *SubmachineBuilder {
	b.state.SetEventForwarding(enabled)
	return b
}

// WithEntryAction adds an entry action
func (b *SubmachineBuilder) WithEntryAction(action Action) *SubmachineBuilder {
	b.state.AddEntryAction(action)
	return b
}

// WithExitAction adds an exit action
func (b *SubmachineBuilder) WithExitAction(action Action) *SubmachineBuilder {
	b.state.AddExitAction(action)
	return b
}

// Build creates the submachine state
func (b *SubmachineBuilder) Build() *SubmachineState {
	return b.state
}
