package flux

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StateMachineState represents the current state of the state machine
type StateMachineState int

const (
	// StateStopped indicates the state machine is stopped
	StateStopped StateMachineState = iota
	// StateRunning indicates the state machine is running
	StateRunning
	// StateCompleted indicates the state machine has completed
	StateCompleted
	// StateError indicates the state machine encountered an error
	StateError
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

// StateMachineObserver observes state machine events
type StateMachineObserver interface {
	OnStateEnter(sm *StateMachine, state State)
	OnStateExit(sm *StateMachine, state State)
	OnTransition(sm *StateMachine, from, to State, event *Event)
	OnEventProcessed(sm *StateMachine, event *Event)
	OnError(sm *StateMachine, err error)
}

// NewStateMachine creates a new state machine
func NewStateMachine(name string) *StateMachine {
	ctx := context.Background()
	sm := &StateMachine{
		name:          name,
		states:        make(map[string]State),
		transitions:   make([]*Transition, 0),
		finalStates:   make(map[string]State),
		eventQueue:    make(chan *Event, 100),
		eventDeferrer: NewEventDeferrer(),
		state:         StateStopped,
		observers:     make([]StateMachineObserver, 0),
		stopChan:      make(chan struct{}),
	}

	sm.context = NewContext(ctx, sm)
	return sm
}

// Name returns the state machine name
func (sm *StateMachine) Name() string {
	return sm.name
}

// AddState adds a state to the state machine
// The first state added is automatically set as the initial state
func (sm *StateMachine) AddState(state State) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.states[state.Name()] = state

	// Automatically set first state as initial state
	if sm.initialState == nil {
		sm.initialState = state
	}

	// Automatically track final states for completion detection
	if finalState, ok := state.(*FinalState); ok {
		sm.finalStates[finalState.Name()] = finalState
	}

	return sm
}

// RemoveState removes a state from the state machine
func (sm *StateMachine) RemoveState(stateName string) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	delete(sm.states, stateName)
	delete(sm.finalStates, stateName)

	return sm
}

// GetState returns a state by name
func (sm *StateMachine) GetState(name string) State {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.states[name]
}

// SetInitialState sets the initial state
// Note: The first state added is automatically set as initial state.
// Use this method only when you need to override the automatic behavior.
func (sm *StateMachine) SetInitialState(state State) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.initialState = state
	return sm
}

// AddTransition adds a transition to the state machine
func (sm *StateMachine) AddTransition(from, to State, event string, guard GuardCondition) *StateMachine {
	transition := NewTransition(from, to, event).WithGuard(guard)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.transitions = append(sm.transitions, transition)

	return sm
}

// AddTransitionWithAction adds a transition with an action
func (sm *StateMachine) AddTransitionWithAction(from, to State, event string, guard GuardCondition, action Action) *StateMachine {
	transition := NewTransition(from, to, event).WithGuard(guard).WithAction(action)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.transitions = append(sm.transitions, transition)

	return sm
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

// IsCompleted returns whether the state machine has completed
func (sm *StateMachine) IsCompleted() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.isCompleted
}

// IsStarted returns whether the state machine is currently running
func (sm *StateMachine) IsStarted() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.state == StateRunning
}

// SetCompleted sets the completion status
func (sm *StateMachine) SetCompleted(completed bool) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.isCompleted = completed
	if completed {
		sm.state = StateCompleted
	}
}

// GetState returns the current state machine state
func (sm *StateMachine) GetStateMachineState() StateMachineState {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.state
}

// AddObserver adds a state machine observer
func (sm *StateMachine) AddObserver(observer StateMachineObserver) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.observers = append(sm.observers, observer)
	return sm
}

// RemoveObserver removes a state machine observer
func (sm *StateMachine) RemoveObserver(observer StateMachineObserver) *StateMachine {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for i, obs := range sm.observers {
		if obs == observer {
			sm.observers = append(sm.observers[:i], sm.observers[i+1:]...)
			break
		}
	}
	return sm
}

// Start starts the state machine
func (sm *StateMachine) Start() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.state == StateRunning {
		return fmt.Errorf("state machine %s is already running", sm.name)
	}

	if sm.initialState == nil {
		return fmt.Errorf("no initial state set for state machine %s", sm.name)
	}

	sm.state = StateRunning
	sm.currentState = sm.initialState
	sm.stopChan = make(chan struct{})

	if err := sm.currentState.Enter(sm.context); err != nil {
		sm.state = StateError
		sm.lastError = err
		return err
	}

	sm.notifyStateEnter(sm.currentState)

	sm.wg.Add(1)
	go sm.eventLoop()

	return nil
}

// Stop stops the state machine
func (sm *StateMachine) Stop() error {
	sm.mutex.Lock()
	if sm.state != StateRunning {
		sm.mutex.Unlock()
		return fmt.Errorf("state machine %s is not running", sm.name)
	}

	sm.state = StateStopped
	close(sm.stopChan)
	sm.mutex.Unlock()

	sm.wg.Wait()

	if sm.currentState != nil {
		if err := sm.currentState.Exit(sm.context); err != nil {
			return err
		}
		sm.notifyStateExit(sm.currentState)
	}

	return nil
}

// SendEvent sends an event to the state machine
func (sm *StateMachine) SendEvent(event *Event) error {
	sm.mutex.RLock()
	if sm.state != StateRunning {
		sm.mutex.RUnlock()
		return fmt.Errorf("state machine %s is not running", sm.name)
	}
	sm.mutex.RUnlock()

	select {
	case sm.eventQueue <- event:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("event queue is full, event %s was not sent", event.Name)
	}
}

// SendEventSync sends an event synchronously and waits for processing
func (sm *StateMachine) SendEventSync(event *Event) error {
	if err := sm.SendEvent(event); err != nil {
		return err
	}

	time.Sleep(10 * time.Millisecond)
	return nil
}

// eventLoop processes events from the queue
func (sm *StateMachine) eventLoop() {
	defer sm.wg.Done()

	for {
		select {
		case <-sm.stopChan:
			return
		case event := <-sm.eventQueue:
			if err := sm.processEvent(event); err != nil {
				sm.mutex.Lock()
				sm.state = StateError
				sm.lastError = err
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
		return fmt.Errorf("no current state to process event %s", event.Name)
	}

	sm.context.Event = event

	if sm.eventDeferrer.ShouldDefer(event, currentState) {
		sm.eventDeferrer.DeferEvent(event, sm.context)
		return nil
	}

	newState, err := currentState.HandleEvent(event, sm.context)
	if err != nil {
		return err
	}

	if newState != nil {
		if err := sm.transitionTo(newState, event); err != nil {
			return err
		}
	} else {
		for _, transition := range sm.transitions {
			if transition.From == currentState && transition.Event == event.Name {
				if transition.CanExecute(sm.context) {
					if err := sm.executeTransition(transition, event); err != nil {
						return err
					}
					break
				}
			}
		}
	}

	sm.notifyEventProcessed(event)
	return nil
}

// executeTransition executes a transition
func (sm *StateMachine) executeTransition(transition *Transition, event *Event) error {
	sm.notifyTransition(transition.From, transition.To, event)

	if err := transition.Execute(sm.context); err != nil {
		return err
	}

	return sm.transitionTo(transition.To, event)
}

// transitionTo transitions to a new state
func (sm *StateMachine) transitionTo(newState State, event *Event) error {
	sm.mutex.Lock()
	currentState := sm.currentState
	sm.mutex.Unlock()

	if currentState != nil {
		if err := currentState.Exit(sm.context); err != nil {
			return err
		}
		sm.notifyStateExit(currentState)
	}

	sm.SetCurrentState(newState)

	if err := newState.Enter(sm.context); err != nil {
		return err
	}

	sm.notifyStateEnter(newState)

	// Special handling for exit points - automatically trigger external transitions
	if _, isExitPoint := newState.(*ExitPointState); isExitPoint {
		// Look for external transitions from this exit point with empty event trigger
		for _, transition := range sm.transitions {
			if transition.From == newState && transition.Event == "" {
				if transition.CanExecute(sm.context) {
					// Execute the external transition
					if err := sm.executeTransition(transition, event); err != nil {
						return err
					}
					return nil // Successfully transitioned to external state
				}
			}
		}
	}

	if _, isFinal := newState.(*FinalState); isFinal {
		sm.SetCompleted(true)
	}

	if err := sm.eventDeferrer.ProcessDeferredEvents(sm); err != nil {
		return err
	}

	return nil
}

// Notification methods
func (sm *StateMachine) notifyStateEnter(state State) {
	for _, observer := range sm.observers {
		observer.OnStateEnter(sm, state)
	}
}

func (sm *StateMachine) notifyStateExit(state State) {
	for _, observer := range sm.observers {
		observer.OnStateExit(sm, state)
	}
}

func (sm *StateMachine) notifyTransition(from, to State, event *Event) {
	for _, observer := range sm.observers {
		observer.OnTransition(sm, from, to, event)
	}
}

func (sm *StateMachine) notifyEventProcessed(event *Event) {
	for _, observer := range sm.observers {
		observer.OnEventProcessed(sm, event)
	}
}

func (sm *StateMachine) notifyError(err error) {
	for _, observer := range sm.observers {
		observer.OnError(sm, err)
	}
}

// GetLastError returns the last error that occurred
func (sm *StateMachine) GetLastError() error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.lastError
}

// Reset resets the state machine to its initial state
func (sm *StateMachine) Reset() error {
	if err := sm.Stop(); err != nil {
		return err
	}

	sm.mutex.Lock()
	sm.currentState = nil
	sm.isCompleted = false
	sm.lastError = nil
	sm.state = StateStopped
	sm.eventDeferrer.ClearDeferredEvents()
	sm.mutex.Unlock()

	return nil
}
