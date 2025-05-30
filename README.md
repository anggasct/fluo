# Fluo

Fluo is a modern, comprehensive finite state machine library for Go that implements UML state machine concepts including hierarchical states, parallel regions, choice points, and event deferring.

[![Go Report Card](https://goreportcard.com/badge/github.com/anggasct/fluo)](https://goreportcard.com/report/github.com/anggasct/fluo)
[![GoDoc](https://godoc.org/github.com/anggasct/fluo?status.svg)](https://godoc.org/github.com/anggasct/fluo)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Features

- **Hierarchical State Machines**: Support for composite states that contain child states
- **Parallel Regions**: Execute multiple state machines concurrently
- **History States**: Remember previously active states
- **Choice Points**: Dynamically select transitions based on conditions
- **Junction Points**: Simplify complex transitions
- **Event Deferral**: Postpone events until they can be handled
- **Submachines**: Encapsulate full state machines within a state
- **Guards and Actions**: Control transitions with conditions and actions
- **Observable**: Monitor state machine events and transitions with observers
- **Builder Pattern**: Fluent API for constructing state machines
- **Thread-Safe**: Safe for concurrent use
- **Context Support**: Full support for Go's context package
- **Workflow Patterns**: Ready-made patterns for common workflows
- **Metrics and Validation**: Built-in support for monitoring and validation

## Installation

To install the Fluo library, use the standard Go package installation command:

```bash
go get -u github.com/anggasct/fluo
```

For updating to the latest version:

```bash
go get -u github.com/anggasct/fluo@latest
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/anggasct/fluo"
)

func main() {
    // Create a traffic light state machine
    builder := fluo.NewStateMachineBuilder("traffic-light")
    
    // Define states with entry actions
    builder.WithState("red").
        WithEntryAction(func(ctx *fluo.Context) error {
            fmt.Println("Light turned RED - Stop")
            return nil
        })
        
    builder.WithState("yellow").
        WithEntryAction(func(ctx *fluo.Context) error {
            fmt.Println("Light turned YELLOW - Prepare to stop")
            return nil
        })
        
    builder.WithState("green").
        WithEntryAction(func(ctx *fluo.Context) error {
            fmt.Println("Light turned GREEN - Go")
            return nil
        })
    
    // Define transitions
    builder.WithTransition("red", "green", "NEXT")
    builder.WithTransition("green", "yellow", "NEXT")
    builder.WithTransition("yellow", "red", "NEXT")
    
    // Set initial state and add logging
    builder.WithInitialState("red")
    builder.WithObserver(fluo.NewLoggingObserver())
    
    // Build and start the state machine
    sm, err := builder.Build()
    if err != nil {
        fmt.Printf("Error building state machine: %v\n", err)
        return
    }
    
    ctx := context.Background()
    if err = sm.Start(ctx); err != nil {
        fmt.Printf("Error starting state machine: %v\n", err)
        return
    }
    
    // Run through one complete traffic light cycle
    for i := 0; i < 3; i++ {
        fmt.Printf("Current state: %s\n", sm.CurrentState().Name())
        sm.HandleEvent(ctx, fluo.NewEvent("NEXT"))
        time.Sleep(1 * time.Second)
    }
    
    // Stop the state machine
    sm.Stop(ctx)
}
```

## Advanced Features

### Hierarchical State Machines

Fluo supports nested states for modeling complex behaviors:

```go
// Create a hierarchical state machine for a mobile phone
builder := fluo.NewStateMachineBuilder("phone")

// Define top-level states
builder.WithState("Idle")
     .WithEntryAction(func(ctx *fluo.Context) error {
         fmt.Println("Phone is in standby mode")
         return nil
     })

builder.WithCompositeState("Active")
builder.WithInitialState("Idle")

// Define sub-states of the Active state
builder.WithChildState("Active", "Calling")
     .WithEntryAction(func(ctx *fluo.Context) error {
         fmt.Println("Phone call in progress")
         return nil
     })

builder.WithChildState("Active", "AppRunning")
     .WithEntryAction(func(ctx *fluo.Context) error {
         fmt.Println("Application is running")
         return nil
     })

builder.WithInitialChildState("Active", "AppRunning")

// Define transitions with meaningful events
builder.WithTransition("Idle", "Active", "WAKE_UP")
builder.WithTransition("Active.AppRunning", "Active.Calling", "INCOMING_CALL")
builder.WithTransition("Active", "Idle", "SLEEP")
```

### Guards and Actions

Use guards to control transitions based on conditions, and actions to execute logic during transitions:

```go
// Document approval workflow with guards and actions
workflow := fluo.NewWorkflowBuilder("DocumentApproval")

// Define the states
workflow.WithState("Draft")
workflow.WithState("Pending")
workflow.WithState("Approved")
workflow.WithState("Rejected")
workflow.WithInitialState("Draft")

// Add a transition with a guard condition and action
workflow.WithTransition("Pending", "Approved", "REVIEW")
    .WithGuard(func(ctx *fluo.Context) bool {
        // Only allow approval if user has proper role
        return ctx.GetData("userRole") == "admin"
    })
    .WithAction(func(ctx *fluo.Context) error {
        // Record the approval with timestamp
        approver := ctx.GetData("username")
        timestamp := time.Now().Format(time.RFC3339)
        ctx.SetData("approvedBy", approver)
        ctx.SetData("approvedAt", timestamp)
        
        fmt.Printf("Document approved by %s at %s\n", approver, timestamp)
        return nil
    })

// Add a rejection transition with guard
workflow.WithTransition("Pending", "Rejected", "REVIEW")
    .WithGuard(func(ctx *fluo.Context) bool {
        // Document doesn't meet requirements
        return ctx.GetData("score").(int) < 70
    })
```

### Event Deferral

Defer events until a state can properly handle them:

```go
// Create a state machine for handling notifications
builder := fluo.NewStateMachineBuilder("NotificationSystem")

// Define states
builder.WithState("Ready")
builder.WithState("ProcessingUpdate")
builder.WithInitialState("Ready")

// Create a state that defers specific events while busy
builder.AddDeferState("ProcessingUpdate")
    .WithEntryAction(func(ctx *fluo.Context) error {
        fmt.Println("Processing update, deferring new notifications")
        // Register events to defer while in this state
        ctx.StateMachine.DeferEvents("ProcessingUpdate", "NEW_NOTIFICATION", "LOW_PRIORITY")
        return nil
    })
    .WithExitAction(func(ctx *fluo.Context) error {
        fmt.Println("Update completed, resuming deferred notifications")
        return nil
    })

// Define transitions
builder.WithTransition("Ready", "ProcessingUpdate", "SYSTEM_UPDATE")
builder.WithTransition("ProcessingUpdate", "Ready", "UPDATE_COMPLETE")

// When we return to Ready state, deferred events will automatically be processed
```

### Observability

Monitor and track state machine behavior using observers:

```go
// Create a state machine with observability
builder := fluo.NewStateMachineBuilder("ObservableProcess")

// Define states
builder.WithState("Initializing")
builder.WithState("Running")
builder.WithState("Paused")
builder.WithState("Completed")
builder.WithInitialState("Initializing")

// Define transitions
builder.WithTransition("Initializing", "Running", "START")
builder.WithTransition("Running", "Paused", "PAUSE")
builder.WithTransition("Paused", "Running", "RESUME")
builder.WithTransition("Running", "Completed", "FINISH")

// Add multiple observers for different purposes
builder.WithObserver(fluo.NewLoggingObserver())  // Standard log output

// Add a custom observer to track metrics or send notifications
customObserver := &MyCustomObserver{
    stateTimes: make(map[string]time.Time),
    metrics:    initMetricsClient(),
}
builder.WithObserver(customObserver)

// Or create a custom observer
metrics := &fluo.Observer{
    OnTransitionHandler: func(sm *fluo.StateMachine, from fluo.State, to fluo.State, event *fluo.Event) {
        // Record transition metrics
        recordMetric("transition", from.GetName(), to.GetName(), event.Name)
    },
}
builder.WithObserver(metrics)
```

Here's an example implementation of a custom observer:

```go
// MyCustomObserver tracks state machine metrics
type MyCustomObserver struct {
    stateTimes map[string]time.Time
    metrics    MetricsClient
}

// OnStateEnter is called when entering a state
func (o *MyCustomObserver) OnStateEnter(sm *fluo.StateMachine, state fluo.State) {
    stateName := state.Name()
    o.stateTimes[stateName] = time.Now()
    o.metrics.Increment("state.enter." + stateName)
    
    // Could also send notifications for important state changes
    if stateName == "Error" || stateName == "Completed" {
        notifyAdministrator(stateName, sm.Context().GetData("processId"))
    }
}

// OnStateExit is called when exiting a state
func (o *MyCustomObserver) OnStateExit(sm *fluo.StateMachine, state fluo.State) {
    stateName := state.Name()
    duration := time.Since(o.stateTimes[stateName])
    o.metrics.Gauge("state.duration." + stateName, duration.Milliseconds())
}
```

### Workflow Patterns

Fluo provides specialized builders for common workflow patterns:

```go
// Create a document approval workflow
workflow := fluo.NewWorkflowBuilder("DocumentApproval")

// Define the main workflow states
workflow.WithState("Draft")
    .WithEntryAction(func(ctx *fluo.Context) error {
        fmt.Println("Document is in draft stage")
        return nil
    })

workflow.WithState("InReview")
    .WithEntryAction(func(ctx *fluo.Context) error {
        fmt.Println("Document is being reviewed")
        return nil
    })

workflow.WithState("Approved")
    .WithEntryAction(func(ctx *fluo.Context) error {
        fmt.Println("Document has been approved")
        return nil
    })

workflow.WithState("Published").AsFinal()
    .WithEntryAction(func(ctx *fluo.Context) error {
        fmt.Println("Document has been published")
        return nil
    })

// Create a choice junction for the review outcome
workflow.WithChoiceJunction("ReviewDecision")
    .WithTransition("ReviewDecision", "Approved", "DECIDE")
    .WithGuard(func(ctx *fluo.Context) bool {
        decision := ctx.Event.Data.(string)
        return decision == "approve"
    })
    .WithTransition("ReviewDecision", "Draft", "DECIDE")
    .WithGuard(func(ctx *fluo.Context) bool {
        decision := ctx.Event.Data.(string)
        return decision == "reject"
    })

// Define the main workflow transitions
workflow.WithTransition("Draft", "InReview", "SUBMIT")
workflow.WithTransition("InReview", "ReviewDecision", "COMPLETE_REVIEW")
workflow.WithTransition("Approved", "Published", "PUBLISH")

// Configure the workflow
workflow.WithInitialState("Draft")
workflow.WithObserver(fluo.NewLoggingObserver())
```

## License

MIT License - See [LICENSE](LICENSE) for details.
