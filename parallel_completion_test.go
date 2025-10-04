package fluo

import (
	"testing"
)

func TestParallelStateAutomaticCompletion(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().
		To("parallel_work").On("begin")

	parallel := builder.ParallelState("parallel_work")

	region1 := parallel.Region("region1")
	region1.State("r1_init").Initial().
		To("r1_working").On("r1_start")
	region1.State("r1_working").
		To("r1_final").On("r1_complete")
	region1.State("r1_final").Final()

	region2 := parallel.Region("region2")
	region2.State("r2_init").Initial().
		To("r2_working").On("r2_start")
	region2.State("r2_working").
		To("r2_final").On("r2_complete")
	region2.State("r2_final").Final()

	builder.ParallelState("parallel_work").
		To("completed").OnCompletion()

	builder.State("completed")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("begin", nil)
	AssertEventProcessed(t, result, true)

	t.Logf("After entering parallel - Active states: %v", machine.GetActiveStates())
	t.Logf("Parallel regions: %v", machine.GetParallelRegions())

	if !machine.IsStateActive("parallel_work.region1.r1_init") {
		t.Error("Expected region1 to be in initial state")
	}
	if !machine.IsStateActive("parallel_work.region2.r2_init") {
		t.Error("Expected region2 to be in initial state")
	}

	result1 := machine.HandleEvent("r1_start", nil)
	t.Logf("r1_start result: %v -> %v", result1.PreviousState, result1.CurrentState)
	result2 := machine.HandleEvent("r2_start", nil)
	t.Logf("r2_start result: %v -> %v", result2.PreviousState, result2.CurrentState)

	t.Logf("After starting both: Active states: %v", machine.GetActiveStates())

	result3 := machine.HandleEvent("r1_complete", nil)
	t.Logf("r1_complete result: %v -> %v (StateChanged: %v)", result3.PreviousState, result3.CurrentState, result3.StateChanged)

	t.Logf("After r1_complete - Active states: %v", machine.GetActiveStates())
	t.Logf("r1_final active: %v", machine.IsStateActive("parallel_work.region1.r1_final"))
	t.Logf("r2 still working: %v", machine.IsStateActive("parallel_work.region2.r2_working"))

	if !machine.IsStateActive("parallel_work") {
		t.Error("Parallel state should still be active after one region completes")
	}

	_ = machine.HandleEvent("r2_complete", nil)

	t.Logf("After r2_complete - Active states: %v", machine.GetActiveStates())

	if !machine.IsStateActive("completed") {
		t.Error("Expected automatic transition to completed state")
	}
	if machine.IsStateActive("parallel_work") {
		t.Error("Parallel state should no longer be active")
	}
}

func TestParallelStateCompletionWithGuard(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().
		To("parallel_processing").On("start")

	parallel := builder.ParallelState("parallel_processing")

	r1 := parallel.Region("validator")
	r1.State("validating").Initial().
		To("valid").On("validate_ok").Do(func(ctx Context) error {
		ctx.Set("validation_passed", true)
		return nil
	})
	r1.State("valid").Final()

	r2 := parallel.Region("processor")
	r2.State("processing").Initial().
		To("processed").On("process_done")
	r2.State("processed").Final()

	builder.ParallelState("parallel_processing").
		To("success").OnCompletion().When(func(ctx Context) bool {
		passed, _ := ctx.Get("validation_passed")
		return passed == true
	})

	builder.ParallelState("parallel_processing").
		To("failure").OnCompletion().When(func(ctx Context) bool {
		passed, _ := ctx.Get("validation_passed")
		return passed != true
	})

	builder.State("success")
	builder.State("failure")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	_ = machine.HandleEvent("start", nil)

	_ = machine.HandleEvent("validate_ok", nil)

	_ = machine.HandleEvent("process_done", nil)

	if !machine.IsStateActive("success") {
		t.Error("Expected transition to success state")
	}
}

func TestParallelStateNoCompletionWithoutFinalStates(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().
		To("parallel").On("begin")

	parallel := builder.ParallelState("parallel")

	r1 := parallel.Region("r1")
	r1.State("active").Initial()
	r1.State("done")

	builder.ParallelState("parallel").
		To("completed").OnCompletion()

	builder.State("completed")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	_ = machine.HandleEvent("begin", nil)

	if machine.IsStateActive("completed") {
		t.Error("Should not transition without final states")
	}
}

func TestParallelStateNestedCompletion(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().
		To("outer_parallel").On("start")

	outer := builder.ParallelState("outer_parallel")

	r1 := outer.Region("simple")
	r1.State("working").Initial().
		To("done").On("finish_simple")
	r1.State("done").Final()

	r2 := outer.Region("complex")
	r2.State("init").Initial().
		To("processing").On("start_complex")
	r2.State("processing").
		To("complex_done").On("finish_complex")
	r2.State("complex_done").Final()

	builder.ParallelState("outer_parallel").
		To("all_done").OnCompletion()

	builder.State("all_done")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	_ = machine.HandleEvent("start", nil)

	_ = machine.HandleEvent("finish_simple", nil)

	_ = machine.HandleEvent("start_complex", nil)
	_ = machine.HandleEvent("finish_complex", nil)

	if !machine.IsStateActive("all_done") {
		t.Error("Expected full completion through nested parallel states")
	}
}

func TestParallelStateMultipleCompletionTransitions(t *testing.T) {

	builder := NewMachine()

	builder.State("start").Initial().
		To("parallel").On("begin")

	parallel := builder.ParallelState("parallel")

	r1 := parallel.Region("counter1")
	r1.State("counting").Initial().
		To("done").On("count1_done").Do(func(ctx Context) error {
		ctx.Set("count1", 10)
		return nil
	})
	r1.State("done").Final()

	r2 := parallel.Region("counter2")
	r2.State("counting").Initial().
		To("done").On("count2_done").Do(func(ctx Context) error {
		ctx.Set("count2", 20)
		return nil
	})
	r2.State("done").Final()

	builder.ParallelState("parallel").
		To("low_total").OnCompletion().When(func(ctx Context) bool {
		c1, _ := ctx.Get("count1")
		c2, _ := ctx.Get("count2")
		count1, ok1 := c1.(int)
		count2, ok2 := c2.(int)
		if !ok1 || !ok2 {
			return false
		}
		return count1+count2 <= 25
	})

	builder.ParallelState("parallel").
		To("high_total").OnCompletion().When(func(ctx Context) bool {
		c1, _ := ctx.Get("count1")
		c2, _ := ctx.Get("count2")
		count1, ok1 := c1.(int)
		count2, ok2 := c2.(int)
		if !ok1 || !ok2 {
			return false
		}
		return count1+count2 > 25
	})

	builder.State("high_total")
	builder.State("low_total")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	_ = machine.HandleEvent("begin", nil)
	_ = machine.HandleEvent("count1_done", nil)
	_ = machine.HandleEvent("count2_done", nil)

	t.Logf("Active states after completion: %v", machine.GetActiveStates())
	ctx := machine.Context()
	c1, _ := ctx.Get("count1")
	c2, _ := ctx.Get("count2")
	t.Logf("count1=%v (type %T), count2=%v (type %T)", c1, c1, c2, c2)

	if !machine.IsStateActive("high_total") {
		t.Error("Expected transition to high_total state")
	}
}
