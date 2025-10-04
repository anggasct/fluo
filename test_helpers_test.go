package fluo

import (
	"context"
	"testing"
)

func TestTestHelpers_Functions(t *testing.T) {
	t.Run("TestObserver Basic Functionality", func(t *testing.T) {
		observer := NewTestObserver()

		// Test initial state
		if observer.TransitionCount() != 0 {
			t.Errorf("Expected 0 transitions initially, got %d", observer.TransitionCount())
		}

		if observer.StateEnterCount() != 0 {
			t.Errorf("Expected 0 state enters initially, got %d", observer.StateEnterCount())
		}

		if observer.StateExitCount() != 0 {
			t.Errorf("Expected 0 state exits initially, got %d", observer.StateExitCount())
		}

		// Test event recording
		testEvent := NewEvent("test_event", "test_data")
		machine := createTestMachine()
		testCtx := NewContext(context.Background(), machine)

		observer.OnTransition("state_a", "state_b", testEvent, testCtx)
		observer.OnStateEnter("state_b", testCtx)
		observer.OnStateExit("state_a", testCtx)

		if observer.TransitionCount() != 1 {
			t.Errorf("Expected 1 transition, got %d", observer.TransitionCount())
		}

		if observer.StateEnterCount() != 1 {
			t.Errorf("Expected 1 state enter, got %d", observer.StateEnterCount())
		}

		if observer.StateExitCount() != 1 {
			t.Errorf("Expected 1 state exit, got %d", observer.StateExitCount())
		}

		// Test event rejection
		observer.OnEventRejected(testEvent, "test rejection", testCtx)
		if len(observer.EventRejects) != 1 {
			t.Errorf("Expected 1 event rejection, got %d", len(observer.EventRejects))
		}

		// Test guard evaluation
		observer.OnGuardEvaluation("state_a", "state_b", testEvent, true, testCtx)
		if len(observer.Guards) != 1 {
			t.Errorf("Expected 1 guard evaluation, got %d", len(observer.Guards))
		}

		// Test action execution
		observer.OnActionExecution("test_action", "state_b", testEvent, testCtx)
		if len(observer.Actions) != 1 {
			t.Errorf("Expected 1 action execution, got %d", len(observer.Actions))
		}

		// Test error
		testError := NewStateNotFoundError("test_state")
		observer.OnError(testError, testCtx)
		if len(observer.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(observer.Errors))
		}

		// Test lifecycle events
		observer.OnMachineStarted(testCtx)
		observer.OnMachineStopped(testCtx)
		if len(observer.Started) != 1 {
			t.Errorf("Expected 1 started event, got %d", len(observer.Started))
		}
		if len(observer.Stopped) != 1 {
			t.Errorf("Expected 1 stopped event, got %d", len(observer.Stopped))
		}
	})

	t.Run("TestObserver Event Access", func(t *testing.T) {
		observer := NewTestObserver()
		testEvent := NewEvent("test", "data")
		machine := createTestMachine()
		testCtx := NewContext(context.Background(), machine)

		// Add some events
		observer.OnTransition("a", "b", testEvent, testCtx)
		observer.OnStateEnter("b", testCtx)
		observer.OnStateExit("a", testCtx)

		// Test direct access to event arrays
		if len(observer.Transitions) != 1 {
			t.Errorf("Expected 1 transition, got %d", len(observer.Transitions))
		}

		if len(observer.StateEnters) != 1 {
			t.Errorf("Expected 1 state enter, got %d", len(observer.StateEnters))
		}

		if len(observer.StateExits) != 1 {
			t.Errorf("Expected 1 state exit, got %d", len(observer.StateExits))
		}

		// Verify event data
		if observer.Transitions[0].From != "a" || observer.Transitions[0].To != "b" {
			t.Error("Transition data mismatch")
		}

		if observer.StateEnters[0].State != "b" {
			t.Error("State enter data mismatch")
		}

		if observer.StateExits[0].State != "a" {
			t.Error("State exit data mismatch")
		}
	})

	t.Run("TestObserver Reset", func(t *testing.T) {
		observer := NewTestObserver()
		testEvent := NewEvent("test", "data")
		machine := createTestMachine()
		testCtx := NewContext(context.Background(), machine)

		// Add events
		observer.OnTransition("a", "b", testEvent, testCtx)
		observer.OnStateEnter("b", testCtx)

		// Verify events exist
		if observer.TransitionCount() != 1 {
			t.Error("Expected 1 transition before reset")
		}

		// Reset observer
		observer.Reset()

		// Verify events are cleared
		if observer.TransitionCount() != 0 {
			t.Error("Expected 0 transitions after reset")
		}

		if observer.StateEnterCount() != 0 {
			t.Error("Expected 0 state enters after reset")
		}
	})
}
