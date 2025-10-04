package fluo

import (
	"testing"
)

// Test to reproduce the multiple transitions chaining issue
func TestMultipleTransitionsChaining(t *testing.T) {
	// This should work but currently doesn't
	definition := NewMachine().
		State("maintenance_mode").Initial().
		OnEntry(func(ctx Context) error {
			return nil
		}).
		To("normal_operation").On("maintenance_complete").
		To("off").On("power_off").
		State("normal_operation").
		State("off").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Force transition to maintenance_mode to test the transitions
	machine.SetState("maintenance_mode")

	// Test first transition
	result := machine.HandleEvent("maintenance_complete", nil)
	if !result.Processed {
		t.Error("Expected maintenance_complete event to be processed")
	}
	if machine.CurrentState() != "normal_operation" {
		t.Errorf("Expected to be in normal_operation, got %s", machine.CurrentState())
	}

	// Reset and test second transition
	machine2 := definition.CreateInstance()
	_ = machine2.Start()
	machine2.SetState("maintenance_mode")

	result2 := machine2.HandleEvent("power_off", nil)
	if !result2.Processed {
		t.Error("Expected power_off event to be processed")
	}
	if machine2.CurrentState() != "off" {
		t.Errorf("Expected to be in off, got %s", machine2.CurrentState())
	}
}

// Test to specifically check if both transitions are properly saved
func TestMultipleTransitionsAreSaved(t *testing.T) {
	definition := NewMachine().
		State("maintenance_mode").Initial().
		To("normal_operation").On("maintenance_complete").
		To("off").On("power_off").
		State("normal_operation").
		State("off").
		Build()

	// Check that both transitions are in the definition
	transitions := definition.GetTransitions()
	maintenanceTransitions := transitions["maintenance_mode"]

	if len(maintenanceTransitions) != 2 {
		t.Errorf("Expected 2 transitions from maintenance_mode, got %d", len(maintenanceTransitions))
		for _, transition := range maintenanceTransitions {
			t.Logf("Transition: %s -> %s on %s", transition.SourceState, transition.TargetState, transition.EventName)
		}
	}

	// Verify both events are present
	hasMaintenanceComplete := false
	hasPowerOff := false
	for _, transition := range maintenanceTransitions {
		if transition.EventName == "maintenance_complete" && transition.TargetState == "normal_operation" {
			hasMaintenanceComplete = true
		}
		if transition.EventName == "power_off" && transition.TargetState == "off" {
			hasPowerOff = true
		}
	}

	if !hasMaintenanceComplete {
		t.Error("Missing transition for maintenance_complete -> normal_operation")
	}

	if !hasPowerOff {
		t.Error("Missing transition for power_off -> off")
	}
}
