// Package main demonstrates a document approval workflow using the Fluo library.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/anggasct/fluo"
)

func main() {
	// Create a document approval workflow
	workflow := fluo.NewWorkflowBuilder("DocumentApproval")

	// Define workflow states with entry actions
	workflow.WithState("Draft").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Document is in draft stage")
			return nil
		})

	workflow.WithState("Submitted").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Document has been submitted for review")
			return nil
		})

	workflow.WithState("InReview").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Printf("Document '%s' is being reviewed by %s\n",
				ctx.GetData("documentName"),
				ctx.GetData("reviewer"))
			return nil
		})

	workflow.WithState("Approved").
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Document has been approved!")
			return nil
		})

	workflow.WithState("Rejected").
		WithEntryAction(func(ctx *fluo.Context) error {
			reason := "unspecified reasons"
			if r := ctx.GetData("rejectReason"); r != nil {
				reason = r.(string)
			}
			fmt.Printf("Document was rejected for: %s\n", reason)
			return nil
		})

	workflow.WithState("Published").AsFinal().
		WithEntryAction(func(ctx *fluo.Context) error {
			fmt.Println("Document has been published and is now available")
			return nil
		})

	// Define a choice junction for review decision outcomes
	workflow.WithChoiceJunction("ReviewOutcome").
		WithTransition("ReviewOutcome", "Approved", "DECISION").
		WithGuard(func(ctx *fluo.Context) bool {
			decision := ctx.Event.Data.(string)
			return decision == "approve"
		}).
		WithTransition("ReviewOutcome", "Rejected", "DECISION").
		WithGuard(func(ctx *fluo.Context) bool {
			decision := ctx.Event.Data.(string)
			return decision == "reject"
		})

	// Define transitions between states
	workflow.WithTransition("Draft", "Submitted", "SUBMIT").
		WithAction(func(ctx *fluo.Context) error {
			fmt.Printf("Document '%s' submitted at %s\n",
				ctx.GetData("documentName"),
				time.Now().Format(time.RFC3339))
			return nil
		})

	workflow.WithTransition("Submitted", "InReview", "ASSIGN_REVIEWER").
		WithAction(func(ctx *fluo.Context) error {
			reviewer := ctx.Event.Data.(string)
			ctx.SetData("reviewer", reviewer)
			fmt.Printf("Assigned reviewer: %s\n", reviewer)
			return nil
		})

	workflow.WithTransition("InReview", "ReviewOutcome", "COMPLETE_REVIEW")
	workflow.WithTransition("Rejected", "Draft", "REVISE")
	workflow.WithTransition("Approved", "Published", "PUBLISH")

	// Configure workflow
	workflow.WithInitialState("Draft")
	workflow.WithObserver(fluo.NewLoggingObserver())

	// Build and start the workflow
	sm, err := workflow.Build()
	if err != nil {
		fmt.Printf("Error building workflow: %v\n", err)
		return
	}

	ctx := context.Background()
	if err = sm.Start(ctx); err != nil {
		fmt.Printf("Error starting workflow: %v\n", err)
		return
	}

	// Set initial document information
	sm.Context().SetData("documentName", "Annual Budget Proposal")

	fmt.Println("\n=== Document Approval Workflow ===\n")

	// Run the workflow simulation with events sequence
	runWorkflowSimulation(ctx, sm)

	fmt.Println("\n=== Workflow Completed ===")
}

// runWorkflowSimulation demonstrates the document approval workflow process
func runWorkflowSimulation(ctx context.Context, sm *fluo.StateMachine) {
	// Initial document submission
	handleEvent(ctx, sm, fluo.NewEvent("SUBMIT"))

	// First review cycle with rejection
	handleEvent(ctx, sm, fluo.NewEventWithData("ASSIGN_REVIEWER", "John Smith"))
	handleEvent(ctx, sm, fluo.NewEvent("COMPLETE_REVIEW"))

	sm.Context().SetData("rejectReason", "Budget numbers need verification")
	handleEvent(ctx, sm, fluo.NewEventWithData("DECISION", "reject"))

	// Document revision and second review cycle
	handleEvent(ctx, sm, fluo.NewEvent("REVISE"))
	handleEvent(ctx, sm, fluo.NewEvent("SUBMIT"))
	handleEvent(ctx, sm, fluo.NewEventWithData("ASSIGN_REVIEWER", "Jane Doe"))
	handleEvent(ctx, sm, fluo.NewEvent("COMPLETE_REVIEW"))
	handleEvent(ctx, sm, fluo.NewEventWithData("DECISION", "approve"))

	// Final publication
	handleEvent(ctx, sm, fluo.NewEvent("PUBLISH"))
}

// handleEvent processes an event and displays the resulting state
func handleEvent(ctx context.Context, sm *fluo.StateMachine, event *fluo.Event) {
	if err := sm.HandleEvent(ctx, event); err != nil {
		fmt.Printf("Error handling event %s: %v\n", event.Name, err)
	}
	printCurrentState(sm)
}

func printCurrentState(sm *fluo.StateMachine) {
	fmt.Printf("\nCurrent state: %s\n", sm.CurrentState().Name())
	fmt.Println("-------------------------")
	time.Sleep(time.Millisecond * 500) // Short delay for readability
}
