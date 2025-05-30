// Package builders provides fluent builders for constructing state machines
package builders

import (
	"strings"
	"time"

	"github.com/anggasct/fluo/pkg/core"
	"github.com/anggasct/fluo/pkg/states"
)

// StateMachineBuilder provides a fluent interface for building state machines
type StateMachineBuilder struct {
	sm            *core.StateMachine
	currentState  core.State
	pendingEvents []string
}

// StateBuilder provides a fluent interface for configuring individual states
type StateBuilder struct {
	builder *StateMachineBuilder
	state   core.State
}

// RegionBuilder provides a fluent interface for configuring parallel regions
type RegionBuilder struct {
	builder       *StateMachineBuilder
	parallelState *states.ParallelState
	region        *states.ParallelRegion
	currentState  core.State
}

// NewStateMachineBuilder creates a new state machine builder
func NewStateMachineBuilder(name string) *StateMachineBuilder {
	return &StateMachineBuilder{
		sm:            core.NewStateMachine(name),
		pendingEvents: make([]string, 0),
	}
}

// AddSimpleState adds a simple state to the state machine
func (b *StateMachineBuilder) AddSimpleState(name string) *StateMachineBuilder {
	state := states.NewSimpleState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddCompositeState adds a composite state to the state machine
func (b *StateMachineBuilder) AddCompositeState(name string) *StateMachineBuilder {
	state := states.NewCompositeState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddParallelState adds a parallel state to the state machine
func (b *StateMachineBuilder) AddParallelState(name string) *StateMachineBuilder {
	state := states.NewParallelState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddChoiceState adds a choice state to the state machine
func (b *StateMachineBuilder) AddChoiceState(name string) *StateMachineBuilder {
	state := states.NewChoiceState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddFinalState adds a final state to the state machine
func (b *StateMachineBuilder) AddFinalState(name string) *StateMachineBuilder {
	state := states.NewFinalState(name)
	b.sm.AddFinalState(state)
	b.currentState = state
	return b
}

// AddHistoryState adds a history state to the state machine
func (b *StateMachineBuilder) AddHistoryState(name string, historyType states.HistoryType) *StateMachineBuilder {
	state := states.NewHistoryState(name, historyType)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddTimeoutState adds a timeout state to the state machine
func (b *StateMachineBuilder) AddTimeoutState(name string, duration int) *StateMachineBuilder {
	// Convert duration to time.Duration
	timeDuration := states.NewTimeoutState(name, time.Duration(duration)*time.Millisecond)
	b.sm.AddState(timeDuration)
	b.currentState = timeDuration
	return b
}

// AddDeferState adds a defer state to the state machine
func (b *StateMachineBuilder) AddDeferState(name string) *StateMachineBuilder {
	state := states.NewDeferState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddEntryPoint adds an entry point state to the state machine
func (b *StateMachineBuilder) AddEntryPoint(name string) *StateMachineBuilder {
	state := states.NewEntryPointState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddExitPoint adds an exit point state to the state machine
func (b *StateMachineBuilder) AddExitPoint(name string) *StateMachineBuilder {
	state := states.NewExitPointState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// SetInitialState sets the initial state of the state machine
func (b *StateMachineBuilder) SetInitialState(name string) *StateMachineBuilder {
	b.sm.SetInitialStateByName(name)
	return b
}

// WithEntryAction adds an entry action to the current state
func (b *StateMachineBuilder) WithEntryAction(action core.Action) *StateMachineBuilder {
	if baseState, ok := b.currentState.(*states.SimpleState); ok {
		baseState.AddEntryAction(action)
	} else if baseState, ok := b.currentState.(*states.BaseState); ok {
		baseState.AddEntryAction(action)
	}
	return b
}

// WithExitAction adds an exit action to the current state
func (b *StateMachineBuilder) WithExitAction(action core.Action) *StateMachineBuilder {
	if baseState, ok := b.currentState.(*states.SimpleState); ok {
		baseState.AddExitAction(action)
	} else if baseState, ok := b.currentState.(*states.BaseState); ok {
		baseState.AddExitAction(action)
	}
	return b
}

// AddTransition adds a transition between states
func (b *StateMachineBuilder) AddTransition(fromName, toName, event string) *StateMachineBuilder {
	from := b.sm.GetState(fromName)
	to := b.sm.GetState(toName)
	if from != nil && to != nil {
		b.sm.AddTransition(from, to, event)
	}
	return b
}

// AddTransitionWithGuard adds a transition with a guard condition
func (b *StateMachineBuilder) AddTransitionWithGuard(fromName, toName, event string, guard core.GuardCondition) *StateMachineBuilder {
	from := b.sm.GetState(fromName)
	to := b.sm.GetState(toName)
	if from != nil && to != nil {
		b.sm.AddTransitionWithGuard(from, to, event, guard)
	}
	return b
}

// AddTransitionWithAction adds a transition with an action
func (b *StateMachineBuilder) AddTransitionWithAction(fromName, toName, event string, action core.Action) *StateMachineBuilder {
	from := b.sm.GetState(fromName)
	to := b.sm.GetState(toName)
	if from != nil && to != nil {
		b.sm.AddTransitionWithAction(from, to, event, nil, action)
	}
	return b
}

// AddLoggingObserver adds a logging observer
func (b *StateMachineBuilder) AddLoggingObserver(level int) *StateMachineBuilder {
	// This will be implemented when we add the logger
	return b
}

// AddMetricsObserver adds a metrics observer
func (b *StateMachineBuilder) AddMetricsObserver() *StateMachineBuilder {
	// This will be implemented when we add metrics
	return b
}

// WithState adds a simple state to the state machine and returns a fluent StateBuilder
func (b *StateMachineBuilder) WithState(name string) *StateBuilder {
	state := states.NewSimpleState(name)
	b.sm.AddState(state)
	return &StateBuilder{
		builder: b,
		state:   state,
	}
}

// WithCompositeState adds a composite state to the state machine
func (b *StateMachineBuilder) WithCompositeState(name string) *StateMachineBuilder {
	return b.AddCompositeState(name)
}

// WithChildState adds a child state to a composite state
func (b *StateMachineBuilder) WithChildState(compositeName, childName string) *StateMachineBuilder {
	parent := b.sm.GetState(compositeName)
	if compositeState, ok := parent.(*states.CompositeState); ok {
		child := states.NewSimpleState(childName)
		compositeState.AddChild(child)
	}
	return b
}

// WithInitialChildState sets the initial child state of a composite state
func (b *StateMachineBuilder) WithInitialChildState(compositeName, childName string) *StateMachineBuilder {
	parent := b.sm.GetState(compositeName)
	if compositeState, ok := parent.(*states.CompositeState); ok {
		child := compositeState.GetChild(childName)
		if child != nil {
			compositeState.SetInitialChild(child)
		}
	}
	return b
}

// WithInitialState sets the initial state of the state machine
func (b *StateMachineBuilder) WithInitialState(name string) *StateMachineBuilder {
	return b.SetInitialState(name)
}

// WithTransition adds a transition between states
func (b *StateMachineBuilder) WithTransition(fromName, toName, event string) *TransitionBuilder {
	// Handling for composite state transitions
	// fromName can be in format "ParentState.ChildState"
	from := b.getStateByPath(fromName)
	to := b.getStateByPath(toName)

	if from != nil && to != nil {
		transition := b.sm.AddTransition(from, to, event)
		return &TransitionBuilder{
			builder:    b,
			transition: transition,
		}
	}
	return &TransitionBuilder{builder: b}
}

// TransitionBuilder provides a fluent interface for configuring transitions
type TransitionBuilder struct {
	builder    *StateMachineBuilder
	transition *core.Transition
}

// WithGuard adds a guard condition to the transition
func (tb *TransitionBuilder) WithGuard(guard core.GuardCondition) *TransitionBuilder {
	if tb.transition != nil {
		tb.transition.Guard = guard
	}
	return tb
}

// WithAction adds an action to the transition
func (tb *TransitionBuilder) WithAction(action core.Action) *TransitionBuilder {
	if tb.transition != nil {
		tb.transition.Action = action
	}
	return tb
}

// AsFinal marks the current state as a final state
func (b *StateMachineBuilder) AsFinal() *StateMachineBuilder {
	if b.currentState != nil {
		b.sm.AddFinalState(b.currentState)
	}
	return b
}

// Build validates and returns the constructed state machine
func (b *StateMachineBuilder) Build() (*core.StateMachine, error) {
	// Validate the state machine
	validator := NewValidationBuilder(b.sm)
	if err := validator.ValidateStateMachine(); err != nil {
		return nil, err
	}

	return b.sm, nil
}

// getStateByPath retrieves a state from a path string (e.g., "Parent.Child")
func (b *StateMachineBuilder) getStateByPath(path string) core.State {
	// Split path into components
	parts := strings.Split(path, ".")

	if len(parts) == 1 {
		// Simple state path
		return b.sm.GetState(parts[0])
	} else if len(parts) == 2 {
		// For "Complex.SubState1" format, the state doesn't actually exist in the state machine
		// so we need to create a special proxy state that will be used in transitions
		parentName := parts[0]
		childName := parts[1]

		// Create a special state that just forwards to the composite state
		proxyState := states.NewSimpleState(path)

		// Remember parent and child names in the state machine context for later use
		if b.sm.Context().GetData(path+"_parent") == nil {
			b.sm.Context().SetData(path+"_parent", parentName)
			b.sm.Context().SetData(path+"_child", childName)
		}

		return proxyState
	}

	return nil
}

// WithEntryAction adds an entry action to the state
func (sb *StateBuilder) WithEntryAction(action core.Action) *StateBuilder {
	if baseState, ok := sb.state.(*states.BaseState); ok {
		baseState.AddEntryAction(action)
	} else if simpleState, ok := sb.state.(*states.SimpleState); ok {
		simpleState.AddEntryAction(action)
	}
	return sb
}

// WithExitAction adds an exit action to the state
func (sb *StateBuilder) WithExitAction(action core.Action) *StateBuilder {
	if baseState, ok := sb.state.(*states.BaseState); ok {
		baseState.AddExitAction(action)
	} else if simpleState, ok := sb.state.(*states.SimpleState); ok {
		simpleState.AddExitAction(action)
	}
	return sb
}

// WithDoActivity adds a do activity to the state
func (sb *StateBuilder) WithDoActivity(action core.Action) *StateBuilder {
	if baseState, ok := sb.state.(*states.BaseState); ok {
		baseState.AddDoActivity(action)
	} else if simpleState, ok := sb.state.(*states.SimpleState); ok {
		simpleState.AddDoActivity(action)
	}
	return sb
}

// AsFinal marks the state as a final state
func (sb *StateBuilder) AsFinal() *StateBuilder {
	sb.builder.sm.AddFinalState(sb.state)
	return sb
}

// Done returns the parent builder to continue fluent API chain
func (sb *StateBuilder) Done() *StateMachineBuilder {
	return sb.builder
}

// WithObserver adds an observer to the state machine
func (b *StateMachineBuilder) WithObserver(observer core.StateMachineObserver) *StateMachineBuilder {
	b.sm.AddObserver(observer)
	return b
}

// WithParallelState adds a parallel state to the state machine
func (b *StateMachineBuilder) WithParallelState(name string) *StateMachineBuilder {
	state := states.NewParallelState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddParallelRegion adds a parallel region to a parallel state and returns a region builder
func (b *StateMachineBuilder) AddParallelRegion(parallelStateName, regionName string) *RegionBuilder {
	parallelState, ok := b.sm.GetState(parallelStateName).(*states.ParallelState)
	if !ok {
		// Create the parallel state if it doesn't exist
		parallelState = states.NewParallelState(parallelStateName)
		b.sm.AddState(parallelState)
	}

	// Create a sub-state machine for the region
	regionStateMachine := core.NewStateMachine(regionName)

	// Create the region
	region := states.NewParallelRegion(regionName, regionStateMachine)
	parallelState.AddRegion(region)

	return &RegionBuilder{
		builder:       b,
		parallelState: parallelState,
		region:        region,
		currentState:  nil,
	}
}

// WithState adds a state to the region
func (rb *RegionBuilder) WithState(name string) *RegionBuilder {
	stateMachine := rb.region.GetStateMachine()
	state := states.NewSimpleState(name)
	stateMachine.AddState(state)
	rb.currentState = state
	return rb
}

// WithEntryAction adds an entry action to the current state in the region
func (rb *RegionBuilder) WithEntryAction(action core.Action) *RegionBuilder {
	if rb.currentState != nil {
		if baseState, ok := rb.currentState.(*states.BaseState); ok {
			baseState.AddEntryAction(action)
		} else if simpleState, ok := rb.currentState.(*states.SimpleState); ok {
			simpleState.AddEntryAction(action)
		}
	}
	return rb
}

// WithExitAction adds an exit action to the current state in the region
func (rb *RegionBuilder) WithExitAction(action core.Action) *RegionBuilder {
	if rb.currentState != nil {
		if baseState, ok := rb.currentState.(*states.BaseState); ok {
			baseState.AddExitAction(action)
		} else if simpleState, ok := rb.currentState.(*states.SimpleState); ok {
			simpleState.AddExitAction(action)
		}
	}
	return rb
}

// WithDoActivity adds a do activity to the current state in the region
func (rb *RegionBuilder) WithDoActivity(action core.Action) *RegionBuilder {
	if rb.currentState != nil {
		if baseState, ok := rb.currentState.(*states.BaseState); ok {
			baseState.AddDoActivity(action)
		} else if simpleState, ok := rb.currentState.(*states.SimpleState); ok {
			simpleState.AddDoActivity(action)
		}
	}
	return rb
}

// WithInitialState sets the initial state of the region
func (rb *RegionBuilder) WithInitialState(name string) *RegionBuilder {
	stateMachine := rb.region.GetStateMachine()
	stateMachine.SetInitialStateByName(name)
	return rb
}

// WithTransition adds a transition between states in the region
func (rb *RegionBuilder) WithTransition(fromName, toName, event string) *RegionTransitionBuilder {
	stateMachine := rb.region.GetStateMachine()
	from := stateMachine.GetState(fromName)
	to := stateMachine.GetState(toName)

	if from != nil && to != nil {
		transition := stateMachine.AddTransition(from, to, event)
		return &RegionTransitionBuilder{
			regionBuilder: rb,
			transition:    transition,
		}
	}
	return &RegionTransitionBuilder{regionBuilder: rb}
}

// RegionTransitionBuilder provides a fluent interface for configuring transitions in a region
type RegionTransitionBuilder struct {
	regionBuilder *RegionBuilder
	transition    *core.Transition
}

// WithGuard adds a guard condition to the transition
func (rtb *RegionTransitionBuilder) WithGuard(guard core.GuardCondition) *RegionTransitionBuilder {
	if rtb.transition != nil {
		rtb.transition.Guard = guard
	}
	return rtb
}

// WithAction adds an action to the transition
func (rtb *RegionTransitionBuilder) WithAction(action core.Action) *RegionTransitionBuilder {
	if rtb.transition != nil {
		rtb.transition.Action = action
	}
	return rtb
}

// Done returns to the region builder to continue the fluent chain
func (rtb *RegionTransitionBuilder) Done() *RegionBuilder {
	return rtb.regionBuilder
}
