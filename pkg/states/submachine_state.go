package states

import (
	"context"
	"fmt"
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// SubmachineState represents a state that encapsulates another state machine
type SubmachineState struct {
	BaseState
	submachine      *core.StateMachine
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
	ParentState     string              // State in parent machine
	SubmachineState string              // State in submachine
	EventMapping    map[string]string   // Maps parent events to submachine events
	GuardCondition  core.GuardCondition // Optional guard for the connection
}

// NewSubmachineState creates a new submachine state
func NewSubmachineState(name string) *SubmachineState {
	return &SubmachineState{
		BaseState:    *NewBaseState(name),
		stateMapping: make(map[string]string),
	}
}

// SetSubmachine sets the embedded state machine
func (s *SubmachineState) SetSubmachine(submachine *core.StateMachine) *SubmachineState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.submachine = submachine
	return s
}

// GetSubmachine returns the embedded state machine
func (s *SubmachineState) GetSubmachine() *core.StateMachine {
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
func (s *SubmachineState) Enter(ctx *core.Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	submachine := s.submachine
	entryConnector := s.entryConnector
	s.mutex.RUnlock()

	if submachine == nil {
		return fmt.Errorf("no submachine defined for state %s", s.Name())
	}

	// Start the submachine
	if err := submachine.Start(ctx.Context); err != nil {
		return err
	}

	// Add an observer to the submachine
	submachine.AddObserver(&submachineObserver{
		parent:          s,
		completionEvent: s.completionEvent,
		errorEvent:      s.errorEvent,
	})

	// If we have an entry connector, use it to transition to a specific state
	if entryConnector != nil {
		targetState := submachine.GetState(entryConnector.SubmachineState)
		if targetState != nil {
			if err := submachine.SendEvent(core.NewEvent(fmt.Sprintf("ENTRY_TO_%s", targetState.Name()))); err != nil {
				return err
			}
		}
	}

	return nil
}

// Exit handles exit from the submachine state
func (s *SubmachineState) Exit(ctx *core.Context) error {
	s.mutex.RLock()
	submachine := s.submachine
	s.mutex.RUnlock()

	if submachine != nil && submachine.IsStarted() {
		submachine.Stop(ctx.Context)
	}

	return s.BaseState.Exit(ctx)
}

// HandleEvent processes events for the submachine state
func (s *SubmachineState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	s.mutex.RLock()
	submachine := s.submachine
	eventForwarding := s.eventForwarding
	exitConnector := s.exitConnector
	s.mutex.RUnlock()

	// First check if this is an exit connector event
	if exitConnector != nil && event.Name == fmt.Sprintf("EXIT_FROM_%s", exitConnector.SubmachineState) {
		// Find the target state in the parent
		parentState := ctx.StateMachine.GetState(exitConnector.ParentState)
		if parentState != nil {
			return parentState, nil
		}
	}

	// Then check if this is a completion event
	if event.Name == s.completionEvent {
		// Find the exit connector
		if exitConnector != nil {
			// Find the target state in the parent
			parentState := ctx.StateMachine.GetState(exitConnector.ParentState)
			if parentState != nil {
				return parentState, nil
			}
		}
	}

	// Otherwise forward the event to the submachine
	if submachine != nil && submachine.IsStarted() {
		// Check if we need to map the event
		if exitConnector != nil && exitConnector.EventMapping != nil {
			if mappedEvent, exists := exitConnector.EventMapping[event.Name]; exists {
				// Create a new mapped event
				mappedEventObj := core.NewEventWithData(mappedEvent, event.Data)
				err := submachine.SendEvent(mappedEventObj)
				if err != nil {
					return nil, err
				}
				return nil, nil
			}
		}

		// No mapping, just forward the event
		err := submachine.SendEvent(event)
		if err != nil && !eventForwarding {
			return nil, err
		}
	}

	// If event forwarding is enabled, let the event continue to parent
	if eventForwarding {
		return nil, nil
	}

	// Otherwise consider the event handled
	return nil, nil
}

// GetCurrentSubmachineState returns the current state of the submachine
func (s *SubmachineState) GetCurrentSubmachineState() string {
	s.mutex.RLock()
	submachine := s.submachine
	s.mutex.RUnlock()

	if submachine != nil {
		return submachine.GetCurrentStateName()
	}

	return ""
}

// GetSubmachineHistory returns the state history of the submachine
func (s *SubmachineState) GetSubmachineHistory() []string {
	// This would require state tracking which is not implemented yet
	return []string{}
}

// IsSubmachineInFinalState checks if submachine is in a final state
func (s *SubmachineState) IsSubmachineInFinalState() bool {
	s.mutex.RLock()
	submachine := s.submachine
	s.mutex.RUnlock()

	if submachine != nil {
		return submachine.IsCompleted()
	}

	return false
}

// submachineObserver observes submachine events and forwards them to parent
type submachineObserver struct {
	parent          *SubmachineState
	completionEvent string
	errorEvent      string
}

func (o *submachineObserver) OnStateEnter(sm *core.StateMachine, state core.State) {
	// Check state mapping
	if o.parent != nil {
		o.parent.mutex.RLock()
		if parentState, exists := o.parent.stateMapping[state.Name()]; exists {
			// We could notify the parent machine here
			_ = parentState
		}
		o.parent.mutex.RUnlock()
	}
}

func (o *submachineObserver) OnStateExit(sm *core.StateMachine, state core.State) {
	// No special handling needed
}

func (o *submachineObserver) OnTransition(sm *core.StateMachine, from, to core.State, event *core.Event) {
	// Special handling for transitions to final states
	if to != nil && to.Name() == "FinalState" {
		// If completion event is defined, we could create it here
		if o.completionEvent != "" && o.parent != nil && o.parent.GetParent() != nil {
			// We could notify the parent machine here
			// This would trigger the completion event in the parent state machine
		}
	}
}

func (o *submachineObserver) OnEventProcessed(sm *core.StateMachine, event *core.Event) {
	// Could forward certain events to parent machine if needed
}

func (o *submachineObserver) OnError(sm *core.StateMachine, err error) {
	// If error event is defined, we could create it here
	if o.errorEvent != "" && o.parent != nil && o.parent.GetParent() != nil {
		// We could notify the parent machine here
		// This would trigger the error event in the parent state machine
	}
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
func (c *SubmachineConnector) WithGuard(guard core.GuardCondition) *SubmachineConnector {
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
			submachine.submachine.Stop(context.Background())
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
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	for _, submachine := range sm.submachines {
		if submachine.submachine != nil && submachine.submachine.IsStarted() {
			submachine.submachine.Stop(context.Background())
		}
	}

	return nil
}

// SubmachineBuilder helps build submachine states with fluent API
type SubmachineBuilder struct {
	state *SubmachineState
}

// NewSubmachineBuilder creates a new submachine builder
func NewSubmachineBuilder(id string) *SubmachineBuilder {
	return &SubmachineBuilder{
		state: NewSubmachineState(id),
	}
}

// WithSubmachine sets the embedded state machine
func (b *SubmachineBuilder) WithSubmachine(sm *core.StateMachine) *SubmachineBuilder {
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
func (b *SubmachineBuilder) WithEntryAction(action core.Action) *SubmachineBuilder {
	b.state.AddEntryAction(action)
	return b
}

// WithExitAction adds an exit action
func (b *SubmachineBuilder) WithExitAction(action core.Action) *SubmachineBuilder {
	b.state.AddExitAction(action)
	return b
}

// Build creates the submachine state
func (b *SubmachineBuilder) Build() *SubmachineState {
	return b.state
}
