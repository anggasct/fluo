// Package main demonstrates a hierarchical state machine for an ATM using the Fluo library.
package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/anggasct/fluo"
)

// ATM State Machine Example
// This example models an ATM with the following states:
// - Idle (waiting for card)
// - Active (card inserted)
//   - Validating (validating PIN)
//   - MainMenu (PIN validated, showing options)
//   - Transaction
//     - Withdrawal
//     - Deposit
//     - Balance
//   - Finalizing (completing transaction)

func main() {
	// Create state machine builder
	builder := fluo.NewStateMachineBuilder("ATM")

	// Define top-level states
	builder.WithState("Idle")
	builder.WithCompositeState("Active") // Active is a composite state
	builder.WithInitialState("Idle")

	// Define sub-states of Active state
	builder.WithChildState("Active", "Validating")
	builder.WithChildState("Active", "MainMenu")
	builder.WithCompositeState("Active.Transaction") // Transaction is a nested composite state
	builder.WithChildState("Active", "Finalizing")
	builder.WithInitialChildState("Active", "Validating")

	// Define sub-states of Transaction
	builder.WithChildState("Active.Transaction", "Withdrawal")
	builder.WithChildState("Active.Transaction", "Deposit")
	builder.WithChildState("Active.Transaction", "Balance")
	builder.WithInitialChildState("Active.Transaction", "Balance")

	// Add entry/exit actions for states
	builder.WithState("Idle").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("ATM waiting for card...")
			return nil
		})

	builder.WithState("Active.Validating").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Please enter PIN...")
			return nil
		})

	builder.WithState("Active.MainMenu").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Main Menu - Select a transaction:")
			fmt.Println("1. Withdrawal")
			fmt.Println("2. Deposit")
			fmt.Println("3. Balance Inquiry")
			fmt.Println("4. Exit")
			return nil
		})

	builder.WithState("Active.Transaction.Withdrawal").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Enter withdrawal amount:")
			return nil
		})

	builder.WithState("Active.Transaction.Deposit").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Enter deposit amount:")
			return nil
		})

	builder.WithState("Active.Transaction.Balance").
		WithEntryAction(func(ctx *fluo.Context) error {
			balance := ctx.GetData("balance")
			if balance == nil {
				balance = 1000.00 // Default balance
			}
			fmt.Printf("Your current balance: $%.2f\n", balance)
			return nil
		})

	builder.WithState("Active.Finalizing").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Processing transaction...")
			return nil
		})

	// Define transitions

	// From Idle to Active
	builder.WithTransition("Idle", "Active", "INSERT_CARD").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Card inserted")
			return nil
		})

	// From Validating to MainMenu
	builder.WithTransition("Active.Validating", "Active.MainMenu", "ENTER_PIN").
		WithGuard(func(ctx *fluo.Context) bool {
			pin := ctx.Event.Data.(string)
			return pin == "1234" // Simple PIN validation
		}).
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("PIN accepted")
			return nil
		})

	// Failed PIN attempt stays in Validating
	builder.WithTransition("Active.Validating", "Active.Validating", "ENTER_PIN").
		WithGuard(func(ctx *fluo.Context) bool {
			pin := ctx.Event.Data.(string)
			return pin != "1234"
		}).
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Invalid PIN, please try again")
			return nil
		})

	// From MainMenu to different transaction types
	builder.WithTransition("Active.MainMenu", "Active.Transaction.Withdrawal", "SELECT_OPTION").
		WithGuard(func(ctx *fluo.Context) bool {
			option := ctx.Event.Data.(string)
			return option == "1"
		})

	builder.WithTransition("Active.MainMenu", "Active.Transaction.Deposit", "SELECT_OPTION").
		WithGuard(func(ctx *fluo.Context) bool {
			option := ctx.Event.Data.(string)
			return option == "2"
		})

	builder.WithTransition("Active.MainMenu", "Active.Transaction.Balance", "SELECT_OPTION").
		WithGuard(func(ctx *fluo.Context) bool {
			option := ctx.Event.Data.(string)
			return option == "3"
		})

	builder.WithTransition("Active.MainMenu", "Idle", "SELECT_OPTION").
		WithGuard(func(ctx *fluo.Context) bool {
			option := ctx.Event.Data.(string)
			return option == "4"
		}).
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Card ejected. Thank you for using our ATM.")
			return nil
		})

	// From transactions to finalizing
	builder.WithTransition("Active.Transaction.Withdrawal", "Active.Finalizing", "ENTER_AMOUNT").
		WithAction(func(ctx *fluo.Context) error {
			amount, _ := strconv.ParseFloat(ctx.Event.Data.(string), 64)
			currentBalance := 1000.0 // Default balance
			if bal := ctx.GetData("balance"); bal != nil {
				currentBalance = bal.(float64)
			}

			if amount > currentBalance {
				fmt.Println("Insufficient funds!")
				return nil
			}

			newBalance := currentBalance - amount
			ctx.SetData("balance", newBalance)
			fmt.Printf("Withdrawing $%.2f. New balance: $%.2f\n", amount, newBalance)
			return nil
		})

	builder.WithTransition("Active.Transaction.Deposit", "Active.Finalizing", "ENTER_AMOUNT").
		WithAction(func(ctx *fluo.Context) error {
			amount, _ := strconv.ParseFloat(ctx.Event.Data.(string), 64)
			currentBalance := 1000.0 // Default balance
			if bal := ctx.GetData("balance"); bal != nil {
				currentBalance = bal.(float64)
			}

			newBalance := currentBalance + amount
			ctx.SetData("balance", newBalance)
			fmt.Printf("Depositing $%.2f. New balance: $%.2f\n", amount, newBalance)
			return nil
		})

	builder.WithTransition("Active.Transaction.Balance", "Active.MainMenu", "CONTINUE").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Returning to main menu...")
			return nil
		})

	// From finalizing back to main menu
	builder.WithTransition("Active.Finalizing", "Active.MainMenu", "COMPLETE").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Transaction complete")
			return nil
		})

	// From any Active state to Idle (card ejection)
	builder.WithTransition("Active", "Idle", "EJECT_CARD").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Card ejected. Thank you for using our ATM.")
			return nil
		})

	// Add a logging observer
	builder.WithObserver(fluo.NewLoggingObserver())

	// Build the state machine
	atm, err := builder.Build()
	if err != nil {
		fmt.Printf("Error building state machine: %v\n", err)
		return
	}

	// Start the machine
	ctx := context.Background()
	if err := atm.Start(ctx); err != nil {
		fmt.Printf("Error starting state machine: %v\n", err)
		return
	}

	fmt.Println("\n=== ATM Simulation Started ===\n")

	// Simulate a sequence of interactions
	simulateATM(ctx, atm)

	fmt.Println("\n=== ATM Simulation Completed ===")
}

func simulateATM(ctx context.Context, atm *fluo.StateMachine) {
	// Define a helper function to process events with delay
	processEvent := func(name string, data interface{}) {
		fmt.Printf("\n--- Event: %s ---\n", name)
		event := fluo.NewEventWithData(name, data)
		if err := atm.HandleEvent(ctx, event); err != nil {
			fmt.Printf("Error handling event: %v\n", err)
		}
		time.Sleep(1 * time.Second) // Pause for readability
	}

	// Insert card
	processEvent("INSERT_CARD", nil)

	// First try wrong PIN
	processEvent("ENTER_PIN", "5678")

	// Then correct PIN
	processEvent("ENTER_PIN", "1234")

	// Check balance first
	processEvent("SELECT_OPTION", "3")
	processEvent("CONTINUE", nil)

	// Make a withdrawal
	processEvent("SELECT_OPTION", "1")
	processEvent("ENTER_AMOUNT", "300.00")
	processEvent("COMPLETE", nil)

	// Make a deposit
	processEvent("SELECT_OPTION", "2")
	processEvent("ENTER_AMOUNT", "500.00")
	processEvent("COMPLETE", nil)

	// Check balance again
	processEvent("SELECT_OPTION", "3")
	processEvent("CONTINUE", nil)

	// Exit
	processEvent("SELECT_OPTION", "4")
}
