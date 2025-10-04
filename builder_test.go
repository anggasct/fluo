package fluo

import (
	"testing"
)

func TestMachineBuilder_BasicCreation(t *testing.T) {
	builder := NewMachine()
	if builder == nil {
		t.Error("Expected non-nil machine builder")
	}
}

func TestMachineBuilder_SimpleStateMachine(t *testing.T) {
	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").
		State("running").
		To("idle").On("stop").
		Build()

	if definition == nil {
		t.Error("Expected non-nil machine definition")
	}

	machine := definition.CreateInstance()
	if machine == nil {
		t.Error("Expected non-nil machine instance")
	}

	_ = machine.Start()
	AssertState(t, machine, "idle")

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "running")
}

func TestMachineBuilder_StateConfiguration(t *testing.T) {
	entryActionCalled := false
	exitActionCalled := false

	entryAction := func(ctx Context) error {
		entryActionCalled = true
		return nil
	}

	exitAction := func(ctx Context) error {
		exitActionCalled = true
		return nil
	}

	definition := NewMachine().
		State("test").Initial().
		OnEntry(entryAction).
		OnExit(exitAction).
		To("next").On("go").
		State("next").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	if !entryActionCalled {
		t.Error("Expected entry action to be called on start")
	}

	entryActionCalled = false
	exitActionCalled = false

	_ = machine.HandleEvent("go", nil)

	if !exitActionCalled {
		t.Error("Expected exit action to be called on transition")
	}
}

func TestMachineBuilder_FinalState(t *testing.T) {
	builder := NewMachine()
	builder.State("running").Initial().
		To("finished").On("finish")
	builder.State("finished").Final()
	definition := builder.Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("finish", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "finished")

	states := definition.GetStates()
	if finalState, exists := states["finished"]; exists {
		if !finalState.IsFinal() {
			t.Error("Expected 'finished' to be a final state")
		}
	} else {
		t.Error("Expected to find 'finished' state in definition")
	}
}

func TestMachineBuilder_TransitionConfiguration(t *testing.T) {
	actionCalled := false
	guardCondition := true

	action := func(ctx Context) error {
		actionCalled = true
		return nil
	}

	guard := func(ctx Context) bool {
		return guardCondition
	}

	definition := NewMachine().
		State("start").Initial().
		To("end").On("go").When(guard).Do(action).
		State("end").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	guardCondition = false
	result := machine.HandleEvent("go", nil)
	AssertEventProcessed(t, result, false)
	AssertState(t, machine, "start")

	if actionCalled {
		t.Error("Expected action not to be called when guard returns false")
	}

	guardCondition = true
	result = machine.HandleEvent("go", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "end")

	if !actionCalled {
		t.Error("Expected action to be called when guard returns true")
	}
}

func TestMachineBuilder_MultipleTransitions(t *testing.T) {
	definition := NewMachine().
		State("start").Initial().
		To("path_a").On("go_a").
		To("path_b").On("go_b").
		To("end").On("finish").
		State("path_a").
		To("end").On("finish").
		State("path_b").
		To("end").On("finish").
		State("end").
		Build()

	machine1 := definition.CreateInstance()
	_ = machine1.Start()
	_ = machine1.HandleEvent("go_a", nil)
	AssertState(t, machine1, "path_a")
	_ = machine1.HandleEvent("finish", nil)
	AssertState(t, machine1, "end")

	machine2 := definition.CreateInstance()
	_ = machine2.Start()
	_ = machine2.HandleEvent("go_b", nil)
	AssertState(t, machine2, "path_b")
	_ = machine2.HandleEvent("finish", nil)
	AssertState(t, machine2, "end")

	machine3 := definition.CreateInstance()
	_ = machine3.Start()
	_ = machine3.HandleEvent("finish", nil)
	AssertState(t, machine3, "end")
}

func TestMachineBuilder_SelfTransition(t *testing.T) {
	actionCalled := false
	action := func(ctx Context) error {
		actionCalled = true
		return nil
	}

	definition := NewMachine().
		State("active").Initial().
		ToSelf().On("refresh").Do(action).
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)
	_ = machine.Start()

	observer.Reset()

	result := machine.HandleEvent("refresh", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "active")

	if !actionCalled {
		t.Error("Expected self-transition action to be called")
	}

	if observer.StateExitCount() != 1 || observer.StateEnterCount() != 1 {
		t.Error("Expected exit and enter for self-transition")
	}
}

func TestMachineBuilder_CompositeState(t *testing.T) {

	builder := NewMachine()

	builder.State("simple").Initial()

	compositeBuilder := builder.CompositeState("composite")
	compositeBuilder.OnEntry(func(ctx Context) error {

		return nil
	})

	definition := builder.Build()
	if definition == nil {
		t.Error("Expected non-nil machine definition")
	}

	states := definition.GetStates()
	if compositeState, exists := states["composite"]; exists {
		if !compositeState.IsComposite() {
			t.Error("Expected composite state to be composite")
		}
	}
}

func TestMachineBuilder_ParallelState(t *testing.T) {

	builder := NewMachine()

	builder.State("simple").Initial()

	parallelBuilder := builder.ParallelState("parallel")
	parallelBuilder.OnEntry(func(ctx Context) error { return nil })

	definition := builder.Build()
	if definition == nil {
		t.Error("Expected non-nil machine definition")
	}

	states := definition.GetStates()
	if parallelState, exists := states["parallel"]; exists {
		if !parallelState.IsParallel() {
			t.Error("Expected parallel state to be parallel")
		}
		if !parallelState.IsComposite() {
			t.Error("Expected parallel state to be composite")
		}
	}
}

func TestMachineBuilder_ChoicePseudostate(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("decision").On("decide")

	builder.Choice("decision").
		When(func(ctx Context) bool {
			if value, ok := ctx.Get("path"); ok {
				return value.(string) == "A"
			}
			return false
		}).To("path_a").
		Otherwise("path_b")

	builder.State("path_a")
	builder.State("path_b")

	definition := builder.Build()

	machine1 := definition.CreateInstance()
	machine1.Context().Set("path", "A")
	_ = machine1.Start()

	result1 := machine1.HandleEvent("decide", nil)
	AssertEventProcessed(t, result1, true)
	if machine1.CurrentState() == "" {
		t.Error("Expected machine to have resolved choice to a state")
	}

	machine2 := definition.CreateInstance()
	machine2.Context().Set("path", "B")
	_ = machine2.Start()

	result2 := machine2.HandleEvent("decide", nil)
	AssertEventProcessed(t, result2, true)
	if machine2.CurrentState() == "" {
		t.Error("Expected machine to have resolved choice to default state")
	}
}

func TestMachineBuilder_JunctionPseudostate(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("junction1").On("go")

	builder.Junction("junction1").
		To("end")

	builder.State("end")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("go", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "end")
}

func TestMachineBuilder_ForkPseudostate(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("fork1").On("split")

	builder.Fork("fork1").
		To("path1", "path2")

	builder.State("path1")
	builder.State("path2")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("split", nil)
	AssertEventProcessed(t, result, true)

	activeStates := machine.GetActiveStates()
	if len(activeStates) < 2 {
		t.Errorf("Expected at least 2 active states after fork, got %d", len(activeStates))
	}
}

func TestMachineBuilder_JoinPseudostate(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("fork1").On("split")

	builder.Fork("fork1").
		To("path1", "path2")

	builder.State("path1").
		To("join1").On("sync")

	builder.State("path2").
		To("join1").On("sync")

	builder.Join("join1").
		From("path1", "path2").
		To("end")

	builder.State("end")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	_ = machine.HandleEvent("split", nil)

	states := definition.GetStates()
	if joinState, exists := states["join1"]; !exists || !joinState.IsPseudo() {
		t.Error("Expected join1 to be a pseudostate")
	}
}

func TestMachineBuilder_HistoryPseudostate(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("composite1").On("enter")

	builder.CompositeState("composite1").
		State("composite1.state1").Initial().
		To("composite1.state2").On("next").
		State("composite1.state2").
		History("composite1.history").Default("composite1.state1")

	definition := builder.Build()

	states := definition.GetStates()
	if historyState, exists := states["composite1.history"]; !exists || !historyState.IsPseudo() {
		t.Error("Expected composite1.history to be a pseudostate")
	}
}

func TestMachineBuilder_DeepHistoryPseudostate(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial()

	builder.DeepHistory("deep_history").
		Default("start")

	definition := builder.Build()

	states := definition.GetStates()
	if historyState, exists := states["deep_history"]; exists && historyState.IsPseudo() {
		if pseudoState, ok := historyState.(PseudoState); ok {
			if pseudoState.Kind() != DeepHistory {
				t.Error("Expected deep_history to be DeepHistory kind")
			}
		}
	} else {
		t.Error("Expected deep_history to be a pseudostate")
	}
}

func TestMachineBuilder_FluentChaining(t *testing.T) {

	definition := NewMachine().
		State("s1").Initial().OnEntry(TestAction).OnExit(TestAction).
		To("s2").On("next").When(TestGuard).Do(TestAction).
		State("s2").
		ToSelf().On("refresh").
		To("s3").On("forward").
		State("s3").Final().
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	if machine.CurrentState() != "s1" {
		t.Error("Expected machine to start in s1")
	}
}

func TestMachineBuilder_ValidationErrors(t *testing.T) {

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when building machine without initial state")
		}
	}()

	NewMachine().
		State("state1").
		Build()
}

func TestMachineBuilder_MachineDefinitionReuse(t *testing.T) {
	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").
		State("running").
		To("idle").On("stop").
		Build()

	machine1 := definition.CreateInstance()
	machine2 := definition.CreateInstance()

	_ = machine1.Start()
	_ = machine2.Start()

	_ = machine1.HandleEvent("start", nil)
	AssertState(t, machine1, "running")
	AssertState(t, machine2, "idle")

	_ = machine2.HandleEvent("start", nil)
	AssertState(t, machine1, "running")
	AssertState(t, machine2, "running")
}

func TestMachineBuilder_GettersAndMetadata(t *testing.T) {
	definition := NewMachine().
		State("s1").Initial().
		To("s2").On("go").
		State("s2").
		Build()

	if definition.GetInitialState() != "s1" {
		t.Error("Expected initial state to be s1")
	}

	states := definition.GetStates()
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}

	if _, exists := states["s1"]; !exists {
		t.Error("Expected s1 to exist in states")
	}

	if _, exists := states["s2"]; !exists {
		t.Error("Expected s2 to exist in states")
	}

	transitions := definition.GetTransitions()
	if len(transitions) == 0 {
		t.Error("Expected transitions to be present")
	}
}

func TestMachineBuilder_ComplexHierarchy(t *testing.T) {

	builder := NewMachine()

	builder.State("simple").Initial()

	compositeBuilder1 := builder.CompositeState("composite1")
	compositeBuilder1.OnEntry(func(ctx Context) error {
		ctx.Set("composite1_entered", true)
		return nil
	})

	compositeBuilder2 := builder.CompositeState("composite2")
	compositeBuilder2.OnEntry(func(ctx Context) error {
		ctx.Set("composite2_entered", true)
		return nil
	})

	definition := builder.Build()
	if definition == nil {
		t.Error("Expected successful build of complex hierarchy")
	}

	states := definition.GetStates()
	if len(states) < 3 {
		t.Error("Expected multiple states in complex hierarchy")
	}

	if state1, exists := states["composite1"]; exists && !state1.IsComposite() {
		t.Error("Expected composite1 to be composite")
	}

	if state2, exists := states["composite2"]; exists && !state2.IsComposite() {
		t.Error("Expected composite2 to be composite")
	}
}

func TestMachineBuilder_ErrorHandling(t *testing.T) {

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when referencing non-existent target state")
		}
	}()

	NewMachine().
		State("start").Initial().
		To("nonexistent").On("go").
		Build()
}

func TestMachineBuilder_NewMachineDefinition(t *testing.T) {
	// Test alternative constructor
	builder := NewMachineDefinition()
	if builder == nil {
		t.Error("Expected non-nil machine builder from NewMachineDefinition")
	}

	// Test that it creates the same type as NewMachine
	definition := builder.
		State("initial").Initial().
		State("final").
		To("final").On("complete").
		Build()

	if definition == nil {
		t.Error("Expected successful build from NewMachineDefinition")
	}

	states := definition.GetStates()
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}

	if _, exists := states["initial"]; !exists {
		t.Error("Expected 'initial' state to exist")
	}

	if _, exists := states["final"]; !exists {
		t.Error("Expected 'final' state to exist")
	}
}

func TestMachineBuilder_ToParent(t *testing.T) {
	// Test ToParent functionality with nested states
	builder := NewMachine()

	// Create top-level states
	builder.State("top_level").Initial()
	builder.State("target_state")

	// Create composite state
	composite := builder.CompositeState("composite")
	composite.State("sub_initial").Initial()
	composite.State("sub_state")

	// Test ToParent from nested state to top level
	composite.State("sub_state").
		ToParent("target_state").On("go_to_parent")

	definition := builder.Build()
	if definition == nil {
		t.Error("Expected successful build with ToParent transition")
	}

	// Verify the states were created correctly
	states := definition.GetStates()
	if len(states) < 3 {
		t.Errorf("Expected at least 3 states, got %d", len(states))
	}

	// Check that top-level states exist
	if _, exists := states["top_level"]; !exists {
		t.Error("Expected 'top_level' state to exist")
	}

	if _, exists := states["target_state"]; !exists {
		t.Error("Expected 'target_state' state to exist")
	}

	// Check that composite state exists
	compositeState, exists := states["composite"]
	if !exists {
		t.Error("Expected 'composite' state to exist")
		return
	}

	if !compositeState.IsComposite() {
		t.Error("Expected composite state to be composite")
	}

	// Test that machine can be created and started
	machine := definition.CreateInstance()
	if machine == nil {
		t.Error("Expected non-nil machine instance")
		return
	}

	err := machine.Start()
	if err != nil {
		t.Errorf("Expected machine to start successfully, got error: %v", err)
	}
}
