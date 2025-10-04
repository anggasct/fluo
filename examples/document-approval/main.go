package main

import (
	"fmt"
	"log"
	"time"

	"github.com/anggasct/fluo"
)

func main() {
	fmt.Println("=== Document Approval Workflow Example ===")
	fmt.Println("Demonstrating: Choice, Fork/Join, History, and Parallel States")

	machine := buildDocumentApprovalMachine()
	instance := machine.CreateInstance()

	observer := &DocumentWorkflowObserver{
		TransitionCount: 0,
		StateHistory:    []string{},
		Metrics:         make(map[string]interface{}),
	}
	instance.AddObserver(observer)

	ctx := instance.Context()
	setupDocumentContext(ctx)

	fmt.Println("\n=== Starting Document Approval Workflow ===")
	if err := instance.Start(); err != nil {
		log.Fatalf("Failed to start workflow: %v", err)
	}

	runWorkflowScenarios(instance, observer)
}

func setupDocumentContext(ctx fluo.Context) {
	document := &Document{
		ID:        "DOC-2024-001",
		Title:     "Software Architecture Guidelines",
		Author:    "Senior Architect",
		Type:      TechnicalDoc,
		Priority:  Standard,
		Content:   "Comprehensive guidelines for software architecture...",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		Version:   1,
	}

	reviewContext := &ReviewContext{
		Document:      document,
		LegalDecision: Pending,
		TechDecision:  Pending,
		RejectCount:   0,
		History:       []string{},
		Metadata:      make(map[string]interface{}),
	}

	ctx.Set("review_context", reviewContext)
	ctx.Set("document", document)
}

type DocumentWorkflowObserver struct {
	fluo.BaseObserver
	TransitionCount int
	StateHistory    []string
	Metrics         map[string]interface{}
}

func (o *DocumentWorkflowObserver) OnTransition(from, to string, event fluo.Event, ctx fluo.Context) {
	o.TransitionCount++
	o.StateHistory = append(o.StateHistory, fmt.Sprintf("%s -> %s", from, to))

	eventName := ""
	if event != nil {
		eventName = event.GetName()
	}

	fmt.Printf("Transition #%d: %s --[%s]--> %s\n",
		o.TransitionCount, from, eventName, to)

	o.updateMetrics(from, to, eventName, ctx)
}

func (o *DocumentWorkflowObserver) OnStateEnter(state string, ctx fluo.Context) {
	fmt.Printf("Entered: %s\n", state)

	if state == "review_workflow" {
		fmt.Println("Entering complex review workflow with hierarchical states")
	}
}

func (o *DocumentWorkflowObserver) OnStateExit(state string, ctx fluo.Context) {
	fmt.Printf("Exited: %s\n", state)
}

func (o *DocumentWorkflowObserver) updateMetrics(from, to, eventName string, ctx fluo.Context) {
	transitionKey := fmt.Sprintf("%s_%s", from, to)
	if count, exists := o.Metrics[transitionKey]; exists {
		o.Metrics[transitionKey] = count.(int) + 1
	} else {
		o.Metrics[transitionKey] = 1
	}

	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		if to == "approved" || to == "rejected" {
			duration := time.Since(reviewCtx.ReviewStartTime)
			o.Metrics["workflow_duration"] = duration
		}
	}
}

func (o *DocumentWorkflowObserver) PrintSummary() {
	fmt.Printf("\n=== Workflow Summary ===\n")
	fmt.Printf("Total Transitions: %d\n", o.TransitionCount)
	fmt.Printf("State History: %v\n", o.StateHistory)
	fmt.Printf("Transition Patterns: %v\n", o.Metrics)

	if duration, exists := o.Metrics["workflow_duration"]; exists {
		fmt.Printf("Workflow Duration: %v\n", duration)
	}
}

func runWorkflowScenarios(instance fluo.Machine, observer *DocumentWorkflowObserver) {
	ctx := instance.Context()

	fmt.Println("\n=== Scenario 1: True Parallel State with Regions ===")
	runStandardApprovalScenario(instance, observer)

	fmt.Println("\n=== Scenario 2: Fork/Join Pseudostate Demonstration ===")
	runForkJoinDemo(instance, observer)

	fmt.Println("\n=== Scenario 3: Expedited Approval Flow ===")
	runExpeditedApprovalScenario(instance, observer)

	fmt.Println("\n=== Scenario 4: Revision and Resubmission Flow ===")
	runRevisionScenario(instance, observer)

	observer.PrintSummary()
	printFinalDocumentState(ctx)

	fmt.Printf("\nFinal State: %s\n", instance.CurrentState())
	fmt.Println("=== Document Approval Workflow Demonstration Complete ===")
}

func runStandardApprovalScenario(instance fluo.Machine, observer *DocumentWorkflowObserver) {
	instance.Reset()
	instance.Start()

	ctx := instance.Context()
	setupDocumentContext(ctx)

	fmt.Println("Executing standard approval workflow with parallel processing...")
	events := []struct {
		name        string
		description string
	}{
		{"submit", "Submit document for approval"},
		{"review_complete", "Complete initial review and route by priority"},
		{"standard_review_complete", "Fork to parallel branches with TRUE parallel execution"},
		{"legal_review_done", "Complete legal review in parallel"},
		{"technical_review_done", "Complete technical review in parallel"},
		{"legal_complete", "Legal branch completes"},
		{"technical_complete", "Technical branch completes - triggers Join synchronization"},
	}

	for i, event := range events {
		fmt.Printf("\n--- Parallel Step %d: %s ---\n", i+1, event.description)
		fmt.Printf("Sending event: %s\n", event.name)

		result := instance.SendEvent(event.name, nil)

		if result.Success() {
			fmt.Printf("Success: %s → %s\n", result.PreviousState, result.CurrentState)
			activeStates := instance.GetActiveStates()
			if len(activeStates) > 1 {
				fmt.Printf("Active states (parallel): %v\n", activeStates)
			}
			regions := instance.GetParallelRegions()
			if len(regions) > 0 {
				fmt.Printf("Parallel regions: %v\n", regions)
			}
		} else {
			fmt.Printf("Failed: %s\n", result.RejectionReason)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func runForkJoinDemo(instance fluo.Machine, observer *DocumentWorkflowObserver) {
	instance.Reset()
	instance.Start()

	ctx := instance.Context()
	setupUrgentDocumentContext(ctx) // Use urgent to get quick approval

	instance.SendEvent("submit", nil)             // draft -> expedited_review
	instance.SendEvent("expedited_complete", nil) // expedited_review -> approved

	fmt.Println("Executing Fork/Join demonstration...")

	events := []struct {
		name        string
		description string
	}{
		{"fork_demo_trigger", "Trigger fork demonstration"},
		{"branch_a_done", "Complete branch A"},
		{"branch_b_done", "Complete branch B"},
		{"branch_c_done", "Complete branch C - triggers Join synchronization"},
	}

	for i, event := range events {
		fmt.Printf("\n--- Fork/Join Step %d: %s ---\n", i+1, event.description)

		fmt.Printf("Sending event: %s\n", event.name)

		result := instance.SendEvent(event.name, nil)

		if result.Success() {
			fmt.Printf("Success: %s → %s\n", result.PreviousState, result.CurrentState)
			activeStates := instance.GetActiveStates()
			if len(activeStates) > 1 {
				fmt.Printf("Active states (from fork): %v\n", activeStates)
			}
		} else {
			fmt.Printf("Failed: %s\n", result.RejectionReason)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func runExpeditedApprovalScenario(instance fluo.Machine, observer *DocumentWorkflowObserver) {
	instance.Reset()
	instance.Start()

	ctx := instance.Context()
	setupUrgentDocumentContext(ctx)

	fmt.Println("Executing expedited approval for urgent document...")

	events := []struct {
		name        string
		description string
	}{
		{"submit", "Submit urgent document (bypasses standard review)"},
		{"expedited_complete", "Complete expedited approval"},
	}

	for i, event := range events {
		fmt.Printf("\n--- Expedited Step %d: %s ---\n", i+1, event.description)
		fmt.Printf("Sending event: %s\n", event.name)

		result := instance.SendEvent(event.name, nil)

		if result.Success() {
			fmt.Printf("Success: %s → %s\n", result.PreviousState, result.CurrentState)
		} else {
			fmt.Printf("Failed: %s\n", result.RejectionReason)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func runRevisionScenario(instance fluo.Machine, observer *DocumentWorkflowObserver) {
	if instance.CurrentState() != "revision_required" {
		instance.Reset()
		instance.Start()
		ctx := instance.Context()
		setupDocumentForRevision(ctx) // Use new setup that creates a rejection scenario

		instance.SendEvent("submit", nil)
		instance.SendEvent("review_complete", nil)
		instance.SendEvent("standard_review_complete", nil)
		instance.SendEvent("legal_review_done", nil)
		instance.SendEvent("technical_review_done", nil)
		instance.SendEvent("legal_complete", nil)
		instance.SendEvent("technical_complete", nil)
	}

	events := []struct {
		name        string
		description string
	}{
		{"revise", "Revise document and resubmit"},
		{"submit", "Resubmit revised document"},
	}

	for i, event := range events {
		fmt.Printf("\n--- Revision Step %d: %s ---\n", i+1, event.description)
		fmt.Printf("Sending event: %s\n", event.name)

		result := instance.SendEvent(event.name, nil)

		if result.Success() {
			fmt.Printf("Success: %s → %s\n", result.PreviousState, result.CurrentState)
		} else {
			fmt.Printf("Failed: %s\n", result.RejectionReason)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func setupUrgentDocumentContext(ctx fluo.Context) {
	document := &Document{
		ID:        "DOC-2024-URGENT-001",
		Title:     "Critical Security Policy Update",
		Author:    "Security Officer",
		Type:      PolicyDoc,
		Priority:  Urgent,
		Content:   "Critical security policy updates required immediately...",
		CreatedAt: time.Now().Add(-1 * time.Hour),
		Version:   1,
	}

	reviewContext := &ReviewContext{
		Document:      document,
		LegalDecision: Pending,
		TechDecision:  Pending,
		RejectCount:   0,
		History:       []string{},
		Metadata:      make(map[string]interface{}),
	}

	ctx.Set("review_context", reviewContext)
	ctx.Set("document", document)
}

func setupDocumentForRevision(ctx fluo.Context) {
	document := &Document{
		ID:        "DOC-2024-REVISION-001",
		Title:     "Draft Technical Specification",
		Author:    "Junior Developer",
		Type:      TechnicalDoc,
		Priority:  Standard,
		Content:   "Incomplete technical specification that needs revision...",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		Version:   1,
	}

	reviewContext := &ReviewContext{
		Document:      document,
		LegalDecision: Rejected, // Set to rejected to trigger revision flow
		LegalReviewer: "Legal Team",
		LegalComments: "Legal terms need clarification",
		TechDecision:  Approved, // One approved, one rejected = revision needed
		TechReviewer:  "Tech Lead",
		TechComments:  "Technical approach is sound",
		RejectCount:   0,
		History:       []string{},
		Metadata:      make(map[string]interface{}),
	}

	ctx.Set("review_context", reviewContext)
	ctx.Set("document", document)
}

func printFinalDocumentState(ctx fluo.Context) {
	if reviewCtx := getReviewContext(ctx); reviewCtx != nil {
		fmt.Printf("\n=== Final Document State ===\n")
		fmt.Printf("Title: %s\n", reviewCtx.Document.Title)
		fmt.Printf("Priority: %s\n", reviewCtx.Document.Priority)
		fmt.Printf("Legal Decision: %s (%s)\n", reviewCtx.LegalDecision, reviewCtx.LegalComments)
		fmt.Printf("Technical Decision: %s (%s)\n", reviewCtx.TechDecision, reviewCtx.TechComments)
		fmt.Printf("Revision Count: %d\n", reviewCtx.RejectCount)
		fmt.Printf("Review History: %v\n", reviewCtx.History)
	}
}
