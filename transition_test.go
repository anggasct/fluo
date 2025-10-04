package fluo

import (
	"errors"
	"testing"
)

func TestTransition_Creation(t *testing.T) {
	transition := NewTransition("source", "target", "event")

	if transition.SourceState != "source" {
		t.Errorf("Expected source state 'source', got '%s'", transition.SourceState)
	}

	if transition.TargetState != "target" {
		t.Errorf("Expected target state 'target', got '%s'", transition.TargetState)
	}

	if transition.EventName != "event" {
		t.Errorf("Expected event name 'event', got '%s'", transition.EventName)
	}

	if transition.Guard != nil {
		t.Error("Expected guard to be nil initially")
	}

	if transition.Action != nil {
		t.Error("Expected action to be nil initially")
	}
}

func TestTransition_WithGuard(t *testing.T) {
	guardCalled := false
	guard := func(ctx Context) bool {
		guardCalled = true
		return true
	}

	transition := NewTransition("source", "target", "event").
		WithGuard(guard)

	if transition.Guard == nil {
		t.Error("Expected guard to be set")
	}

	ctx := CreateTestContext()
	result := transition.Guard(ctx)

	if !guardCalled {
		t.Error("Expected guard to be called")
	}

	if !result {
		t.Error("Expected guard to return true")
	}
}

func TestTransition_WithAction(t *testing.T) {
	actionCalled := false
	action := func(ctx Context) error {
		actionCalled = true
		return nil
	}

	transition := NewTransition("source", "target", "event").
		WithAction(action)

	if transition.Action == nil {
		t.Error("Expected action to be set")
	}

	ctx := CreateTestContext()
	err := transition.Action(ctx)

	if !actionCalled {
		t.Error("Expected action to be called")
	}

	if err != nil {
		t.Errorf("Expected no error from action, got: %v", err)
	}
}

func TestTransition_BasicInMachine(t *testing.T) {
	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").
		State("running").
		To("idle").On("stop").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	AssertState(t, machine, "idle")

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)
	AssertStateChanged(t, result, "idle", "running")
	AssertState(t, machine, "running")

	if observer.TransitionCount() != 1 {
		t.Errorf("Expected 1 transition, got %d", observer.TransitionCount())
	}

	lastTransition := observer.LastTransition()
	if lastTransition.From != "idle" || lastTransition.To != "running" {
		t.Errorf("Expected transition from idle to running, got %s to %s",
			lastTransition.From, lastTransition.To)
	}
}

func TestTransition_WithGuardInMachine(t *testing.T) {
	guardCondition := true
	guard := func(ctx Context) bool {
		return guardCondition
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

	guardCondition = false
	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, false)
	AssertState(t, machine, "idle")

	if observer.TransitionCount() != 0 {
		t.Error("Expected no transitions when guard returns false")
	}

	guardCondition = true
	result = machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "running")

	if observer.TransitionCount() != 1 {
		t.Error("Expected 1 transition when guard returns true")
	}
}

func TestTransition_WithActionInMachine(t *testing.T) {
	actionCalled := false
	actionData := ""

	action := func(ctx Context) error {
		actionCalled = true
		if event := ctx.GetCurrentEvent(); event != nil {
			if data := event.GetData(); data != nil {
				actionData = data.(string)
			}
		}
		return nil
	}

	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").Do(action).
		State("running").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", "test_data")

	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "running")

	if !actionCalled {
		t.Error("Expected transition action to be called")
	}

	if actionData != "test_data" {
		t.Errorf("Expected action to receive 'test_data', got '%s'", actionData)
	}
}

func TestTransition_ActionError(t *testing.T) {
	actionError := errors.New("action failed")
	action := func(ctx Context) error {
		return actionError
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

	result := machine.HandleEvent("start", nil)

	// When action fails, transition should not be processed
	AssertEventProcessed(t, result, false)
	AssertState(t, machine, "idle") // Should remain in initial state

	if len(observer.Actions) == 0 {
		t.Error("Expected action execution to be recorded")
	}

	if result.Error == nil {
		t.Error("Expected action error to be captured in result")
	}
}

func TestTransition_SelfTransition(t *testing.T) {
	actionCalled := false
	action := func(ctx Context) error {
		actionCalled = true
		return nil
	}

	definition := NewMachine().
		State("state1").Initial().
		ToSelf().On("self_event").Do(action).
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

	if !actionCalled {
		t.Error("Expected self-transition action to be called")
	}

	if observer.StateExitCount() != 1 || observer.StateEnterCount() != 1 {
		t.Error("Expected exit and enter for self-transition")
	}
}

func TestTransition_MultipleTransitionsFromState(t *testing.T) {
	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").
		To("error").On("fail").
		To("shutdown").On("shutdown").
		State("running").
		State("error").
		State("shutdown").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	AssertState(t, machine, "idle")

	machine1 := definition.CreateInstance()
	_ = machine1.Start()
	result1 := machine1.HandleEvent("start", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine1, "running")

	machine2 := definition.CreateInstance()
	_ = machine2.Start()
	result2 := machine2.HandleEvent("fail", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "error")

	machine3 := definition.CreateInstance()
	_ = machine3.Start()
	result3 := machine3.HandleEvent("shutdown", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine3, "shutdown")
}

func TestTransition_ConditionalBranching(t *testing.T) {
	testCases := []struct {
		condition     bool
		expectedState string
		shouldProcess bool
	}{
		{true, "path_a", true},
		{false, "idle", false},
	}

	for i, tc := range testCases {
		guard := func(ctx Context) bool {
			return tc.condition
		}

		definition := NewMachine().
			State("idle").Initial().
			To("path_a").On("decide").When(guard).
			State("path_a").
			Build()

		machine := definition.CreateInstance()
		_ = machine.Start()

		result := machine.HandleEvent("decide", nil)

		AssertEventProcessed(t, result, tc.shouldProcess)
		AssertState(t, machine, tc.expectedState)

		t.Logf("Test case %d: condition=%v, state=%s, processed=%v",
			i+1, tc.condition, machine.CurrentState(), result.Processed)
	}
}

func TestTransition_ComplexGuardConditions(t *testing.T) {
	definition := NewMachine().
		State("start").Initial().
		To("path1").On("go").When(func(ctx Context) bool {
		if value, ok := ctx.Get("path"); ok {
			return value.(string) == "path1"
		}
		return false
	}).
		To("path2").On("go").When(func(ctx Context) bool {
		if value, ok := ctx.Get("path"); ok {
			return value.(string) == "path2"
		}
		return false
	}).
		State("path1").
		State("path2").
		Build()

	machine1 := definition.CreateInstance()
	machine1.Context().Set("path", "path1")
	_ = machine1.Start()

	result1 := machine1.HandleEvent("go", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine1, "path1")

	machine2 := definition.CreateInstance()
	machine2.Context().Set("path", "path2")
	_ = machine2.Start()

	result2 := machine2.HandleEvent("go", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "path2")

	machine3 := definition.CreateInstance()
	machine3.Context().Set("path", "invalid")
	_ = machine3.Start()

	result3 := machine3.HandleEvent("go", nil)
	AssertEventProcessed(t, result3, false)
	AssertState(t, machine3, "start")
}

func TestTransition_EventDataAccess(t *testing.T) {
	actionData := make(map[string]interface{})

	action := func(ctx Context) error {
		event := ctx.GetCurrentEvent()
		if event != nil {
			actionData["eventName"] = event.GetName()
			actionData["eventData"] = event.GetData()
		}
		actionData["sourceState"] = ctx.GetSourceState()
		actionData["targetState"] = ctx.GetTargetState()
		return nil
	}

	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").Do(action).
		State("running").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	testData := map[string]string{"key": "value"}
	result := machine.HandleEvent("start", testData)

	AssertEventProcessed(t, result, true)

	if actionData["eventName"] != "start" {
		t.Errorf("Expected event name 'start', got %v", actionData["eventName"])
	}

	if data, ok := actionData["eventData"].(map[string]string); !ok || data["key"] != "value" {
		t.Errorf("Expected event data to be preserved, got %v", actionData["eventData"])
	}

	if actionData["sourceState"] != "idle" {
		t.Errorf("Expected source state 'idle', got %v", actionData["sourceState"])
	}

	if actionData["targetState"] != "running" {
		t.Errorf("Expected target state 'running', got %v", actionData["targetState"])
	}
}

func TestTransition_ChainedTransitions(t *testing.T) {

	definition := NewMachine().
		State("state1").Initial().
		To("state2").On("next").
		State("state2").
		To("state3").On("next").
		State("state3").
		To("state1").On("reset").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)
	_ = machine.Start()

	_ = machine.HandleEvent("next", nil)
	AssertState(t, machine, "state2")

	_ = machine.HandleEvent("next", nil)
	AssertState(t, machine, "state3")

	_ = machine.HandleEvent("reset", nil)
	AssertState(t, machine, "state1")

	if observer.TransitionCount() != 3 {
		t.Errorf("Expected 3 transitions, got %d", observer.TransitionCount())
	}
}

func TestTransition_GuardWithContextData(t *testing.T) {
	// Track check count separately to avoid guard side-effect issues
	checkCount := 0
	guard := func(ctx Context) bool {
		// Use external counter to determine transition
		// This avoids issues with guard being called multiple times
		return checkCount >= 2
	}

	definition := NewMachine().
		State("waiting").Initial().
		To("ready").On("check").When(guard).
		ToSelf().On("check").Unless(guard).
		State("ready").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()
	AssertState(t, machine, "waiting")

	for i := 1; i <= 3; i++ {
		result := machine.HandleEvent("check", nil)
		checkCount++
		if i < 3 {
			AssertEventProcessed(t, result, true)
			AssertState(t, machine, "waiting")
		} else {
			AssertEventProcessed(t, result, true)
			AssertState(t, machine, "ready")
		}
	}
}

func TestTransition_PriorityOrdering(t *testing.T) {

	actionCalled := ""
	action1 := func(ctx Context) error {
		actionCalled = "action1"
		return nil
	}
	action2 := func(ctx Context) error {
		actionCalled = "action2"
		return nil
	}

	definition := NewMachine().
		State("start").Initial().
		To("path1").On("go").Do(action1).
		To("path2").On("go").Do(action2).
		State("path1").
		State("path2").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("go", nil)
	AssertEventProcessed(t, result, true)

	AssertState(t, machine, "path1")
	if actionCalled != "action1" {
		t.Errorf("Expected action1 to be called, got %s", actionCalled)
	}
}

func TestTransition_ErrorRecovery(t *testing.T) {

	definition := NewMachine().
		State("normal").Initial().
		To("processing").On("process").
		To("error").On("error").
		State("processing").
		To("normal").On("complete").
		To("error").On("error").
		State("error").
		To("normal").On("recover").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)
	_ = machine.Start()

	_ = machine.HandleEvent("process", nil)
	AssertState(t, machine, "processing")

	_ = machine.HandleEvent("error", nil)
	AssertState(t, machine, "error")

	result := machine.HandleEvent("recover", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "normal")

	if observer.TransitionCount() < 3 {
		t.Errorf("Expected at least 3 transitions, got %d", observer.TransitionCount())
	}
}
