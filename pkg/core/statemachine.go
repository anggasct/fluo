package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// StateMachine represents the main state machine engine
type StateMachine struct {
	name          string
	states        map[string]State
	transitions   []*Transition
	currentState  State
	initialState  State
	finalStates   map[string]State
	context       *Context
	eventQueue    chan *Event
	eventDeferrer *EventDeferrer
	state         StateMachineState
	isCompleted   bool
	lastError     error
	observers     []StateMachineObserver
	mutex         sync.RWMutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewStateMachine creates a new state machine with the given name
func NewStateMachine(name string) *StateMachine {
	sm := &StateMachine{
		name:          name,
		states:        make(map[string]State),
		transitions:   make([]*Transition, 0),
		finalStates:   make(map[string]State),
		state:         StateStopped,
		eventQueue:    make(chan *Event, 100),
		eventDeferrer: NewEventDeferrer(),
		stopChan:      make(chan struct{}),
		observers:     make([]StateMachineObserver, 0),
	}
	return sm
}

// Name returns the name of the state machine
func (sm *StateMachine) Name() string {
	return sm.name
}

// GetAllStates returns all states in this state machine
func (sm *StateMachine) GetAllStates() []State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	states := make([]State, 0, len(sm.states))
	for _, state := range sm.states {
		states = append(states, state)
	}
	return states
}

// AddState adds a state to the state machine
func (sm *StateMachine) AddState(state State) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if state == nil {
		return sm
	}

	sm.states[state.Name()] = state
	return sm
}

// GetState retrieves a state by name
func (sm *StateMachine) GetState(name string) State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.states[name]
}

// SetInitialState sets the initial state
func (sm *StateMachine) SetInitialState(state State) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.initialState = state
	return sm
}

// SetInitialStateByName sets the initial state by name
func (sm *StateMachine) SetInitialStateByName(name string) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if state, exists := sm.states[name]; exists {
		sm.initialState = state
	}
	return sm
}

// AddFinalState adds a final state
func (sm *StateMachine) AddFinalState(state State) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if state != nil {
		sm.states[state.Name()] = state
		sm.finalStates[state.Name()] = state
	}
	return sm
}

// AddTransition adds a transition between states
func (sm *StateMachine) AddTransition(from, to State, event string) *Transition {
	transition := NewTransition(from, to, event)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.transitions = append(sm.transitions, transition)

	return transition
}

// AddTransitionWithGuard adds a transition with a guard condition
func (sm *StateMachine) AddTransitionWithGuard(from, to State, event string, guard GuardCondition) *Transition {
	transition := NewTransition(from, to, event).WithGuard(guard)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.transitions = append(sm.transitions, transition)

	return transition
}

// AddTransitionWithAction adds a transition with an action
func (sm *StateMachine) AddTransitionWithAction(from, to State, event string, guard GuardCondition, action Action) *Transition {
	transition := NewTransition(from, to, event).WithGuard(guard).WithAction(action)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.transitions = append(sm.transitions, transition)

	return transition
}

// GetCurrentState returns the current state
func (sm *StateMachine) GetCurrentState() State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.currentState
}

// GetCurrentStateName returns the name of the current state
func (sm *StateMachine) GetCurrentStateName() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.currentState == nil {
		return ""
	}
	return sm.currentState.Name()
}

// Start starts the state machine
func (sm *StateMachine) Start(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.state == StateRunning {
		return fmt.Errorf("state machine is already running")
	}

	if sm.initialState == nil {
		return fmt.Errorf("no initial state set for state machine %s", sm.name)
	}

	// Initialize the context if not already done
	if sm.context == nil {
		sm.context = NewContext(ctx, sm)
	} else {
		sm.context.Context = ctx
	}

	// Start from initial state
	sm.state = StateRunning
	sm.currentState = sm.initialState

	// Enter the initial state
	if err := sm.currentState.Enter(sm.context); err != nil {
		sm.state = StateError
		sm.lastError = err
		return err
	}

	// Notify observers
	sm.mutex.Unlock()
	sm.notifyStateEnter(sm.currentState)
	sm.mutex.Lock()

	// Start the event processing goroutine
	sm.wg.Add(1)
	go sm.processEvents()

	return nil
}

// Stop stops the state machine
func (sm *StateMachine) Stop(ctx context.Context) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.state != StateRunning {
		return
	}

	// Update context before stopping
	if ctx != nil && sm.context != nil {
		sm.context.Context = ctx
	}

	close(sm.stopChan)
	sm.wg.Wait()

	sm.state = StateStopped
}

// Reset resets the state machine to its initial state
func (sm *StateMachine) Reset(ctx context.Context) error {
	sm.Stop(ctx)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.currentState = nil
	sm.isCompleted = false
	sm.lastError = nil
	sm.eventQueue = make(chan *Event, 100)
	sm.stopChan = make(chan struct{})
	sm.state = StateStopped

	return nil
}

// IsStarted returns whether the state machine is started
func (sm *StateMachine) IsStarted() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.state == StateRunning
}

// IsCompleted returns whether the state machine has completed
func (sm *StateMachine) IsCompleted() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.isCompleted
}

// SendEvent sends an event to the state machine
func (sm *StateMachine) SendEvent(event *Event) error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.state != StateRunning {
		return fmt.Errorf("state machine not running")
	}

	select {
	case sm.eventQueue <- event:
		return nil
	default:
		return fmt.Errorf("event queue full")
	}
}

// SendEventSync sends an event and waits for it to be processed
func (sm *StateMachine) SendEventSync(event *Event, timeout time.Duration) error {
	if err := sm.SendEvent(event); err != nil {
		return err
	}

	// Wait for the event to be processed
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return fmt.Errorf("timeout waiting for event to be processed")
	case <-sm.context.Done():
		return sm.context.Err()
	}
}

// SendEventWithData sends an event with data
func (sm *StateMachine) SendEventWithData(eventName string, data interface{}) error {
	return sm.SendEvent(NewEventWithData(eventName, data))
}

// GetLastError returns the last error that occurred
func (sm *StateMachine) GetLastError() error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.lastError
}

// GetStates returns all states in the state machine
func (sm *StateMachine) GetStates() map[string]State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result := make(map[string]State)
	for k, v := range sm.states {
		result[k] = v
	}

	return result
}

// States returns the number of states in the state machine
func (sm *StateMachine) States() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return len(sm.states)
}

// GetInitialState returns the initial state
func (sm *StateMachine) GetInitialState() State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.initialState
}

// CurrentState returns the current state
func (sm *StateMachine) CurrentState() State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.currentState
}

// SetCurrentState sets the current state
func (sm *StateMachine) SetCurrentState(state State) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.currentState = state
}

// ForceState forces the state machine to a specific state
func (sm *StateMachine) ForceState(state State) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.currentState != nil {
		// No need to call Exit as we're forcing the state
		sm.currentState = state
	}
}

// HandleEvent processes an event with the provided context immediately (synchronously)
func (sm *StateMachine) HandleEvent(ctx context.Context, event *Event) error {
	// Update the context if needed
	if ctx != nil && sm.context != nil {
		sm.context.Context = ctx
	}

	// For testing purposes, process the event directly
	sm.context.SetEvent(event)
	return sm.processEvent(event)
}

// Context returns the context of the state machine
func (sm *StateMachine) Context() *Context {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.context == nil {
		// Create context if it doesn't exist
		sm.mutex.RUnlock()
		sm.mutex.Lock()
		defer sm.mutex.Unlock()

		if sm.context == nil {
			sm.context = NewContext(context.Background(), sm)
		}
	}

	return sm.context
}

// processEvents handles incoming events
func (sm *StateMachine) processEvents() {
	defer sm.wg.Done()

	for {
		select {
		case <-sm.stopChan:
			return
		case event := <-sm.eventQueue:
			if err := sm.processEvent(event); err != nil {
				sm.mutex.Lock()
				sm.lastError = err
				sm.state = StateError
				sm.mutex.Unlock()
				sm.notifyError(err)
			}
		}
	}
}

// processEvent processes a single event
func (sm *StateMachine) processEvent(event *Event) error {
	sm.mutex.Lock()
	currentState := sm.currentState
	sm.mutex.Unlock()

	if currentState == nil {
		return fmt.Errorf("no current state")
	}

	sm.context.SetEvent(event)

	// First, check for transitions at the current state level
	var applicableTransitions []*Transition

	sm.mutex.RLock()
	for _, transition := range sm.transitions {
		// Standard transitions
		if transition.From == currentState && transition.Event == event.Name {
			if transition.CanExecute(sm.context) {
				applicableTransitions = append(applicableTransitions, transition)
			}
			continue
		}

		// Handle composite state child transitions
		// If we're in a composite state, check for transitions from specific child states
		if reflect.TypeOf(currentState).String() == "*states.CompositeState" {
			// Use type assertion to get the composite state
			if compositeState, ok := currentState.(interface {
				GetCurrentChild() State
				Name() string
			}); ok {
				childState := compositeState.GetCurrentChild()
				if childState != nil {
					// Check if this transition is from the current child state of the composite
					// We compare using string format since we can't directly compare the hierarchical state reference
					fromName := transition.From.Name()
					compoundName := fmt.Sprintf("%s.%s", compositeState.Name(), childState.Name())

					if fromName == compoundName && transition.Event == event.Name && transition.CanExecute(sm.context) {
						applicableTransitions = append(applicableTransitions, transition)
					}
				}
			}
		}
	}
	sm.mutex.RUnlock()

	// Sort transitions by priority
	if len(applicableTransitions) > 1 {
		// Simple bubble sort by priority (higher priority first)
		for i := 0; i < len(applicableTransitions)-1; i++ {
			for j := 0; j < len(applicableTransitions)-i-1; j++ {
				if applicableTransitions[j].Priority < applicableTransitions[j+1].Priority {
					applicableTransitions[j], applicableTransitions[j+1] = applicableTransitions[j+1], applicableTransitions[j]
				}
			}
		}
	}

	// Execute the first applicable transition
	if len(applicableTransitions) > 0 {
		return sm.executeTransition(applicableTransitions[0], event)
	}

	// If no transitions are applicable, delegate to current state
	nextState, err := currentState.HandleEvent(event, sm.context)
	if err != nil {
		return err
	}

	// If handleEvent returned a new state, transition to it
	if nextState != nil && nextState != currentState {
		return sm.transitionTo(nextState, event)
	}

	// No transition found, but that's not an error
	sm.notifyEventProcessed(event)
	return nil
}

// executeTransition executes a transition
func (sm *StateMachine) executeTransition(transition *Transition, event *Event) error {
	sm.mutex.Lock()
	currentState := sm.currentState
	sm.mutex.Unlock()

	// Execute exit action first, done by calling Exit on the current state
	if currentState != nil {
		if err := currentState.Exit(sm.context); err != nil {
			return err
		}
		sm.notifyStateExit(currentState)
	}

	// Execute transition action next
	if transition.Action != nil {
		if err := transition.Action(sm.context); err != nil {
			return err
		}
	}

	// Set the new state
	sm.SetCurrentState(transition.To)

	// Execute entry action by calling Enter on the new state
	if err := transition.To.Enter(sm.context); err != nil {
		return err
	}

	// Notify observers
	sm.notifyStateEnter(transition.To)
	sm.notifyTransition(currentState, transition.To, event)
	sm.notifyEventProcessed(event)

	// Check if we've reached a final state
	sm.mutex.Lock()
	_, isFinal := sm.finalStates[transition.To.Name()]
	if isFinal {
		sm.isCompleted = true
		sm.state = StateCompleted
	}
	sm.mutex.Unlock()

	return nil
}

// transitionTo transitions to a new state
func (sm *StateMachine) transitionTo(newState State, event *Event) error {
	// This method is used when there is no explicit transition object
	// Create a simple transition for consistency
	transition := &Transition{
		From:  sm.currentState,
		To:    newState,
		Event: event.Name,
	}

	return sm.executeTransition(transition, event)
}

// Observer notification methods
func (sm *StateMachine) notifyStateEnter(state State) {
	sm.mutex.RLock()
	observers := sm.observers
	sm.mutex.RUnlock()

	for _, observer := range observers {
		observer.OnStateEnter(sm, state)
	}
}

func (sm *StateMachine) notifyStateExit(state State) {
	sm.mutex.RLock()
	observers := sm.observers
	sm.mutex.RUnlock()

	for _, observer := range observers {
		observer.OnStateExit(sm, state)
	}
}

func (sm *StateMachine) notifyTransition(from, to State, event *Event) {
	sm.mutex.RLock()
	observers := sm.observers
	sm.mutex.RUnlock()

	for _, observer := range observers {
		observer.OnTransition(sm, from, to, event)
	}
}

func (sm *StateMachine) notifyEventProcessed(event *Event) {
	sm.mutex.RLock()
	observers := sm.observers
	sm.mutex.RUnlock()

	for _, observer := range observers {
		observer.OnEventProcessed(sm, event)
	}
}

func (sm *StateMachine) notifyError(err error) {
	sm.mutex.RLock()
	observers := sm.observers
	sm.mutex.RUnlock()

	for _, observer := range observers {
		observer.OnError(sm, err)
	}
}

// GetFinalStates returns a map of final states
func (sm *StateMachine) GetFinalStates() map[string]State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result := make(map[string]State)
	for k, v := range sm.finalStates {
		result[k] = v
	}

	return result
}

// AddObserver adds an observer to the state machine
func (sm *StateMachine) AddObserver(observer StateMachineObserver) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.observers = append(sm.observers, observer)
}
