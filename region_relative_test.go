package fluo

import (
	"testing"
)

func TestRegionRelativeTransitions(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().To("parallel").On("begin")

	parallel := builder.ParallelState("parallel")

	payment := parallel.Region("payment")
	payment.State("pending").Initial().
		OnEntry(func(ctx Context) error { return nil }).
		To("processing").On("process_payment")
	payment.State("processing").
		OnEntry(func(ctx Context) error { return nil }).
		To("complete").On("payment_done")
	payment.State("complete").
		OnEntry(func(ctx Context) error { return nil })

	inventory := parallel.Region("inventory")
	inventory.State("checking").Initial().
		OnEntry(func(ctx Context) error { return nil }).
		To("reserved").On("reserve_items")
	inventory.State("reserved").
		OnEntry(func(ctx Context) error { return nil })

	parallel.End()

	definition := builder.Build()
	machine := definition.CreateInstance()

	err := machine.Start()
	if err != nil {
		t.Fatalf("Failed to start machine: %v", err)
	}

	result := machine.HandleEvent("begin", nil)
	if !result.Processed {
		t.Error("Expected 'begin' event to be processed")
	}

	result = machine.HandleEvent("process_payment", nil)
	if !result.Processed {
		t.Error("Expected 'process_payment' event to be processed")
	}

	if !machine.IsStateActive("parallel.payment.processing") {
		t.Error("Expected parallel.payment.processing to be active")
	}

	result = machine.HandleEvent("payment_done", nil)
	if !result.Processed {
		t.Error("Expected 'payment_done' event to be processed")
	}

	if !machine.IsStateActive("parallel.payment.complete") {
		t.Error("Expected parallel.payment.complete to be active")
	}

	result = machine.HandleEvent("reserve_items", nil)
	if !result.Processed {
		t.Error("Expected 'reserve_items' event to be processed")
	}

	if !machine.IsStateActive("parallel.inventory.reserved") {
		t.Error("Expected parallel.inventory.reserved to be active")
	}
}

func TestCompositeStateRelativeTransitions(t *testing.T) {

	builder := NewMachine()

	builder.State("idle").Initial().To("active").On("activate")

	active := builder.CompositeState("active")
	active.State("working").Initial().
		To("paused").On("pause")
	active.State("paused").
		To("working").On("resume")
	active.End()

	definition := builder.Build()
	machine := definition.CreateInstance()

	err := machine.Start()
	if err != nil {
		t.Fatalf("Failed to start machine: %v", err)
	}

	if machine.CurrentState() != "idle" {
		t.Errorf("Expected initial state to be idle, got %s", machine.CurrentState())
	}

	result := machine.HandleEvent("activate", nil)
	if !result.Processed {
		t.Errorf("Expected 'activate' event to be processed. Rejection: %s, Error: %v", result.RejectionReason, result.Error)
	}

	if machine.CurrentState() != "active.working" {
		t.Errorf("Expected current state to be active.working, got %s", machine.CurrentState())
	}

	result = machine.HandleEvent("pause", nil)
	if !result.Processed {
		t.Error("Expected 'pause' event to be processed")
	}

	if machine.CurrentState() != "active.paused" {
		t.Errorf("Expected current state to be active.paused, got %s", machine.CurrentState())
	}

	result = machine.HandleEvent("resume", nil)
	if !result.Processed {
		t.Error("Expected 'resume' event to be processed")
	}

	if machine.CurrentState() != "active.working" {
		t.Errorf("Expected current state to be active.working, got %s", machine.CurrentState())
	}
}
