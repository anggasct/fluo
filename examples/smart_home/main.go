// Package main demonstrates a smart home system with parallel regions using Fluo.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

func main() {
	// Create a smart home state machine with parallel regions for different systems
	builder := fluo.NewStateMachineBuilder("SmartHome")

	// Define main regions for the parallel state
	builder.WithParallelState("Active")

	// Add lighting control region
	lightingRegion := builder.AddParallelRegion("Active", "Lighting")
	lightingRegion.WithState("Off").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Lights turned off")
			return nil
		})
	lightingRegion.WithState("On").
		WithEntryAction(func(ctx *fluo.Context) error {
			brightness := 100
			if b := ctx.GetData("brightness"); b != nil {
				brightness = b.(int)
			}
			fmt.Printf("Lights turned on (brightness: %d%%)\n", brightness)
			return nil
		})
	lightingRegion.WithInitialState("Off")
	// Add transitions for lighting
	lightingRegion.WithTransition("Off", "On", "LIGHTS_ON").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Turning lights on...")
			return nil
		})
	lightingRegion.WithTransition("On", "Off", "LIGHTS_OFF")

	// Add heating system region
	heatingRegion := builder.AddParallelRegion("Active", "Heating")
	heatingRegion.WithState("Off").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Heating system is off")
			return nil
		})
	heatingRegion.WithState("On").
		WithEntryAction(func(ctx *fluo.Context) error {
			temperature := 21
			if t := ctx.GetData("temperature"); t != nil {
				temperature = t.(int)
			}
			fmt.Printf("Heating system on (target: %d°C)\n", temperature)
			return nil
		})
	heatingRegion.WithInitialState("Off")
	// Add transitions for heating
	heatingRegion.WithTransition("Off", "On", "HEAT_ON")
	heatingRegion.WithTransition("On", "Off", "HEAT_OFF")

	// Add security system region
	securityRegion := builder.AddParallelRegion("Active", "Security")
	securityRegion.WithState("Disarmed").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Security system is disarmed")
			return nil
		})
	securityRegion.WithInitialState("Disarmed")
	securityRegion.WithState("Armed").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Security system is armed")
			return nil
		})
	securityRegion.WithState("Alarming").
		WithEntryAction(func(ctx *fluo.Context) error {
			reason := "unknown trigger"
			if r := ctx.GetData("alarmReason"); r != nil {
				reason = r.(string)
			}
			fmt.Printf("⚠️ ALARM TRIGGERED: %s!\n", reason)
			return nil
		})
	// Add transitions for security
	securityRegion.WithTransition("Disarmed", "Armed", "ARM_SECURITY")
	securityRegion.WithTransition("Armed", "Disarmed", "DISARM_SECURITY")
	securityRegion.WithTransition("Armed", "Alarming", "TRIGGER_ALARM")
	securityRegion.WithTransition("Alarming", "Armed", "RESET_ALARM")

	// Set initial state and add logging
	builder.WithInitialState("Active")
	builder.WithObserver(fluo.NewLoggingObserver())

	// Build and start the state machine
	smartHome, err := builder.Build()
	if err != nil {
		fmt.Printf("Error building smart home system: %v\n", err)
		return
	}

	ctx := context.Background()
	if err := smartHome.Start(ctx); err != nil {
		fmt.Printf("Error starting smart home system: %v\n", err)
		return
	}

	// Run a simulation of the smart home system
	fmt.Println("\n=== Smart Home System Simulation ===\n")

	fmt.Println("1. Turn on lighting")
	smartHome.Context().SetData("brightness", 80)
	smartHome.HandleEvent(ctx, fluo.NewEvent("LIGHTS_ON"))
	time.Sleep(1 * time.Second)

	fmt.Println("\n2. Turn on heating")
	smartHome.Context().SetData("temperature", 22)
	smartHome.HandleEvent(ctx, fluo.NewEvent("HEAT_ON"))
	time.Sleep(1 * time.Second)

	fmt.Println("\n3. Arm security system")
	smartHome.HandleEvent(ctx, fluo.NewEvent("ARM_SECURITY"))
	time.Sleep(1 * time.Second)

	fmt.Println("\n4. Trigger alarm")
	smartHome.Context().SetData("alarmReason", "motion detected")
	smartHome.HandleEvent(ctx, fluo.NewEvent("TRIGGER_ALARM"))
	time.Sleep(1 * time.Second)

	fmt.Println("\n5. Reset alarm")
	smartHome.HandleEvent(ctx, fluo.NewEvent("RESET_ALARM"))
	time.Sleep(1 * time.Second)

	fmt.Println("\n6. Night mode: turn off lights and heating")
	smartHome.HandleEvent(ctx, fluo.NewEvent("LIGHTS_OFF"))
	smartHome.HandleEvent(ctx, fluo.NewEvent("HEAT_OFF"))
	time.Sleep(1 * time.Second)

	// Clean shutdown
	smartHome.Stop(ctx)
	fmt.Println("\n=== Simulation Completed ===")
}
