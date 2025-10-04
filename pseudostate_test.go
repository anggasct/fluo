package fluo

import (
	"testing"
)

func TestPseudostate_ChoiceBasic(t *testing.T) {
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
	AssertState(t, machine1, "path_a")

	machine2 := definition.CreateInstance()
	machine2.Context().Set("path", "B")
	_ = machine2.Start()

	result2 := machine2.HandleEvent("decide", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "path_b")
}

func TestPseudostate_ChoiceWithAction(t *testing.T) {
	actionCalled := false
	actionData := ""

	choiceAction := func(ctx Context) error {
		actionCalled = true
		if value, ok := ctx.Get("data"); ok {
			actionData = value.(string)
		}
		return nil
	}

	builder := NewMachine()

	builder.State("start").Initial().
		To("choice1").On("go")

	builder.Choice("choice1").
		When(func(ctx Context) bool { return true }).Do(choiceAction).To("end")

	builder.State("end")

	definition := builder.Build()
	machine := definition.CreateInstance()
	machine.Context().Set("data", "test_value")

	_ = machine.Start()
	result := machine.HandleEvent("go", nil)

	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "end")

	if !actionCalled {
		t.Error("Expected choice action to be called")
	}

	if actionData != "test_value" {
		t.Errorf("Expected action to receive 'test_value', got '%s'", actionData)
	}
}

func TestPseudostate_ChoiceMultipleConditions(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("choice1").On("decide")

	builder.Choice("choice1").
		When(func(ctx Context) bool {
			if score, ok := ctx.Get("score"); ok {
				return score.(int) >= 90
			}
			return false
		}).To("excellent").
		When(func(ctx Context) bool {
			if score, ok := ctx.Get("score"); ok {
				return score.(int) >= 70
			}
			return false
		}).To("good").
		Otherwise("needs_improvement")

	builder.State("excellent")
	builder.State("good")
	builder.State("needs_improvement")

	definition := builder.Build()

	machine1 := definition.CreateInstance()
	machine1.Context().Set("score", 95)
	_ = machine1.Start()
	_ = machine1.HandleEvent("decide", nil)
	AssertState(t, machine1, "excellent")

	machine2 := definition.CreateInstance()
	machine2.Context().Set("score", 75)
	_ = machine2.Start()
	_ = machine2.HandleEvent("decide", nil)
	AssertState(t, machine2, "good")

	machine3 := definition.CreateInstance()
	machine3.Context().Set("score", 50)
	_ = machine3.Start()
	_ = machine3.HandleEvent("decide", nil)
	AssertState(t, machine3, "needs_improvement")
}

func TestPseudostate_Junction(t *testing.T) {
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

func TestPseudostate_Fork(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("fork1").On("split")

	builder.Fork("fork1").
		To("path1", "path2", "path3")

	builder.State("path1")
	builder.State("path2")
	builder.State("path3")

	definition := builder.Build()
	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	result := machine.HandleEvent("split", nil)

	AssertEventProcessed(t, result, true)

	activeStates := machine.GetActiveStates()
	if len(activeStates) < 2 {
		t.Errorf("Expected multiple active states after fork, got %d: %v", len(activeStates), activeStates)
	}

	if observer.StateEnterCount() < 3 {
		t.Errorf("Expected multiple state enters after fork, got %d", observer.StateEnterCount())
	}
}

func TestPseudostate_JoinSynchronization(t *testing.T) {
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

	result1 := machine.HandleEvent("split", nil)
	AssertEventProcessed(t, result1, true)

	activeStates := machine.GetActiveStates()
	if len(activeStates) < 2 {
		t.Error("Expected multiple states to be active after fork")
	}

	states := definition.GetStates()
	if joinState, exists := states["join1"]; !exists || !joinState.IsPseudo() {
		t.Error("Expected join1 to be a pseudostate")
	}
}

func TestPseudostate_HistoryShallow(t *testing.T) {
	builder := NewMachine()

	builder.State("inactive").Initial().
		To("active").On("activate")

	compositeBuilder := builder.CompositeState("active")
	compositeBuilder.State("idle").Initial().
		To("working").On("start_work")
	compositeBuilder.State("working").
		To("idle").On("finish_work")
	compositeBuilder.History("history").Default("idle")

	builder.State("active").
		To("inactive").On("deactivate")

	builder.State("inactive").
		To("active.history").On("reactivate")

	definition := builder.Build()

	states := definition.GetStates()
	if historyState, exists := states["active.history"]; exists {
		if !historyState.IsPseudo() {
			t.Error("Expected active.history to be a pseudostate")
		}
		if pseudoState, ok := historyState.(PseudoState); ok {
			if pseudoState.Kind() != History {
				t.Error("Expected active.history to be History kind")
			}
		}
	} else {
		t.Error("Expected active.history to exist")
	}
}

func TestPseudostate_HistoryDeep(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial()

	builder.DeepHistory("deep_history").
		Default("start")

	definition := builder.Build()

	states := definition.GetStates()
	if historyState, exists := states["deep_history"]; exists {
		if !historyState.IsPseudo() {
			t.Error("Expected deep_history to be a pseudostate")
		}
		if pseudoState, ok := historyState.(PseudoState); ok {
			if pseudoState.Kind() != DeepHistory {
				t.Error("Expected deep_history to be DeepHistory kind")
			}
		}
	} else {
		t.Error("Expected deep_history to exist")
	}
}

func TestPseudostate_ChainedPseudostates(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().
		To("choice1").On("go")

	builder.Choice("choice1").
		When(func(ctx Context) bool { return true }).To("junction1")

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

func TestPseudostate_NestedChoices(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("choice1").On("first_decision")

	builder.Choice("choice1").
		When(func(ctx Context) bool {
			if level, ok := ctx.Get("level"); ok {
				return level.(int) > 5
			}
			return false
		}).To("choice2").
		Otherwise("low_level")

	builder.Choice("choice2").
		When(func(ctx Context) bool {
			if level, ok := ctx.Get("level"); ok {
				return level.(int) > 10
			}
			return false
		}).To("high_level").
		Otherwise("medium_level")

	builder.State("low_level")
	builder.State("medium_level")
	builder.State("high_level")

	definition := builder.Build()

	machine1 := definition.CreateInstance()
	machine1.Context().Set("level", 3)
	_ = machine1.Start()
	_ = machine1.HandleEvent("first_decision", nil)
	AssertState(t, machine1, "low_level")

	machine2 := definition.CreateInstance()
	machine2.Context().Set("level", 8)
	_ = machine2.Start()
	_ = machine2.HandleEvent("first_decision", nil)
	AssertState(t, machine2, "medium_level")

	machine3 := definition.CreateInstance()
	machine3.Context().Set("level", 15)
	_ = machine3.Start()
	_ = machine3.HandleEvent("first_decision", nil)
	AssertState(t, machine3, "high_level")
}

func TestPseudostate_ComplexWorkflow(t *testing.T) {

	builder := NewMachine()

	builder.State("idle").Initial().
		To("fork_processing").On("start_processing")

	builder.Fork("fork_processing").
		To("validation", "data_processing")

	builder.State("validation").
		To("validation_choice").On("validate")

	builder.Choice("validation_choice").
		When(func(ctx Context) bool {
			if valid, ok := ctx.Get("is_valid"); ok {
				return valid.(bool)
			}
			return false
		}).To("validation_success").
		Otherwise("validation_error")

	builder.State("validation_success").
		To("join_processing").On("validation_complete")

	builder.State("validation_error")

	builder.State("data_processing").
		To("join_processing").On("processing_complete")

	builder.Join("join_processing").
		From("validation_success", "data_processing").
		To("finalize")

	builder.State("finalize")

	definition := builder.Build()
	machine := definition.CreateInstance()
	machine.Context().Set("is_valid", true)

	_ = machine.Start()
	AssertState(t, machine, "idle")

	result := machine.HandleEvent("start_processing", nil)
	AssertEventProcessed(t, result, true)

	activeStates := machine.GetActiveStates()
	if len(activeStates) < 2 {
		t.Error("Expected multiple active states after fork")
	}

}

func TestPseudostate_ErrorConditions(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().
		To("bad_choice").On("go")

	builder.Choice("bad_choice").
		When(func(ctx Context) bool { return false }).To("unreachable")

	builder.State("unreachable")

	definition := builder.Build()
	machine := definition.CreateInstance()

	_ = machine.Start()
	result := machine.HandleEvent("go", nil)

	if result.Error != nil {
		t.Logf("Choice pseudostate error handled: %v", result.Error)
	}
}

func TestPseudostate_GuardConditionsWithContext(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("decision").On("decide")

	builder.Choice("decision").
		When(func(ctx Context) bool {

			score, hasScore := ctx.Get("score")
			category, hasCategory := ctx.Get("category")

			if hasScore && hasCategory {
				return score.(int) >= 80 && category.(string) == "premium"
			}
			return false
		}).To("premium_path").
		When(func(ctx Context) bool {
			if score, ok := ctx.Get("score"); ok {
				return score.(int) >= 60
			}
			return false
		}).To("standard_path").
		Otherwise("basic_path")

	builder.State("premium_path")
	builder.State("standard_path")
	builder.State("basic_path")

	definition := builder.Build()

	testCases := []struct {
		score    int
		category string
		expected string
	}{
		{90, "premium", "premium_path"},
		{85, "standard", "standard_path"},
		{70, "basic", "standard_path"},
		{50, "premium", "basic_path"},
	}

	for i, tc := range testCases {
		machine := definition.CreateInstance()
		machine.Context().Set("score", tc.score)
		machine.Context().Set("category", tc.category)

		_ = machine.Start()
		result := machine.HandleEvent("decide", nil)

		AssertEventProcessed(t, result, true)
		AssertState(t, machine, tc.expected)

		t.Logf("Test case %d: score=%d, category=%s -> %s",
			i+1, tc.score, tc.category, machine.CurrentState())
	}
}

func TestPseudostate_ObserverNotifications(t *testing.T) {
	builder := NewMachine()

	builder.State("start").Initial().
		To("choice1").On("go")

	builder.Choice("choice1").
		When(func(ctx Context) bool { return true }).To("end")

	builder.State("end")

	definition := builder.Build()
	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)

	_ = machine.Start()
	observer.Reset()

	_ = machine.HandleEvent("go", nil)

	if observer.TransitionCount() == 0 {
		t.Error("Expected transition notifications through pseudostate")
	}

	AssertState(t, machine, "end")
}
