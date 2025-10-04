package fluo

import (
	"testing"
)

func TestAtomicState_Creation(t *testing.T) {
	state := NewAtomicState("test_state")

	if state.ID() != "test_state" {
		t.Errorf("Expected ID 'test_state', got '%s'", state.ID())
	}

	if state.IsComposite() {
		t.Error("Expected atomic state not to be composite")
	}

	if state.IsParallel() {
		t.Error("Expected atomic state not to be parallel")
	}

	if state.IsPseudo() {
		t.Error("Expected atomic state not to be pseudo")
	}

	if state.IsFinal() {
		t.Error("Expected atomic state not to be final by default")
	}

	if state.Parent() != nil {
		t.Error("Expected atomic state to have no parent initially")
	}
}

func TestAtomicState_FinalState(t *testing.T) {
	state := NewFinalState("final_state")

	if state.ID() != "final_state" {
		t.Errorf("Expected ID 'final_state', got '%s'", state.ID())
	}

	if !state.IsFinal() {
		t.Error("Expected final state to be final")
	}
}

func TestAtomicState_EntryExitActions(t *testing.T) {
	ResetTestAction()

	state := NewAtomicState("test_state").
		WithEntryAction(TestAction).
		WithExitAction(TestAction)

	ctx := CreateTestContext()

	TestActionCalled = false
	state.Enter(ctx)
	if !TestActionCalled {
		t.Error("Expected entry action to be called")
	}

	TestActionCalled = false
	state.Exit(ctx)
	if !TestActionCalled {
		t.Error("Expected exit action to be called")
	}
}

func TestAtomicState_ParentRelationship(t *testing.T) {
	parent := NewCompositeState("parent")
	child := NewAtomicState("child").WithParent(parent)

	if child.Parent() != parent {
		t.Error("Expected child to have correct parent")
	}
}

func TestCompositeState_Creation(t *testing.T) {
	state := NewCompositeState("composite_state")

	if state.ID() != "composite_state" {
		t.Errorf("Expected ID 'composite_state', got '%s'", state.ID())
	}

	if !state.IsComposite() {
		t.Error("Expected composite state to be composite")
	}

	if state.IsParallel() {
		t.Error("Expected composite state not to be parallel")
	}

	if state.IsPseudo() {
		t.Error("Expected composite state not to be pseudo")
	}

	if state.InitialState() != nil {
		t.Error("Expected composite state to have no initial state initially")
	}

	if len(state.Substates()) != 0 {
		t.Error("Expected composite state to have no substates initially")
	}
}

func TestCompositeState_AddSubstate(t *testing.T) {
	composite := NewCompositeState("composite")
	child1 := NewAtomicState("child1")
	child2 := NewAtomicState("child2")

	composite.AddSubstate(child1)
	composite.AddSubstate(child2)

	substates := composite.Substates()
	if len(substates) != 2 {
		t.Errorf("Expected 2 substates, got %d", len(substates))
	}

	if child1.Parent() != composite {
		t.Error("Expected child1 to have composite as parent")
	}

	if child2.Parent() != composite {
		t.Error("Expected child2 to have composite as parent")
	}
}

func TestCompositeState_InitialState(t *testing.T) {
	composite := NewCompositeState("composite")
	initial := NewAtomicState("initial")

	composite.WithInitialState(initial)

	if composite.InitialState() != initial {
		t.Error("Expected correct initial state")
	}
}

func TestSequentialState_Creation(t *testing.T) {
	state := NewSequentialState("sequential")

	if state.ID() != "sequential" {
		t.Errorf("Expected ID 'sequential', got '%s'", state.ID())
	}

	if !state.IsComposite() {
		t.Error("Expected sequential state to be composite")
	}

	if state.IsParallel() {
		t.Error("Expected sequential state not to be parallel")
	}
}

func TestParallelState_Creation(t *testing.T) {
	state := NewParallelState("parallel")

	if state.ID() != "parallel" {
		t.Errorf("Expected ID 'parallel', got '%s'", state.ID())
	}

	if !state.IsComposite() {
		t.Error("Expected parallel state to be composite")
	}

	if !state.IsParallel() {
		t.Error("Expected parallel state to be parallel")
	}

	if len(state.Regions()) != 0 {
		t.Error("Expected parallel state to have no regions initially")
	}
}

func TestParallelState_AddRegion(t *testing.T) {
	parallel := NewParallelState("parallel")
	region1 := NewRegion("region1", parallel)
	region2 := NewRegion("region2", parallel)

	parallel.AddRegion(region1)
	parallel.AddRegion(region2)

	regions := parallel.Regions()
	if len(regions) != 2 {
		t.Errorf("Expected 2 regions, got %d", len(regions))
	}
}

func TestRegion_Creation(t *testing.T) {
	parallel := NewParallelState("parallel")
	region := NewRegion("region", parallel)

	if region.ID() != "region" {
		t.Errorf("Expected ID 'region', got '%s'", region.ID())
	}

	if region.ParentState() != parallel {
		t.Error("Expected region to have correct parent state")
	}

	if region.CurrentState() != nil {
		t.Error("Expected region to have no current state initially")
	}

	if region.InitialState() != nil {
		t.Error("Expected region to have no initial state initially")
	}

	if len(region.States()) != 0 {
		t.Error("Expected region to have no states initially")
	}
}

func TestRegion_AddState(t *testing.T) {
	parallel := NewParallelState("parallel")
	region := NewRegion("region", parallel)
	state1 := NewAtomicState("state1")
	state2 := NewAtomicState("state2")

	region.AddState(state1)
	region.AddState(state2)

	states := region.States()
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}
}

func TestRegion_InitialState(t *testing.T) {
	parallel := NewParallelState("parallel")
	region := NewRegion("region", parallel)
	initial := NewAtomicState("initial")

	region.WithInitialState(initial)

	if region.InitialState() != initial {
		t.Error("Expected correct initial state")
	}
}

func TestPseudoState_Creation(t *testing.T) {
	choice := NewPseudoState("choice", Choice)

	if choice.ID() != "choice" {
		t.Errorf("Expected ID 'choice', got '%s'", choice.ID())
	}

	if !choice.IsPseudo() {
		t.Error("Expected pseudostate to be pseudo")
	}

	if choice.IsComposite() {
		t.Error("Expected pseudostate not to be composite")
	}

	if choice.IsParallel() {
		t.Error("Expected pseudostate not to be parallel")
	}

	if choice.Kind() != Choice {
		t.Errorf("Expected kind Choice, got %v", choice.Kind())
	}
}

func TestPseudoState_AllKinds(t *testing.T) {
	testCases := []struct {
		name string
		kind PseudoStateKind
	}{
		{"initial", Initial},
		{"choice", Choice},
		{"junction", Junction},
		{"fork", Fork},
		{"join", Join},
		{"terminate", Terminate},
		{"history", History},
		{"deep_history", DeepHistory},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := NewPseudoState(tc.name, tc.kind)
			if state.Kind() != tc.kind {
				t.Errorf("Expected kind %v, got %v", tc.kind, state.Kind())
			}
		})
	}
}

func TestHistoryState_Creation(t *testing.T) {

	shallow := NewHistoryState("shallow", false)
	if shallow.Kind() != History {
		t.Errorf("Expected kind History, got %v", shallow.Kind())
	}

	deep := NewHistoryState("deep", true)
	if deep.Kind() != DeepHistory {
		t.Errorf("Expected kind DeepHistory, got %v", deep.Kind())
	}
}

func TestPseudoState_ChoiceConfiguration(t *testing.T) {
	choice := NewPseudoState("choice", Choice)

	choice.AddChoiceCondition(
		func(ctx Context) bool { return true },
		"target1",
		func(ctx Context) error { return nil },
	)

	choice.AddChoiceCondition(
		func(ctx Context) bool { return false },
		"target2",
		nil,
	)

	choice.SetDefaultTarget("default")

	if len(choice.choiceConditions) != 2 {
		t.Errorf("Expected 2 choice conditions, got %d", len(choice.choiceConditions))
	}

	if choice.defaultTarget != "default" {
		t.Errorf("Expected default target 'default', got '%s'", choice.defaultTarget)
	}
}

func TestPseudoState_ForkConfiguration(t *testing.T) {
	fork := NewPseudoState("fork", Fork)

	targets := []string{"target1", "target2", "target3"}
	fork.SetForkTargets(targets)

	if len(fork.forkTargets) != 3 {
		t.Errorf("Expected 3 fork targets, got %d", len(fork.forkTargets))
	}

	fork.AddForkTarget("target4")
	if len(fork.forkTargets) != 4 {
		t.Errorf("Expected 4 fork targets after adding one, got %d", len(fork.forkTargets))
	}
}

func TestPseudoState_JoinConfiguration(t *testing.T) {
	join := NewPseudoState("join", Join)

	sources := []string{"source1", "source2", "source3"}
	join.SetJoinSources(sources)
	join.SetJoinTarget("target")

	if len(join.joinSourceCombinations) != 1 {
		t.Errorf("Expected 1 join source combination, got %d", len(join.joinSourceCombinations))
	}

	if len(join.joinSourceCombinations[0]) != 3 {
		t.Errorf("Expected 3 join sources in combination, got %d", len(join.joinSourceCombinations[0]))
	}

	if join.joinTarget != "target" {
		t.Errorf("Expected join target 'target', got '%s'", join.joinTarget)
	}
}

func TestPseudoState_HistoryConfiguration(t *testing.T) {
	history := NewHistoryState("history", false)

	history.SetHistoryDefault("default_state")

	if history.historyDefault != "default_state" {
		t.Errorf("Expected history default 'default_state', got '%s'", history.historyDefault)
	}
}

func TestStateHierarchy_ComplexNesting(t *testing.T) {

	root := NewCompositeState("root")

	composite1 := NewCompositeState("composite1")
	atomic1 := NewAtomicState("atomic1")
	composite2 := NewCompositeState("composite2")
	atomic2 := NewAtomicState("atomic2")
	atomic3 := NewAtomicState("atomic3")

	root.AddSubstate(composite1)
	composite1.AddSubstate(atomic1)
	composite1.AddSubstate(composite2)
	composite2.AddSubstate(atomic2)
	composite2.AddSubstate(atomic3)

	parallel1 := NewParallelState("parallel1")
	region1 := NewRegion("region1", parallel1)
	atomic4 := NewAtomicState("atomic4")
	atomic5 := NewAtomicState("atomic5")
	region2 := NewRegion("region2", parallel1)
	atomic6 := NewAtomicState("atomic6")

	root.AddSubstate(parallel1)
	parallel1.AddRegion(region1)
	parallel1.AddRegion(region2)
	region1.AddState(atomic4)
	region1.AddState(atomic5)
	region2.AddState(atomic6)

	if atomic2.Parent() != composite2 {
		t.Error("Expected atomic2 to have composite2 as parent")
	}

	if composite2.Parent() != composite1 {
		t.Error("Expected composite2 to have composite1 as parent")
	}

	if composite1.Parent() != root {
		t.Error("Expected composite1 to have root as parent")
	}

	if parallel1.Parent() != root {
		t.Error("Expected parallel1 to have root as parent")
	}

	if len(root.Substates()) != 2 {
		t.Errorf("Expected root to have 2 substates, got %d", len(root.Substates()))
	}

	if len(composite1.Substates()) != 2 {
		t.Errorf("Expected composite1 to have 2 substates, got %d", len(composite1.Substates()))
	}

	if len(composite2.Substates()) != 2 {
		t.Errorf("Expected composite2 to have 2 substates, got %d", len(composite2.Substates()))
	}

	if len(parallel1.Regions()) != 2 {
		t.Errorf("Expected parallel1 to have 2 regions, got %d", len(parallel1.Regions()))
	}

	if len(region1.States()) != 2 {
		t.Errorf("Expected region1 to have 2 states, got %d", len(region1.States()))
	}

	if len(region2.States()) != 1 {
		t.Errorf("Expected region2 to have 1 state, got %d", len(region2.States()))
	}
}

func TestStateActions_EntryExit(t *testing.T) {
	actionCallCount := 0
	entryAction := func(ctx Context) error {
		actionCallCount++
		return nil
	}
	exitAction := func(ctx Context) error {
		actionCallCount += 10
		return nil
	}

	state := NewAtomicState("test").
		WithEntryAction(entryAction).
		WithExitAction(exitAction)

	ctx := CreateTestContext()

	actionCallCount = 0
	state.Enter(ctx)
	if actionCallCount != 1 {
		t.Errorf("Expected entry action to be called once, got %d", actionCallCount)
	}

	actionCallCount = 0
	state.Exit(ctx)
	if actionCallCount != 10 {
		t.Errorf("Expected exit action to be called once, got %d", actionCallCount)
	}
}

func TestStateIntegration_WithMachine(t *testing.T) {

	builder := NewMachine()

	builder.State("root").Initial().To("parent").On("enter_parent")

	parentBuilder := builder.CompositeState("parent")
	parentBuilder.State("child1").Initial()
	parentBuilder.State("child2")

	definition := builder.Build()
	machine := definition.CreateInstance()

	_ = machine.Start()
	AssertState(t, machine, "root")

	builder2 := NewMachine()
	builder2.State("start").Initial().To("parent.child1").On("go_nested")
	parent2 := builder2.CompositeState("parent")
	parent2.State("child1").Initial().
		To("child2").On("switch")
	parent2.State("child2")

	definition2 := builder2.Build()
	machine2 := definition2.CreateInstance()
	_ = machine2.Start()

	result := machine2.HandleEvent("go_nested", nil)
	AssertEventProcessed(t, result, true)

	currentState := machine2.CurrentState()
	if currentState == "" {
		t.Error("Expected machine to have a current state after hierarchical transition")
	}
}

func TestStateBehavior_MultipleParents(t *testing.T) {

	parent1 := NewCompositeState("parent1")
	parent2 := NewCompositeState("parent2")
	child := NewAtomicState("child")

	parent1.AddSubstate(child)
	if child.Parent() != parent1 {
		t.Error("Expected child to have parent1 as parent initially")
	}

	parent2.AddSubstate(child)
	if child.Parent() != parent2 {
		t.Error("Expected child to have parent2 as parent after adding to parent2")
	}

	parent1Substates := parent1.Substates()
	found := false
	for _, substate := range parent1Substates {
		if substate == child {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected parent1 to still contain child in substates")
	}
}
