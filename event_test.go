package fluo

import (
	"reflect"
	"testing"
	"time"
)

func TestEvent_BasicCreation(t *testing.T) {
	event := NewEvent("test_event", "test_data")

	if event.GetName() != "test_event" {
		t.Errorf("Expected event name 'test_event', got '%s'", event.GetName())
	}

	if event.GetData() != "test_data" {
		t.Errorf("Expected event data 'test_data', got '%v'", event.GetData())
	}

	if event.GetTimestamp().IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	if event.GetMetadata() == nil {
		t.Error("Expected non-nil metadata map")
	}

	if len(event.GetMetadata()) != 0 {
		t.Error("Expected empty metadata map initially")
	}
}

func TestEvent_WithMetadata(t *testing.T) {
	metadata := map[string]any{
		"source":   "test_system",
		"priority": 1,
		"tags":     []string{"important", "urgent"},
	}

	event := NewEventWithMetadata("test_event", "test_data", metadata)

	if event.GetName() != "test_event" {
		t.Errorf("Expected event name 'test_event', got '%s'", event.GetName())
	}

	if event.GetData() != "test_data" {
		t.Errorf("Expected event data 'test_data', got '%v'", event.GetData())
	}

	retrievedMetadata := event.GetMetadata()
	if len(retrievedMetadata) != len(metadata) {
		t.Errorf("Expected metadata length %d, got %d", len(metadata), len(retrievedMetadata))
	}

	for key, expectedValue := range metadata {
		if actualValue, exists := retrievedMetadata[key]; !exists {
			t.Errorf("Expected metadata key '%s' to exist", key)
		} else if !compareValues(expectedValue, actualValue) {
			t.Errorf("Expected metadata[%s] = %v, got %v", key, expectedValue, actualValue)
		}
	}
}

func TestEvent_TypedEvent(t *testing.T) {

	testCases := []struct {
		name      string
		eventName string
		data      any
	}{
		{"string", "string_event", "hello world"},
		{"int", "int_event", 42},
		{"bool", "bool_event", true},
		{"float", "float_event", 3.14},
		{"nil", "nil_event", nil},
		{"map", "map_event", map[string]string{"key": "value"}},
		{"slice", "slice_event", []int{1, 2, 3}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := NewTypedEvent(tc.eventName, tc.data)

			if event.GetName() != tc.eventName {
				t.Errorf("Expected event name '%s', got '%s'", tc.eventName, event.GetName())
			}

			if !compareValues(event.GetData(), tc.data) {
				t.Errorf("Expected event data %v, got %v", tc.data, event.GetData())
			}
		})
	}
}

func TestEvent_Timestamp(t *testing.T) {
	beforeCreate := time.Now()
	event := NewEvent("timestamp_test", nil)
	afterCreate := time.Now()

	timestamp := event.GetTimestamp()

	if timestamp.Before(beforeCreate) || timestamp.After(afterCreate) {
		t.Errorf("Expected timestamp to be between %v and %v, got %v",
			beforeCreate, afterCreate, timestamp)
	}
}

func TestEvent_MetadataIsolation(t *testing.T) {
	originalMetadata := map[string]any{
		"original": "value",
	}

	event := NewEventWithMetadata("test", "data", originalMetadata)

	retrievedMetadata := event.GetMetadata()
	if len(retrievedMetadata) != 1 || retrievedMetadata["original"] != "value" {
		t.Error("Expected to retrieve original metadata correctly")
	}

	retrievedMetadata["external_modification"] = "should_not_affect_event"

	newRetrievedMetadata := event.GetMetadata()
	if _, exists := newRetrievedMetadata["external_modification"]; exists {
		t.Error("Expected event metadata to be protected from external modifications")
	}

	if newRetrievedMetadata["original"] != "value" {
		t.Error("Expected original metadata to remain accessible")
	}
}

func TestEventResult_Creation(t *testing.T) {
	result := NewEventResult(true, true, "prev_state", "curr_state")

	if !result.Processed {
		t.Error("Expected processed to be true")
	}

	if !result.StateChanged {
		t.Error("Expected state changed to be true")
	}

	if result.PreviousState != "prev_state" {
		t.Errorf("Expected previous state 'prev_state', got '%s'", result.PreviousState)
	}

	if result.CurrentState != "curr_state" {
		t.Errorf("Expected current state 'curr_state', got '%s'", result.CurrentState)
	}

	if result.Error != nil {
		t.Error("Expected no error initially")
	}

	if result.RejectionReason != "" {
		t.Error("Expected empty rejection reason initially")
	}
}

func TestEventResult_WithError(t *testing.T) {
	testError := NewStateError(ErrCodeStateNotFound, "test_state", "test error")

	result := NewEventResult(false, false, "", "").WithError(testError)

	if result == nil {
		t.Error("Expected WithError to return the result for chaining")
		return
	}

	if result.Error != testError {
		t.Error("Expected error to be set correctly")
	}
}

func TestEventResult_WithRejection(t *testing.T) {
	result := NewEventResult(false, false, "current", "current").
		WithRejection("no valid transition")

	if result.Processed {
		t.Error("Expected processed to be false for rejected event")
	}

	if result.StateChanged {
		t.Error("Expected state changed to be false for rejected event")
	}

	if result.RejectionReason != "no valid transition" {
		t.Errorf("Expected rejection reason 'no valid transition', got '%s'", result.RejectionReason)
	}
}

func TestEvent_InMachineContext(t *testing.T) {
	eventData := map[string]string{
		"action": "process_data",
		"target": "database",
	}

	actionCalled := false
	capturedEvent := Event(nil)

	action := func(ctx Context) error {
		actionCalled = true
		capturedEvent = ctx.GetCurrentEvent()
		return nil
	}

	definition := NewMachine().
		State("idle").Initial().
		To("processing").On("process").Do(action).
		State("processing").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("process", eventData)

	AssertEventProcessed(t, result, true)

	if !actionCalled {
		t.Error("Expected action to be called")
	}

	if capturedEvent == nil {
		t.Error("Expected action to receive current event")
	}

	if capturedEvent.GetName() != "process" {
		t.Errorf("Expected event name 'process', got '%s'", capturedEvent.GetName())
	}

	if !compareValues(capturedEvent.GetData(), eventData) {
		t.Errorf("Expected event data %v, got %v", eventData, capturedEvent.GetData())
	}
}

func TestEvent_EventDataTypes(t *testing.T) {
	testCases := []struct {
		name string
		data any
	}{
		{"string", "hello"},
		{"int", 42},
		{"float", 3.14159},
		{"bool", true},
		{"nil", nil},
		{"map", map[string]int{"count": 10}},
		{"slice", []string{"a", "b", "c"}},
		{"struct", struct {
			Name string
			Age  int
		}{"John", 30}},
		{"pointer", &struct{ Value int }{42}},
	}

	definition := NewMachine().
		State("start").Initial().
		To("end").On("test").
		State("end").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			_ = machine.SetState("start")

			result := machine.HandleEvent("test", tc.data)
			AssertEventProcessed(t, result, true)

			event := machine.Context().GetCurrentEvent()
			if event == nil {
				t.Error("Expected current event to be available")
				return
			}

			if !compareValues(event.GetData(), tc.data) {
				t.Errorf("Expected event data %v, got %v", tc.data, event.GetData())
			}
		})
	}
}

func TestEvent_ConcurrentEvents(t *testing.T) {
	definition := NewMachine().
		State("idle").Initial().
		To("processing").On("process").
		State("processing").
		To("idle").On("complete").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	results := make(chan *EventResult, 100)

	for i := 0; i < 50; i++ {
		go func(id int) {
			eventName := "process"
			if id%2 == 0 {
				eventName = "complete"
			}
			result := machine.HandleEvent(eventName, id)
			results <- result
		}(i)
	}

	processedCount := 0
	for i := 0; i < 50; i++ {
		result := <-results
		if result.Processed {
			processedCount++
		}
	}

	if processedCount == 0 {
		t.Error("Expected some events to be processed")
	}

	currentState := machine.CurrentState()
	if currentState != "idle" && currentState != "processing" {
		t.Errorf("Expected machine to be in valid state, got '%s'", currentState)
	}
}

func TestEvent_EventSequencing(t *testing.T) {
	events := []string{"event1", "event2", "event3", "event4"}
	receivedEvents := make([]string, 0, len(events))

	action := func(ctx Context) error {
		if event := ctx.GetCurrentEvent(); event != nil {
			receivedEvents = append(receivedEvents, event.GetName())
		}
		return nil
	}

	definition := NewMachine().
		State("state").Initial().
		ToSelf().On("event1").Do(action).
		ToSelf().On("event2").Do(action).
		ToSelf().On("event3").Do(action).
		ToSelf().On("event4").Do(action).
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	for _, eventName := range events {
		result := machine.HandleEvent(eventName, nil)
		AssertEventProcessed(t, result, true)
	}

	if len(receivedEvents) != len(events) {
		t.Errorf("Expected %d events, received %d", len(events), len(receivedEvents))
	}

	for i, expected := range events {
		if i < len(receivedEvents) && receivedEvents[i] != expected {
			t.Errorf("Expected event[%d] = '%s', got '%s'", i, expected, receivedEvents[i])
		}
	}
}

func TestEvent_LargeEventData(t *testing.T) {

	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	event := NewEvent("large_data_event", largeData)

	if event.GetName() != "large_data_event" {
		t.Error("Expected event name to be preserved with large data")
	}

	retrievedData := event.GetData().([]byte)
	if len(retrievedData) != len(largeData) {
		t.Errorf("Expected data length %d, got %d", len(largeData), len(retrievedData))
	}

	for i, expected := range largeData {
		if retrievedData[i] != expected {
			t.Errorf("Data corruption at index %d: expected %d, got %d", i, expected, retrievedData[i])
			break
		}
	}
}

func TestEvent_ComplexMetadata(t *testing.T) {
	complexMetadata := map[string]any{
		"nested": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep_value",
			},
		},
		"array":     []int{1, 2, 3, 4, 5},
		"timestamp": time.Now(),
		"mixed": []interface{}{
			"string",
			42,
			true,
			map[string]string{"inner": "value"},
		},
	}

	event := NewEventWithMetadata("complex_event", "data", complexMetadata)
	retrievedMetadata := event.GetMetadata()

	if nested, ok := retrievedMetadata["nested"].(map[string]interface{}); ok {
		if level2, ok := nested["level2"].(map[string]interface{}); ok {
			if level3, ok := level2["level3"].(string); !ok || level3 != "deep_value" {
				t.Error("Expected deeply nested metadata to be preserved")
			}
		} else {
			t.Error("Expected level2 nested metadata to be accessible")
		}
	} else {
		t.Error("Expected nested metadata to be accessible")
	}

	if array, ok := retrievedMetadata["array"].([]int); !ok || len(array) != 5 {
		t.Error("Expected array metadata to be preserved")
	}
}

func compareValues(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if sliceA, okA := a.([]string); okA {
		if sliceB, okB := b.([]string); okB {
			if len(sliceA) != len(sliceB) {
				return false
			}
			for i, v := range sliceA {
				if sliceB[i] != v {
					return false
				}
			}
			return true
		}
		return false
	}

	if sliceA, okA := a.([]int); okA {
		if sliceB, okB := b.([]int); okB {
			if len(sliceA) != len(sliceB) {
				return false
			}
			for i, v := range sliceA {
				if sliceB[i] != v {
					return false
				}
			}
			return true
		}
		return false
	}

	if mapA, okA := a.(map[string]string); okA {
		if mapB, okB := b.(map[string]string); okB {
			if len(mapA) != len(mapB) {
				return false
			}
			for k, v := range mapA {
				if mapB[k] != v {
					return false
				}
			}
			return true
		}
		return false
	}

	if mapA, okA := a.(map[string]int); okA {
		if mapB, okB := b.(map[string]int); okB {
			if len(mapA) != len(mapB) {
				return false
			}
			for k, v := range mapA {
				if mapB[k] != v {
					return false
				}
			}
			return true
		}
		return false
	}

	return reflect.DeepEqual(a, b)
}
