package fluo

import (
	"sync"
	"testing"
	"time"
)

func TestObserver_BasicInterface(t *testing.T) {
	observer := NewTestObserver()

	var _ Observer = observer

	var _ ExtendedObserver = observer
}

func TestObserver_StateTransitions(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)
	_ = machine.Start()

	observer.Reset()

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	if observer.TransitionCount() != 1 {
		t.Errorf("Expected 1 transition, got %d", observer.TransitionCount())
	}

	lastTransition := observer.LastTransition()
	if lastTransition == nil {
		t.Error("Expected last transition to be recorded")
		return
	}

	if lastTransition.From != "idle" {
		t.Errorf("Expected transition from 'idle', got '%s'", lastTransition.From)
	}

	if lastTransition.To != "running" {
		t.Errorf("Expected transition to 'running', got '%s'", lastTransition.To)
	}

	if lastTransition.Event == nil {
		t.Error("Expected transition event to be recorded")
	}

	if lastTransition.Event.GetName() != "start" {
		t.Errorf("Expected event name 'start', got '%s'", lastTransition.Event.GetName())
	}
}

func TestObserver_StateEnterExit(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)
	_ = machine.Start()

	if observer.StateEnterCount() != 1 {
		t.Errorf("Expected 1 state enter, got %d", observer.StateEnterCount())
	}

	lastEnter := observer.LastStateEnter()
	if lastEnter == nil || lastEnter.State != "idle" {
		t.Error("Expected initial state enter to be 'idle'")
	}

	observer.Reset()

	_ = machine.HandleEvent("start", nil)

	if observer.StateExitCount() != 1 {
		t.Errorf("Expected 1 state exit, got %d", observer.StateExitCount())
	}

	if observer.StateEnterCount() != 1 {
		t.Errorf("Expected 1 state enter, got %d", observer.StateEnterCount())
	}

	if len(observer.StateExits) == 0 || observer.StateExits[0].State != "idle" {
		t.Error("Expected state exit from 'idle'")
	}

	lastEnter = observer.LastStateEnter()
	if lastEnter == nil || lastEnter.State != "running" {
		t.Error("Expected state enter to 'running'")
	}
}

func TestObserver_SelfTransition(t *testing.T) {
	definition := NewMachine().
		State("active").Initial().
		ToSelf().On("refresh").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	observer.Reset()

	_ = machine.HandleEvent("refresh", nil)

	if observer.StateExitCount() != 1 {
		t.Errorf("Expected 1 state exit for self-transition, got %d", observer.StateExitCount())
	}

	if observer.StateEnterCount() != 1 {
		t.Errorf("Expected 1 state enter for self-transition, got %d", observer.StateEnterCount())
	}

	if observer.TransitionCount() != 1 {
		t.Errorf("Expected 1 transition for self-transition, got %d", observer.TransitionCount())
	}

	lastTransition := observer.LastTransition()
	if lastTransition.From != "active" || lastTransition.To != "active" {
		t.Error("Expected self-transition from 'active' to 'active'")
	}
}

func TestObserver_EventRejection(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)
	_ = machine.Start()

	result := machine.HandleEvent("invalid_event", nil)
	AssertEventProcessed(t, result, false)

	if len(observer.EventRejects) != 1 {
		t.Errorf("Expected 1 event rejection, got %d", len(observer.EventRejects))
	}

	rejection := observer.EventRejects[0]
	if rejection.Event.GetName() != "invalid_event" {
		t.Errorf("Expected rejected event name 'invalid_event', got '%s'", rejection.Event.GetName())
	}

	if rejection.Reason == "" {
		t.Error("Expected rejection reason to be provided")
	}
}

func TestObserver_MachineLifecycle(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)

	_ = machine.Start()

	if len(observer.Started) != 1 {
		t.Errorf("Expected 1 machine start notification, got %d", len(observer.Started))
	}

	_ = machine.Stop()

	if len(observer.Stopped) != 1 {
		t.Errorf("Expected 1 machine stop notification, got %d", len(observer.Stopped))
	}
}

func TestObserver_ActionExecution(t *testing.T) {
	actionCalled := false
	action := func(ctx Context) error {
		actionCalled = true
		return nil
	}

	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").Do(action).
		State("running").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	_ = machine.HandleEvent("start", nil)

	if !actionCalled {
		t.Error("Expected action to be called")
	}

	if len(observer.Actions) == 0 {
		t.Error("Expected action execution to be observed")
	}

	actionEvent := observer.Actions[0]
	if actionEvent.ActionType != "transition" {
		t.Errorf("Expected action type 'transition', got '%s'", actionEvent.ActionType)
	}

	if actionEvent.State != "idle" {
		t.Errorf("Expected action state 'idle', got '%s'", actionEvent.State)
	}
}

func TestObserver_MultipleObservers(t *testing.T) {
	machine := CreateSimpleMachine()

	observer1 := NewTestObserver()
	observer2 := NewTestObserver()
	observer3 := NewTestObserver()

	machine.AddObserver(observer1)
	machine.AddObserver(observer2)
	machine.AddObserver(observer3)

	_ = machine.Start()
	_ = machine.HandleEvent("start", nil)

	observers := []*TestObserver{observer1, observer2, observer3}
	for i, obs := range observers {
		if obs.TransitionCount() != 1 {
			t.Errorf("Observer %d: expected 1 transition, got %d", i+1, obs.TransitionCount())
		}

		if obs.StateEnterCount() != 2 {
			t.Errorf("Observer %d: expected 2 state enters, got %d", i+1, obs.StateEnterCount())
		}

		if obs.StateExitCount() != 1 {
			t.Errorf("Observer %d: expected 1 state exit, got %d", i+1, obs.StateExitCount())
		}
	}
}

func TestObserver_RemoveObserver(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)
	_ = machine.Start()

	if observer.StateEnterCount() == 0 {
		t.Error("Expected observer to receive initial notifications")
	}

	observer.Reset()
	machine.RemoveObserver(observer)

	_ = machine.HandleEvent("start", nil)

	if observer.TransitionCount() != 0 {
		t.Error("Expected removed observer not to receive transition notifications")
	}

	if observer.StateEnterCount() != 0 {
		t.Error("Expected removed observer not to receive state enter notifications")
	}
}

func TestObserver_ContextAccess(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)
	machine.Context().Set("test_data", "observer_test")

	_ = machine.Start()
	_ = machine.HandleEvent("start", nil)

	if observer.TransitionCount() > 0 {
		transition := observer.Transitions[0]
		if transition.Ctx == nil {
			t.Error("Expected transition to include context")
		}

		if value, ok := transition.Ctx.Get("test_data"); !ok || value != "observer_test" {
			t.Error("Expected context data to be accessible in observer")
		}
	}

	if observer.StateEnterCount() > 0 {
		stateEnter := observer.StateEnters[len(observer.StateEnters)-1]
		if stateEnter.Ctx == nil {
			t.Error("Expected state enter to include context")
		}
	}
}

func TestObserver_ConcurrentNotifications(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)
	_ = machine.Start()

	const numEvents = 100
	var wg sync.WaitGroup

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(eventID int) {
			defer wg.Done()
			eventName := "start"
			if eventID%2 == 0 {
				eventName = "stop"
			}
			machine.HandleEvent(eventName, eventID)
		}(i)
	}

	wg.Wait()

	totalNotifications := observer.TransitionCount() +
		observer.StateEnterCount() +
		observer.StateExitCount()

	if totalNotifications == 0 {
		t.Error("Expected some notifications to be received")
	}
}

func TestObserver_BaseObserverNoOp(t *testing.T) {
	baseObserver := &BaseObserver{}

	testEvent := CreateTestEvent("test", nil)
	testCtx := CreateTestContext()

	baseObserver.OnTransition("from", "to", testEvent, testCtx)
	baseObserver.OnStateEnter("state", testCtx)
	baseObserver.OnStateExit("state", testCtx)
	baseObserver.OnGuardEvaluation("from", "to", testEvent, true, testCtx)
	baseObserver.OnEventRejected(testEvent, "reason", testCtx)
	baseObserver.OnError(NewStateError(ErrCodeStateNotFound, "test", "error"), testCtx)
	baseObserver.OnActionExecution("action", "state", testEvent, testCtx)
	baseObserver.OnMachineStarted(testCtx)
	baseObserver.OnMachineStopped(testCtx)

}

func TestObserver_GuardEvaluation(t *testing.T) {
	guardResult := true
	guard := func(ctx Context) bool {
		return guardResult
	}

	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").When(guard).
		State("running").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()

	guardResult = true
	_ = machine.HandleEvent("start", nil)

}

func TestObserver_ErrorHandling(t *testing.T) {

	errorObserver := NewTestObserver()

	machine := CreateSimpleMachine()
	machine.AddObserver(errorObserver)

	_ = machine.Start()

	err := machine.SetState("nonexistent_state")
	if err == nil {
		t.Error("Expected error when setting nonexistent state")
	}

}

func TestObserver_ComplexScenario(t *testing.T) {

	definition := NewMachine().
		State("init").Initial().
		To("loading").On("load").
		State("loading").
		To("ready").On("loaded").
		To("error").On("fail").
		State("ready").
		To("processing").On("process").
		State("processing").
		To("ready").On("complete").
		To("error").On("fail").
		State("error").
		To("init").On("retry").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	observer.Reset()

	events := []string{"load", "loaded", "process", "complete", "process", "fail", "retry"}

	for _, eventName := range events {
		result := machine.HandleEvent(eventName, nil)
		if !result.Processed {

			continue
		}
	}

	if observer.TransitionCount() == 0 {
		t.Error("Expected some transitions to be observed")
	}

	if observer.StateEnterCount() == 0 {
		t.Error("Expected some state enters to be observed")
	}

	if observer.StateExitCount() == 0 {
		t.Error("Expected some state exits to be observed")
	}

	for i, transition := range observer.Transitions {
		if transition.From == "" || transition.To == "" {
			t.Errorf("Transition %d: expected non-empty from/to states", i)
		}

		if transition.Event == nil {
			t.Errorf("Transition %d: expected non-nil event", i)
		}

		if transition.Ctx == nil {
			t.Errorf("Transition %d: expected non-nil context", i)
		}
	}
}

func TestObserver_ObserverManagerDirect(t *testing.T) {
	manager := NewObserverManager()

	if manager == nil {
		t.Error("Expected non-nil observer manager")
	}

	observer1 := NewTestObserver()
	observer2 := NewTestObserver()

	manager.AddObserver(observer1)
	manager.AddObserver(observer2)

	testEvent := CreateTestEvent("test", nil)
	testCtx := CreateTestContext()

	manager.NotifyTransition("from", "to", testEvent, testCtx)

	if observer1.TransitionCount() != 1 {
		t.Error("Expected observer1 to receive transition notification")
	}

	if observer2.TransitionCount() != 1 {
		t.Error("Expected observer2 to receive transition notification")
	}

	manager.RemoveObserver(observer1)

	manager.NotifyStateEnter("state", testCtx)

	if observer1.StateEnterCount() != 0 {
		t.Error("Expected removed observer not to receive notifications")
	}

	if observer2.StateEnterCount() != 1 {
		t.Error("Expected remaining observer to receive notification")
	}
}

func TestObserver_PerformanceWithManyObservers(t *testing.T) {
	machine := CreateSimpleMachine()

	const numObservers = 100
	observers := make([]*TestObserver, numObservers)

	for i := 0; i < numObservers; i++ {
		observers[i] = NewTestObserver()
		machine.AddObserver(observers[i])
	}

	start := time.Now()

	_ = machine.Start()
	_ = machine.HandleEvent("start", nil)
	_ = machine.HandleEvent("stop", nil)

	duration := time.Since(start)

	if duration > time.Second {
		t.Errorf("Observer notifications took too long: %v", duration)
	}

	for i, obs := range observers {
		if obs.StateEnterCount() < 2 {
			t.Errorf("Observer %d: expected multiple state enters, got %d", i, obs.StateEnterCount())
		}
	}

	t.Logf("Notified %d observers in %v", numObservers, duration)
}
