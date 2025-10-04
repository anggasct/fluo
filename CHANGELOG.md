# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-04

### Added
- **Hierarchical State Machine** - Complete HSM implementation with support for:
  - Atomic states (simple states with no substates)
  - Composite states (states containing nested substates)
  - Parallel states (concurrent regions executing simultaneously)
- **UML Pseudostates** - Full support for UML pseudostates:
  - Initial states
  - Choice states (dynamic conditional branching)
  - Junction states (static merge points)
  - Fork states (split execution into parallel paths)
  - Join states (synchronize parallel execution)
  - History states (shallow and deep history)
- **Fluent Builder API** - Method chaining for intuitive state machine construction
- **Observer Pattern** - Comprehensive monitoring system:
  - State enter/exit notifications
  - Transition events
  - Action execution tracking
  - Error monitoring
- **Thread Safety** - Concurrent-safe operations with internal synchronization
- **Context System** - Rich execution context with:
  - Event metadata and data storage
  - User-defined data storage
  - Integration with Go's standard context package
- **Visualization** - DOT and SVG generation for state machine diagrams
- **Comprehensive Examples** - Four complete examples:
  - Traffic Light (basic composite states)
  - Document Approval (complex workflow with parallel states)
  - Order Pipeline (business process with fork/join)
  - Smart Home (IoT device control with hierarchical states)

### Features
- **Zero Dependencies** - No external dependencies required (Graphviz optional for SVG)
- **Type Safe** - Full Go type safety throughout the API
- **JSON Serialization** - Machine state serialization/deserialization
- **Error Handling** - Comprehensive error types and handling
- **Performance** - Optimized for high-throughput event processing
- **Testing Support** - Built-in testing utilities and helpers

### Technical Specifications
- **Go Version**: Requires Go 1.21+
- **Test Coverage**: 45.8% statement coverage
- **Architecture**: Interface-driven design with clean separation of concerns
- **Concurrency**: Safe for concurrent use from multiple goroutines

### API Highlights
```go
// Builder pattern
definition := fluo.NewMachine().
    State("idle").Initial().
        To("working").On("start").
    Build()

// Runtime usage
machine := definition.CreateInstance()
machine.Start()
machine.SendEvent("start", data)

// Observer pattern
machine.AddObserver(&MyObserver{})

// Context integration
machine.SendEventWithContext(ctx, "event", data)
```

## Development Statistics
- **Total Go Files**: 25 source files
- **Test Files**: 18 comprehensive test files
- **Examples**: 4 production-ready examples
- **CI/CD**: GitHub Actions with multi-version Go testing
- **Documentation**: Complete README with API reference