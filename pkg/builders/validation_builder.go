package builders

import (
	"fmt"

	"github.com/anggasct/fluo/pkg/core"
	"github.com/anggasct/fluo/pkg/observers"
	"github.com/anggasct/fluo/pkg/states"
)

// ValidationBuilder helps build validation rules for state machines
type ValidationBuilder struct {
	observer *observers.ValidationObserver
	sm       *core.StateMachine
}

// NewValidationBuilder creates a new validation builder
func NewValidationBuilder(sm *core.StateMachine) *ValidationBuilder {
	return &ValidationBuilder{
		observer: observers.NewValidationObserver(),
		sm:       sm,
	}
}

// ExpectState adds an expected state to validation
func (v *ValidationBuilder) ExpectState(stateName string) *ValidationBuilder {
	v.observer.AddExpectedState(stateName)
	return v
}

// AllowTransition adds an allowed transition to validation
func (v *ValidationBuilder) AllowTransition(from, to string) *ValidationBuilder {
	v.observer.AddAllowedTransition(from, to)
	return v
}

// Build returns the validation observer
func (v *ValidationBuilder) Build() *observers.ValidationObserver {
	return v.observer
}

// ValidateStateMachine validates the state machine configuration
func (v *ValidationBuilder) ValidateStateMachine() error {
	// Validate that there's an initial state
	if v.sm.GetInitialState() == nil {
		return fmt.Errorf("state machine '%s' has no initial state", v.sm.Name())
	}

	// Validate all states
	stateErrors := make([]error, 0)
	allStates := v.sm.GetAllStates()

	for _, state := range allStates {
		// Validate parallel states
		if parallelState, ok := state.(*states.ParallelState); ok {
			if err := v.validateParallelState(parallelState); err != nil {
				stateErrors = append(stateErrors, fmt.Errorf("parallel state '%s' is invalid: %w", state.Name(), err))
			}
		}

		// More state validations can be added here...
	}

	if len(stateErrors) > 0 {
		return fmt.Errorf("found %d state validation errors", len(stateErrors))
	}

	return nil
}

// validateParallelState validates a parallel state configuration
func (v *ValidationBuilder) validateParallelState(ps *states.ParallelState) error {
	// Check if it has regions
	regions := ps.GetAllRegions()
	if len(regions) == 0 {
		return fmt.Errorf("parallel state has no regions")
	}

	// Check each region
	var regionErrors []error
	for name, region := range regions {
		// Check if region has a state machine
		if region.GetStateMachine() == nil {
			regionErrors = append(regionErrors, fmt.Errorf("region '%s' has no state machine", name))
			continue
		}

		// Check if region state machine has an initial state
		if region.GetStateMachine().GetInitialState() == nil {
			regionErrors = append(regionErrors, fmt.Errorf("region '%s' has no initial state", name))
		}
	}

	if len(regionErrors) > 0 {
		return fmt.Errorf("%d region validation errors", len(regionErrors))
	}

	return nil
}

// Validate validates and returns any errors found
func (v *ValidationBuilder) Validate() error {
	return v.ValidateStateMachine()
}
