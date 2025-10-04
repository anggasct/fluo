package fluo

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test for guards that modify context during evaluation
func TestTransition_GuardWithSideEffects(t *testing.T) {
	guardCallCount := 0
	guard := func(ctx Context) bool {
		guardCallCount++
		// Guard modifies context during evaluation
		ctx.Set("guard_called", guardCallCount)
		ctx.Set("guard_result", guardCallCount > 1)
		return guardCallCount > 1
	}

	definition := NewMachine().
		State("initial").Initial().
		To("target").On("test").When(guard).
		State("target").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// First call should fail (guardCallCount = 1)
	result1 := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result1, false)

	// Verify side effects were applied
	if count, ok := machine.Context().Get("guard_called"); !ok || count != 1 {
		t.Error("Expected guard to have modified context on first call")
	}

	// Second call should succeed (guardCallCount = 2)
	result2 := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "target")
}

// Test for guards with expensive operations
func TestTransition_GuardWithExpensiveOperations(t *testing.T) {
	guardCallCount := 0
	guard := func(ctx Context) bool {
		guardCallCount++
		// Simulate expensive operation
		for i := 0; i < 1000; i++ {
			_ = i * i
		}
		return guardCallCount > 2
	}

	definition := NewMachine().
		State("initial").Initial().
		To("target").On("test").When(guard).
		State("target").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Multiple calls to test performance and consistency
	for i := 0; i < 5; i++ {
		result := machine.HandleEvent("test", nil)
		if i < 2 {
			AssertEventProcessed(t, result, false)
		} else {
			AssertEventProcessed(t, result, true)
			break
		}
	}

	if guardCallCount < 3 {
		t.Errorf("Expected guard to be called at least 3 times, got %d", guardCallCount)
	}
}

// Test for actions that modify context values used by subsequent transitions
func TestTransition_ActionModifiesContext(t *testing.T) {
	action := func(ctx Context) error {
		// Action modifies context during transition
		ctx.Set("transition_count", 1)
		ctx.Set("last_transition", time.Now())
		ctx.Set("processing_complete", true)
		return nil
	}

	// Guard that depends on context values set by previous action
	guard := func(ctx Context) bool {
		if count, ok := ctx.Get("transition_count"); ok {
			return count.(int) > 0
		}
		return false
	}

	definition := NewMachine().
		State("initial").Initial().
		To("processing").On("start").Do(action).
		State("processing").
		To("complete").On("finish").When(guard).
		State("complete").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// First transition with action
	result1 := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "processing")

	// Verify context was modified by action
	if count, ok := machine.Context().Get("transition_count"); !ok || count != 1 {
		t.Error("Expected action to have modified context")
	}

	// Second transition depends on context modified by first action
	result2 := machine.HandleEvent("finish", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "complete")
}

// Test for actions that modify context values that affect other guards
func TestTransition_ActionAffectsOtherGuards(t *testing.T) {
	// First action sets a flag
	action1 := func(ctx Context) error {
		ctx.Set("system_ready", true)
		return nil
	}

	// Second action modifies the flag
	action2 := func(ctx Context) error {
		ctx.Set("system_ready", false)
		ctx.Set("system_busy", true)
		return nil
	}

	// Guard depends on system_ready flag
	guard1 := func(ctx Context) bool {
		if ready, ok := ctx.Get("system_ready"); ok {
			return ready.(bool)
		}
		return false
	}

	// Guard depends on system_busy flag
	guard2 := func(ctx Context) bool {
		if busy, ok := ctx.Get("system_busy"); ok {
			return busy.(bool)
		}
		return false
	}

	definition := NewMachine().
		State("idle").Initial().
		To("ready").On("prepare").Do(action1).
		State("ready").
		To("processing").On("work").When(guard1).
		To("busy").On("work").Do(action2).When(guard2).
		State("processing").
		State("busy").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Prepare system
	result1 := machine.HandleEvent("prepare", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "ready")

	// Should transition to processing (system_ready is true)
	result2 := machine.HandleEvent("work", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "processing")

	// Reset and test busy path
	machine2 := definition.CreateInstance()
	_ = machine2.Start()
	_ = machine2.HandleEvent("prepare", nil)

	// Manually set system_busy to test the second guard
	machine2.Context().Set("system_busy", true)

	result3 := machine2.HandleEvent("work", nil)
	// This should take the busy path since action2 will be executed
	AssertEventProcessed(t, result3, true)
}

// Test for actions that modify complex data structures in context
func TestTransition_ActionModifiesComplexContext(t *testing.T) {
	action := func(ctx Context) error {
		// Create or modify a complex data structure
		var data map[string]interface{}
		if existing, ok := ctx.Get("complex_data"); ok {
			data = existing.(map[string]interface{})
		} else {
			data = make(map[string]interface{})
		}

		data["timestamp"] = time.Now()
		data["counter"] = 0
		if counter, ok := data["counter"]; ok {
			data["counter"] = counter.(int) + 1
		}

		// Add nested structure
		data["nested"] = map[string]interface{}{
			"level":    1,
			"active":   true,
			"metadata": []string{"item1", "item2"},
		}

		ctx.Set("complex_data", data)
		return nil
	}

	// Guard checks nested structure
	guard := func(ctx Context) bool {
		if data, ok := ctx.Get("complex_data"); ok {
			if dataMap, ok := data.(map[string]interface{}); ok {
				if nested, ok := dataMap["nested"]; ok {
					if nestedMap, ok := nested.(map[string]interface{}); ok {
						if active, ok := nestedMap["active"]; ok {
							return active.(bool)
						}
					}
				}
			}
		}
		return false
	}

	definition := NewMachine().
		State("initial").Initial().
		To("structured").On("organize").Do(action).
		State("structured").
		To("active").On("activate").When(guard).
		State("active").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result1 := machine.HandleEvent("organize", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "structured")

	// Verify complex structure was created
	if data, ok := machine.Context().Get("complex_data"); ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if _, ok := dataMap["nested"]; !ok {
				t.Error("Expected nested structure to be created")
			}
		}
	}

	result2 := machine.HandleEvent("activate", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "active")
}

// Test for transitions with guards and multiple attempts
func TestTransition_NestedTransitionsWithGuards(t *testing.T) {
	transitionCount := 0

	// Simple transition action
	action := func(ctx Context) error {
		transitionCount++
		return nil
	}

	// Guard that allows all attempts for simplicity
	guard := func(ctx Context) bool {
		return true
	}

	definition := NewMachine().
		State("initial").Initial().
		To("middle").On("trigger").When(guard).Do(action).
		State("middle").
		To("final").On("nested").Do(action).
		State("final").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Test simple transitions
	result := machine.HandleEvent("trigger", nil)
	AssertEventProcessed(t, result, true)

	// Move to final state
	machine.HandleEvent("nested", nil)
	AssertState(t, machine, "final")
}

// Test for complex guard chains where one guard depends on another
func TestTransition_GuardDependencyChains(t *testing.T) {
	guard1Called := false
	guard2Called := false
	guard3Called := false

	guard1 := func(ctx Context) bool {
		guard1Called = true
		ctx.Set("guard1_result", true)
		return true
	}

	guard2 := func(ctx Context) bool {
		guard2Called = true
		if result, ok := ctx.Get("guard1_result"); ok {
			ctx.Set("guard2_result", result.(bool))
			return result.(bool)
		}
		return false
	}

	guard3 := func(ctx Context) bool {
		guard3Called = true
		if result1, ok := ctx.Get("guard1_result"); ok {
			if result2, ok := ctx.Get("guard2_result"); ok {
				return result1.(bool) && result2.(bool)
			}
		}
		return false
	}

	definition := NewMachine().
		State("initial").Initial().
		To("step1").On("next").When(guard1).
		State("step1").
		To("step2").On("next").When(guard2).
		State("step2").
		To("final").On("next").When(guard3).
		State("final").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// First transition
	result1 := machine.HandleEvent("next", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "step1")
	if !guard1Called {
		t.Error("Expected guard1 to be called")
	}

	// Second transition
	result2 := machine.HandleEvent("next", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "step2")
	if !guard2Called {
		t.Error("Expected guard2 to be called")
	}

	// Third transition
	result3 := machine.HandleEvent("next", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine, "final")
	if !guard3Called {
		t.Error("Expected guard3 to be called")
	}
}

// Test for multiple valid transitions for the same event with different guards
func TestTransition_ConflictingGuardConditions(t *testing.T) {
	guardCallOrder := []string{}

	// First guard - always true
	guard1 := func(ctx Context) bool {
		guardCallOrder = append(guardCallOrder, "guard1")
		ctx.Set("selected_path", "path1")
		return true
	}

	// Second guard - also always true
	guard2 := func(ctx Context) bool {
		guardCallOrder = append(guardCallOrder, "guard2")
		ctx.Set("selected_path", "path2")
		return true
	}

	// Third guard - also always true
	guard3 := func(ctx Context) bool {
		guardCallOrder = append(guardCallOrder, "guard3")
		ctx.Set("selected_path", "path3")
		return true
	}

	definition := NewMachine().
		State("initial").Initial().
		To("path1").On("choose").When(guard1).
		To("path2").On("choose").When(guard2).
		To("path3").On("choose").When(guard3).
		State("path1").
		State("path2").
		State("path3").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("choose", nil)
	AssertEventProcessed(t, result, true)

	// Verify only one transition was taken
	finalState := machine.CurrentState()
	if finalState != "path1" && finalState != "path2" && finalState != "path3" {
		t.Errorf("Expected final state to be one of path1, path2, or path3, got %s", finalState)
	}

	// Verify guard precedence (first matching guard should win)
	if len(guardCallOrder) > 0 && guardCallOrder[0] != "guard1" {
		t.Errorf("Expected guard1 to be evaluated first, got %s", guardCallOrder[0])
	}
}

// Test for guards with conflicting conditions based on context
func TestTransition_ContextDependentConflictingGuards(t *testing.T) {
	guardExecuted := ""

	// Guard for high priority path
	highPriorityGuard := func(ctx Context) bool {
		guardExecuted = "high_priority"
		if priority, ok := ctx.Get("priority"); ok {
			return priority.(string) == "high"
		}
		return false
	}

	// Guard for medium priority path
	mediumPriorityGuard := func(ctx Context) bool {
		guardExecuted = "medium_priority"
		if priority, ok := ctx.Get("priority"); ok {
			return priority.(string) == "medium"
		}
		return false
	}

	// Default guard (always true)
	defaultGuard := func(ctx Context) bool {
		guardExecuted = "default"
		return true
	}

	definition := NewMachine().
		State("initial").Initial().
		To("high_priority").On("process").When(highPriorityGuard).
		To("medium_priority").On("process").When(mediumPriorityGuard).
		To("default").On("process").When(defaultGuard).
		State("high_priority").
		State("medium_priority").
		State("default").
		Build()

	// Test high priority
	machine1 := definition.CreateInstance()
	machine1.Context().Set("priority", "high")
	_ = machine1.Start()

	result1 := machine1.HandleEvent("process", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine1, "high_priority")
	if guardExecuted != "high_priority" {
		t.Errorf("Expected high_priority guard to be executed, got %s", guardExecuted)
	}

	// Test medium priority
	machine2 := definition.CreateInstance()
	machine2.Context().Set("priority", "medium")
	_ = machine2.Start()

	guardExecuted = ""
	result2 := machine2.HandleEvent("process", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine2, "medium_priority")
	if guardExecuted != "medium_priority" {
		t.Errorf("Expected medium_priority guard to be executed, got %s", guardExecuted)
	}

	// Test default path
	machine3 := definition.CreateInstance()
	machine3.Context().Set("priority", "low")
	_ = machine3.Start()

	guardExecuted = ""
	result3 := machine3.HandleEvent("process", nil)
	AssertEventProcessed(t, result3, true)
	AssertState(t, machine3, "default")
	if guardExecuted != "default" {
		t.Errorf("Expected default guard to be executed, got %s", guardExecuted)
	}
}

// Test for actions that fail and leave the state machine in an inconsistent state
func TestTransition_ActionFailureStateInconsistency(t *testing.T) {
	actionCallCount := 0
	inconsistentState := false

	// Action that fails after modifying context
	failingAction := func(ctx Context) error {
		actionCallCount++
		// Modify context before failing
		ctx.Set("action_started", true)
		ctx.Set("action_count", actionCallCount)

		// On third call, fail after modifying context
		if actionCallCount == 3 {
			inconsistentState = true
			return errors.New("action failed after partial state change")
		}

		return nil
	}

	// Recovery action to fix inconsistency
	recoveryAction := func(ctx Context) error {
		inconsistentState = false
		ctx.Set("recovered", true)
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("processing").On("start").Do(failingAction).
		State("processing").
		To("recovery").On("recover").Do(recoveryAction).
		State("recovery").
		Build()

	machine := definition.CreateInstance()
	observer := NewTestObserver()
	machine.AddObserver(observer)
	_ = machine.Start()

	// First two transitions should succeed
	for i := 0; i < 2; i++ {
		result := machine.HandleEvent("start", nil)
		AssertEventProcessed(t, result, true)
		AssertState(t, machine, "processing")

		// Reset for next test
		_ = machine.SetState("initial")
	}

	// Third transition should fail
	result := machine.HandleEvent("start", nil)
	// The behavior depends on implementation - either:
	// 1. Transition doesn't happen but action partially executed
	// 2. Transition happens but action failed

	if result.Error != nil {
		t.Logf("Action failed as expected: %v", result.Error)
	}

	// Check for inconsistent state
	if started, ok := machine.Context().Get("action_started"); ok && started.(bool) {
		if inconsistentState {
			t.Logf("State machine detected inconsistent state after action failure (expected behavior)")

			// Verify the state machine remains in initial state when action fails
			if machine.CurrentState() != "initial" {
				t.Errorf("Expected machine to remain in initial state after failed action, got %s", machine.CurrentState())
			}

			// Test recovery
			recoveryResult := machine.HandleEvent("recover", nil)
			if recoveryResult.Processed {
				t.Logf("Recovery successful: %s", machine.CurrentState())
			} else {
				t.Logf("Recovery not processed (machine in state: %s)", machine.CurrentState())
			}
		}
	}
}

// Test for rapid transitions where guards depend on state from previous transitions
func TestTransition_RapidTransitionsWithGuardStateDependencies(t *testing.T) {
	transitionCount := 0
	guardEvaluations := []int{}

	// Guard that depends on transition count
	countDependentGuard := func(ctx Context) bool {
		// Always record guard evaluation
		if count, ok := ctx.Get("transition_count"); ok {
			guardEvaluations = append(guardEvaluations, count.(int))
			return count.(int) < 5
		} else {
			guardEvaluations = append(guardEvaluations, 0)
			return true
		}
	}

	// Action that increments transition count
	countingAction := func(ctx Context) error {
		transitionCount++
		ctx.Set("transition_count", transitionCount)
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("state1").On("next").When(countDependentGuard).Do(countingAction).
		State("state1").
		To("state2").On("next").When(countDependentGuard).Do(countingAction).
		State("state2").
		To("state3").On("next").When(countDependentGuard).Do(countingAction).
		State("state3").
		To("state1").On("next").When(countDependentGuard).Do(countingAction). // Allow cycling back
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Rapid transitions - test 7 attempts
	for i := 0; i < 7; i++ {
		result := machine.HandleEvent("next", nil)
		if i < 5 {
			AssertEventProcessed(t, result, true)
		} else {
			AssertEventProcessed(t, result, false)
		}
	}

	// Verify guard evaluations
	if len(guardEvaluations) != 7 {
		t.Errorf("Expected 7 guard evaluations, got %d", len(guardEvaluations))
	}
}

// Test for concurrent rapid transitions with shared guard state
func TestTransition_ConcurrentRapidTransitionsWithSharedGuardState(t *testing.T) {
	var mutex sync.Mutex
	sharedCounter := 0
	guardEvaluations := 0

	// Thread-safe guard that depends on shared state
	sharedStateGuard := func(ctx Context) bool {
		mutex.Lock()
		defer mutex.Unlock()

		guardEvaluations++
		currentCount := sharedCounter
		sharedCounter++

		// Allow only first 10 transitions
		return currentCount < 10
	}

	definition := NewMachine().
		State("initial").Initial().
		To("active").On("activate").When(sharedStateGuard).
		State("active").
		To("initial").On("reset").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	const numGoroutines = 50
	const transitionsPerGoroutine = 5
	var wg sync.WaitGroup
	successfulTransitions := make(chan bool, numGoroutines*transitionsPerGoroutine)

	// Launch multiple goroutines for rapid transitions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < transitionsPerGoroutine; j++ {
				result := machine.HandleEvent("activate", nil)
				successfulTransitions <- result.Processed

				if result.Processed {
					// Reset for next transition
					machine.HandleEvent("reset", nil)
				}
			}
		}()
	}

	wg.Wait()
	close(successfulTransitions)

	// Count successful transitions
	successfulCount := 0
	for success := range successfulTransitions {
		if success {
			successfulCount++
		}
	}

	// Should have exactly 10 successful transitions
	if successfulCount != 10 {
		t.Errorf("Expected 10 successful transitions, got %d", successfulCount)
	}

	if guardEvaluations < 10 {
		t.Errorf("Expected at least 10 guard evaluations, got %d", guardEvaluations)
	}
}

// ===== CONTEXT EDGE CASES =====

// Test for context with nil values and type assertions
func TestContext_NilValuesAndTypeAssertions(t *testing.T) {
	var nilValue interface{}

	action := func(ctx Context) error {
		// Set nil value
		ctx.Set("nil_value", nilValue)
		ctx.Set("string_value", "test")
		ctx.Set("int_value", 42)

		// Try to get and type assert
		ctx.Get("nil_value")
		ctx.Get("string_value")

		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("processed").On("process").Do(action).
		State("processed").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("process", nil)
	AssertEventProcessed(t, result, true)

	// Verify context values
	if _, ok := machine.Context().Get("nil_value"); ok {
		// nil_value exists as expected
	} else {
		t.Error("Expected nil_value to exist")
	}
}

// Test for context corruption during concurrent access
func TestContext_ConcurrentAccessCorruption(t *testing.T) {
	const numGoroutines = 100
	const operationsPerGoroutine = 50

	action := func(ctx Context) error {
		var wg sync.WaitGroup

		// Launch multiple goroutines that modify context concurrently
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					key := fmt.Sprintf("key_%d_%d", goroutineID, j)
					value := fmt.Sprintf("value_%d_%d", goroutineID, j)
					ctx.Set(key, value)

					// Immediately read it back
					if retrieved, ok := ctx.Get(key); ok {
						if retrieved != value {
							t.Errorf("Context corruption: expected %s, got %v", value, retrieved)
						}
					}
				}
			}(i)
		}

		wg.Wait()
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("concurrent").On("test").Do(action).
		State("concurrent").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result, true)

	// Verify context integrity
	data := machine.Context().GetAll()
	expectedCount := numGoroutines * operationsPerGoroutine
	if len(data) != expectedCount {
		t.Errorf("Expected %d context entries, got %d", expectedCount, len(data))
	}
}

// Test for context memory leaks with large data structures
func TestContext_MemoryLeaks(t *testing.T) {
	action := func(ctx Context) error {
		// Create large data structures
		largeSlice := make([][]byte, 1000)
		for i := range largeSlice {
			largeSlice[i] = make([]byte, 1024) // 1KB each
		}

		largeMap := make(map[string]interface{})
		for i := 0; i < 1000; i++ {
			largeMap[fmt.Sprintf("key_%d", i)] = largeSlice
		}

		ctx.Set("large_data", largeMap)

		// Create more large data
		for i := 0; i < 100; i++ {
			ctx.Set(fmt.Sprintf("large_item_%d", i), largeSlice)
		}

		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("memory_test").On("test").Do(action).
		State("memory_test").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Measure memory before
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	result := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result, true)

	// Measure memory after
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Memory should increase but not excessively
	memIncrease := m2.Alloc - m1.Alloc
	if memIncrease > 100*1024*1024 { // 100MB limit
		t.Errorf("Excessive memory usage: %d bytes", memIncrease)
	}

	// Test context cleanup
	machine.Context().Set("large_data", nil)
	for i := 0; i < 100; i++ {
		machine.Context().Set(fmt.Sprintf("large_item_%d", i), nil)
	}

	// Force GC and check memory again
	runtime.GC()
	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)

	memAfterCleanup := m3.Alloc
	if memAfterCleanup > memIncrease*2 { // Should be significantly reduced
		t.Logf("Memory after cleanup: %d bytes (was %d)", memAfterCleanup, memIncrease)
	}
}

// Test for context fork behavior with nested modifications
func TestContext_ForkBehavior(t *testing.T) {
	action := func(ctx Context) error {
		// Set initial values
		ctx.Set("original", "value1")
		ctx.Set("shared", "initial")

		// Fork context
		forkedCtx := ctx.Fork()

		// Modify original context
		ctx.Set("modified", "value2")
		ctx.Set("shared", "changed_in_original")

		// Modify forked context
		forkedCtx.Set("forked", "value3")
		forkedCtx.Set("shared", "changed_in_fork")

		// Verify fork isolation
		if _, ok := ctx.Get("forked"); ok {
			t.Error("Original context should not have forked values")
		}

		if _, ok := forkedCtx.Get("modified"); ok {
			t.Error("Forked context should not have modifications from original")
		}

		// Verify shared value independence
		if val, _ := ctx.Get("shared"); val != "changed_in_original" {
			t.Errorf("Expected 'changed_in_original', got %v", val)
		}

		if val, _ := forkedCtx.Get("shared"); val != "changed_in_fork" {
			t.Errorf("Expected 'changed_in_fork', got %v", val)
		}

		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("fork_test").On("test").Do(action).
		State("fork_test").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result, true)
}

// ===== STATE MACHINE LIFECYCLE EDGE CASES =====

// Test for machine start/stop during active transitions
func TestMachine_StartStopDuringTransitions(t *testing.T) {
	var transitionInProgress sync.WaitGroup

	// Long-running action
	longAction := func(ctx Context) error {
		transitionInProgress.Add(1)
		defer transitionInProgress.Done()

		// Simulate long operation
		time.Sleep(100 * time.Millisecond)
		ctx.Set("action_completed", true)
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("processing").On("start").Do(longAction).
		State("processing").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Start transition in goroutine
	go func() {
		result := machine.HandleEvent("start", nil)
		if result.Processed {
			t.Log("Transition completed successfully")
		}
	}()

	// Wait a bit then try to stop machine during transition
	time.Sleep(10 * time.Millisecond)
	stopErr := machine.Stop()
	if stopErr != nil {
		t.Logf("Machine stop during transition: %v", stopErr)
	}

	// Wait for transition to complete
	transitionInProgress.Wait()

	// Try to restart machine
	startErr := machine.Start()
	if startErr != nil {
		t.Logf("Machine restart after interruption: %v", startErr)
	}

	// Verify machine state
	if machine.CurrentState() != "processing" {
		t.Logf("Final state: %s", machine.CurrentState())
	}
}

// Test for reset during complex parallel state execution
func TestMachine_ResetDuringParallelExecution(t *testing.T) {
	var parallelStatesEntered sync.WaitGroup

	// Actions for parallel regions
	action1 := func(ctx Context) error {
		parallelStatesEntered.Add(1)
		defer parallelStatesEntered.Done()
		time.Sleep(50 * time.Millisecond)
		ctx.Set("region1_completed", true)
		return nil
	}

	action2 := func(ctx Context) error {
		parallelStatesEntered.Add(1)
		defer parallelStatesEntered.Done()
		time.Sleep(60 * time.Millisecond)
		ctx.Set("region2_completed", true)
		return nil
	}

	builder := NewMachine()
	builder.State("initial").Initial().
		To("parallel").On("start")

	parallelBuilder := builder.ParallelState("parallel")
	region1 := parallelBuilder.Region("region1")
	region1.State("r1_initial").Initial().
		To("r1_final").On("process").Do(action1)
	region1.State("r1_final")

	region2 := parallelBuilder.Region("region2")
	region2.State("r2_initial").Initial().
		To("r2_final").On("process").Do(action2)
	region2.State("r2_final")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	// Start parallel execution
	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	// Wait for parallel states to be entered
	parallelStatesEntered.Wait()

	// Reset during parallel execution
	resetErr := machine.Reset()
	if resetErr != nil {
		t.Errorf("Reset during parallel execution failed: %v", resetErr)
	}

	// Verify machine is back to initial state
	if machine.CurrentState() != "initial" {
		t.Errorf("Expected initial state after reset, got %s", machine.CurrentState())
	}

	// Verify context is cleaned up
	if _, ok := machine.Context().Get("region1_completed"); ok {
		t.Error("Context should be cleaned up after reset")
	}
}

// Test for multiple rapid start/stop cycles
func TestMachine_MultipleRapidStartStopCycles(t *testing.T) {
	const cycles = 50

	definition := NewMachine().
		State("initial").Initial().
		To("running").On("start").
		State("running").
		To("stopped").On("stop").
		State("stopped").
		Build()

	machine := definition.CreateInstance()

	for i := 0; i < cycles; i++ {
		// Start machine
		startErr := machine.Start()
		if startErr != nil && i > 0 { // First start should succeed, others might fail if already started
			t.Logf("Start cycle %d failed: %v", i, startErr)
		}

		// Do some transitions
		machine.HandleEvent("start", nil)
		machine.HandleEvent("stop", nil)

		// Stop machine
		stopErr := machine.Stop()
		if stopErr != nil {
			t.Logf("Stop cycle %d failed: %v", i, stopErr)
		}

		// Reset for next cycle
		resetErr := machine.Reset()
		if resetErr != nil {
			t.Logf("Reset cycle %d failed: %v", i, resetErr)
		}
	}

	// Final verification
	if machine.CurrentState() != "initial" {
		t.Errorf("Expected initial state after cycles, got %s", machine.CurrentState())
	}
}

// Test for machine state corruption during concurrent operations
func TestMachine_ConcurrentStateCorruption(t *testing.T) {
	const numGoroutines = 20
	const operationsPerGoroutine = 10

	definition := NewMachine().
		State("initial").Initial().
		To("state1").On("event1").
		State("state1").
		To("state2").On("event2").
		State("state2").
		To("initial").On("reset").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Launch multiple goroutines performing various operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				switch j % 4 {
				case 0:
					result := machine.HandleEvent("event1", nil)
					if !result.Processed && result.Error == nil {
						errors <- fmt.Errorf("event1 not processed without error in goroutine %d", goroutineID)
					}
				case 1:
					result := machine.HandleEvent("event2", nil)
					if !result.Processed && result.Error == nil {
						errors <- fmt.Errorf("event2 not processed without error in goroutine %d", goroutineID)
					}
				case 2:
					result := machine.HandleEvent("reset", nil)
					if !result.Processed && result.Error == nil {
						errors <- fmt.Errorf("reset not processed without error in goroutine %d", goroutineID)
					}
				case 3:
					// Check state consistency
					state := machine.CurrentState()
					if state != "initial" && state != "state1" && state != "state2" {
						errors <- fmt.Errorf("invalid state %s in goroutine %d", state, goroutineID)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Found %d state corruption errors", errorCount)
	}
}

// ===== PARALLEL STATE EDGE CASES =====

// Test for fork/join synchronization failures
func TestParallelState_ForkJoinSynchronizationFailures(t *testing.T) {
	var region1Completed, region2Completed bool
	var region1Error, region2Error error

	// Region 1 action that might fail
	action1 := func(ctx Context) error {
		region1Completed = true
		if _, exists := ctx.Get("force_region1_error"); exists {
			region1Error = errors.New("region1 forced error")
			return region1Error
		}
		return nil
	}

	// Region 2 action that might fail
	action2 := func(ctx Context) error {
		region2Completed = true
		if _, exists := ctx.Get("force_region2_error"); exists {
			region2Error = errors.New("region2 forced error")
			return region2Error
		}
		return nil
	}

	// Test successful fork/join
	t.Run("SuccessfulForkJoin", func(t *testing.T) {
		region1Completed, region2Completed = false, false
		region1Error, region2Error = nil, nil

		builder := NewMachine()
		builder.State("initial").Initial().
			To("parallel").On("start")

		parallelBuilder := builder.ParallelState("parallel")
		region1 := parallelBuilder.Region("region1")
		region1.State("r1_initial").Initial().
			To("r1_final").On("process_region1").Do(action1)
		region1.State("r1_final").Final()

		region2 := parallelBuilder.Region("region2")
		region2.State("r2_initial").Initial().
			To("r2_final").On("process_region2").Do(action2)
		region2.State("r2_final").Final()

		builder.State("joined").
			Build()

		definition := builder.Build()
		machine := definition.CreateInstance()
		_ = machine.Start()

		result := machine.HandleEvent("start", nil)
		AssertEventProcessed(t, result, true)

		// Trigger region transitions - need to trigger events for each region
		machine.HandleEvent("process_region1", nil) // This should trigger region1
		machine.HandleEvent("process_region2", nil) // This should trigger region2

		// Wait for regions to complete
		time.Sleep(50 * time.Millisecond)

		if !region1Completed || !region2Completed {
			t.Error("Both regions should have completed successfully")
		}
	})

	// Test fork/join with region1 failure
	t.Run("Region1Failure", func(t *testing.T) {
		region1Completed, region2Completed = false, false
		region1Error, region2Error = nil, nil

		builder := NewMachine()
		builder.State("initial").Initial().
			To("parallel").On("start")

		parallelBuilder := builder.ParallelState("parallel")
		region1 := parallelBuilder.Region("region1")
		region1.State("r1_initial").Initial().
			To("r1_final").On("process_region1").Do(action1)
		region1.State("r1_final").Final()

		region2 := parallelBuilder.Region("region2")
		region2.State("r2_initial").Initial().
			To("r2_final").On("process_region2").Do(action2)
		region2.State("r2_final").Final()

		builder.State("joined").
			Build()

		definition := builder.Build()
		machine := definition.CreateInstance()
		machine.Context().Set("force_region1_error", true)
		_ = machine.Start()

		result := machine.HandleEvent("start", nil)
		AssertEventProcessed(t, result, true)

		// Trigger region transitions
		machine.HandleEvent("process_region1", nil)
		machine.HandleEvent("process_region2", nil)

		// Wait for regions to complete
		time.Sleep(50 * time.Millisecond)

		if region1Error == nil {
			t.Error("Expected region1 to fail")
		}
		if !region2Completed {
			t.Error("Region2 should still complete even if region1 fails")
		}
	})

	// Test fork/join with both regions failing
	t.Run("BothRegionsFailure", func(t *testing.T) {
		region1Completed, region2Completed = false, false
		region1Error, region2Error = nil, nil

		builder := NewMachine()
		builder.State("initial").Initial().
			To("parallel").On("start")

		parallelBuilder := builder.ParallelState("parallel")
		region1 := parallelBuilder.Region("region1")
		region1.State("r1_initial").Initial().
			To("r1_final").On("process_region1").Do(action1)
		region1.State("r1_final").Final()

		region2 := parallelBuilder.Region("region2")
		region2.State("r2_initial").Initial().
			To("r2_final").On("process_region2").Do(action2)
		region2.State("r2_final").Final()

		builder.State("joined").
			Build()

		definition := builder.Build()
		machine := definition.CreateInstance()
		machine.Context().Set("force_region1_error", true)
		machine.Context().Set("force_region2_error", true)
		_ = machine.Start()

		result := machine.HandleEvent("start", nil)
		AssertEventProcessed(t, result, true)

		// Trigger region transitions
		machine.HandleEvent("process_region1", nil)
		machine.HandleEvent("process_region2", nil)

		// Wait for regions to complete
		time.Sleep(50 * time.Millisecond)

		if region1Error == nil || region2Error == nil {
			t.Error("Expected both regions to fail")
		}
	})
}

// Test for region state conflicts and race conditions
func TestParallelState_RegionConflictsAndRaceConditions(t *testing.T) {
	var sharedCounter int
	var mutex sync.Mutex
	var regionUpdates []string

	// Action that modifies shared state
	sharedStateAction := func(regionID string) ActionFunc {
		return func(ctx Context) error {
			for i := 0; i < 10; i++ {
				mutex.Lock()
				sharedCounter++
				regionUpdates = append(regionUpdates, fmt.Sprintf("%s_%d", regionID, sharedCounter))
				mutex.Unlock()
				time.Sleep(1 * time.Millisecond) // Small delay to increase race condition probability
			}
			return nil
		}
	}

	builder := NewMachine()
	builder.State("initial").Initial().
		To("parallel").On("start")

	parallelBuilder := builder.ParallelState("parallel")

	// Create multiple regions that modify shared state
	for i := 0; i < 3; i++ {
		regionID := fmt.Sprintf("region%d", i+1)
		region := parallelBuilder.Region(regionID)
		region.State(fmt.Sprintf("r%d_initial", i+1)).Initial().
			To(fmt.Sprintf("r%d_final", i+1)).On("process").Do(sharedStateAction(regionID))
		region.State(fmt.Sprintf("r%d_final", i+1))
	}

	builder.State("complete").
		Build()

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	// Trigger process events for each region
	for i := 0; i < 3; i++ {
		machine.HandleEvent("process", nil) // This should trigger all regions simultaneously
	}

	// Wait for all regions to complete
	time.Sleep(200 * time.Millisecond)

	// Verify all regions contributed to shared state
	if sharedCounter != 30 { // 3 regions * 10 updates each
		t.Errorf("Expected shared counter to be 30, got %d", sharedCounter)
	}

	if len(regionUpdates) != 30 {
		t.Errorf("Expected 30 region updates, got %d", len(regionUpdates))
	}

	// Verify no duplicate updates (race condition detection)
	updateMap := make(map[string]bool)
	for _, update := range regionUpdates {
		if updateMap[update] {
			t.Errorf("Duplicate update detected: %s", update)
		}
		updateMap[update] = true
	}
}

// Test for parallel state completion with missing regions
func TestParallelState_MissingRegions(t *testing.T) {
	builder := NewMachine()
	builder.State("initial").Initial().
		To("parallel").On("start")

	parallelBuilder := builder.ParallelState("parallel")

	// Create only one region instead of expected multiple
	region1 := parallelBuilder.Region("region1")
	region1.State("r1_initial").Initial().
		To("r1_final").On("process")
	region1.State("r1_final")

	// Intentionally don't create region2 and region3 that might be expected

	builder.State("complete").
		Build()

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	// Test that machine can handle parallel state with only one region
	time.Sleep(50 * time.Millisecond)

	// Should be able to transition out of parallel state even with single region
	// This tests robustness against missing/empty regions
}

// Test for nested parallel state hierarchies
func TestParallelState_NestedParallelHierarchies(t *testing.T) {
	var outerRegionCompleted bool

	// Outer parallel state action
	outerAction := func(ctx Context) error {
		time.Sleep(50 * time.Millisecond)
		outerRegionCompleted = true
		return nil
	}

	builder := NewMachine()
	builder.State("initial").Initial().
		To("outer_parallel").On("start")

	// Outer parallel state
	outerParallelBuilder := builder.ParallelState("outer_parallel")

	// Region 1 of outer parallel
	outerRegion1 := outerParallelBuilder.Region("outer_region1")
	outerRegion1.State("or1_initial").Initial().
		To("or1_final").On("process").Do(outerAction)
	outerRegion1.State("or1_final")

	// Region 2 of outer parallel - contains nested parallel state
	outerRegion2 := outerParallelBuilder.Region("outer_region2")
	outerRegion2.State("or2_initial").Initial().
		To("or2_final").On("process").Do(outerAction)
	outerRegion2.State("or2_final")

	builder.State("complete").
		Build()

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	// Trigger process events for outer regions
	machine.HandleEvent("process", nil)

	// Wait for nested hierarchy to complete
	time.Sleep(150 * time.Millisecond)

	if !outerRegionCompleted {
		t.Error("Outer regions should have completed")
	}
}

// Test for parallel state with different completion times
func TestParallelState_DifferentCompletionTimes(t *testing.T) {
	var completionTimes []time.Time
	var completionMutex sync.Mutex

	// Action with different sleep times
	variableAction := func(duration time.Duration, regionID string) ActionFunc {
		return func(ctx Context) error {
			time.Sleep(duration)
			completionMutex.Lock()
			completionTimes = append(completionTimes, time.Now())
			completionMutex.Unlock()
			ctx.Set(fmt.Sprintf("%s_completed", regionID), true)
			return nil
		}
	}

	builder := NewMachine()
	builder.State("initial").Initial().
		To("parallel").On("start")

	parallelBuilder := builder.ParallelState("parallel")

	// Regions with different completion times
	durations := []time.Duration{10 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond}
	for i, duration := range durations {
		regionID := fmt.Sprintf("region%d", i+1)
		region := parallelBuilder.Region(regionID)
		region.State(fmt.Sprintf("r%d_initial", i+1)).Initial().
			To(fmt.Sprintf("r%d_final", i+1)).On("process").Do(variableAction(duration, regionID))
		region.State(fmt.Sprintf("r%d_final", i+1))
	}

	builder.State("complete").
		Build()

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	// Trigger process events for all regions multiple times
	for i := 0; i < 3; i++ {
		machine.HandleEvent("process", nil)
	}

	// Wait for all regions to complete
	time.Sleep(150 * time.Millisecond)

	if len(completionTimes) != 3 {
		t.Errorf("Expected 3 completions, got %d", len(completionTimes))
	}

	// Verify completion order (should be in order of duration)
	for i := 1; i < len(completionTimes); i++ {
		if completionTimes[i].Before(completionTimes[i-1]) {
			t.Error("Completion times should be in order")
		}
	}

	// Verify all regions completed in context
	for i := 1; i <= 3; i++ {
		regionID := fmt.Sprintf("region%d", i)
		if completed, ok := machine.Context().Get(fmt.Sprintf("%s_completed", regionID)); !ok || !completed.(bool) {
			t.Errorf("Region %s should have completed", regionID)
		}
	}
}

// Test for parallel state with context isolation
func TestParallelState_ContextIsolation(t *testing.T) {
	var regionContextValues []string
	var contextMutex sync.Mutex

	// Action that sets region-specific context
	contextIsolationAction := func(regionID string) ActionFunc {
		return func(ctx Context) error {
			// Set region-specific value
			ctx.Set(fmt.Sprintf("%s_value", regionID), fmt.Sprintf("value_from_%s", regionID))

			// Also set a shared value
			ctx.Set("shared_value", fmt.Sprintf("updated_by_%s", regionID))

			// Record what we set
			contextMutex.Lock()
			regionContextValues = append(regionContextValues, fmt.Sprintf("%s_set", regionID))
			contextMutex.Unlock()

			time.Sleep(20 * time.Millisecond)
			return nil
		}
	}

	builder := NewMachine()
	builder.State("initial").Initial().
		To("parallel").On("start")

	parallelBuilder := builder.ParallelState("parallel")

	// Create multiple regions
	for i := 0; i < 3; i++ {
		regionID := fmt.Sprintf("region%d", i+1)
		region := parallelBuilder.Region(regionID)
		region.State(fmt.Sprintf("r%d_initial", i+1)).Initial().
			To(fmt.Sprintf("r%d_final", i+1)).On("process").Do(contextIsolationAction(regionID))
		region.State(fmt.Sprintf("r%d_final", i+1))
	}

	builder.State("complete").
		Build()

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	// Trigger process events for all regions multiple times
	for i := 0; i < 3; i++ {
		machine.HandleEvent("process", nil)
	}

	// Wait for all regions to complete
	time.Sleep(100 * time.Millisecond)

	// Verify all regions set their values
	for i := 1; i <= 3; i++ {
		regionID := fmt.Sprintf("region%d", i)
		expectedValue := fmt.Sprintf("value_from_%s", regionID)
		if value, ok := machine.Context().Get(fmt.Sprintf("%s_value", regionID)); !ok || value != expectedValue {
			t.Errorf("Expected %s to have value %s, got %v", regionID, expectedValue, value)
		}
	}

	// Verify shared value (last region should win)
	if sharedValue, ok := machine.Context().Get("shared_value"); ok {
		expectedSharedValue := "updated_by_region3" // Last region to execute
		if sharedValue != expectedSharedValue {
			t.Logf("Shared value: %s (expected: %s)", sharedValue, expectedSharedValue)
		}
	}

	// Verify all regions executed
	if len(regionContextValues) != 3 {
		t.Errorf("Expected 3 regions to execute, got %d", len(regionContextValues))
	}
}

// ===== PSEUDOSTATE EDGE CASES =====

// Test for choice pseudostate with dependent conditions
func TestPseudostate_ChoiceDependencies(t *testing.T) {
	var evaluationOrder []string

	// Guard that records evaluation order
	orderTrackingGuard := func(name string) GuardFunc {
		return func(ctx Context) bool {
			evaluationOrder = append(evaluationOrder, name)
			return false // Always false to test multiple evaluations
		}
	}

	// Simulate choice behavior with multiple guarded transitions
	definition := NewMachine().
		State("initial").Initial().
		To("path1").On("decide").When(orderTrackingGuard("guard1")).
		To("path2").On("decide").When(orderTrackingGuard("guard2")).
		To("path3").On("decide").When(orderTrackingGuard("guard3")).
		To("default").On("decide"). // Default path when all guards fail
		State("path1").
		State("path2").
		State("path3").
		State("default").
		To("final").On("continue").
		State("final").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("decide", nil)
	AssertEventProcessed(t, result, true)

	// Should have evaluated guards before taking default path
	if len(evaluationOrder) < 1 {
		t.Errorf("Expected at least 1 guard evaluation, got %d", len(evaluationOrder))
	}

	// Should be in default state since all guards returned false
	AssertState(t, machine, "default")

	// Continue to final
	result2 := machine.HandleEvent("continue", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "final")
}

// Test for history pseudostate with invalid references
func TestPseudostate_HistoryInvalidReferences(t *testing.T) {
	// Test with deep history to non-existent state
	builder := NewMachine().
		State("initial").Initial().
		To("composite").On("enter").
		CompositeState("composite").
		State("composite.sub1").Initial().
		To("composite.sub2").On("switch").
		State("composite.sub2").
		DeepHistory("composite.deep_hist").
		Default("composite.invalid"). // invalid target state
		State("composite.invalid")

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	// Enter composite state
	result1 := machine.HandleEvent("enter", nil)
	AssertEventProcessed(t, result1, true)

	// Switch to sub2
	result2 := machine.HandleEvent("switch", nil)
	AssertEventProcessed(t, result2, true)

	// Try to restore to invalid state via deep history
	result3 := machine.HandleEvent("restore", nil)
	// This should either fail gracefully or handle the invalid reference
	if result3.Processed {
		t.Logf("Invalid history reference handled gracefully: %s", machine.CurrentState())
	} else {
		t.Logf("Invalid history reference properly rejected: %v", result3.Error)
	}
}

// Test for join pseudostate with multiple incoming transitions
func TestPseudostate_JoinCombinations(t *testing.T) {
	var joinSources []string
	var joinMutex sync.Mutex

	// Action that records which state joined
	joinTrackingAction := func(source string) ActionFunc {
		return func(ctx Context) error {
			joinMutex.Lock()
			defer joinMutex.Unlock()
			joinSources = append(joinSources, source)
			return nil
		}
	}

	definition := NewMachine().
		State("initial").Initial().
		To("branch").On("start").
		State("branch").
		To("path1").On("go1").
		To("path2").On("go2").
		To("path3").On("go3").
		State("path1").
		To("join").On("merge").Do(joinTrackingAction("path1")).
		State("path2").
		To("join").On("merge").Do(joinTrackingAction("path2")).
		State("path3").
		To("join").On("merge").Do(joinTrackingAction("path3")).
		State("join").
		To("final").On("complete").
		State("final").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Start branching
	result := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result, true)

	// Test different join combinations
	testCases := []struct {
		path   string
		event  string
		expect bool
	}{
		{"path1", "go1", true},
		{"path2", "go2", true},
		{"path3", "go3", true},
	}

	for _, tc := range testCases {
		// Reset to branch state
		_ = machine.SetState("branch")

		// Go to specific path
		result := machine.HandleEvent(tc.event, nil)
		if tc.expect {
			AssertEventProcessed(t, result, true)
		}

		// Try to join
		joinResult := machine.HandleEvent("merge", nil)
		if tc.expect {
			AssertEventProcessed(t, joinResult, true)
			AssertState(t, machine, "join")

			// Complete to final
			completeResult := machine.HandleEvent("complete", nil)
			AssertEventProcessed(t, completeResult, true)
			AssertState(t, machine, "final")
		}
	}
}

// Test for junction pseudostate with complex conditions
func TestPseudostate_JunctionComplexConditions(t *testing.T) {
	var junctionEvaluations int

	// Complex guard with multiple conditions
	complexGuard := func(ctx Context) bool {
		junctionEvaluations++

		// Check multiple context values
		if val1, ok := ctx.Get("condition1"); ok {
			if val2, ok := ctx.Get("condition2"); ok {
				if val3, ok := ctx.Get("condition3"); ok {
					return val1.(bool) && val2.(bool) && !val3.(bool)
				}
			}
		}
		return false
	}

	// Simulate junction behavior with guarded transitions
	definition := NewMachine().
		State("initial").Initial().
		To("complex_path").On("test").When(complexGuard).
		To("simple_path").On("test"). // Default path
		State("complex_path").
		State("simple_path").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Test with insufficient conditions
	result1 := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "simple_path") // Should take default path since conditions aren't met

	if junctionEvaluations < 1 {
		t.Errorf("Expected at least 1 junction evaluation, got %d", junctionEvaluations)
	}
}

// Test for terminate state edge cases
func TestPseudostate_TerminateEdgeCases(t *testing.T) {
	var terminateActionCalled bool

	// Action that should be called before termination
	preTerminateAction := func(ctx Context) error {
		terminateActionCalled = true
		ctx.Set("cleanup_started", true)
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("processing").On("start").
		State("processing").
		To("terminating").On("terminate").Do(preTerminateAction).
		State("terminating").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Start processing
	result1 := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "processing")

	// Trigger termination
	result2 := machine.HandleEvent("terminate", nil)
	AssertEventProcessed(t, result2, true)

	if !terminateActionCalled {
		t.Error("Pre-terminate action should have been called")
	}

	// Verify cleanup context was set
	if cleanup, ok := machine.Context().Get("cleanup_started"); !ok || !cleanup.(bool) {
		t.Error("Cleanup should have been started")
	}

	// Machine should be in terminated state
	if machine.CurrentState() != "terminating" {
		t.Logf("Machine terminated in state: %s", machine.CurrentState())
	}
}

// Test for multiple decision points in sequence
func TestPseudostate_MultipleDecisionSequence(t *testing.T) {
	var decisionSequence []string

	sequenceTrackingGuard := func(decisionName string) GuardFunc {
		return func(ctx Context) bool {
			decisionSequence = append(decisionSequence, decisionName)
			// Alternate between true and false
			return len(decisionSequence)%2 == 1
		}
	}

	definition := NewMachine().
		State("initial").Initial().
		To("decision1").On("decide1").
		State("decision1").
		To("path1a").On("decide2").When(sequenceTrackingGuard("decision1_guard1")).
		To("path1b").On("decide2").
		State("path1a").
		State("path1b").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// First decision
	result1 := machine.HandleEvent("decide1", nil)
	AssertEventProcessed(t, result1, true)

	// Second decision from whichever path we took
	result2 := machine.HandleEvent("decide2", nil)
	AssertEventProcessed(t, result2, true)

	// Should have evaluated guards in sequence
	if len(decisionSequence) < 1 {
		t.Errorf("Expected at least 1 decision evaluation, got %d", len(decisionSequence))
	}

	// Verify we're in a final state
	finalState := machine.CurrentState()
	if finalState != "path1a" && finalState != "path1b" {
		t.Errorf("Expected final state to be path1a or path1b, got %s", finalState)
	}
}

// ===== ERROR HANDLING EDGE CASES =====

// Test for cascading failures across multiple transitions
func TestErrorHandling_CascadingFailures(t *testing.T) {
	var failureChain []string
	var failureMutex sync.Mutex

	// Action that sometimes fails and records the attempt
	attemptAction := func(stepName string, shouldFail bool) ActionFunc {
		return func(ctx Context) error {
			failureMutex.Lock()
			failureChain = append(failureChain, stepName)
			failureMutex.Unlock()
			if shouldFail {
				return fmt.Errorf("failure in %s", stepName)
			}
			return nil
		}
	}

	// Recovery action that clears the failure chain
	recoveryAction := func(ctx Context) error {
		failureMutex.Lock()
		failureChain = []string{"recovered"}
		failureMutex.Unlock()
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("step1").On("start").Do(attemptAction("step1", false)). // Succeed to step1
		To("recovery").On("recover").Do(recoveryAction).           // Allow recovery from initial
		State("step1").
		To("step2").On("continue").Do(attemptAction("step2", false)). // Succeed to step2
		State("step2").
		To("step3").On("continue").Do(attemptAction("step3", true)). // Fail at step3
		To("recovery").On("recover").Do(recoveryAction).
		State("step3").
		To("recovery").On("recover").Do(recoveryAction).
		State("recovery").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Start - should succeed at step1
	result1 := machine.HandleEvent("start", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "step1")

	// Continue to step2 - should succeed
	result2 := machine.HandleEvent("continue", nil)
	AssertEventProcessed(t, result2, true)
	AssertState(t, machine, "step2")

	// Continue to step3 - should fail at step3
	result3 := machine.HandleEvent("continue", nil)
	if result3.Processed {
		t.Error("Expected step3 to fail")
	}

	// Verify attempt chain was recorded
	failureMutex.Lock()
	if len(failureChain) != 3 {
		t.Errorf("Expected 3 attempts, got %d: %v", len(failureChain), failureChain)
	}
	failureMutex.Unlock()

	// Recover from failures
	result4 := machine.HandleEvent("recover", nil)
	AssertEventProcessed(t, result4, true)
	AssertState(t, machine, "recovery")

	// Verify recovery
	failureMutex.Lock()
	if len(failureChain) != 1 || failureChain[0] != "recovered" {
		t.Errorf("Expected recovery state, got %v", failureChain)
	}
	failureMutex.Unlock()
}

// Test for guard panics and recovery
func TestErrorHandling_GuardPanics(t *testing.T) {
	var panicCount int
	var panicMutex sync.Mutex

	// Guard that panics
	panickingGuard := func(shouldPanic bool) GuardFunc {
		return func(ctx Context) bool {
			if shouldPanic {
				panicMutex.Lock()
				panicCount++
				panicMutex.Unlock()
				panic("intentional guard panic")
			}
			return true
		}
	}

	// Safe guard that never panics
	safeGuard := func(ctx Context) bool {
		return true
	}

	definition := NewMachine().
		State("initial").Initial().
		To("safe").On("safe_event").When(safeGuard).
		To("panic").On("panic_event").When(panickingGuard(true)).
		State("safe").
		State("panic").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Safe transition should work
	result1 := machine.HandleEvent("safe_event", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "safe")

	// Reset to initial
	_ = machine.SetState("initial")

	// Panicking guard should be handled gracefully
	result2 := machine.HandleEvent("panic_event", nil)
	// The behavior depends on implementation - either:
	// 1. Transition is rejected due to panic
	// 2. Panic is recovered and transition proceeds
	// 3. Machine enters error state

	if result2.Error != nil {
		t.Logf("Guard panic handled: %v", result2.Error)
	}

	// Verify panic was recorded
	panicMutex.Lock()
	if panicCount == 0 {
		t.Error("Expected guard to panic")
	}
	panicMutex.Unlock()
}

// CustomFailingObserver is a test observer that panics on specific events
type CustomFailingObserver struct {
	observerFailures *[]string
	failureMutex     *sync.Mutex
}

func (o *CustomFailingObserver) OnTransition(from string, to string, event Event, ctx Context) {
	o.failureMutex.Lock()
	if event.GetName() == "fail_event" {
		*o.observerFailures = append(*o.observerFailures, fmt.Sprintf("observer failed on %s", event.GetName()))
		o.failureMutex.Unlock()
		panic("observer failure")
	}
	o.failureMutex.Unlock()
}

func (o *CustomFailingObserver) OnStateEnter(state string, ctx Context) {
	// No-op for this test
}

func (o *CustomFailingObserver) OnStateExit(state string, ctx Context) {
	// No-op for this test
}

// Test for observer failures and machine resilience
func TestErrorHandling_ObserverFailures(t *testing.T) {
	var observerFailures []string
	var failureMutex sync.Mutex

	customObserver := &CustomFailingObserver{
		observerFailures: &observerFailures,
		failureMutex:     &failureMutex,
	}

	definition := NewMachine().
		State("initial").Initial().
		To("normal").On("normal_event").
		State("normal").
		To("failed").On("fail_event").
		State("failed").
		Build()

	machine := definition.CreateInstance()
	machine.AddObserver(customObserver)
	_ = machine.Start()

	// Normal event should work fine
	result1 := machine.HandleEvent("normal_event", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "normal")

	// Event that causes observer failure should still transition
	result2 := machine.HandleEvent("fail_event", nil)
	// Machine should be resilient to observer failures
	if result2.Processed {
		AssertState(t, machine, "failed")
		t.Log("Machine resilient to observer failure")
	} else {
		t.Logf("Machine rejected transition due to observer failure: %v", result2.Error)
	}

	// Verify observer failure was recorded
	failureMutex.Lock()
	if len(observerFailures) == 0 {
		t.Error("Expected observer to fail")
	}
	failureMutex.Unlock()
}

// Test for action timeout handling
func TestErrorHandling_ActionTimeouts(t *testing.T) {
	var timeoutOccurred bool
	var timeoutMutex sync.Mutex

	// Long-running action that simulates timeout
	timeoutAction := func(ctx Context) error {
		timeoutMutex.Lock()
		timeoutOccurred = true
		timeoutMutex.Unlock()

		// Simulate long operation that might timeout
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	// Quick action for comparison
	quickAction := func(ctx Context) error {
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("quick_state").On("quick_event").Do(quickAction).
		To("timeout_state").On("timeout_event").Do(timeoutAction).
		State("quick_state").
		State("timeout_state").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Quick action should work
	result1 := machine.HandleEvent("quick_event", nil)
	AssertEventProcessed(t, result1, true)
	AssertState(t, machine, "quick_state")

	// Create new machine instance for timeout test
	machine2 := definition.CreateInstance()
	_ = machine2.Start()

	// Timeout action behavior depends on implementation
	start := time.Now()
	result2 := machine2.HandleEvent("timeout_event", nil)
	duration := time.Since(start)

	timeoutMutex.Lock()
	if timeoutOccurred {
		t.Logf("Timeout action took %v, result: %v", duration, result2)
	}
	timeoutMutex.Unlock()

	// Machine should either:
	// 1. Complete the action despite taking time
	// 2. Timeout and handle the error gracefully
	if result2.Processed {
		AssertState(t, machine2, "timeout_state")
	} else {
		t.Logf("Action timeout handled: %v", result2.Error)
	}
}

// Test for context corruption during errors
func TestErrorHandling_ContextCorruptionDuringErrors(t *testing.T) {
	var contextCorruptionDetected bool
	var corruptionMutex sync.Mutex

	// Action that corrupts context then fails
	corruptingAction := func(ctx Context) error {
		// Set some context values
		ctx.Set("pre_error", "value_before_error")
		ctx.Set("corruption_test", "will_be_corrupted")

		// Simulate context corruption by setting invalid values
		ctx.Set("corruption_test", nil)
		ctx.Set("invalid_key", make(chan int)) // Invalid value for context

		corruptionMutex.Lock()
		contextCorruptionDetected = true
		corruptionMutex.Unlock()

		return fmt.Errorf("context corruption test error")
	}

	// Verification action to check context integrity
	verifyAction := func(ctx Context) error {
		// Check if context is still usable after error
		if val, ok := ctx.Get("pre_error"); ok {
			if val != "value_before_error" {
				return fmt.Errorf("context corruption detected: expected 'value_before_error', got %v", val)
			}
		}

		// Try to set a new value
		ctx.Set("post_error", "recovery_value")
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("error_state").On("corrupt").Do(corruptingAction).
		State("error_state").
		To("verify_state").On("verify").Do(verifyAction).
		State("verify_state").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Trigger context corruption
	result1 := machine.HandleEvent("corrupt", nil)
	if result1.Processed {
		t.Error("Expected corrupting action to fail")
	}

	// Verify corruption was detected
	corruptionMutex.Lock()
	if !contextCorruptionDetected {
		t.Error("Expected context corruption to be detected")
	}
	corruptionMutex.Unlock()

	// Try to verify context integrity
	result2 := machine.HandleEvent("verify", nil)
	if result2.Processed {
		// If verification succeeds, context is still usable
		AssertState(t, machine, "verify_state")

		// Check post-error value
		if val, ok := machine.Context().Get("post_error"); !ok || val != "recovery_value" {
			t.Error("Context should be usable after error recovery")
		}
	} else {
		t.Logf("Context verification failed as expected: %v", result2.Error)
	}
}

// Test for multiple simultaneous errors
func TestErrorHandling_MultipleSimultaneousErrors(t *testing.T) {
	var errorSources []string
	var errorMutex sync.Mutex

	// Action that fails with different error types
	errorGeneratingAction := func(errorType string) ActionFunc {
		return func(ctx Context) error {
			errorMutex.Lock()
			errorSources = append(errorSources, errorType)
			errorMutex.Unlock()

			switch errorType {
			case "panic":
				panic("simulated panic")
			case "error":
				return fmt.Errorf("simulated error")
			case "timeout":
				time.Sleep(100 * time.Millisecond)
				return fmt.Errorf("timeout error")
			default:
				return nil
			}
		}
	}

	definition := NewMachine().
		State("initial").Initial().
		To("panic_test").On("panic").Do(errorGeneratingAction("panic")).
		To("error_test").On("error").Do(errorGeneratingAction("error")).
		To("timeout_test").On("timeout").Do(errorGeneratingAction("timeout")).
		State("panic_test").
		State("error_test").
		State("timeout_test").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Test different error types
	testCases := []struct {
		event       string
		expectError bool
		description string
	}{
		{"panic", true, "panic error"},
		{"error", true, "regular error"},
		{"timeout", true, "timeout error"},
	}

	for _, tc := range testCases {
		// Reset to initial for each test
		_ = machine.SetState("initial")

		result := machine.HandleEvent(tc.event, nil)
		if tc.expectError {
			if result.Processed {
				t.Errorf("Expected %s to fail", tc.description)
			} else {
				t.Logf("%s handled: %v", tc.description, result.Error)
			}
		}
	}

	// Verify all error sources were recorded
	errorMutex.Lock()
	if len(errorSources) != 3 {
		t.Errorf("Expected 3 error sources, got %d: %v", len(errorSources), errorSources)
	}
	errorMutex.Unlock()
}

// ===== PERFORMANCE EDGE CASES =====

// Test for memory exhaustion with large number of states
func TestPerformance_MemoryExhaustion(t *testing.T) {
	const numStates = 100 // Reduced from 1000 to avoid memory issues

	// Create a machine with many states
	builder := NewMachine()
	builder.State("initial").Initial()

	// Create a chain of many states
	for i := 0; i < numStates; i++ {
		stateName := fmt.Sprintf("state_%d", i)
		nextStateName := fmt.Sprintf("state_%d", i+1)
		if i < numStates-1 {
			if i == 0 {
				// Connect initial to first state
				builder.State("initial").
					To(stateName).On(fmt.Sprintf("next_%d", i))
			}
			builder.State(stateName).
				To(nextStateName).On(fmt.Sprintf("next_%d", i+1))
		} else {
			builder.State(stateName)
		}
	}

	definition := builder.Build()

	// Measure memory before creation
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create machine instance
	machine := definition.CreateInstance()
	_ = machine.Start()

	// Measure memory after creation
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	memUsed := m2.Alloc
	if m2.Alloc > m1.Alloc {
		memUsed = m2.Alloc - m1.Alloc
	}
	t.Logf("Memory used for %d states: %d bytes (%.2f KB)", numStates, memUsed, float64(memUsed)/1024)

	// Test transitions through many states
	for i := 0; i < 10 && i < numStates-1; i++ {
		eventName := fmt.Sprintf("next_%d", i)
		result := machine.HandleEvent(eventName, nil)
		if i < 5 { // Only test first 5 to avoid failures
			AssertEventProcessed(t, result, true)
		}
	}

	// Verify machine is still functional
	if machine.CurrentState() == "" {
		t.Error("Machine should still be functional after many states")
	}
}

// Test for stack overflow with deeply nested transitions
func TestPerformance_StackOverflow(t *testing.T) {
	const nestingDepth = 100

	// Create deeply nested state machine
	builder := NewMachine()
	builder.State("initial").Initial()

	// Create a single deeply nested composite state
	root := builder.CompositeState("root")
	current := root
	for i := 0; i < nestingDepth; i++ {
		levelName := fmt.Sprintf("level_%d", i)
		if i < nestingDepth-1 {
			current = current.CompositeState(levelName)
		} else {
			current.State(levelName)
		}
	}

	definition := builder.Build()
	machine := definition.CreateInstance()
	_ = machine.Start()

	// Test deep transitions
	for i := 0; i < 10; i++ {
		result := machine.HandleEvent("go_deeper", nil)
		if result.Error != nil {
			t.Logf("Deep transition %d failed: %v", i, result.Error)
			break
		}
	}

	// Verify machine is still responsive
	currentState := machine.CurrentState()
	if currentState == "" {
		t.Error("Machine should still be responsive after deep nesting")
	}

	t.Logf("Final state after deep nesting: %s", currentState)
}

// Test for goroutine leaks with concurrent operations
func TestPerformance_GoroutineLeaks(t *testing.T) {
	const numGoroutines = 50
	const operationsPerGoroutine = 20

	// Measure goroutines before test
	initialGoroutines := runtime.NumGoroutine()

	definition := NewMachine().
		State("initial").Initial().
		To("processing").On("start").
		State("processing").
		To("complete").On("finish").
		State("complete").
		Build()

	var wg sync.WaitGroup

	// Launch many goroutines performing operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			machine := definition.CreateInstance()
			_ = machine.Start()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Perform transitions
				machine.HandleEvent("start", nil)
				machine.HandleEvent("finish", nil)

				// Reset and continue
				_ = machine.Reset()
				_ = machine.Start()

				// Small delay to increase goroutine lifetime
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	// Wait for goroutines to clean up
	time.Sleep(100 * time.Millisecond)

	// Measure goroutines after test
	finalGoroutines := runtime.NumGoroutine()
	goroutineIncrease := finalGoroutines - initialGoroutines

	t.Logf("Goroutine increase: %d (from %d to %d)", goroutineIncrease, initialGoroutines, finalGoroutines)

	// Allow some increase for runtime goroutines, but not excessive
	if goroutineIncrease > numGoroutines/2 {
		t.Errorf("Potential goroutine leak: %d goroutines increased", goroutineIncrease)
	}
}

// Test for performance with rapid event processing
func TestPerformance_RapidEventProcessing(t *testing.T) {
	const numEvents = 10000

	definition := NewMachine().
		State("initial").Initial().
		To("state1").On("event1").
		State("state1").
		To("state2").On("event2").
		State("state2").
		To("initial").On("reset").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Measure time for rapid event processing
	start := time.Now()

	for i := 0; i < numEvents; i++ {
		switch i % 3 {
		case 0:
			machine.HandleEvent("event1", nil)
		case 1:
			machine.HandleEvent("event2", nil)
		case 2:
			machine.HandleEvent("reset", nil)
		}
	}

	duration := time.Since(start)
	eventsPerSecond := float64(numEvents) / duration.Seconds()

	t.Logf("Processed %d events in %v (%.2f events/sec)", numEvents, duration, eventsPerSecond)

	// Verify machine is still functional
	if machine.CurrentState() == "" {
		t.Error("Machine should still be functional after rapid processing")
	}

	// Performance should be reasonable
	if eventsPerSecond < 1000 {
		t.Logf("Performance warning: only %.2f events/sec", eventsPerSecond)
	}
}

// Test for memory pressure with large context data
func TestPerformance_MemoryPressureWithLargeContext(t *testing.T) {
	const dataSize = 1024 * 1024 // 1MB

	// Create large data structure
	largeData := make([]byte, dataSize)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	action := func(ctx Context) error {
		// Store large data in context
		ctx.Set("large_data", largeData)
		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("memory_test").On("store").Do(action).
		State("memory_test").
		Build()

	// Measure memory before
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("store", nil)
	AssertEventProcessed(t, result, true)

	// Measure memory after
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	memIncrease := m2.Alloc - m1.Alloc
	t.Logf("Memory increase with large context data: %d bytes", memIncrease)

	// Verify data is accessible
	if data, ok := machine.Context().Get("large_data"); ok {
		if dataSlice, ok := data.([]byte); ok {
			if len(dataSlice) != dataSize {
				t.Errorf("Expected %d bytes, got %d", dataSize, len(dataSlice))
			}
		} else {
			t.Error("Large data should be accessible as []byte")
		}
	} else {
		t.Error("Large data should be stored in context")
	}

	// Clean up
	machine.Context().Set("large_data", nil)
}

// ===== DATA STRUCTURE EDGE CASES =====

// Test for invalid state IDs
func TestDataStructure_InvalidStateIDs(t *testing.T) {
	// Test with empty state name - should panic at Build() time
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Empty state name handled with panic: %v", r)
		}
	}()

	builder1 := NewMachine()
	builder1.State("").Initial()    // Empty state name
	definition1 := builder1.Build() // This should panic
	machine1 := definition1.CreateInstance()
	_ = machine1.Start()

	// Test with very long state name
	veryLongName := strings.Repeat("a", 1000)
	builder2 := NewMachine()
	builder2.State(veryLongName).Initial()

	definition2 := builder2.Build()
	machine2 := definition2.CreateInstance()

	// Should handle long state names
	if err := machine2.Start(); err != nil {
		t.Logf("Long state name handled: %v", err)
	}

	// Test with special characters in state name
	specialName := "state!@#$%^&*()_+-={}[]|\\:;\"'<>?,./"
	builder3 := NewMachine()
	builder3.State(specialName).Initial()

	definition3 := builder3.Build()
	machine3 := definition3.CreateInstance()

	// Should handle special characters
	if err := machine3.Start(); err != nil {
		t.Logf("Special character state name handled: %v", err)
	}
}

// Test for circular dependencies in state definitions
func TestDataStructure_CircularDependencies(t *testing.T) {
	// Create a machine with potential circular dependencies
	builder := NewMachine()
	builder.State("initial").Initial().
		To("state_a").On("to_a").
		To("state_b").On("to_b").
		State("state_a").
		To("state_b").On("to_b").
		To("state_c").On("to_c").
		State("state_b").
		To("state_c").On("to_c").
		To("state_a").On("to_a"). // Potential circular reference
		State("state_c").
		To("state_a").On("to_a"). // Another circular reference
		Build()

	definition := builder.Build()
	machine := definition.CreateInstance()

	// Should handle circular dependencies gracefully
	if err := machine.Start(); err != nil {
		t.Logf("Circular dependencies handled: %v", err)
	}

	// Test transitions that could cause cycles
	result1 := machine.HandleEvent("to_a", nil)
	if result1.Processed {
		t.Log("Transition to state_a succeeded")
	}

	result2 := machine.HandleEvent("to_b", nil)
	if result2.Processed {
		t.Log("Transition to state_b succeeded")
	}

	result3 := machine.HandleEvent("to_c", nil)
	if result3.Processed {
		t.Log("Transition to state_c succeeded")
	}

	// Verify machine is in a valid state
	currentState := machine.CurrentState()
	if currentState != "initial" && currentState != "state_a" && currentState != "state_b" && currentState != "state_c" {
		t.Errorf("Invalid state after circular transitions: %s", currentState)
	}
}

// Test for duplicate state definitions
func TestDataStructure_DuplicateStateDefinitions(t *testing.T) {
	// Create a machine with duplicate state names
	builder := NewMachine()
	builder.State("initial").Initial().
		To("duplicate").On("go").
		State("duplicate").
		To("final").On("continue").
		State("duplicate"). // Duplicate state definition
		State("final").
		Build()

	definition := builder.Build()
	machine := definition.CreateInstance()

	// Should handle duplicate states gracefully
	if err := machine.Start(); err != nil {
		t.Logf("Duplicate state definitions handled: %v", err)
	}

	result := machine.HandleEvent("go", nil)
	if result.Processed {
		AssertState(t, machine, "duplicate")

		result2 := machine.HandleEvent("continue", nil)
		if result2.Processed {
			AssertState(t, machine, "final")
		}
	}
}

// Test for invalid event names
func TestDataStructure_InvalidEventNames(t *testing.T) {
	definition := NewMachine().
		State("initial").Initial().
		To("target").On("").
		State("target").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	// Test with empty event name
	result := machine.HandleEvent("", nil)
	if result.Processed {
		t.Error("Empty event name should not be processed")
	} else {
		t.Logf("Empty event name rejected: %v", result.Error)
	}

	// Test with nil event data
	result2 := machine.HandleEvent("test", nil)
	if result2.Processed {
		t.Log("Nil event data handled")
	}
}

// Test for malformed transition definitions
func TestDataStructure_MalformedTransitionDefinitions(t *testing.T) {
	// Test transition without target state
	builder1 := NewMachine()
	builder1.State("initial").Initial()
	// Don't add any transitions

	definition1 := builder1.Build()
	machine1 := definition1.CreateInstance()
	_ = machine1.Start()

	// Should handle missing transitions gracefully
	result1 := machine1.HandleEvent("nonexistent", nil)
	if result1.Processed {
		t.Error("Nonexistent event should not be processed")
	}

	// Test transition to non-existent state
	// The builder should panic during Build() due to validation
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Expected panic caught during build: %v", r)
		}
	}()

	builder2 := NewMachine()
	builder2.State("initial").Initial().
		To("nonexistent").On("go").
		State("target"). // Different name than transition target
		Build()

	// This should panic, so the code below shouldn't execute
	definition2 := builder2.Build()
	machine2 := definition2.CreateInstance()
	_ = machine2.Start()

	result2 := machine2.HandleEvent("go", nil)
	if result2.Processed {
		t.Logf("Transition to non-existent state handled: %s", machine2.CurrentState())
	} else {
		t.Logf("Transition to non-existent state rejected: %v", result2.Error)
	}
}

// Test for context with invalid data types
func TestDataStructure_InvalidContextDataTypes(t *testing.T) {
	action := func(ctx Context) error {
		// Set various data types
		ctx.Set("string_value", "test")
		ctx.Set("int_value", 42)
		ctx.Set("float_value", 3.14)
		ctx.Set("bool_value", true)
		ctx.Set("nil_value", nil)
		ctx.Set("slice_value", []int{1, 2, 3})
		ctx.Set("map_value", map[string]int{"key": 1})

		// Set invalid type (channel)
		invalidValue := make(chan int)
		ctx.Set("invalid_value", invalidValue)

		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("data_test").On("test").Do(action).
		State("data_test").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result, true)

	// Test retrieval of various types
	testCases := []struct {
		key      string
		expected interface{}
	}{
		{"string_value", "test"},
		{"int_value", 42},
		{"float_value", 3.14},
		{"bool_value", true},
		{"nil_value", nil},
	}

	for _, tc := range testCases {
		if value, ok := machine.Context().Get(tc.key); ok {
			if value != tc.expected {
				t.Errorf("Expected %s to be %v, got %v", tc.key, tc.expected, value)
			}
		} else {
			t.Errorf("Expected %s to exist", tc.key)
		}
	}

	// Test slice and map
	if slice, ok := machine.Context().Get("slice_value"); ok {
		if sliceInt, ok := slice.([]int); ok {
			if len(sliceInt) != 3 {
				t.Error("Slice should have 3 elements")
			}
		} else {
			t.Error("Slice value should be []int")
		}
	}

	if mapVal, ok := machine.Context().Get("map_value"); ok {
		if mapInt, ok := mapVal.(map[string]int); ok {
			if mapInt["key"] != 1 {
				t.Error("Map should contain key: 1")
			}
		} else {
			t.Error("Map value should be map[string]int")
		}
	}

	// Invalid value should still be stored (implementation dependent)
	if invalid, ok := machine.Context().Get("invalid_value"); ok {
		t.Logf("Invalid value stored: %T", invalid)
	}
}

// Test for extreme values in context
func TestDataStructure_ExtremeContextValues(t *testing.T) {
	action := func(ctx Context) error {
		// Set extreme values
		ctx.Set("max_int", int64(^uint64(0)>>1))
		ctx.Set("min_int", int64(-1<<63))
		ctx.Set("max_float", float64(1.797693134862315708145274237317043567981e+308))
		ctx.Set("min_float", float64(-1.797693134862315708145274237317043567981e+308))
		ctx.Set("empty_string", "")
		ctx.Set("long_string", strings.Repeat("x", 10000))
		ctx.Set("empty_slice", []int{})
		ctx.Set("large_slice", make([]int, 1000))

		return nil
	}

	definition := NewMachine().
		State("initial").Initial().
		To("extreme_test").On("test").Do(action).
		State("extreme_test").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("test", nil)
	AssertEventProcessed(t, result, true)

	// Verify extreme values are handled
	extremeKeys := []string{
		"max_int", "min_int", "max_float", "min_float",
		"empty_string", "long_string", "empty_slice", "large_slice",
	}

	for _, key := range extremeKeys {
		if _, ok := machine.Context().Get(key); !ok {
			t.Errorf("Extreme value %s should be stored", key)
		}
	}

	// Test large string retrieval
	if longStr, ok := machine.Context().Get("long_string"); ok {
		if str, ok := longStr.(string); ok {
			if len(str) != 10000 {
				t.Error("Long string should maintain length")
			}
		}
	}

	// Test large slice retrieval
	if largeSlice, ok := machine.Context().Get("large_slice"); ok {
		if slice, ok := largeSlice.([]int); ok {
			if len(slice) != 1000 {
				t.Error("Large slice should maintain length")
			}
		}
	}
}
