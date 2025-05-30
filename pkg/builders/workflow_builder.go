package builders

import (
	"fmt"

	"github.com/anggasct/fluo/pkg/core"
	"github.com/anggasct/fluo/pkg/states"
)

// WorkflowBuilder provides specialized builder for workflow patterns
type WorkflowBuilder struct {
	*StateMachineBuilder
	stepCount int
}

// NewWorkflowBuilder creates a new workflow builder
func NewWorkflowBuilder(name string) *WorkflowBuilder {
	return &WorkflowBuilder{
		StateMachineBuilder: NewStateMachineBuilder(name),
		stepCount:           0,
	}
}

// AddSequentialStep adds a sequential workflow step
func (w *WorkflowBuilder) AddSequentialStep(name string, action core.Action) *WorkflowBuilder {
	stepName := fmt.Sprintf("%s_step_%d", name, w.stepCount)
	w.AddSimpleState(stepName)
	w.currentState.(*states.SimpleState).AddEntryAction(action)

	// If this isn't the first step, add a transition from the previous step
	if w.stepCount > 0 {
		prevStepName := fmt.Sprintf("%s_step_%d", name, w.stepCount-1)
		w.AddTransition(prevStepName, stepName, "NEXT")
	} else {
		// If it's the first step, make it the initial state
		w.SetInitialState(stepName)
	}

	w.stepCount++
	return w
}

// AddParallelBranch adds a parallel branch to the workflow
func (w *WorkflowBuilder) AddParallelBranch(branchName string, steps []string) *WorkflowBuilder {
	// Create a parallel state
	parallelName := fmt.Sprintf("parallel_%s", branchName)
	parallelState := states.NewParallelState(parallelName)
	w.sm.AddState(parallelState)

	// Create a join state
	joinName := fmt.Sprintf("join_%s", branchName)
	joinState := states.NewSimpleState(joinName)
	w.sm.AddState(joinState)

	// Set the join state
	parallelState.SetJoinState(joinState)

	// If this isn't the first step, add a transition from the previous step
	if w.stepCount > 0 {
		prevStepName := fmt.Sprintf("step_%d", w.stepCount-1)
		w.AddTransition(prevStepName, parallelName, "NEXT")
	} else {
		// If it's the first step, make it the initial state
		w.SetInitialState(parallelName)
	}

	// Add a transition from the join state to the next sequential step
	w.stepCount++
	nextStepName := fmt.Sprintf("step_%d", w.stepCount)
	w.AddTransition(joinName, nextStepName, "NEXT")

	// Keep track of the current state
	w.currentState = joinState

	return w
}

// AddConditionalBranch adds a conditional branch using choice state
func (w *WorkflowBuilder) AddConditionalBranch(name string, conditions map[string]core.GuardCondition) *WorkflowBuilder {
	// Create a choice state
	choiceName := fmt.Sprintf("choice_%s", name)
	choiceState := states.NewChoiceState(choiceName)
	w.sm.AddState(choiceState)

	// If this isn't the first step, add a transition from the previous step
	if w.stepCount > 0 {
		prevStepName := fmt.Sprintf("step_%d", w.stepCount-1)
		w.AddTransition(prevStepName, choiceName, "NEXT")
	} else {
		// If it's the first step, make it the initial state
		w.SetInitialState(choiceName)
	}

	// Create destination states for each condition
	for destName, condition := range conditions {
		destState := states.NewSimpleState(destName)
		w.sm.AddState(destState)
		choiceState.AddChoice(condition, destState)

		// Add transition from the destination back to the main flow
		w.stepCount++
		nextStepName := fmt.Sprintf("step_%d", w.stepCount)
		w.AddTransition(destName, nextStepName, "NEXT")
	}

	w.currentState = choiceState
	return w
}

// WithChoiceJunction adds a choice junction to the workflow
func (w *WorkflowBuilder) WithChoiceJunction(name string) *WorkflowBuilder {
	choiceState := states.NewChoiceState(name)
	w.sm.AddState(choiceState)
	return w
}

// FinishWorkflow adds a final state to complete the workflow
func (w *WorkflowBuilder) FinishWorkflow() *WorkflowBuilder {
	finalState := states.NewFinalState("workflow_completed")
	w.sm.AddFinalState(finalState)

	// Add transition from the last step to the final state
	if w.stepCount > 0 {
		lastStepName := fmt.Sprintf("step_%d", w.stepCount-1)
		w.AddTransition(lastStepName, "workflow_completed", "COMPLETE")
	}

	return w
}
