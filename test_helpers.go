package fluo

import (
	"sync"
	"testing"
)

// TestObserver is a mock observer for testing that captures all observer events
type TestObserver struct {
	mutex        sync.RWMutex
	Transitions  []TransitionEvent
	StateEnters  []StateEvent
	StateExits   []StateEvent
	EventRejects []EventRejectEvent
	Errors       []ErrorEvent
	Actions      []ActionEvent
	Started      []ContextEvent
	Stopped      []ContextEvent
	Guards       []GuardEvent
}

type TransitionEvent struct {
	From  string
	To    string
	Event Event
	Ctx   Context
}

type StateEvent struct {
	State string
	Ctx   Context
}

type EventRejectEvent struct {
	Event  Event
	Reason string
	Ctx    Context
}

type ErrorEvent struct {
	Error error
	Ctx   Context
}

type ActionEvent struct {
	ActionType string
	State      string
	Event      Event
	Ctx        Context
}

type ContextEvent struct {
	Ctx Context
}

type GuardEvent struct {
	From   string
	To     string
	Event  Event
	Result bool
	Ctx    Context
}

// NewTestObserver creates a new test observer
func NewTestObserver() *TestObserver {
	return &TestObserver{
		Transitions:  make([]TransitionEvent, 0),
		StateEnters:  make([]StateEvent, 0),
		StateExits:   make([]StateEvent, 0),
		EventRejects: make([]EventRejectEvent, 0),
		Errors:       make([]ErrorEvent, 0),
		Actions:      make([]ActionEvent, 0),
		Started:      make([]ContextEvent, 0),
		Stopped:      make([]ContextEvent, 0),
		Guards:       make([]GuardEvent, 0),
	}
}

// Observer interface implementations
func (o *TestObserver) OnTransition(from string, to string, event Event, ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.Transitions = append(o.Transitions, TransitionEvent{From: from, To: to, Event: event, Ctx: ctx})
}

func (o *TestObserver) OnStateEnter(state string, ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.StateEnters = append(o.StateEnters, StateEvent{State: state, Ctx: ctx})
}

// ExtendedObserver interface implementations
func (o *TestObserver) OnStateExit(state string, ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.StateExits = append(o.StateExits, StateEvent{State: state, Ctx: ctx})
}

func (o *TestObserver) OnGuardEvaluation(from string, to string, event Event, result bool, ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.Guards = append(o.Guards, GuardEvent{From: from, To: to, Event: event, Result: result, Ctx: ctx})
}

func (o *TestObserver) OnEventRejected(event Event, reason string, ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.EventRejects = append(o.EventRejects, EventRejectEvent{Event: event, Reason: reason, Ctx: ctx})
}

func (o *TestObserver) OnError(err error, ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.Errors = append(o.Errors, ErrorEvent{Error: err, Ctx: ctx})
}

func (o *TestObserver) OnActionExecution(actionType string, state string, event Event, ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.Actions = append(o.Actions, ActionEvent{ActionType: actionType, State: state, Event: event, Ctx: ctx})
}

func (o *TestObserver) OnMachineStarted(ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.Started = append(o.Started, ContextEvent{Ctx: ctx})
}

func (o *TestObserver) OnMachineStopped(ctx Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.Stopped = append(o.Stopped, ContextEvent{Ctx: ctx})
}

// Helper methods for test assertions
func (o *TestObserver) Reset() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.Transitions = nil
	o.StateEnters = nil
	o.StateExits = nil
	o.EventRejects = nil
	o.Errors = nil
	o.Actions = nil
	o.Started = nil
	o.Stopped = nil
	o.Guards = nil
}

func (o *TestObserver) TransitionCount() int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return len(o.Transitions)
}

func (o *TestObserver) StateEnterCount() int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return len(o.StateEnters)
}

func (o *TestObserver) StateExitCount() int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return len(o.StateExits)
}

func (o *TestObserver) LastTransition() *TransitionEvent {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	if len(o.Transitions) == 0 {
		return nil
	}
	return &o.Transitions[len(o.Transitions)-1]
}

func (o *TestObserver) LastStateEnter() *StateEvent {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	if len(o.StateEnters) == 0 {
		return nil
	}
	return &o.StateEnters[len(o.StateEnters)-1]
}

// Test machine builders - common machine configurations for testing

// CreateSimpleMachine creates a basic state machine for testing
func CreateSimpleMachine() Machine {
	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").
		State("running").
		To("stopped").On("stop").
		State("stopped").
		To("idle").On("reset").
		Build()

	return definition.CreateInstance()
}

// CreateHierarchicalMachine creates a hierarchical state machine for testing
func CreateHierarchicalMachine() Machine {
	definition := NewMachine().
		State("offline").Initial().
		To("online").On("connect").
		CompositeState("online").
		State("online.idle").Initial().
		To("online.processing").On("process").
		State("online.processing").
		To("online.idle").On("complete").
		Build()

	return definition.CreateInstance()
}

// CreateParallelMachine creates a parallel state machine for testing
func CreateParallelMachine() Machine {
	builder := NewMachine()
	builder.State("inactive").Initial().
		To("active").On("activate")

	parallelBuilder := builder.ParallelState("active")
	motorRegion := parallelBuilder.Region("motor")
	motorRegion.State("stopped").Initial().
		To("running").On("start_motor")
	motorRegion.State("running")

	lightsRegion := parallelBuilder.Region("lights")
	lightsRegion.State("off").Initial().
		To("on").On("turn_on_lights")
	lightsRegion.State("on")

	definition := builder.Build()
	return definition.CreateInstance()
}

// CreatePseudostateMachine creates a machine with pseudostates for testing
func CreatePseudostateMachine() Machine {
	builder := NewMachine()
	builder.State("start").Initial().
		To("choice1").On("decide")

	builder.Choice("choice1").
		When(func(ctx Context) bool {
			if value, ok := ctx.Get("condition"); ok {
				return value.(bool)
			}
			return false
		}).To("path_a").
		Otherwise("path_b")

	builder.State("path_a")
	builder.State("path_b")

	definition := builder.Build()
	return definition.CreateInstance()
}

// Test assertions and utilities

// AssertState checks if machine is in expected state
func AssertState(t *testing.T, machine Machine, expectedState string) {
	t.Helper()
	currentState := machine.CurrentState()
	if currentState != expectedState {
		t.Errorf("Expected state %s, got %s", expectedState, currentState)
	}
}

// AssertStateChanged checks if state transition occurred
func AssertStateChanged(t *testing.T, result *EventResult, expectedPrevious, expectedCurrent string) {
	t.Helper()
	if !result.StateChanged {
		t.Error("Expected state to change")
	}
	if result.PreviousState != expectedPrevious {
		t.Errorf("Expected previous state %s, got %s", expectedPrevious, result.PreviousState)
	}
	if result.CurrentState != expectedCurrent {
		t.Errorf("Expected current state %s, got %s", expectedCurrent, result.CurrentState)
	}
}

// AssertEventProcessed checks if event was processed successfully
func AssertEventProcessed(t *testing.T, result *EventResult, shouldProcess bool) {
	t.Helper()
	if result.Processed != shouldProcess {
		if shouldProcess {
			t.Error("Expected event to be processed")
		} else {
			t.Error("Expected event to be rejected")
		}
	}
}

// AssertObserverCalled checks if observer methods were called expected number of times
func AssertObserverCalled(t *testing.T, observer *TestObserver, transitions, enters, exits int) {
	t.Helper()
	if observer.TransitionCount() != transitions {
		t.Errorf("Expected %d transitions, got %d", transitions, observer.TransitionCount())
	}
	if observer.StateEnterCount() != enters {
		t.Errorf("Expected %d state enters, got %d", enters, observer.StateEnterCount())
	}
	if observer.StateExitCount() != exits {
		t.Errorf("Expected %d state exits, got %d", exits, observer.StateExitCount())
	}
}

// CreateTestContext creates a simple test context
func CreateTestContext() Context {
	return NewSimpleContext()
}

// CreateTestEvent creates a test event
func CreateTestEvent(name string, data any) Event {
	return NewEvent(name, data)
}

// Test action functions for testing
var TestActionCalled bool
var TestActionError error

func TestAction(ctx Context) error {
	TestActionCalled = true
	return TestActionError
}

func ResetTestAction() {
	TestActionCalled = false
	TestActionError = nil
}

// Test guard functions for testing
var TestGuardResult bool

func TestGuard(ctx Context) bool {
	return TestGuardResult
}

func SetTestGuard(result bool) {
	TestGuardResult = result
}

// Concurrent testing utilities

// ConcurrentEventSender sends events concurrently for testing thread safety
func ConcurrentEventSender(machine Machine, eventName string, count int, done chan bool) {
	for i := 0; i < count; i++ {
		machine.HandleEvent(eventName, i)
	}
	done <- true
}

// ConcurrentStateChecker checks states concurrently for testing thread safety
func ConcurrentStateChecker(machine Machine, checks int, results chan string) {
	for i := 0; i < checks; i++ {
		results <- machine.CurrentState()
	}
}

// AssertContextValue checks if context contains expected value
func AssertContextValue(t *testing.T, ctx Context, key string, expected interface{}) {
	t.Helper()
	if value, ok := ctx.Get(key); !ok {
		t.Errorf("Expected context to contain key '%s'", key)
	} else if value != expected {
		t.Errorf("Expected context[%s] to be %v, got %v", key, expected, value)
	}
}

// AssertGuardEvaluationCount checks if guard was evaluated expected number of times
func AssertGuardEvaluationCount(t *testing.T, count int, expected int) {
	t.Helper()
	if count != expected {
		t.Errorf("Expected guard to be evaluated %d times, got %d", expected, count)
	}
}

// AssertTransitionSequence checks if machine followed expected state sequence
func AssertTransitionSequence(t *testing.T, machine Machine, expectedStates []string) {
	t.Helper()
	currentState := machine.CurrentState()
	for _, expected := range expectedStates {
		if currentState == expected {
			return // Found match
		}
	}
	t.Errorf("Expected current state to be one of %v, got %s", expectedStates, currentState)
}

// CreateEdgeCaseMachine creates a machine configured for edge case testing
func CreateEdgeCaseMachine() Machine {
	definition := NewMachine().
		State("initial").Initial().
		To("processing").On("start").
		State("processing").
		To("complete").On("finish").
		State("complete").
		Build()
	return definition.CreateInstance()
}
