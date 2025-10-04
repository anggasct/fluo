# Fluo - Hierarchical State Machine Library for Go

Fluo is a flexible hierarchical state machine library for Go. It features a fluent builder for creating statecharts with atomic, composite, and parallel states, along with comprehensive support for UML pseudostates.

## Key Features

- **Hierarchical State Machines**: Atomic, composite, and parallel states
- **Fluent Builder API**: Method chaining for state machine construction
- **Thread Safe**: Concurrent-safe operations with internal synchronization
- **Observer Pattern**: Monitor transitions and state changes
- **Visualization**: DOT and SVG generation for state machine diagrams
- **Zero Dependencies**: No external dependencies required (Graphviz optional for SVG)

## Installation

```bash
go get github.com/anggasct/fluo
```

**Requirements:**

- Go 1.21+

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/anggasct/fluo"
)

func main() {
    // Build state machine
    definition := fluo.NewMachine().
        State("red").Initial().
            To("green").On("timer").
        State("green").
            To("yellow").On("timer").
        State("yellow").
            To("red").On("timer").
        Build()

    // Create and start machine
    machine := definition.CreateInstance()
    machine.Start()
    
    // Send events
    machine.SendEvent("timer", nil) // red -> green
    machine.SendEvent("timer", nil) // green -> yellow  
    machine.SendEvent("timer", nil) // yellow -> red
    
    fmt.Printf("Current state: %s\n", machine.CurrentState())
}
```

## Core Concepts

### MachineDefinition vs Machine

- **MachineDefinition**: Immutable configuration created by the builder
- **Machine**: Runtime instance with current state, context, and observers

```go
// Build definition (immutable)
definition := fluo.NewMachine().
    State("idle").Initial().
        To("working").On("start").
    Build()

// Create runtime instances (can have multiple)
machine1 := definition.CreateInstance()
machine2 := definition.CreateInstance()
```

### Context and Events

Context carries machine state, event metadata, and user data:

```go
// Send event with data
machine.SendEvent("process", map[string]interface{}{
    "input": "data",
    "priority": 1,
})

// Access context in actions
.To("completed").On("finish").Do(func(ctx fluo.Context) error {
    if data, exists := ctx.Get("input"); exists {
        fmt.Printf("Processing: %v\n", data)
    }
    return nil
})
```

## State Types and Examples

| Element | Description | Use Case |
|---------|-------------|----------|
| **AtomicState** | Simple state with no substates | Basic workflow steps |
| **CompositeState** | Hierarchical state with substates | Complex nested behaviors |
| **ParallelState** | Concurrent regions executing simultaneously | Parallel processing, multi-tasking |
| **FinalState** | Marks completion of a state or region | Workflow termination, region completion |
| **ChoiceState** | Dynamic conditional branching | Runtime decision points |
| **JunctionState** | Static merge point for transitions | Multiple path convergence |
| **ForkState** | Split execution into parallel states | Concurrent workflow initiation |
| **JoinState** | Synchronize parallel execution paths | Barrier synchronization |
| **HistoryState** | Remember last state at current level | Resume interrupted workflows |
| **DeepHistoryState** | Remember complete state hierarchy | Restore nested state configurations |

### Atomic State

Simple state with no substates:

```go
builder.State("idle").Initial().
    OnEntry(func(ctx fluo.Context) error {
        fmt.Println("Entered idle state")
        return nil
    }).
    To("working").On("start")
```

### Composite State

Hierarchical state containing substates:

```go
composite := builder.CompositeState("order_processing")

composite.State("validation").Initial().
    OnEntry(func(ctx fluo.Context) error {
        fmt.Println("Validating order")
        return nil
    }).
    To("payment").On("valid")

composite.State("payment").
    To("shipping").On("paid")

composite.State("shipping").
    To("complete").On("delivered")

composite.State("complete").Final()

// The composite state itself can have transitions
builder.CompositeState("order_processing").
    To("canceled").On("cancel")  // Can exit from any substate
```

### Parallel State

Concurrent regions executing simultaneously:

```go
parallel := builder.ParallelState("parallel_work")

taskA := parallel.Region("task_a")
taskA.State("start").Initial().
    To("done").On("a_complete")
taskA.State("done").Final()

taskB := parallel.Region("task_b")
taskB.State("start").Initial().
    To("done").On("b_complete")
taskB.State("done").Final()

parallel.End()

// Transition when all regions complete
builder.ParallelState("parallel_work").
    To("next_state").OnCompletion()
```

### Choice Pseudostate

Dynamic conditional branching:

```go
builder.Choice("payment_router").
    When(func(ctx fluo.Context) bool {
        if amount, exists := ctx.Get("amount"); exists {
            if amt, ok := amount.(int); ok {
                return amt < 100
            }
        }
        return false
    }).To("fast_payment").
    When(func(ctx fluo.Context) bool {
        if amount, exists := ctx.Get("amount"); exists {
            if amt, ok := amount.(int); ok {
                return amt >= 100
            }
        }
        return false
    }).To("secure_payment")
```

### Final State

Mark states as final to indicate completion:

```go
builder.State("processing").
    OnEntry(func(ctx fluo.Context) error {
        fmt.Println("Processing data")
        return nil
    }).
    To("completed").On("finish")

builder.State("completed").Final().
    OnEntry(func(ctx fluo.Context) error {
        fmt.Println("Process completed")
        return nil
    })

// In parallel regions, final states trigger OnCompletion
region.State("task_done").Final()
```

### Junction Pseudostate

Static merge point for multiple transitions:

```go
// Multiple paths converge at junction
builder.State("path1").
    To("merge_point").On("complete")

builder.State("path2").
    To("merge_point").On("complete")

builder.State("path3").
    To("merge_point").On("complete")

builder.Junction("merge_point").
    To("consolidated_result").
    Do(func(ctx fluo.Context) error {
        fmt.Println("Merging results from multiple paths")
        return nil
    })

builder.State("consolidated_result")
```

### Fork and Join Pseudostates

Split and synchronize parallel execution:

```go
builder.State("start").
    To("fork_parallel").On("begin")

// Fork splits execution to multiple states simultaneously
builder.Fork("fork_parallel").
    To("task1", "task2", "task3").
    Do(func(ctx fluo.Context) error {
        fmt.Println("Starting parallel tasks")
        return nil
    })

builder.State("task1").
    To("join_tasks").On("done")

builder.State("task2").
    To("join_tasks").On("done")

builder.State("task3").
    To("join_tasks").On("done")

// Join waits for all source states before proceeding
builder.Join("join_tasks").
    From("task1", "task2", "task3").
    To("all_complete").
    Do(func(ctx fluo.Context) error {
        fmt.Println("All parallel tasks completed")
        return nil
    })

builder.State("all_complete").Final()
```

### History State

Shallow history - remember last state at current level:

```go
composite := builder.CompositeState("workflow")

composite.State("step1").Initial().
    To("step2").On("next")

composite.State("step2").
    To("step3").On("next")

composite.State("step3")

// Shallow history remembers the last active substate
composite.History("memory").Default("step1")

// Can interrupt and resume workflow
builder.CompositeState("workflow").
    To("paused").On("pause")

builder.State("paused").
    To("workflow.memory").On("resume")  // Returns to last active state
```

### Deep History State  

Deep history - remember state including nested substates:

```go
outer := builder.CompositeState("multi_level")

nested := outer.CompositeState("nested")
nested.State("sub1").Initial().
    To("sub2").On("next")
nested.State("sub2")

outer.State("other_state")

// Deep history remembers state at all nesting levels
outer.DeepHistory("deep_memory").Default("nested.sub1")

builder.CompositeState("multi_level").
    To("suspended").On("suspend")

builder.State("suspended").
    To("multi_level.deep_memory").On("restore")  // Restores complete state hierarchy
```

## Observer Pattern

Monitor state machine lifecycle events:

```go
type DebugObserver struct {
    fluo.BaseObserver
}

func (o *DebugObserver) OnTransition(from, to string, event fluo.Event, ctx fluo.Context) {
    fmt.Printf("Transition: %s -> %s (event: %s)\n", from, to, event.GetName())
}

func (o *DebugObserver) OnStateEnter(state string, ctx fluo.Context) {
    fmt.Printf("Entering state: %s\n", state)
}

func (o *DebugObserver) OnStateExit(state string, ctx fluo.Context) {
    fmt.Printf("Exiting state: %s\n", state)
}

// Add observer to machine
machine := definition.CreateInstance()
machine.AddObserver(&DebugObserver{})
```

### Extended Observer

For more comprehensive monitoring:

```go
type ExtendedObserver struct {
    fluo.BaseObserver
}

func (o *ExtendedObserver) OnGuardEvaluation(from, to string, event fluo.Event, result bool, ctx fluo.Context) {
    fmt.Printf("Guard %s -> %s: %v\n", from, to, result)
}

func (o *ExtendedObserver) OnEventRejected(event fluo.Event, reason string, ctx fluo.Context) {
    fmt.Printf("Event rejected: %s (%s)\n", event.GetName(), reason)
}

func (o *ExtendedObserver) OnError(err error, ctx fluo.Context) {
    fmt.Printf("Error: %v\n", err)
}

func (o *ExtendedObserver) OnActionExecution(actionType, state string, event fluo.Event, ctx fluo.Context) {
    fmt.Printf("Action '%s' in state '%s'\n", actionType, state)
}
```

## Visualization

Generate DOT and SVG diagrams of your state machines:

```go
import "github.com/anggasct/fluo/visualization"

// Create DOT generator
dotGen := visualization.NewDOTGenerator(definition)

// Generate DOT format
dotContent, err := dotGen.Generate()
if err != nil {
    panic(err)
}
fmt.Println(dotContent)

// Generate SVG (requires Graphviz)
svgGen := visualization.NewSVGGenerator(definition)
svgContent, err := svgGen.Generate()
if err != nil {
    panic(err)
}
fmt.Println(svgContent)

// Save to file
err = dotGen.GenerateToFile("machine.dot")
if err != nil {
    panic(err)
}
```

### Custom Visualization Options

```go
options := visualization.DefaultDOTOptions()
options.ShowGuardConditions = true
options.ShowActions = true
options.CompactMode = false
options.RankDirection = "LR"  // Left to right
options.NodeShape = "ellipse"
options.CompositeStateStyle = "rounded,filled"

dotGen := visualization.NewDOTGenerator(definition, options)
```

## Concurrency and Thread Safety

Fluo provides thread-safe operations with internal synchronization:

```go
// Safe to call from multiple goroutines
go func() {
    machine.SendEvent("event1", data)
}()

go func() {
    machine.SendEvent("event2", data)
}()

// Query current state safely
currentState := machine.CurrentState()
activeStates := machine.GetActiveStates()
```

### Parallel State Execution

Parallel states execute concurrently with proper synchronization:

```go
// Parallel regions run simultaneously
parallel := builder.ParallelState("data_processing")

region1 := parallel.Region("validation")
region1.State("validate").Initial().
    To("validated").On("complete")

region2 := parallel.Region("transformation")
region2.State("transform").Initial().
    To("transformed").On("complete")

// Both regions execute in parallel
// Transition occurs when both reach final states
parallel.To("next_phase").OnCompletion()
```

## Examples

The project includes several comprehensive examples:

- **Traffic Light** (`examples/traffic-light/`) - Basic state machine with composite states
- **Document Approval** (`examples/document-approval/`) - Complex workflow with parallel states and choice logic
- **Order Pipeline** (`examples/order-pipeline/`) - Business process with fork/join patterns
- **Smart Home** (`examples/smart-home/`) - IoT device control with hierarchical states

Run examples:

```bash
# Run traffic light example
cd examples/traffic-light
go run main.go

# Run document approval example
cd examples/document-approval
go run main.go

# Run order pipeline example
cd examples/order-pipeline
go run main.go

# Run smart home example
cd examples/smart-home
go run main.go
```

## API Reference

### Core Interfaces

#### Machine

```go
type Machine interface {
    Start() error
    Stop() error
    Reset() error
    
    CurrentState() string
    SetState(state string) error
    SetRegionState(regionID string, stateID string) error
    RegionState(regionID string) string
    GetStateHierarchy() []string
    IsInState(stateID string) bool
    GetActiveStates() []string
    IsStateActive(stateID string) bool
    GetParallelRegions() map[string][]string
    
    SendEvent(eventName string, eventData any) *EventResult
    SendEventWithContext(ctx context.Context, eventName string, eventData any) *EventResult
    HandleEvent(eventName string, eventData any) *EventResult
    HandleEventWithContext(ctx context.Context, eventName string, eventData any) *EventResult
    
    AddObserver(observer Observer)
    RemoveObserver(observer Observer)
    
    Context() Context
    WithContext(ctx Context) Machine
    
    MarshalJSON() ([]byte, error)
    UnmarshalJSON(data []byte) error
}
```

#### MachineDefinition

```go
type MachineDefinition interface {
    CreateInstance() Machine
    Build() MachineDefinition
    
    GetInitialState() string
    GetStates() map[string]State
    GetTransitions() map[string][]Transition
}
```

#### Observer

```go
type Observer interface {
    OnTransition(from string, to string, event Event, ctx Context)
    OnStateEnter(state string, ctx Context)
}

type ExtendedObserver interface {
    Observer
    OnStateExit(state string, ctx Context)
    OnGuardEvaluation(from string, to string, event Event, result bool, ctx Context)
    OnEventRejected(event Event, reason string, ctx Context)
    OnError(err error, ctx Context)
    OnActionExecution(actionType string, state string, event Event, ctx Context)
    OnMachineStarted(ctx Context)
    OnMachineStopped(ctx Context)
}
```

## Testing

Test your state machines with the provided utilities:

```go
func TestStateMachine(t *testing.T) {
    definition := BuildTestMachine()
    machine := definition.CreateInstance()
    
    // Start machine
    err := machine.Start()
    assert.NoError(t, err)
    assert.Equal(t, "initial", machine.CurrentState())
    
    // Send event and verify transition
    result := machine.SendEvent("test", nil)
    assert.True(t, result.Success())
    assert.Equal(t, "next", machine.CurrentState())
}
```

## Development

### Build Commands

```bash
# Build the library
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Clean build artifacts
make clean

# Lint the code
make lint

# Format code
make fmt

# Vet code
make vet

# Install dependencies
make deps

# Run all checks
make check
```

### Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

#### Reporting Issues

When reporting issues, please include:

- Go version
- Operating system
- Minimal code example that reproduces the issue
- Expected vs actual behavior

## License

MIT License - see [LICENSE](LICENSE) file.
