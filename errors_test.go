package fluo

import (
	"testing"
)

func TestErrors_ErrorCode(t *testing.T) {
	testCases := []ErrorCode{
		ErrCodeNone,
		ErrCodeStateNotFound,
		ErrCodeTransitionNotAllowed,
		ErrCodeGuardRejected,
		ErrCodeInvalidEvent,
		ErrCodeMachineNotStarted,
		ErrCodeActionFailed,
		ErrCodeInvalidConfiguration,
		ErrCodeInvalidState,
		ErrCodeConcurrentModification,
	}

	for i, code := range testCases {
		if int(code) != i {
			t.Errorf("Expected error code %d to have value %d", i, int(code))
		}
	}
}

func TestStateError_Creation(t *testing.T) {
	err := NewStateNotFoundError("test_state")

	if err.Code != ErrCodeStateNotFound {
		t.Errorf("Expected error code %v, got %v", ErrCodeStateNotFound, err.Code)
	}

	if err.StateID != "test_state" {
		t.Errorf("Expected state ID 'test_state', got '%s'", err.StateID)
	}

	if err.Message == "" {
		t.Error("Expected non-empty error message")
	}

	errorString := err.Error()
	if errorString == "" {
		t.Error("Expected non-empty error string")
	}

	if !contains(errorString, "test_state") {
		t.Error("Expected error string to contain state ID")
	}
}

func TestStateError_CustomError(t *testing.T) {
	err := NewStateError(ErrCodeInvalidState, "custom_state", "custom message")

	if err.Code != ErrCodeInvalidState {
		t.Error("Expected custom error code")
	}

	if err.StateID != "custom_state" {
		t.Error("Expected custom state ID")
	}

	if err.Message != "custom message" {
		t.Error("Expected custom message")
	}
}

func TestStateError_InvalidStateError(t *testing.T) {
	err := NewInvalidStateError("invalid_state", "state is invalid")

	if err.Code != ErrCodeInvalidState {
		t.Error("Expected invalid state error code")
	}

	if err.StateID != "invalid_state" {
		t.Error("Expected correct state ID")
	}

	if err.Message != "state is invalid" {
		t.Error("Expected correct reason")
	}
}

func TestTransitionError_Creation(t *testing.T) {
	err := NewTransitionNotAllowedError("from_state", "to_state", "event")

	if err.Code != ErrCodeTransitionNotAllowed {
		t.Error("Expected transition not allowed error code")
	}

	if err.From != "from_state" {
		t.Error("Expected correct from state")
	}

	if err.To != "to_state" {
		t.Error("Expected correct to state")
	}

	if err.Event != "event" {
		t.Error("Expected correct event name")
	}

	if err.Reason == "" {
		t.Error("Expected non-empty reason")
	}

	errorString := err.Error()
	if !contains(errorString, "from_state") || !contains(errorString, "to_state") || !contains(errorString, "event") {
		t.Error("Expected error string to contain transition details")
	}
}

func TestTransitionError_CustomError(t *testing.T) {
	err := NewTransitionError(ErrCodeGuardRejected, "state1", "state2", "test_event", "guard failed")

	if err.Code != ErrCodeGuardRejected {
		t.Error("Expected guard rejected error code")
	}

	if err.Reason != "guard failed" {
		t.Error("Expected custom reason")
	}
}

func TestNoTransitionError(t *testing.T) {
	err := NewNoTransitionError("current_state", "invalid_event")

	if err == nil {
		t.Error("Expected non-nil error")
		return
	}

	errorString := err.Error()
	if !contains(errorString, "current_state") || !contains(errorString, "invalid_event") {
		t.Error("Expected error to contain state and event information")
	}
}

func TestMachineError_Creation(t *testing.T) {
	err := NewMachineError(ErrCodeMachineNotStarted, "Start", "machine not ready")

	if err == nil {
		t.Error("Expected non-nil machine error")
		return
	}

	errorString := err.Error()
	if errorString == "" {
		t.Error("Expected non-empty error string")
	}
}

func TestMachineNotStartedError(t *testing.T) {
	err := NewMachineNotStartedError("HandleEvent")

	if err == nil {
		t.Error("Expected non-nil error")
		return
	}

	errorString := err.Error()
	if !contains(errorString, "HandleEvent") {
		t.Error("Expected error to contain operation name")
	}
}

func TestConfigurationError(t *testing.T) {
	err := NewConfigurationError("StateMachine", "invalid configuration")

	if err == nil {
		t.Error("Expected non-nil configuration error")
		return
	}

	errorString := err.Error()
	if !contains(errorString, "StateMachine") || !contains(errorString, "invalid configuration") {
		t.Error("Expected error to contain component and message")
	}
}

func TestErrors_InMachineOperations(t *testing.T) {
	machine := CreateSimpleMachine()

	result := machine.HandleEvent("start", nil)
	if result.Processed {
		t.Error("Expected event to be rejected when machine not started")
	}

	if result.RejectionReason == "" {
		t.Error("Expected rejection reason")
	}

	_ = machine.Start()

	err := machine.SetState("nonexistent")
	if err == nil {
		t.Error("Expected error when setting nonexistent state")
	}

	if stateErr, ok := err.(*StateError); ok {
		if stateErr.Code != ErrCodeStateNotFound {
			t.Error("Expected state not found error code")
		}
	}

	result = machine.HandleEvent("invalid_event", nil)
	if result.Processed {
		t.Error("Expected invalid event to be rejected")
	}
}

func TestErrors_StartStopErrors(t *testing.T) {
	machine := CreateSimpleMachine()

	_ = machine.Start()
	err := machine.Start()
	if err == nil {
		t.Error("Expected error when starting already started machine")
	}

	newMachine := CreateSimpleMachine()
	err = newMachine.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-started machine")
	}
}

func TestErrors_BuilderValidationErrors(t *testing.T) {

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when building machine without initial state")
		}
	}()

	NewMachine().
		State("state1").
		Build()
}

func TestErrors_InvalidTransitionReferences(t *testing.T) {

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when building machine with invalid target state")
		}
	}()

	NewMachine().
		State("start").Initial().
		To("nonexistent").On("go").
		Build()
}

func TestErrors_ActionErrors(t *testing.T) {
	originalErr := NewStateError(ErrCodeActionFailed, "test_state", "action failed")
	actionError := NewActionError("test_action", "test_state", originalErr)

	definition := NewMachine().
		State("idle").Initial().
		To("running").On("start").Do(func(ctx Context) error {
		return actionError
	}).
		State("running").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	result := machine.HandleEvent("start", nil)

	if result.Error != nil {
		t.Logf("Action error captured: %v", result.Error)
	}
}

func TestErrors_ConcurrentAccess(t *testing.T) {

	machine := CreateSimpleMachine()
	_ = machine.Start()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in goroutine %d: %v", id, r)
				}
				done <- true
			}()

			for j := 0; j < 10; j++ {
				machine.HandleEvent("start", nil)
				machine.HandleEvent("stop", nil)
				machine.CurrentState()
				machine.GetActiveStates()
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestErrors_ErrorTypeAssertions(t *testing.T) {
	stateErr := NewStateNotFoundError("test")
	transitionErr := NewTransitionNotAllowedError("from", "to", "event")

	if !IsStateError(stateErr) {
		t.Error("Expected StateError to be identified correctly")
	}

	if !IsTransitionError(transitionErr) {
		t.Error("Expected TransitionError to be identified correctly")
	}

	var err1 error = stateErr
	var err2 error = transitionErr

	if err1.Error() == "" {
		t.Error("Expected StateError to implement error interface")
	}

	if err2.Error() == "" {
		t.Error("Expected TransitionError to implement error interface")
	}
}

func TestErrors_ErrorRecovery(t *testing.T) {

	definition := NewMachine().
		State("normal").Initial().
		To("error").On("cause_error").
		To("recovery").On("recover").
		State("error").
		To("normal").On("fix").
		State("recovery").
		To("normal").On("complete").
		Build()

	machine := definition.CreateInstance()
	_ = machine.Start()

	AssertState(t, machine, "normal")

	result := machine.HandleEvent("cause_error", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "error")

	result = machine.HandleEvent("fix", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "normal")

	result = machine.HandleEvent("recover", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "recovery")

	result = machine.HandleEvent("complete", nil)
	AssertEventProcessed(t, result, true)
	AssertState(t, machine, "normal")
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr ||
			(len(str) > len(substr) &&
				(str[:len(substr)] == substr ||
					str[len(str)-len(substr):] == substr ||
					containsMiddle(str, substr))))
}

func containsMiddle(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestErrorCreation_Functions(t *testing.T) {
	t.Run("NewStateNotFoundError", func(t *testing.T) {
		err := NewStateNotFoundError("test_state")
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Code != ErrCodeStateNotFound {
			t.Errorf("Expected ErrCodeStateNotFound, got %v", err.Code)
		}

		if err.StateID != "test_state" {
			t.Errorf("Expected 'test_state', got '%s'", err.StateID)
		}

		expectedMsg := "state 'test_state' not found"
		if err.Message != expectedMsg {
			t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Message)
		}

		expectedError := "state error [test_state]: state 'test_state' not found"
		if err.Error() != expectedError {
			t.Errorf("Expected '%s', got '%s'", expectedError, err.Error())
		}
	})

	t.Run("NewInvalidStateError", func(t *testing.T) {
		err := NewInvalidStateError("invalid_state", "test message")
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Code != ErrCodeInvalidState {
			t.Errorf("Expected ErrCodeInvalidState, got %v", err.Code)
		}

		if err.StateID != "invalid_state" {
			t.Errorf("Expected 'invalid_state', got '%s'", err.StateID)
		}

		if err.Message != "test message" {
			t.Errorf("Expected 'test message', got '%s'", err.Message)
		}
	})

	t.Run("NewTransitionError", func(t *testing.T) {
		err := NewTransitionError(ErrCodeTransitionNotAllowed, "state_a", "state_b", "test_event", "test message")
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Code != ErrCodeTransitionNotAllowed {
			t.Errorf("Expected ErrCodeTransitionNotAllowed, got %v", err.Code)
		}

		expectedMsg := "transition error [state_a->state_b on test_event]: test message"
		if err.Error() != expectedMsg {
			t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("NewNoTransitionError", func(t *testing.T) {
		err := NewNoTransitionError("test_state", "test_event")
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Code != ErrCodeTransitionNotAllowed {
			t.Errorf("Expected ErrCodeTransitionNotAllowed, got %v", err.Code)
		}

		expectedMsg := "transition error [test_state-> on test_event]: no transition found from state 'test_state' for event 'test_event'"
		if err.Error() != expectedMsg {
			t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("NewMachineError", func(t *testing.T) {
		err := NewMachineError(ErrCodeMachineNotStarted, "test_operation", "test message")
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Code != ErrCodeMachineNotStarted {
			t.Errorf("Expected ErrCodeMachineNotStarted, got %v", err.Code)
		}

		if err.Operation != "test_operation" {
			t.Errorf("Expected 'test_operation', got '%s'", err.Operation)
		}

		if err.Message != "test message" {
			t.Errorf("Expected 'test message', got '%s'", err.Message)
		}

		expectedMsg := "machine error during test_operation: test message"
		if err.Error() != expectedMsg {
			t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("NewMachineNotStartedError", func(t *testing.T) {
		err := NewMachineNotStartedError("test_operation")
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Code != ErrCodeMachineNotStarted {
			t.Errorf("Expected ErrCodeMachineNotStarted, got %v", err.Code)
		}

		expectedMsg := "machine error during test_operation: state machine is not started"
		if err.Error() != expectedMsg {
			t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("NewConfigurationError", func(t *testing.T) {
		err := NewConfigurationError("test_operation", "validation failed")
		if err == nil {
			t.Error("Expected non-nil error")
		}

		// ConfigurationError has different structure
		expectedMsg := "configuration error in test_operation: validation failed"
		if err.Error() != expectedMsg {
			t.Errorf("Expected '%s', got '%s'", expectedMsg, err.Error())
		}
	})
}
