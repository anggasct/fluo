// Package main demonstrates a payment processing system using Fluo state machine.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

// PaymentData represents payment information
type PaymentData struct {
	Amount       float64
	Currency     string
	PaymentID    string
	CustomerID   string
	Method       string
	ErrorMessage string
}

func main() {
	// Create a payment processing state machine
	builder := fluo.NewStateMachineBuilder("PaymentProcessor")

	// Define states with entry actions
	builder.WithState("Initiated").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Payment initiated, validating details")
			return nil
		})

	builder.WithState("Validating").
		WithEntryAction(func(ctx *fluo.Context) error {
			paymentData := ctx.GetData("payment").(*PaymentData)
			fmt.Printf("Validating payment %s for %.2f %s\n",
				paymentData.PaymentID, paymentData.Amount, paymentData.Currency)
			return nil
		})

	builder.WithState("Processing").
		WithEntryAction(func(ctx *fluo.Context) error {
			paymentData := ctx.GetData("payment").(*PaymentData)
			fmt.Printf("Processing payment via %s\n", paymentData.Method)
			return nil
		})

	builder.WithState("Authorized").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Payment authorized successfully")
			return nil
		})

	builder.WithState("Completed").AsFinal().
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Payment completed and recorded")
			return nil
		})

	builder.WithState("Failed").AsFinal().
		WithEntryAction(func(ctx *fluo.Context) error {
			paymentData := ctx.GetData("payment").(*PaymentData)
			fmt.Printf("Payment failed: %s\n", paymentData.ErrorMessage)
			return nil
		})

	// Add transitions with guards and actions
	builder.WithTransition("Initiated", "Validating", "SUBMIT").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Println("Submitting payment for validation")
			return nil
		})

	// Validation can succeed or fail
	builder.WithTransition("Validating", "Processing", "VALIDATE").
		WithGuard(func(ctx *fluo.Context) bool {
			paymentData := ctx.GetData("payment").(*PaymentData)
			// Simulate validation logic (e.g., amount > 0)
			return paymentData.Amount > 0
		})

	builder.WithTransition("Validating", "Failed", "VALIDATE").
		WithGuard(func(ctx *fluo.Context) bool {
			paymentData := ctx.GetData("payment").(*PaymentData)
			// Failed validation
			isValid := paymentData.Amount > 0
			if !isValid {
				paymentData.ErrorMessage = "Invalid payment amount"
			}
			return !isValid
		})

	// Processing can succeed or fail
	builder.WithTransition("Processing", "Authorized", "PROCESS").
		WithGuard(func(ctx *fluo.Context) bool {
			paymentData := ctx.GetData("payment").(*PaymentData)
			// Simulate processing success (card payments might fail)
			return paymentData.Method != "card" || time.Now().Nanosecond()%2 == 0
		})

	builder.WithTransition("Processing", "Failed", "PROCESS").
		WithGuard(func(ctx *fluo.Context) bool {
			paymentData := ctx.GetData("payment").(*PaymentData)
			// Simulate processing failure for some card payments
			failed := paymentData.Method == "card" && time.Now().Nanosecond()%2 != 0
			if failed {
				paymentData.ErrorMessage = "Payment declined by provider"
			}
			return failed
		})

	// Final settlement
	builder.WithTransition("Authorized", "Completed", "SETTLE")

	// Set initial state and add observer
	builder.WithInitialState("Initiated")
	builder.WithObserver(fluo.NewLoggingObserver())

	// Build and start the state machine
	paymentProcessor, err := builder.Build()
	if err != nil {
		fmt.Printf("Error building payment processor: %v\n", err)
		return
	}

	ctx := context.Background()
	if err := paymentProcessor.Start(ctx); err != nil {
		fmt.Printf("Error starting payment processor: %v\n", err)
		return
	}

	// Run simulation for different payment scenarios
	fmt.Println("\n=== Payment Processing Simulation ===\n")

	// Successful bank payment
	processPayment(ctx, paymentProcessor, &PaymentData{
		Amount:     125.50,
		Currency:   "USD",
		PaymentID:  "PAY-123456",
		CustomerID: "CUST-789",
		Method:     "bank_transfer",
	})

	// Invalid payment amount
	processPayment(ctx, paymentProcessor, &PaymentData{
		Amount:     0,
		Currency:   "USD",
		PaymentID:  "PAY-789012",
		CustomerID: "CUST-456",
		Method:     "wallet",
	})

	// Card payment (randomly succeeds or fails)
	processPayment(ctx, paymentProcessor, &PaymentData{
		Amount:     75.25,
		Currency:   "EUR",
		PaymentID:  "PAY-345678",
		CustomerID: "CUST-123",
		Method:     "card",
	})

	fmt.Println("\n=== Simulation Completed ===")
}

// processPayment runs a payment through the payment processing state machine
func processPayment(ctx context.Context, processor *fluo.StateMachine, payment *PaymentData) {
	fmt.Printf("\n--- Processing Payment %s ---\n", payment.PaymentID)

	// Reset state machine to initial state for a new payment
	processor.Reset(ctx)

	// Store payment data in context
	processor.Context().SetData("payment", payment)

	// Submit payment for validation
	processor.HandleEvent(ctx, fluo.NewEvent("SUBMIT"))

	// Run validation
	processor.HandleEvent(ctx, fluo.NewEvent("VALIDATE"))

	// If still active, process the payment
	if !processor.IsCompleted() {
		processor.HandleEvent(ctx, fluo.NewEvent("PROCESS"))
	}

	// If authorized, settle the payment
	if processor.CurrentState().Name() == "Authorized" {
		processor.HandleEvent(ctx, fluo.NewEvent("SETTLE"))
	}

	fmt.Printf("Final state: %s\n", processor.CurrentState().Name())
	fmt.Println("----------------------------")
}
