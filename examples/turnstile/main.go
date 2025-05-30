// Package main demonstrates a simple turnstile state machine using the Fluo library.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

func main() {
	// Create a state machine for a turnstile
	builder := fluo.NewStateMachineBuilder("Turnstile")

	// Define states
	builder.WithState("Locked")
	builder.WithState("Unlocked")

	// Define transitions with events and actions
	builder.WithTransition("Locked", "Unlocked", "COIN").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Accepting coin and unlocking turnstile")
			return nil
		})

	builder.WithTransition("Unlocked", "Locked", "PUSH").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Person passed through, locking turnstile")
			return nil
		})

	// Define self-transitions for handling invalid operations
	builder.WithTransition("Locked", "Locked", "PUSH").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Turnstile is locked. Insert coin first!")
			return nil
		})

	builder.WithTransition("Unlocked", "Unlocked", "COIN").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Turnstile is already unlocked. Push to go through!")
			return nil
		})

	// Set initial state and add logging
	builder.WithInitialState("Locked")
	builder.WithObserver(fluo.NewLoggingObserver())

	// Build and start the state machine
	turnstile, err := builder.Build()
	if err != nil {
		fmt.Printf("Failed to build state machine: %v\n", err)
		return
	}

	ctx := context.Background()
	if err := turnstile.Start(ctx); err != nil {
		fmt.Printf("Failed to start state machine: %v\n", err)
		return
	}

	// Simulate a sequence of events
	events := []string{
		"PUSH", // Should stay locked with message
		"COIN", // Should unlock
		"COIN", // Should stay unlocked with message
		"PUSH", // Should lock again
	}

	// Process events with slight delay for readability
	fmt.Println("Starting turnstile simulation...")
	for i, eventName := range events {
		fmt.Printf("\n--- Event %d: %s ---\n", i+1, eventName)

		// Create and handle the event
		event := fluo.NewEvent(eventName)
		if err := turnstile.HandleEvent(ctx, event); err != nil {
			fmt.Printf("Error handling event: %v\n", err)
		}

		// Show current state after event
		fmt.Printf("Current state: %s\n", turnstile.CurrentState().Name())

		// Wait a moment for readability
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\nTurnstile simulation completed.")
}
