# Running Fluo Examples

This document provides instructions on how to run the example applications included in the Fluo library.

## Prerequisites

- Go 1.18 or higher installed on your system
- The Fluo library installed (`go get -u github.com/anggasct/fluo`)

## Available Examples

The following examples are available in the `examples` directory:

1. **Turnstile** - A simple state machine demonstrating basic transitions
2. **Traffic Light** - Sequential state machine with timed transitions
3. **ATM** - A hierarchical state machine showing composite states
4. **Workflow** - Process workflows with approval steps and conditional paths
5. **Smart Home** - Parallel regions controlling different systems independently
6. **Payment Processing** - Business logic with guarded transitions

## Running the Examples

### 1. Turnstile Example

This example demonstrates a basic turnstile with locked/unlocked states that responds to coin and push events.

```bash
cd examples/turnstile
go run main.go
```

Expected output shows the turnstile transitioning between locked and unlocked states.

### 2. ATM Example

This example demonstrates a hierarchical state machine modeling an ATM with nested states for different operations.

```bash
cd examples/atm
go run main.go
```

The output shows a simulated ATM session with PIN validation, menu selection, and transaction processing.

### 3. Workflow Example

This example demonstrates a document approval workflow with multiple states and decision points.

```bash
cd examples/workflow
go run main.go
```

The output shows a document going through various approval stages with different actors and decisions.

### 4. Parallel Example

This example demonstrates parallel regions in a smart home system with multiple subsystems operating concurrently.

```bash
cd examples/parallel
go run main.go
```

The output shows multiple systems (heating, lighting, security) operating independently within the same state machine.

### 5. Smart Home Example

This example demonstrates parallel regions to model a smart home system that manages multiple subsystems simultaneously.

```bash
cd examples/smart_home
go run main.go
```

The output shows multiple systems (heating, lighting, security) operating independently within the same state machine.

## Using Parallel Regions

Parallel regions allow you to create state machines where multiple parts operate independently:

```go
// Create a state machine with parallel regions
builder := fluo.NewStateMachineBuilder("SmartHome")

// Define the main parallel state
builder.WithParallelState("Active")

// Add the first region for lighting control
lightingRegion := builder.AddParallelRegion("Active", "Lighting")
lightingRegion.WithState("Off")
lightingRegion.WithState("On")
lightingRegion.WithInitialState("Off")
lightingRegion.WithTransition("Off", "On", "LIGHTS_ON")
lightingRegion.WithTransition("On", "Off", "LIGHTS_OFF")

// Add a second region for heating control
heatingRegion := builder.AddParallelRegion("Active", "Heating")
heatingRegion.WithState("Off")
heatingRegion.WithState("On")
heatingRegion.WithInitialState("Off")
heatingRegion.WithTransition("Off", "On", "HEAT_ON")
heatingRegion.WithTransition("On", "Off", "HEAT_OFF")

// Set the parallel state as the initial state
builder.WithInitialState("Active")
```

Events are dispatched to all active regions, allowing them to react independently.

## Modifying the Examples

Feel free to modify the examples to explore different state machine patterns:

1. Add new states and transitions
2. Modify the guard conditions to create different paths
3. Add new actions to perform operations during transitions
4. Add observers to monitor state machine behavior

## Creating Your Own State Machines

To create your own state machine using the Fluo library:

1. Define your states and their behavior
2. Define transitions between states
3. Set up guard conditions and actions
4. Configure initial states
5. Build and run the state machine

See the README.md for more detailed documentation on how to use the library.
