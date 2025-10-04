package fluo

import (
	"sync"
	"testing"
	"time"
)

func TestStateMachine_Start(t *testing.T) {
	machine := CreateSimpleMachine()

	err := machine.Start()
	if err != nil {
		t.Fatalf("Expected no error starting machine, got: %v", err)
	}

	AssertState(t, machine, "idle")
}

func TestStateMachine_StartAlreadyStarted(t *testing.T) {
	machine := CreateSimpleMachine()

	_ = machine.Start()
	err := machine.Start()

	if err == nil {
		t.Error("Expected error when starting already started machine")
	}
}

func TestStateMachine_Stop(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	err := machine.Stop()

	if err != nil {
		t.Fatalf("Expected no error stopping machine, got: %v", err)
	}

	if len(observer.Stopped) != 1 {
		t.Error("Expected machine stopped notification")
	}
}

func TestStateMachine_StopNotStarted(t *testing.T) {
	machine := CreateSimpleMachine()

	err := machine.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-started machine")
	}
}

func TestStateMachine_Reset(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	_ = machine.HandleEvent("start", nil)
	AssertState(t, machine, "running")

	err := machine.Reset()
	if err != nil {
		t.Fatalf("Expected no error resetting machine, got: %v", err)
	}

	AssertState(t, machine, "idle")
}

func TestStateMachine_BasicTransition(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()

	result := machine.HandleEvent("start", nil)

	AssertEventProcessed(t, result, true)
	AssertStateChanged(t, result, "idle", "running")
	AssertState(t, machine, "running")
	AssertObserverCalled(t, observer, 1, 2, 1)
}

func TestStateMachine_InvalidTransition(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()

	result := machine.HandleEvent("invalid", nil)

	AssertEventProcessed(t, result, false)
	AssertState(t, machine, "idle")

	if len(observer.EventRejects) != 1 {
		t.Error("Expected event rejection notification")
	}
}

func TestStateMachine_SelfTransition(t *testing.T) {
	definition := NewMachine().
		State("state1").Initial().
		ToSelf().On("self_event").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	observer.Reset()

	result := machine.HandleEvent("self_event", nil)

	AssertEventProcessed(t, result, true)
	AssertStateChanged(t, result, "state1", "state1")
	AssertState(t, machine, "state1")

	if observer.StateExitCount() != 1 || observer.StateEnterCount() != 1 {
		t.Error("Expected exit and enter for self-transition")
	}
}

func TestStateMachine_TransitionWithGuard(t *testing.T) {
	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").When(TestGuard).
		State("running").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	SetTestGuard(false)
	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, false)
	AssertState(t, machine, "idle")

	SetTestGuard(true)
	result = machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "running")
}

func TestStateMachine_TransitionWithAction(t *testing.T) {
	ResetTestAction()

	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").Do(TestAction).
		State("running").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", nil)

	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "running")

	if !TestActionCalled {
		t.Error("Expected transition action to be called")
	}
}

func TestStateMachine_CurrentState(t *testing.T) {
	machine := CreateSimpleMachine()

	currentState := machine.CurrentState()
	if currentState != "" && currentState != "idle" {
		t.Errorf("Expected empty or idle current state before start, got '%s'", currentState)
	}

	_ = machine.Start()
	AssertState(t, machine, "idle")

	_ = machine.HandleEvent("start", nil)
	AssertState(t, machine, "running")
}

func TestStateMachine_SetState(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()
	machine.AddObserver(observer)
	_ = machine.Start()

	err := machine.SetState("running")
	if err != nil {
		t.Fatalf("Expected no error setting state, got: %v", err)
	}

	AssertState(t, machine, "running")

	if len(observer.StateExits) == 0 || len(observer.StateEnters) == 0 {
		t.Error("Expected state exit and enter notifications")
	}
}

func TestStateMachine_SetStateInvalid(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	err := machine.SetState("nonexistent")
	if err == nil {
		t.Error("Expected error setting invalid state")
	}
}

func TestStateMachine_IsInState(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	if !machine.IsInState("idle") {
		t.Error("Expected machine to be in idle state")
	}

	if machine.IsInState("running") {
		t.Error("Expected machine not to be in running state")
	}

	_ = machine.HandleEvent("start", nil)

	if !machine.IsInState("running") {
		t.Error("Expected machine to be in running state")
	}

	if machine.IsInState("idle") {
		t.Error("Expected machine not to be in idle state")
	}
}

func TestStateMachine_GetActiveStates(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	activeStates := machine.GetActiveStates()
	if len(activeStates) != 1 || activeStates[0] != "idle" {
		t.Errorf("Expected active states [idle], got %v", activeStates)
	}

	_ = machine.HandleEvent("start", nil)

	activeStates = machine.GetActiveStates()
	if len(activeStates) != 1 || activeStates[0] != "running" {
		t.Errorf("Expected active states [running], got %v", activeStates)
	}
}

func TestStateMachine_IsStateActive(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	if !machine.IsStateActive("idle") {
		t.Error("Expected idle state to be active")
	}

	if machine.IsStateActive("running") {
		t.Error("Expected running state not to be active")
	}
}

func TestStateMachine_GetStateHierarchy(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	hierarchy := machine.GetStateHierarchy()
	if len(hierarchy) != 1 || hierarchy[0] != "idle" {
		t.Errorf("Expected hierarchy [idle], got %v", hierarchy)
	}
}

func TestStateMachine_Context(t *testing.T) {
	machine := CreateSimpleMachine()

	ctx := machine.Context()
	if ctx == nil {
		t.Error("Expected non-nil context")
	}

	ctx.Set("test", "value")
	if value, ok := ctx.Get("test"); !ok || value != "value" {
		t.Error("Expected context to store and retrieve values")
	}
}

func TestStateMachine_WithContext(t *testing.T) {
	machine := CreateSimpleMachine()
	newCtx := CreateTestContext()
	newCtx.Set("test", "value")

	newMachine := machine.WithContext(newCtx)
	if newMachine.Context() != newCtx {
		t.Error("Expected context to be updated")
	}
}

func TestStateMachine_AddRemoveObserver(t *testing.T) {
	machine := CreateSimpleMachine()
	observer := NewTestObserver()

	machine.AddObserver(observer)
	_ = machine.Start()

	if len(observer.StateEnters) == 0 {
		t.Error("Expected observer to receive notifications")
	}

	machine.RemoveObserver(observer)
	observer.Reset()

	_ = machine.HandleEvent("start", nil)

	if len(observer.StateEnters) > 0 {
		t.Error("Expected observer to be removed and not receive notifications")
	}
}

func TestStateMachine_HandleEventNotStarted(t *testing.T) {
	machine := CreateSimpleMachine()

	result := machine.HandleEvent("start", nil)

	AssertEventProcessed(t, result, false)
	if result.RejectionReason == "" {
		t.Error("Expected rejection reason for event on non-started machine")
	}
}

func TestStateMachine_SendEvent(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	result := machine.SendEvent("start", nil)

	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "running")
}

func TestStateMachine_SendEventWithContext(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	result := machine.SendEventWithContext(machine.Context(), "start", nil)

	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "running")
}

func TestStateMachine_ThreadSafety(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	const numGoroutines = 100
	const eventsPerGoroutine = 10

	var wg sync.WaitGroup
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ConcurrentEventSender(machine, "start", eventsPerGoroutine, done)
		}()
	}

	results := make(chan string, numGoroutines*eventsPerGoroutine)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ConcurrentStateChecker(machine, eventsPerGoroutine, results)
		}()
	}

	wg.Wait()
	close(done)
	close(results)

	finalState := machine.CurrentState()
	if finalState != "idle" && finalState != "running" {
		t.Errorf("Expected final state to be valid, got %s", finalState)
	}
}

func TestStateMachine_ComplexScenario(t *testing.T) {

	definition := NewMachine().
		State("init").Initial().
		To("loading").On("load").
		State("loading").
		To("ready").On("loaded").
		To("error").On("fail").
		State("ready").
		To("processing").On("process").
		To("shutdown").On("shutdown").
		State("processing").
		To("ready").On("complete").
		To("error").On("fail").
		State("error").
		To("init").On("retry").
		State("shutdown").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	AssertState(t, machine, "init")

	_ = machine.HandleEvent("load", nil)
	AssertState(t, machine, "loading")

	_ = machine.HandleEvent("loaded", nil)
	AssertState(t, machine, "ready")

	_ = machine.HandleEvent("process", nil)
	AssertState(t, machine, "processing")

	_ = machine.HandleEvent("complete", nil)
	AssertState(t, machine, "ready")

	_ = machine.HandleEvent("process", nil)
	AssertState(t, machine, "processing")

	_ = machine.HandleEvent("fail", nil)
	AssertState(t, machine, "error")

	_ = machine.HandleEvent("retry", nil)
	AssertState(t, machine, "init")

	_ = machine.HandleEvent("load", nil)
	_ = machine.HandleEvent("loaded", nil)
	_ = machine.HandleEvent("shutdown", nil)
	AssertState(t, machine, "shutdown")

	if observer.TransitionCount() < 8 {
		t.Errorf("Expected at least 8 transitions, got %d", observer.TransitionCount())
	}
}

func TestStateMachine_JSONSerialization(t *testing.T) {
	machine := CreateSimpleMachine()
	machine.Context().Set("test_data", "serializable")
	_ = machine.Start()
	_ = machine.HandleEvent("start", nil)

	jsonData, err := machine.MarshalJSON()
	if err != nil {
		t.Fatalf("Expected no error marshaling to JSON, got: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON data")
	}

	newMachine := CreateSimpleMachine()
	err = newMachine.UnmarshalJSON(jsonData)
	if err != nil {
		t.Fatalf("Expected no error unmarshaling from JSON, got: %v", err)
	}

	if newMachine.CurrentState() != machine.CurrentState() {
		t.Error("Expected unmarshaled machine to have same current state")
	}

	if value, ok := newMachine.Context().Get("test_data"); !ok || value != "serializable" {
		t.Error("Expected context data to be restored")
	}
}

func TestStateMachine_PerformanceBasic(t *testing.T) {
	machine := CreateSimpleMachine()
	_ = machine.Start()

	const numEvents = 10000
	start := time.Now()

	for i := 0; i < numEvents; i++ {
		eventName := "start"
		if i%2 == 1 {
			eventName = "stop"
		}
		machine.HandleEvent(eventName, i)
	}

	duration := time.Since(start)
	eventsPerSecond := float64(numEvents) / duration.Seconds()

	t.Logf("Processed %d events in %v (%.0f events/sec)", numEvents, duration, eventsPerSecond)

	if eventsPerSecond < 1000 {
		t.Errorf("Performance too low: %.0f events/sec, expected at least 1000", eventsPerSecond)
	}
}

func createTestMachine() Machine {
	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").
		State("running").
		To("idle").On("stop").
		Build()
	return definition.CreateInstance()
}

func TestStateMachine_EdgeCases(t *testing.T) {
	t.Run("ConcurrentStartStop", func(t *testing.T) {
		machine := createTestMachine()

		// Test multiple start calls
		err1 := machine.Start()
		err2 := machine.Start()

		if err1 != nil {
			t.Errorf("First start should succeed, got: %v", err1)
		}

		if err2 == nil {
			t.Error("Second start should fail")
		}

		// Test multiple stop calls
		err3 := machine.Stop()
		err4 := machine.Stop()

		if err3 != nil {
			t.Errorf("First stop should succeed, got: %v", err3)
		}

		if err4 == nil {
			t.Error("Second stop should fail")
		}
	})

	t.Run("EventHandlingInInvalidStates", func(t *testing.T) {
		machine := createTestMachine()

		// Test handling events before start
		result := machine.HandleEvent("start", nil)
		if result.Success() {
			t.Error("Event handling should fail before machine start")
		}

		_ = machine.Start()

		// Test handling invalid event
		result = machine.HandleEvent("invalid_event", nil)
		if result.Success() {
			t.Error("Invalid event should not be processed")
		}

		_ = machine.Stop()

		// Test handling events after stop
		result = machine.HandleEvent("start", nil)
		if result.Success() {
			t.Error("Event handling should fail after machine stop")
		}
	})

	t.Run("StateQueryOperations", func(t *testing.T) {
		machine := createTestMachine()

		// Test queries before start
		currentState := machine.CurrentState()
		if currentState == "" {
			t.Errorf("Current state should not be empty after machine creation, got: %v", currentState)
		}

		if !machine.IsInState("idle") {
			t.Error("IsInState should return true for initial state before start")
		}

		_ = machine.Start()

		// Test queries after start
		currentState = machine.CurrentState()
		if currentState == "" {
			t.Error("Current state should not be empty after start")
		}

		if !machine.IsInState("idle") {
			t.Error("Should be in idle state after start")
		}

		if !machine.IsStateActive("idle") {
			t.Error("Idle state should be active")
		}

		_ = machine.Stop()

		// Test queries after stop
		currentState = machine.CurrentState()
		if currentState == "" {
			t.Errorf("Current state should not be empty after stop, got: %v", currentState)
		}
	})
}
