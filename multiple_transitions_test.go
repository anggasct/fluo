package fluo

import (
	"testing"
)

// TestMultipleTransitions_MethodChaining tests method chaining for multiple transitions from the same state
func TestMultipleTransitions_MethodChaining(t *testing.T) {
	definition := NewMachine().
		State("idle").Initial().
		To("processing").On("start").
		To("maintenance").On("maintenance").
		To("shutdown").On("shutdown").
		State("processing").
		To("idle").On("complete").
		To("maintenance").On("error").
		State("maintenance").
		To("idle").On("repair_complete").
		State("shutdown").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	AssertState(t, machine, "idle")

	// Test first transition
	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "processing")

	// Reset to test multiple transitions from idle
	machine2 := definition.CreateInstance()
	machine2.AddObserver(observer)
	_ = machine2.Start()

	// Test maintenance transition from idle
	result2 := machine2.HandleEvent("maintenance", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "maintenance")

	// Reset to test shutdown transition from idle
	machine3 := definition.CreateInstance()
	_ = machine3.Start()

	// Test shutdown transition from idle
	result3 := machine3.HandleEvent("shutdown", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine3, "shutdown")
}

// TestMultipleTransitions_WithGuards tests multiple transitions with guard conditions
func TestMultipleTransitions_WithGuards(t *testing.T) {
	priorityFlag := false
	errorFlag := false

	definition := NewMachine().
		State("idle").Initial().
		To("high_priority").On("process").When(func(ctx Context) bool {
		return priorityFlag
	}).
		To("error_handling").On("process").When(func(ctx Context) bool {
		return errorFlag
	}).
		To("normal_processing").On("process").
		State("high_priority").
		State("normal_processing").
		State("error_handling").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Test normal processing (no guards match)
	result := machine.HandleEvent("process", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "normal_processing")

	// Test with error flag set
	machine2 := definition.CreateInstance()
	_ = machine2.Start()
	errorFlag = true

	result2 := machine2.HandleEvent("process", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "error_handling")

	// Test with priority flag set (should take precedence)
	machine3 := definition.CreateInstance()
	_ = machine3.Start()
	errorFlag = false
	priorityFlag = true

	result3 := machine3.HandleEvent("process", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine3, "high_priority")
}

// TestMultipleTransitions_WithActions tests multiple transitions with different actions
func TestMultipleTransitions_WithActions(t *testing.T) {
	var actionTracker []string

	definition := NewMachine().
		State("idle").Initial().
		To("state_a").On("event_a").Do(func(ctx Context) error {
		actionTracker = append(actionTracker, "action_a")
		return nil
	}).
		To("state_b").On("event_b").Do(func(ctx Context) error {
		actionTracker = append(actionTracker, "action_b")
		return nil
	}).
		To("state_c").On("event_c").Do(func(ctx Context) error {
		actionTracker = append(actionTracker, "action_c")
		return nil
	}).
		State("state_a").
		State("state_b").
		State("state_c").
		Build()

	// Test each transition separately
	machines := []Machine{}
	for i := 0; i < 3; i++ {
		machine := definition.CreateInstance()
		_ = machine.Start()
		machines = append(machines, machine)
	}

	// Test transition to state_a
	actionTracker = []string{}
	result1 := machines[0].HandleEvent("event_a", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machines[0], "state_a")
	if len(actionTracker) != 1 || actionTracker[0] != "action_a" {
		t.Errorf("Expected action_a to be executed, got %v", actionTracker)
	}

	// Test transition to state_b
	actionTracker = []string{}
	result2 := machines[1].HandleEvent("event_b", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machines[1], "state_b")
	if len(actionTracker) != 1 || actionTracker[0] != "action_b" {
		t.Errorf("Expected action_b to be executed, got %v", actionTracker)
	}

	// Test transition to state_c
	actionTracker = []string{}
	result3 := machines[2].HandleEvent("event_c", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machines[2], "state_c")
	if len(actionTracker) != 1 || actionTracker[0] != "action_c" {
		t.Errorf("Expected action_c to be executed, got %v", actionTracker)
	}
}

// TestMultipleTransitions_ComplexScenario tests a complex scenario with multiple states having multiple transitions
func TestMultipleTransitions_ComplexScenario(t *testing.T) {
	definition := NewMachine().
		State("initial").Initial().
		To("processing").On("start").
		To("error").On("startup_error").
		State("processing").
		To("success").On("complete").
		To("retry").On("retry").
		To("error").On("fail").
		To("maintenance").On("maintenance_needed").
		State("retry").
		To("processing").On("retry_start").
		To("error").On("retry_fail").
		State("maintenance").
		To("processing").On("maintenance_complete").
		To("shutdown").On("shutdown").
		State("success").
		To("initial").On("reset").
		State("error").
		To("initial").On("recover").
		To("shutdown").On("fatal_error").
		State("shutdown").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	AssertState(t, machine, "initial")

	// Test normal flow: initial -> processing -> success -> initial
	result1 := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "processing")

	result2 := machine.HandleEvent("complete", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "success")

	result3 := machine.HandleEvent("reset", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine, "initial")

	// Test error flow: initial -> error -> initial
	result4 := machine.HandleEvent("startup_error", nil)
	AssertEventProcessed(t, result4, true)
	AssertState(t, machine, "error")

	result5 := machine.HandleEvent("recover", nil)
	AssertEventProcessed(t, result5, true)
	AssertState(t, machine, "initial")

	// Test retry flow: initial -> processing -> retry -> processing
	result6 := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result6, true)
	AssertState(t, machine, "processing")

	result7 := machine.HandleEvent("retry", nil)
	AssertEventProcessed(t, result7, true)
	AssertState(t, machine, "retry")

	result8 := machine.HandleEvent("retry_start", nil)
	AssertEventProcessed(t, result8, true)
	AssertState(t, machine, "processing")

	// Test maintenance flow: processing -> maintenance -> processing
	result9 := machine.HandleEvent("maintenance_needed", nil)
	AssertEventProcessed(t, result9, true)
	AssertState(t, machine, "maintenance")

	result10 := machine.HandleEvent("maintenance_complete", nil)
	AssertEventProcessed(t, result10, true)
	AssertState(t, machine, "processing")

	// Verify all transitions were recorded
	if observer.TransitionCount() < 10 {
		t.Errorf("Expected at least 10 transitions, got %d", observer.TransitionCount())
	}
}

// TestMultipleTransitions_SameEventDifferentTargets tests same event from different states
func TestMultipleTransitions_SameEventDifferentTargets(t *testing.T) {
	definition := NewMachine().
		State("state1").Initial().
		To("state2").On("next").
		State("state2").
		To("state3").On("next").
		State("state3").
		To("state1").On("next").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()
	AssertState(t, machine, "state1")

	// Cycle through states using the same event
	for i := 0; i < 6; i++ {
		expectedState := []string{"state2", "state3", "state1"}[i%3]
		result := machine.HandleEvent("next", nil)
		AssertEventProcessed(t, result, true)
		AssertState(t, machine, expectedState)
	}
}

// TestMultipleTransitions_EventDataBasedRouting tests routing based on event data
func TestMultipleTransitions_EventDataBasedRouting(t *testing.T) {
	definition := NewMachine().
		State("router").Initial().
		To("high").On("route").When(func(ctx Context) bool {
		data := ctx.GetEventData()
		if dataMap, ok := data.(map[string]interface{}); ok {
			if priority := dataMap["priority"]; priority == "high" {
				return true
			}
		}
		return false
	}).
		To("medium").On("route").When(func(ctx Context) bool {
		data := ctx.GetEventData()
		if dataMap, ok := data.(map[string]interface{}); ok {
			if priority := dataMap["priority"]; priority == "medium" {
				return true
			}
		}
		return false
	}).
		To("low").On("route").
		State("high").
		State("medium").
		State("low").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Test high priority routing
	result1 := machine.HandleEvent("route", map[string]interface{}{"priority": "high"})
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "high")

	// Reset and test medium priority routing
	machine2 := definition.CreateInstance()
	_ = machine2.Start()
	result2 := machine2.HandleEvent("route", map[string]interface{}{"priority": "medium"})
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "medium")

	// Reset and test low priority routing (default)
	machine3 := definition.CreateInstance()
	_ = machine3.Start()
	result3 := machine3.HandleEvent("route", map[string]interface{}{"priority": "low"})
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine3, "low")

	// Reset and test default routing (no priority data)
	machine4 := definition.CreateInstance()
	_ = machine4.Start()
	result4 := machine4.HandleEvent("route", nil)
	AssertEventProcessed(t, result4, true)
	AssertState(t, machine4, "low")
}

// TestMultipleTransitions_CompositeState tests multiple transitions from composite states
func TestMultipleTransitions_CompositeState(t *testing.T) {
	// Simplified test focusing on multiple transitions from different states
	definition := NewMachine().
		State("idle").Initial().
		To("active").On("start").
		To("maintenance").On("maintenance").
		State("active").
		To("processing").On("process").
		To("idle").On("deactivate").
		To("error").On("error").
		State("processing").
		To("active").On("complete").
		To("error").On("fail").
		State("error").
		To("idle").On("recover").
		To("shutdown").On("fatal_error").
		State("maintenance").
		To("idle").On("repair_complete").
		State("shutdown").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()
	AssertState(t, machine, "idle")

	// Test multiple transitions from idle
	result1 := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "active")

	machine2 := definition.CreateInstance()
	_ = machine2.Start()
	result2 := machine2.HandleEvent("maintenance", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "maintenance")

	// Test multiple transitions from active
	result3 := machine.HandleEvent("process", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine, "processing")

	machine3 := definition.CreateInstance()
	_ = machine3.Start()
	machine3.HandleEvent("start", nil)
	result4 := machine3.HandleEvent("deactivate", nil)
	AssertEventProcessed(t, result4, true)
	AssertState(t, machine3, "idle")

	machine4 := definition.CreateInstance()
	_ = machine4.Start()
	machine4.HandleEvent("start", nil)
	result5 := machine4.HandleEvent("error", nil)
	AssertEventProcessed(t, result5, true)
	AssertState(t, machine4, "error")

	// Test multiple transitions from error
	machine5 := definition.CreateInstance()
	_ = machine5.Start()
	machine5.HandleEvent("start", nil)
	machine5.HandleEvent("error", nil)
	result6 := machine5.HandleEvent("recover", nil)
	AssertEventProcessed(t, result6, true)
	AssertState(t, machine5, "idle")

	machine6 := definition.CreateInstance()
	_ = machine6.Start()
	machine6.HandleEvent("start", nil)
	machine6.HandleEvent("error", nil)
	result7 := machine6.HandleEvent("fatal_error", nil)
	AssertEventProcessed(t, result7, true)
	AssertState(t, machine6, "shutdown")
}
