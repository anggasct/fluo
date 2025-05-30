// Package main demonstrates a modern traffic light state machine using the Fluo library.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

func main() {
	// Create a traffic light state machine
	builder := fluo.NewStateMachineBuilder("TrafficLight")

	// Define states with entry actions
	builder.WithState("Red").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Light turned RED - Stop")
			return nil
		})

	builder.WithState("Yellow").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Light turned YELLOW - Prepare to stop")
			return nil
		})

	builder.WithState("Green").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Light turned GREEN - Go")
			return nil
		})

	// Define transitions
	builder.WithTransition("Red", "Green", "NEXT").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Changing from Red to Green")
			return nil
		})

	builder.WithTransition("Green", "Yellow", "NEXT").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Changing from Green to Yellow")
			return nil
		})

	builder.WithTransition("Yellow", "Red", "NEXT").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Changing from Yellow to Red")
			return nil
		})

	// Set initial state and add logging
	builder.WithInitialState("Red")
	builder.WithObserver(fluo.NewLoggingObserver())

	// Build state machine
	trafficLight, err := builder.Build()
	if err != nil {
		fmt.Printf("Error building state machine: %v\n", err)
		return
	}

	// Start the state machine
	ctx := context.Background()
	if err := trafficLight.Start(ctx); err != nil {
		fmt.Printf("Error starting state machine: %v\n", err)
		return
	}

	// Run through a complete traffic light cycle
	fmt.Println("\n=== Traffic Light Simulation ===\n")
	fmt.Printf("Initial state: %s\n", trafficLight.CurrentState().Name())

	// Complete cycle of traffic light
	for i := 0; i < 4; i++ {
		time.Sleep(1 * time.Second)
		nextEvent := fluo.NewEvent("NEXT")

		if err := trafficLight.HandleEvent(ctx, nextEvent); err != nil {
			fmt.Printf("Error handling event: %v\n", err)
			continue
		}

		fmt.Printf("Current state: %s\n", trafficLight.CurrentState().Name())
	}

	trafficLight.Stop(ctx)
	fmt.Println("\n=== Simulation Completed ===")
}
