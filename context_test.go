package fluo

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type testContextKey string

func TestContext_Creation(t *testing.T) {
	parentCtx := context.Background()
	machine := CreateSimpleMachine()

	ctx := NewContext(parentCtx, machine)

	if ctx == nil {
		t.Error("Expected non-nil context")
	}

	if ctx.GetMachine() != machine {
		t.Error("Expected context to reference correct machine")
	}

	if ctx.GetCurrentState() != "" {
		t.Error("Expected empty current state initially")
	}

	if ctx.GetSourceState() != "" {
		t.Error("Expected empty source state initially")
	}

	if ctx.GetTargetState() != "" {
		t.Error("Expected empty target state initially")
	}

	if ctx.GetCurrentEvent() != nil {
		t.Error("Expected nil current event initially")
	}
}

func TestContext_SimpleCreation(t *testing.T) {
	ctx := NewSimpleContext()

	if ctx == nil {
		t.Error("Expected non-nil simple context")
	}

	if ctx.GetMachine() != nil {
		t.Error("Expected nil machine for simple context")
	}

	ctx.Set("test", "value")
	if value, ok := ctx.Get("test"); !ok || value != "value" {
		t.Error("Expected simple context to support data operations")
	}
}

func TestContext_DataOperations(t *testing.T) {
	ctx := CreateTestContext()

	ctx.Set("key1", "value1")
	ctx.Set("key2", 42)
	ctx.Set("key3", true)

	if value, ok := ctx.Get("key1"); !ok || value != "value1" {
		t.Error("Expected to retrieve string value")
	}

	if value, ok := ctx.Get("key2"); !ok || value != 42 {
		t.Error("Expected to retrieve int value")
	}

	if value, ok := ctx.Get("key3"); !ok || value != true {
		t.Error("Expected to retrieve bool value")
	}

	if _, ok := ctx.Get("nonexistent"); ok {
		t.Error("Expected non-existent key to return false")
	}
}

func TestContext_GetAll(t *testing.T) {
	ctx := CreateTestContext()

	testData := map[string]interface{}{
		"string": "value",
		"int":    42,
		"bool":   true,
		"float":  3.14,
	}

	for key, value := range testData {
		ctx.Set(key, value)
	}

	allData := ctx.GetAll()

	if len(allData) != len(testData) {
		t.Errorf("Expected %d items, got %d", len(testData), len(allData))
	}

	for key, expectedValue := range testData {
		if actualValue, exists := allData[key]; !exists || actualValue != expectedValue {
			t.Errorf("Expected key '%s' with value %v, got %v (exists: %v)",
				key, expectedValue, actualValue, exists)
		}
	}
}

func TestContext_EventOperations(t *testing.T) {
	ctx := CreateTestContext()

	if ctx.GetCurrentEvent() != nil {
		t.Error("Expected no current event initially")
	}

	if ctx.GetEventName() != "" {
		t.Error("Expected empty event name initially")
	}

	if ctx.GetEventData() != nil {
		t.Error("Expected nil event data initially")
	}

	testEvent := CreateTestEvent("test_event", "test_data")
	if smCtx, ok := ctx.(*StateMachineContext); ok {
		smCtx.updateCurrentEvent(testEvent)
	}

	if ctx.GetCurrentEvent() != testEvent {
		t.Error("Expected current event to be set")
	}

	if ctx.GetEventName() != "test_event" {
		t.Error("Expected event name to be 'test_event'")
	}

	if ctx.GetEventData() != "test_data" {
		t.Error("Expected event data to be 'test_data'")
	}
}

func TestContext_GetEventDataAs(t *testing.T) {
	ctx := CreateTestContext()

	testCases := []struct {
		name        string
		eventData   interface{}
		targetType  interface{}
		shouldWork  bool
		expectedVal interface{}
	}{
		{"string", "hello", new(string), true, "hello"},
		{"int", 42, new(int), true, 42},
		{"bool", true, new(bool), true, true},
		{"float64", 3.14, new(float64), true, 3.14},
		{"wrong_type", "hello", new(int), false, 0},
		{"nil_data", nil, new(string), false, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			testEvent := CreateTestEvent("test", tc.eventData)
			if smCtx, ok := ctx.(*StateMachineContext); ok {
				smCtx.updateCurrentEvent(testEvent)
			}

			success := ctx.GetEventDataAs(tc.targetType)

			if success != tc.shouldWork {
				t.Errorf("Expected success=%v, got %v", tc.shouldWork, success)
				return
			}

			if tc.shouldWork {
				switch target := tc.targetType.(type) {
				case *string:
					if *target != tc.expectedVal.(string) {
						t.Errorf("Expected %v, got %v", tc.expectedVal, *target)
					}
				case *int:
					if *target != tc.expectedVal.(int) {
						t.Errorf("Expected %v, got %v", tc.expectedVal, *target)
					}
				case *bool:
					if *target != tc.expectedVal.(bool) {
						t.Errorf("Expected %v, got %v", tc.expectedVal, *target)
					}
				case *float64:
					if *target != tc.expectedVal.(float64) {
						t.Errorf("Expected %v, got %v", tc.expectedVal, *target)
					}
				}
			}
		})
	}
}

func TestContext_StateOperations(t *testing.T) {
	ctx := CreateTestContext()

	if smCtx, ok := ctx.(*StateMachineContext); ok {
		smCtx.updateCurrentState("current_state")
		smCtx.updateTransitionInfo("current_state", "source_state", "target_state", nil)
	}

	if ctx.GetCurrentState() != "current_state" {
		t.Error("Expected current state to be 'current_state'")
	}

	if ctx.GetSourceState() != "source_state" {
		t.Error("Expected source state to be 'source_state'")
	}

	if ctx.GetTargetState() != "target_state" {
		t.Error("Expected target state to be 'target_state'")
	}
}

func TestContext_WithValue(t *testing.T) {
	originalCtx := CreateTestContext()
	originalCtx.Set("original", "value")

	newCtx := originalCtx.WithValue("new", "new_value")

	if _, ok := originalCtx.Get("new"); ok {
		t.Error("Expected original context not to have new value")
	}

	if value, ok := newCtx.Get("original"); !ok || value != "value" {
		t.Error("Expected new context to have original value")
	}

	if value, ok := newCtx.Get("new"); !ok || value != "new_value" {
		t.Error("Expected new context to have new value")
	}

	originalCtx.Set("modified", "after")
	if _, ok := newCtx.Get("modified"); ok {
		t.Error("Expected new context not to be affected by original modifications")
	}
}

func TestContext_Fork(t *testing.T) {
	originalCtx := CreateTestContext()
	originalCtx.Set("shared", "value")
	originalCtx.Set("number", 42)

	if smCtx, ok := originalCtx.(*StateMachineContext); ok {
		smCtx.updateCurrentState("test_state")
	}

	forkedCtx := originalCtx.Fork()

	if value, ok := forkedCtx.Get("shared"); !ok || value != "value" {
		t.Error("Expected forked context to have shared data")
	}

	if value, ok := forkedCtx.Get("number"); !ok || value != 42 {
		t.Error("Expected forked context to have number data")
	}

	if forkedCtx.GetCurrentState() != "test_state" {
		t.Error("Expected forked context to have same current state")
	}

	originalCtx.Set("original_only", "value")
	forkedCtx.Set("forked_only", "value")

	if _, ok := forkedCtx.Get("original_only"); ok {
		t.Error("Expected forked context not to see original-only modifications")
	}

	if _, ok := originalCtx.Get("forked_only"); ok {
		t.Error("Expected original context not to see forked-only modifications")
	}
}

func TestContext_ThreadSafety(t *testing.T) {
	ctx := CreateTestContext()

	const numGoroutines = 100
	const opsPerGoroutine = 100

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				ctx.Set(key, value)
			}
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)

				ctx.Get(key)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = ctx.GetAll()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	ctx.Set("final_test", "success")
	if value, ok := ctx.Get("final_test"); !ok || value != "success" {
		t.Error("Expected context to remain functional after concurrent access")
	}
}

func TestContext_IntegrationWithMachine(t *testing.T) {
	actionCalled := false
	testAction := func(ctx Context) error {
		actionCalled = true

		if ctx.GetMachine() == nil {
			t.Error("Expected context to reference a machine")
		}

		if ctx.GetSourceState() == "" {
			t.Error("Expected source state to be set during transition")
		}

		if ctx.GetTargetState() == "" {
			t.Error("Expected target state to be set during transition")
		}

		if ctx.GetCurrentEvent() == nil {
			t.Error("Expected current event to be set during transition")
		}

		ctx.Set("transition_data", "stored_during_transition")

		return nil
	}

	builder := NewMachine()
	builder.State("idle").Initial().
		To("running").On("start").Do(testAction)
	builder.State("running")
	definition := builder.Build()

	testMachine := definition.CreateInstance()
	_ = testMachine.Start()

	result := testMachine.HandleEvent("start", "test_event_data")
	AssertEventProcessed(t, result, true)

	if !actionCalled {
		t.Error("Expected transition action to be called")
	}

	if value, ok := testMachine.Context().Get("transition_data"); !ok || value != "stored_during_transition" {
		t.Error("Expected context data to persist after transition")
	}
}

func TestContext_PreviousStateTracking(t *testing.T) {
	definition := NewMachine().
		State("state1").Initial().
		To("state2").On("next").
		State("state2").
		To("state3").On("next").
		State("state3").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	if machine.Context().GetPreviousState() != "" {
		t.Error("Expected no previous state initially")
	}

	_ = machine.HandleEvent("next", nil)

	_ = machine.HandleEvent("next", nil)

	if machine.Context().GetCurrentState() != "state3" {
		t.Error("Expected current state to be properly tracked")
	}
}

func TestContext_ContextWithValue(t *testing.T) {

	parentCtx := context.WithValue(context.Background(), testContextKey("parent_key"), "parent_value")
	machine := CreateSimpleMachine()

	ctx := NewContext(parentCtx, machine)

	if value := ctx.Value(testContextKey("parent_key")); value != "parent_value" {
		t.Error("Expected to inherit parent context values")
	}

	ctx.Set("local_key", "local_value")
	if value, ok := ctx.Get("local_key"); !ok || value != "local_value" {
		t.Error("Expected to support local context data")
	}
}

func TestContext_ContextCancellation(t *testing.T) {

	parentCtx, cancel := context.WithCancel(context.Background())
	machine := CreateSimpleMachine()

	ctx := NewContext(parentCtx, machine)

	select {
	case <-ctx.Done():
		t.Error("Expected context not to be cancelled initially")
	default:

	}

	cancel()

	select {
	case <-ctx.Done():

	case <-time.After(100 * time.Millisecond):
		t.Error("Expected context to be cancelled after parent cancellation")
	}

	if ctx.Err() == nil {
		t.Error("Expected context error after cancellation")
	}
}

func TestContext_ComplexDataStructures(t *testing.T) {
	ctx := CreateTestContext()

	complexData := map[string]interface{}{
		"nested": map[string]string{
			"inner": "value",
		},
		"array": []string{"item1", "item2", "item3"},
		"struct": struct {
			Name string
			Age  int
		}{"test", 25},
	}

	ctx.Set("complex", complexData)

	retrieved, ok := ctx.Get("complex")
	if !ok {
		t.Error("Expected to retrieve complex data")
	}

	retrievedMap, ok := retrieved.(map[string]interface{})
	if !ok {
		t.Error("Expected retrieved data to maintain type")
	}

	if nested, ok := retrievedMap["nested"].(map[string]string); ok {
		if nested["inner"] != "value" {
			t.Error("Expected nested data to be preserved")
		}
	} else {
		t.Error("Expected nested data to be accessible")
	}
}
