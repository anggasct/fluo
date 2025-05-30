// Package main demonstrates parallel regions in state machines using the Fluo library.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

// Parallel Regions Example
// This example demonstrates how to use parallel regions to model a system
// where multiple components operate concurrently, such as a smart home
// with heating, lighting, and security systems running in parallel.

func main() {
	// Create the main state machine builder
	builder := fluo.NewStateMachineBuilder("SmartHome")

	// Define top-level states
	builder.WithState("Off").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Smart home system is OFF")
			return nil
		})

	// Define a parallel state for when the system is on
	builder.WithParallelState("On").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Smart home system is ON - all subsystems starting")
			return nil
		})

	// Define the main transition
	builder.WithTransition("Off", "On", "POWER_ON")
	builder.WithTransition("On", "Off", "POWER_OFF")

	// Set initial state
	builder.WithInitialState("Off")

	// Configure the first parallel region: Heating System
	heatingRegion := builder.AddParallelRegion("On", "Heating")

	// States for heating region
	heatingRegion.WithState("Idle").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Heating system: IDLE")
			return nil
		})

	heatingRegion.WithState("Heating").
		WithEntryAction(func(ctx *fluo.Context) error {
			temperature := ctx.GetData("temperature")
			fmt.Printf("Heating system: HEATING - Target temp: %v°C\n", temperature)
			return nil
		})

	heatingRegion.WithState("Cooling").
		WithEntryAction(func(ctx *fluo.Context) error {
			temperature := ctx.GetData("temperature")
			fmt.Printf("Heating system: COOLING - Target temp: %v°C\n", temperature)
			return nil
		})

	// Transitions for heating region
	heatingRegion.WithTransition("Idle", "Heating", "TEMP_CHANGE").
		WithGuard(func(ctx *fluo.Context) bool {
			currentTemp := ctx.Event.Data.(float64)
			targetTemp := 21.0 // Default target temperature
			if t := ctx.GetData("temperature"); t != nil {
				targetTemp = t.(float64)
			}
			return currentTemp < targetTemp-1.0
		})

	heatingRegion.WithTransition("Idle", "Cooling", "TEMP_CHANGE").
		WithGuard(func(ctx *fluo.Context) bool {
			currentTemp := ctx.Event.Data.(float64)
			targetTemp := 21.0 // Default target temperature
			if t := ctx.GetData("temperature"); t != nil {
				targetTemp = t.(float64)
			}
			return currentTemp > targetTemp+1.0
		})

	heatingRegion.WithTransition("Heating", "Idle", "TEMP_CHANGE").
		WithGuard(func(ctx *fluo.Context) bool {
			currentTemp := ctx.Event.Data.(float64)
			targetTemp := 21.0 // Default target temperature
			if t := ctx.GetData("temperature"); t != nil {
				targetTemp = t.(float64)
			}
			return currentTemp >= targetTemp
		})

	heatingRegion.WithTransition("Cooling", "Idle", "TEMP_CHANGE").
		WithGuard(func(ctx *fluo.Context) bool {
			currentTemp := ctx.Event.Data.(float64)
			targetTemp := 21.0 // Default target temperature
			if t := ctx.GetData("temperature"); t != nil {
				targetTemp = t.(float64)
			}
			return currentTemp <= targetTemp
		})

	// Set initial state for heating region
	heatingRegion.WithInitialState("Idle")

	// Configure the second parallel region: Lighting System
	lightingRegion := builder.AddParallelRegion("On", "Lighting")

	// States for lighting region
	lightingRegion.WithState("Off").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Lighting system: OFF")
			return nil
		})

	lightingRegion.WithState("On").
		WithEntryAction(func(ctx *fluo.Context) error {
			brightness := "50%"
			if b := ctx.GetData("brightness"); b != nil {
				brightness = b.(string)
			}
			fmt.Printf("Lighting system: ON - Brightness: %s\n", brightness)
			return nil
		})

	lightingRegion.WithState("Auto").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Lighting system: AUTO - adjusting based on time and motion")
			return nil
		})

	// Transitions for lighting region
	lightingRegion.WithTransition("Off", "On", "LIGHTS_ON")
	lightingRegion.WithTransition("On", "Off", "LIGHTS_OFF")
	lightingRegion.WithTransition("Off", "Auto", "AUTO_MODE")
	lightingRegion.WithTransition("On", "Auto", "AUTO_MODE")
	lightingRegion.WithTransition("Auto", "On", "LIGHTS_ON")
	lightingRegion.WithTransition("Auto", "Off", "LIGHTS_OFF")

	// Set initial state for lighting region
	lightingRegion.WithInitialState("Off")

	// Configure the third parallel region: Security System
	securityRegion := builder.AddParallelRegion("On", "Security")

	// States for security region
	securityRegion.WithState("Disarmed").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Security system: DISARMED")
			return nil
		})

	securityRegion.WithState("Armed").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Security system: ARMED")
			return nil
		})

	securityRegion.WithState("Alarming").
		WithEntryAction(func(ctx *fluo.Context) error {
			reason := "unknown trigger"
			if r := ctx.GetData("alarmTrigger"); r != nil {
				reason = r.(string)
			}
			fmt.Printf("Security system: !!! ALARM !!! - Reason: %s\n", reason)
			return nil
		})

	// Transitions for security region
	securityRegion.WithTransition("Disarmed", "Armed", "ARM")
	securityRegion.WithTransition("Armed", "Disarmed", "DISARM")
	securityRegion.WithTransition("Armed", "Alarming", "TRIGGER_ALARM")
	securityRegion.WithTransition("Alarming", "Armed", "RESET_ALARM")
	securityRegion.WithTransition("Alarming", "Disarmed", "DISARM")

	// Set initial state for security region
	securityRegion.WithInitialState("Disarmed")

	// Add a logging observer
	builder.WithObserver(fluo.NewLoggingObserver())

	// Build the state machine
	sm, err := builder.Build()
	if err != nil {
		fmt.Printf("Error building state machine: %v\n", err)
		return
	}

	// Start the state machine
	ctx := context.Background()
	if err := sm.Start(ctx); err != nil {
		fmt.Printf("Error starting state machine: %v\n", err)
		return
	}

	fmt.Println("\n=== Smart Home Simulation Started ===\n")

	// Power on the system
	processEvent(ctx, sm, "POWER_ON", nil)

	// Set the target temperature
	sm.Context.SetData("temperature", 22.0)

	// Simulate temperature changes
	processEvent(ctx, sm, "TEMP_CHANGE", 18.0)
	processEvent(ctx, sm, "TEMP_CHANGE", 21.5)
	processEvent(ctx, sm, "TEMP_CHANGE", 24.0)
	processEvent(ctx, sm, "TEMP_CHANGE", 22.0)

	// Turn on lights and set brightness
	processEvent(ctx, sm, "LIGHTS_ON", nil)
	sm.Context.SetData("brightness", "75%")

	// Switch lights to auto mode
	processEvent(ctx, sm, "AUTO_MODE", nil)

	// Arm the security system
	processEvent(ctx, sm, "ARM", nil)

	// Trigger alarm
	sm.Context.SetData("alarmTrigger", "motion detected in living room")
	processEvent(ctx, sm, "TRIGGER_ALARM", nil)

	// Reset alarm
	processEvent(ctx, sm, "RESET_ALARM", nil)

	// Disarm security
	processEvent(ctx, sm, "DISARM", nil)

	// Turn off the whole system
	processEvent(ctx, sm, "POWER_OFF", nil)

	fmt.Println("\n=== Smart Home Simulation Completed ===")
}

// Helper function to process events with a small delay
func processEvent(ctx context.Context, sm *fluo.StateMachine, eventName string, eventData interface{}) {
	fmt.Printf("\n--- Event: %s %v ---\n", eventName, eventData)
	var event *fluo.Event
	if eventData != nil {
		event = fluo.NewEventWithData(eventName, eventData)
	} else {
		event = fluo.NewEvent(eventName)
	}

	if err := sm.HandleEvent(ctx, event); err != nil {
		fmt.Printf("Error handling event: %v\n", err)
	}

	time.Sleep(time.Millisecond * 500) // Short delay for readability
}
