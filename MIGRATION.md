# Migration Guide: Fluo v1 to v2

This document provides guidance on how to migrate your code from the previous flat package structure (v1) to the new hierarchical package structure (v2).

## Package Name Change

The package name has been updated from `flux` to `fluo` for consistency with the repository name. Update your import statements:

```go
// Old
import "github.com/anggasct/fluo"

// New
import "github.com/anggasct/fluo"
```

While the import path remains the same, you'll need to update any direct package references in your code.

## Type References

In the new structure, all types are re-exported from the main package. You can continue to use them directly:

```go
// This still works
sm := fluo.NewStateMachine("MyMachine")

// These still work
state := fluo.SimpleState{...}
event := fluo.NewEvent("MyEvent")
```

However, if you were directly using types that are now internal or have been refactored, you might need to update your code.

## Advanced Usage

For advanced usage where you need direct access to sub-packages, you can import them directly:

```go
// For core types
import "github.com/anggasct/fluo/pkg/core"

// For state types
import "github.com/anggasct/fluo/pkg/states"

// For builder patterns
import "github.com/anggasct/fluo/pkg/builders" 

// For observer patterns
import "github.com/anggasct/fluo/pkg/observers"

// For utility functions
import "github.com/anggasct/fluo/pkg/utils"
```

## API Changes

### State Machine Creation

The state machine creation API now encourages using the builder pattern:

```go
// Old way
sm := flux.NewStateMachine("MyMachine")
state1 := &flux.SimpleState{Name: "State1"}
state2 := &flux.SimpleState{Name: "State2"}
sm.AddState(state1)
sm.AddState(state2)
sm.AddTransition(state1, state2, "NEXT", nil, nil)
sm.SetInitialState(state1)

// New way (preferred)
builder := fluo.NewStateMachineBuilder("MyMachine")
builder.WithState("State1")
builder.WithState("State2")
builder.WithTransition("State1", "State2", "NEXT")
builder.WithInitialState("State1")
sm, err := builder.Build()
```

### State Definitions

State definitions are now more structured:

```go
// Old way
state := &flux.SimpleState{
    Name: "MyState",
    OnEntry: func(ctx *flux.Context) error {
        // Entry logic
        return nil
    },
    OnExit: func(ctx *flux.Context) error {
        // Exit logic
        return nil
    },
}

// New way
state := &fluo.SimpleState{
    BaseState: fluo.BaseState{
        Name: "MyState",
        OnEntry: func(ctx *fluo.Context) error {
            // Entry logic
            return nil
        },
        OnExit: func(ctx *fluo.Context) error {
            // Exit logic
            return nil
        },
    },
}
```

### Composite States

Composite states are now more explicit in their hierarchy:

```go
// Old way
composite := &flux.CompositeState{
    Name: "Composite",
}
child := &flux.SimpleState{
    Name: "Child",
    Parent: composite,
}
composite.AddChildState(child)

// New way
composite := &fluo.CompositeState{
    BaseState: fluo.BaseState{
        Name: "Composite",
    },
}
child := &fluo.SimpleState{
    BaseState: fluo.BaseState{
        Name: "Child",
        ParentState: composite,
    },
}
composite.AddChildState(child)
```

## Observer Pattern

The observer pattern has been expanded with specialized observers:

```go
// Old way
sm.AddObserver(func(sm *flux.StateMachine, event string, data interface{}) {
    // Observer logic
})

// New way
logger := fluo.NewLoggingObserver()
sm.AddObserver(logger)

// Or create custom observers
customObserver := &fluo.Observer{
    OnTransitionHandler: func(sm *fluo.StateMachine, from fluo.State, to fluo.State, event *fluo.Event) {
        // Custom transition logic
    },
    OnEventHandler: func(sm *fluo.StateMachine, event *fluo.Event) {
        // Custom event logic
    },
    // Other handlers...
}
sm.AddObserver(customObserver)
```

## Error Handling

Error handling has been improved:

```go
// Old way
if err := sm.HandleEvent(ctx, event); err != nil {
    // Handle generic error
}

// New way
if err := sm.HandleEvent(ctx, event); err != nil {
    switch e := err.(type) {
    case *fluo.NoTransitionError:
        // Handle no transition found
    case *fluo.GuardRejectedError:
        // Handle guard condition rejection
    case *fluo.StateError:
        // Handle state-related error
    default:
        // Handle other errors
    }
}
```

## Workflow Builders

For workflow-specific state machines, consider using the specialized workflow builders:

```go
// New way for workflow patterns
workflow := fluo.NewWorkflowBuilder("Approval")
workflow.WithSteps("Submit", "Review", "Approve")
workflow.WithOptionalStep("Revise").After("Review").Before("Approve")
workflow.WithTransition("Review", "Reject", "REJECT")
sm, err := workflow.Build()
```

## Testing

The new structure provides better testability:

```go
// Example of testing with the new structure
func TestMyStateMachine(t *testing.T) {
    builder := fluo.NewStateMachineBuilder("TestMachine")
    builder.WithState("State1")
    builder.WithState("State2")
    builder.WithTransition("State1", "State2", "NEXT")
    builder.WithInitialState("State1")
    
    sm, err := builder.Build()
    assert.NoError(t, err)
    
    ctx := context.Background()
    err = sm.Start(ctx)
    assert.NoError(t, err)
    
    err = sm.HandleEvent(ctx, fluo.NewEvent("NEXT"))
    assert.NoError(t, err)
    assert.Equal(t, "State2", sm.CurrentState().GetName())
}
```

## Need More Help?

If you have specific migration questions or issues, please:
1. Check the [documentation](https://github.com/anggasct/fluo)
2. Look at the [examples](https://github.com/anggasct/fluo/examples)
3. Open an issue on GitHub for assistance
