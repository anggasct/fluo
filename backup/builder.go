package flux

import (
	"fmt"
	"time"
)

// StateMachineBuilder provides a fluent interface for building state machines
type StateMachineBuilder struct {
	sm            *StateMachine
	currentState  State
	pendingEvents []string
}

// NewBuilder creates a new state machine builder
func NewBuilder(name string) *StateMachineBuilder {
	return &StateMachineBuilder{
		sm: NewStateMachine(name),
	}
}

// AddSimpleState adds a simple state to the state machine
func (b *StateMachineBuilder) AddSimpleState(name string) *StateMachineBuilder {
	state := NewSimpleState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddCompositeState adds a composite state to the state machine
func (b *StateMachineBuilder) AddCompositeState(name string) *StateMachineBuilder {
	state := NewCompositeState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddParallelState adds a parallel state to the state machine
func (b *StateMachineBuilder) AddParallelState(name string) *StateMachineBuilder {
	state := NewParallelState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddChoiceState adds a choice state to the state machine
func (b *StateMachineBuilder) AddChoiceState(name string) *StateMachineBuilder {
	state := NewChoiceState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddFinalState adds a final state to the state machine
// When entered, this state automatically marks the state machine as completed
func (b *StateMachineBuilder) AddFinalState(name string) *StateMachineBuilder {
	state := NewFinalState(name)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// AddTimeoutState adds a timeout state to the state machine
func (b *StateMachineBuilder) AddTimeoutState(name string, timeout time.Duration) *StateMachineBuilder {
	state := NewTimeoutState(name, timeout)
	b.sm.AddState(state)
	b.currentState = state
	return b
}

// WithEntryAction adds an entry action to the current state
func (b *StateMachineBuilder) WithEntryAction(action Action) *StateMachineBuilder {
	if baseState, ok := b.currentState.(*SimpleState); ok {
		baseState.WithEntryAction(action)
	}
	return b
}

// WithExitAction adds an exit action to the current state
func (b *StateMachineBuilder) WithExitAction(action Action) *StateMachineBuilder {
	if baseState, ok := b.currentState.(*SimpleState); ok {
		baseState.WithExitAction(action)
	}
	return b
}

// SetAsInitial sets the current state as the initial state
// Deprecated: The first state added is automatically set as initial state
func (b *StateMachineBuilder) SetAsInitial() *StateMachineBuilder {
	b.sm.SetInitialState(b.currentState)
	return b
}

// AddTransition adds a transition between states
func (b *StateMachineBuilder) AddTransition(fromName, toName, event string) *StateMachineBuilder {
	from := b.sm.GetState(fromName)
	to := b.sm.GetState(toName)
	if from != nil && to != nil {
		b.sm.AddTransition(from, to, event, nil)
	}
	return b
}

// AddTransitionWithGuard adds a transition with a guard condition
func (b *StateMachineBuilder) AddTransitionWithGuard(fromName, toName, event string, guard GuardCondition) *StateMachineBuilder {
	from := b.sm.GetState(fromName)
	to := b.sm.GetState(toName)
	if from != nil && to != nil {
		b.sm.AddTransition(from, to, event, guard)
	}
	return b
}

// AddTransitionWithAction adds a transition with an action
func (b *StateMachineBuilder) AddTransitionWithAction(fromName, toName, event string, action Action) *StateMachineBuilder {
	from := b.sm.GetState(fromName)
	to := b.sm.GetState(toName)
	if from != nil && to != nil {
		b.sm.AddTransitionWithAction(from, to, event, nil, action)
	}
	return b
}

// AddLoggingObserver adds a logging observer
func (b *StateMachineBuilder) AddLoggingObserver(level LogLevel) *StateMachineBuilder {
	observer := NewLoggingObserver(level, b.sm.Name())
	b.sm.AddObserver(observer)
	return b
}

// AddMetricsObserver adds a metrics observer
func (b *StateMachineBuilder) AddMetricsObserver() *StateMachineBuilder {
	observer := NewMetricsObserver()
	b.sm.AddObserver(observer)
	return b
}

// Build returns the constructed state machine
func (b *StateMachineBuilder) Build() *StateMachine {
	return b.sm
}

// WorkflowBuilder provides specialized builder for workflow patterns
type WorkflowBuilder struct {
	*StateMachineBuilder
	stepCount int
}

// NewWorkflowBuilder creates a new workflow builder
func NewWorkflowBuilder(name string) *WorkflowBuilder {
	return &WorkflowBuilder{
		StateMachineBuilder: NewBuilder(name),
		stepCount:           0,
	}
}

// AddSequentialStep adds a sequential workflow step
func (w *WorkflowBuilder) AddSequentialStep(name string, action Action) *WorkflowBuilder {
	stepName := fmt.Sprintf("step_%d_%s", w.stepCount, name)
	w.AddSimpleState(stepName).WithEntryAction(action)

	if w.stepCount > 0 {
		prevStepName := fmt.Sprintf("step_%d", w.stepCount-1)
		w.AddTransition(prevStepName, stepName, "next")
	}
	// First step is automatically set as initial state

	w.stepCount++
	return w
}

// AddParallelBranch adds a parallel branch to the workflow
func (w *WorkflowBuilder) AddParallelBranch(branchName string, steps []string) *WorkflowBuilder {
	parallelState := fmt.Sprintf("parallel_%s", branchName)
	w.AddParallelState(parallelState)

	region := NewParallelRegion(branchName)
	for i, _ := range steps {
		stepState := NewSimpleState(fmt.Sprintf("%s_step_%d", branchName, i))
		region.GetStateMachine().AddState(stepState)

		if i == 0 {
			region.GetStateMachine().SetInitialState(stepState)
		} else {
			prevStep := fmt.Sprintf("%s_step_%d", branchName, i-1)
			region.GetStateMachine().AddTransition(
				region.GetStateMachine().GetState(prevStep),
				stepState,
				"next",
				nil,
			)
		}
	}

	if parallelState, ok := w.currentState.(*ParallelState); ok {
		parallelState.AddRegion(region)
	}

	return w
}

// AddConditionalBranch adds a conditional branch using choice state
func (w *WorkflowBuilder) AddConditionalBranch(name string, conditions map[string]GuardCondition) *WorkflowBuilder {
	choiceName := fmt.Sprintf("choice_%s", name)
	w.AddChoiceState(choiceName)

	if choiceState, ok := w.currentState.(*ChoiceState); ok {
		for targetStateName, condition := range conditions {
			targetState := w.sm.GetState(targetStateName)
			if targetState != nil {
				choiceState.AddChoice(condition, targetState)
			}
		}
	}

	return w
}

// FinishWorkflow adds a final state to complete the workflow
func (w *WorkflowBuilder) FinishWorkflow() *WorkflowBuilder {
	w.AddFinalState("completed")

	if w.stepCount > 0 {
		lastStepName := fmt.Sprintf("step_%d", w.stepCount-1)
		w.AddTransition(lastStepName, "completed", "finish")
	}

	return w
}

// ValidationBuilder helps build validation rules for state machines
type ValidationBuilder struct {
	observer *ValidationObserver
}

// NewValidationBuilder creates a new validation builder
func NewValidationBuilder() *ValidationBuilder {
	return &ValidationBuilder{
		observer: NewValidationObserver(),
	}
}

// ExpectState adds an expected state to validation
func (v *ValidationBuilder) ExpectState(stateName string) *ValidationBuilder {
	v.observer.AddExpectedState(stateName)
	return v
}

// AllowTransition adds an allowed transition to validation
func (v *ValidationBuilder) AllowTransition(from, to string) *ValidationBuilder {
	v.observer.AddAllowedTransition(from, to)
	return v
}

// Build returns the validation observer
func (v *ValidationBuilder) Build() *ValidationObserver {
	return v.observer
}

// ConditionalActions provides helper functions for common conditional actions
type ConditionalActions struct{}

// IfDataEquals creates a guard condition that checks if context data equals a value
func (ConditionalActions) IfDataEquals(key string, value interface{}) GuardCondition {
	return func(ctx *Context) bool {
		if val, exists := ctx.Get(key); exists {
			return val == value
		}
		return false
	}
}

// IfDataExists creates a guard condition that checks if context data exists
func (ConditionalActions) IfDataExists(key string) GuardCondition {
	return func(ctx *Context) bool {
		_, exists := ctx.Get(key)
		return exists
	}
}

// IfEventDataEquals creates a guard condition that checks if event data equals a value
func (ConditionalActions) IfEventDataEquals(value interface{}) GuardCondition {
	return func(ctx *Context) bool {
		if ctx.Event != nil {
			return ctx.Event.Data == value
		}
		return false
	}
}

// SetData creates an action that sets context data
func (ConditionalActions) SetData(key string, value interface{}) Action {
	return func(ctx *Context) error {
		ctx.Set(key, value)
		return nil
	}
}

// LogMessage creates an action that logs a message
func (ConditionalActions) LogMessage(message string) Action {
	return func(ctx *Context) error {
		fmt.Printf("[State Machine] %s\n", message)
		return nil
	}
}

// Global helper instance
var Actions ConditionalActions
