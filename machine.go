package fluo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Machine represents a state machine instance
type Machine interface {
	Start() error
	Stop() error
	Reset() error

	CurrentState() string
	SetState(state string) error

	SetRegionState(regionID string, stateID string) error
	RegionState(regionID string) string
	GetStateHierarchy() []string
	IsInState(stateID string) bool
	GetActiveStates() []string
	IsStateActive(stateID string) bool
	GetParallelRegions() map[string][]string

	SendEvent(eventName string, eventData any) *EventResult
	SendEventWithContext(ctx context.Context, eventName string, eventData any) *EventResult
	HandleEvent(eventName string, eventData any) *EventResult
	HandleEventWithContext(ctx context.Context, eventName string, eventData any) *EventResult

	AddObserver(observer Observer)
	RemoveObserver(observer Observer)

	Context() Context
	WithContext(ctx Context) Machine

	MarshalJSON() ([]byte, error)
	UnmarshalJSON(data []byte) error
}

// MachineDefinition represents the configuration of a state machine
type MachineDefinition interface {
	CreateInstance() Machine
	Build() MachineDefinition

	GetInitialState() string
	GetStates() map[string]State
	GetTransitions() map[string][]Transition
}

// MachineState represents the current state of the machine
type MachineState int

const (
	// Machine is stopped and not processing events
	MachineStateStopped MachineState = iota
	// Machine is running and processing events
	MachineStateStarted
	// Machine is in error state
	MachineStateError
)

// StateMachine implements the Machine interface
type StateMachine struct {
	currentState string
	activeStates map[string]bool // Track multiple active states for parallel execution
	initialState string
	states       map[string]State
	transitions  map[string][]Transition
	context      Context
	observers    *ObserverManager
	machineState MachineState
	mutex        sync.RWMutex

	stateHistory map[string]string

	// Parallel execution support
	parallelRegions map[string][]string        // Track active states per region
	joinConditions  map[string][][]string      // Track required source state combinations for join pseudostates
	joinTracking    map[string]map[string]bool // Track which source states have arrived at each join
}

// newStateMachine creates a new state machine instance
func newStateMachine() *StateMachine {
	sm := &StateMachine{
		states:          make(map[string]State),
		transitions:     make(map[string][]Transition),
		observers:       NewObserverManager(),
		machineState:    MachineStateStopped,
		stateHistory:    make(map[string]string),
		activeStates:    make(map[string]bool),
		parallelRegions: make(map[string][]string),
		joinConditions:  make(map[string][][]string),
		joinTracking:    make(map[string]map[string]bool),
	}

	sm.context = NewContext(context.Background(), sm)
	return sm
}

// safeEvaluateGuard safely evaluates a guard function with panic recovery
func safeEvaluateGuard(guard GuardFunc, ctx Context) (result bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = false
			err = fmt.Errorf("guard panic: %v", r)
		}
	}()

	result = guard(ctx)
	return result, nil
}

// safeExecuteAction safely executes an action function with panic recovery
func safeExecuteAction(action ActionFunc, ctx Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("action panic: %v", r)
		}
	}()

	err = action(ctx)
	return err
}

// Start starts the state machine
func (sm *StateMachine) Start() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.machineState == MachineStateStarted {
		return NewMachineError(ErrCodeInvalidState, "Start", "machine is already started")
	}

	if sm.initialState == "" {
		return NewConfigurationError("StateMachine", "no initial state defined")
	}

	// Validate that initial state exists
	if _, exists := sm.states[sm.initialState]; !exists {
		return NewConfigurationError("StateMachine", fmt.Sprintf("initial state '%s' does not exist", sm.initialState))
	}

	sm.machineState = MachineStateStarted

	actualInitialState := sm.executeCompositeStateEntry(sm.initialState, nil)
	sm.currentState = actualInitialState

	if smCtx, ok := sm.context.(*StateMachineContext); ok {
		smCtx.updateCurrentState(sm.currentState)
	}

	sm.executeEntryActions("", sm.currentState, nil)
	sm.observers.NotifyStateEnter(sm.currentState, sm.context)
	sm.observers.NotifyMachineStarted(sm.context)

	return nil
}

// Stop stops the state machine
func (sm *StateMachine) Stop() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.machineState != MachineStateStarted {
		return NewMachineNotStartedError("Stop")
	}

	if sm.currentState != "" {
		sm.observers.NotifyStateExit(sm.currentState, sm.context)
	}
	sm.observers.NotifyMachineStopped(sm.context)

	sm.machineState = MachineStateStopped
	return nil
}

// Reset resets the state machine
func (sm *StateMachine) Reset() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	previousState := sm.currentState
	sm.currentState = sm.initialState
	sm.machineState = MachineStateStopped

	if smCtx, ok := sm.context.(*StateMachineContext); ok {
		smCtx.updateCurrentState(sm.currentState)
	}

	if previousState != sm.currentState {
		if previousState != "" {
			sm.observers.NotifyStateExit(previousState, sm.context)
		}
		if sm.currentState != "" {
			sm.observers.NotifyStateEnter(sm.currentState, sm.context)
			sm.observers.NotifyTransition(previousState, sm.currentState, nil, sm.context)
		}
	}

	return nil
}

// CurrentState returns the current state
func (sm *StateMachine) CurrentState() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.currentState
}

// SetState sets the current state
func (sm *StateMachine) SetState(state string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.states[state]; !exists {
		return NewStateNotFoundError(state)
	}

	previousState := sm.currentState
	sm.currentState = state

	sm.updateStateHistory(previousState)

	if smCtx, ok := sm.context.(*StateMachineContext); ok {
		smCtx.updateCurrentState(sm.currentState)
	}

	if previousState != "" && previousState != state {
		sm.observers.NotifyStateExit(previousState, sm.context)
	}

	sm.observers.NotifyStateEnter(state, sm.context)

	if previousState != state {
		sm.observers.NotifyTransition(previousState, state, nil, sm.context)
	}

	return nil
}

// SendEvent sends an event asynchronously
func (sm *StateMachine) SendEvent(eventName string, eventData any) *EventResult {
	return sm.SendEventWithContext(context.Background(), eventName, eventData)
}

// SendEventWithContext sends an event asynchronously with context
func (sm *StateMachine) SendEventWithContext(ctx context.Context, eventName string, eventData any) *EventResult {
	return sm.HandleEventWithContext(ctx, eventName, eventData)
}

// HandleEvent handles an event synchronously
func (sm *StateMachine) HandleEvent(eventName string, eventData any) *EventResult {
	return sm.HandleEventWithContext(context.Background(), eventName, eventData)
}

// HandleEventWithContext handles an event synchronously with context
func (sm *StateMachine) HandleEventWithContext(ctx context.Context, eventName string, eventData any) *EventResult {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.machineState != MachineStateStarted {
		return NewEventResult(false, false, sm.currentState, sm.currentState).
			WithRejection("machine is not started")
	}

	event := NewEvent(eventName, eventData)

	// Validate event name
	if strings.TrimSpace(eventName) == "" {
		reason := "event name cannot be empty"
		sm.observers.NotifyEventRejected(event, reason, sm.context)
		return NewEventResult(false, false, sm.currentState, sm.currentState).
			WithRejection(reason).
			WithError(errors.New(reason))
	}

	if smCtx, ok := sm.context.(*StateMachineContext); ok {
		smCtx.updateCurrentEvent(event)
	}

	matchingTransition, sourceStateID, err := sm.findMatchingTransition(eventName, event)
	if err != nil {
		reason := fmt.Sprintf("no valid transition found for event '%s' in state '%s'", eventName, sm.currentState)
		sm.observers.NotifyEventRejected(event, reason, sm.context)
		return NewEventResult(false, false, sm.currentState, sm.currentState).
			WithRejection(reason).
			WithError(fmt.Errorf("%s", reason))
	}

	previousState := sm.currentState
	targetState := matchingTransition.TargetState
	isRegionTransition := sm.isRegionTransition(sourceStateID, targetState)

	if smCtx, ok := sm.context.(*StateMachineContext); ok {
		smCtx.updateTransitionInfo(sourceStateID, previousState, targetState, event)
	}

	if isRegionTransition {
		// Handle region state transition - update the region's current state but not the machine's current state
		// Store the target state info before updating to check for completion later
		var targetRegion Region
		var isFinalState bool
		if targetStateObj, exists := sm.states[targetState]; exists {
			isFinalState = targetStateObj.IsFinal()
			targetRegion = sm.findRegionForState(targetState)
		}

		// Execute transition action BEFORE state change - if it fails, abort transition
		if matchingTransition.Action != nil {
			// Record action execution regardless of outcome
			sm.observers.NotifyActionExecution("transition", sourceStateID, event, sm.context)
			if err := safeExecuteAction(matchingTransition.Action, sm.context); err != nil {
				reason := fmt.Sprintf("transition action failed: %v", err)
				sm.observers.NotifyEventRejected(event, reason, sm.context)
				return NewEventResult(false, false, sourceStateID, sourceStateID).
					WithError(err)
			}
		}

		sm.updateRegionStateWithoutCompletionCheck(sourceStateID, targetState)

		delete(sm.activeStates, sourceStateID)
		sm.activeStates[targetState] = true

		if sourceState, exists := sm.states[sourceStateID]; exists {
			sourceState.Exit(sm.context)
		}

		if targetStateObj, exists := sm.states[targetState]; exists {
			targetStateObj.Enter(sm.context)
		}

		sm.observers.NotifyStateExit(sourceStateID, sm.context)
		sm.observers.NotifyTransition(sourceStateID, targetState, event, sm.context)
		sm.observers.NotifyStateEnter(targetState, sm.context)

		// Check for parallel state completion AFTER action execution
		if isFinalState && targetRegion != nil {
			sm.checkParallelStateCompletion(targetRegion.ParentState())
		}

		return NewEventResult(true, true, sourceStateID, targetState)
	} else {
		// Execute transition action BEFORE state change - if it fails, abort transition
		if matchingTransition.Action != nil {
			// Record action execution regardless of outcome
			sm.observers.NotifyActionExecution("transition", previousState, event, sm.context)
			if err := safeExecuteAction(matchingTransition.Action, sm.context); err != nil {
				reason := fmt.Sprintf("transition action failed: %v", err)
				sm.observers.NotifyEventRejected(event, reason, sm.context)
				return NewEventResult(false, false, previousState, previousState).
					WithError(err)
			}
		}

		// Handle normal state transition - complex hierarchical state change with exit/entry actions and pseudostate processing
		sm.executeExitActions(previousState, targetState, event)

		if prevStateObj, ok := sm.states[previousState]; ok && prevStateObj.IsParallel() {
			if parallelState, ok := prevStateObj.(ParallelState); ok {
				for _, region := range parallelState.Regions() {
					if region.CurrentState() != nil {
						region.CurrentState().Exit(sm.context)
						sm.observers.NotifyStateExit(region.CurrentState().ID(), sm.context)
						delete(sm.activeStates, region.CurrentState().ID())
						if regionImpl, ok := region.(*RegionImpl); ok {
							regionImpl.currentState = nil
						}
					}
				}
				delete(sm.activeStates, previousState)
			}
		}

		sm.updateStateHistory(previousState)

		actualTargetState := sm.executeCompositeStateEntry(targetState, event)

		// Check if target is a fork pseudostate
		isForkState := false
		if state, exists := sm.states[actualTargetState]; exists {
			if pseudoState, ok := state.(PseudoState); ok {
				if pseudoState.Kind() == Fork {
					isForkState = true
				}
			}
		}

		if finalTargetState, err := sm.executePseudoStateWithSource(actualTargetState, event, sourceStateID); err == nil {
			actualTargetState = finalTargetState
			actualTargetState = sm.executeCompositeStateEntry(actualTargetState, event)
		}

		// For Fork states, don't change currentState since all targets are active
		if !isForkState {
			sm.currentState = actualTargetState
		}

		if sm.activeStates[sourceStateID] {
			delete(sm.activeStates, sourceStateID)
			if !isForkState {
				sm.activeStates[actualTargetState] = true
			}
		}

		if smCtx, ok := sm.context.(*StateMachineContext); ok {
			smCtx.updateCurrentState(sm.currentState)
		}

		sm.executeEntryActions(previousState, actualTargetState, event)

		if previousState != "" {
			sm.observers.NotifyStateExit(sourceStateID, sm.context)
			sm.observers.NotifyTransition(sourceStateID, actualTargetState, event, sm.context)
		}
		sm.observers.NotifyStateEnter(actualTargetState, sm.context)

		// For self-transitions, StateChanged should be true because exit/entry actions are executed
		stateChanged := previousState != actualTargetState || previousState == targetState
		return NewEventResult(true, stateChanged, sourceStateID, actualTargetState)
	}
}

// executeExitActions executes exit actions for states in hierarchical order
func (sm *StateMachine) executeExitActions(fromState, toState string, _ Event) {
	commonAncestor := sm.findCommonAncestor(fromState, toState)

	currentStateID := fromState
	for currentStateID != "" && currentStateID != commonAncestor {
		if state, exists := sm.states[currentStateID]; exists {
			state.Exit(sm.context)

			if state.Parent() != nil {
				currentStateID = state.Parent().ID()
			} else {
				break
			}
		} else {
			break
		}
	}
}

// executeEntryActions executes entry actions for states in hierarchical order
func (sm *StateMachine) executeEntryActions(fromState, toState string, _ Event) {
	commonAncestor := sm.findCommonAncestor(fromState, toState)
	entryPath := sm.buildEntryPath(commonAncestor, toState)

	for _, stateID := range entryPath {
		if state, exists := sm.states[stateID]; exists {
			state.Enter(sm.context)
		}
	}
}

// findCommonAncestor finds the common ancestor of two states
func (sm *StateMachine) findCommonAncestor(state1, state2 string) string {
	if state1 == state2 {
		return state1
	}

	hierarchy1 := sm.getStateHierarchy(state1)
	hierarchy2 := sm.getStateHierarchy(state2)

	var commonAncestor string
	minLen := len(hierarchy1)
	if len(hierarchy2) < minLen {
		minLen = len(hierarchy2)
	}

	for i := 0; i < minLen; i++ {
		if hierarchy1[i] == hierarchy2[i] {
			commonAncestor = hierarchy1[i]
		} else {
			break
		}
	}

	return commonAncestor
}

// getStateHierarchy returns the hierarchy path for a given state
func (sm *StateMachine) getStateHierarchy(stateID string) []string {
	if stateID == "" {
		return []string{}
	}

	state, exists := sm.states[stateID]
	if !exists {
		return []string{stateID}
	}

	hierarchy := []string{stateID}

	for parent := state.Parent(); parent != nil; parent = parent.Parent() {
		hierarchy = append([]string{parent.ID()}, hierarchy...)
	}

	return hierarchy
}

// buildEntryPath builds the path from ancestor to target state
func (sm *StateMachine) buildEntryPath(ancestor, target string) []string {
	if ancestor == target {
		return []string{}
	}

	targetHierarchy := sm.getStateHierarchy(target)

	if ancestor == "" {
		return targetHierarchy
	}

	ancestorHierarchy := sm.getStateHierarchy(ancestor)
	startIndex := len(ancestorHierarchy)

	if startIndex < len(targetHierarchy) {
		return targetHierarchy[startIndex:]
	}

	return []string{}
}

// executeCompositeStateEntry handles entering composite states with proper initial state activation
func (sm *StateMachine) executeCompositeStateEntry(stateID string, event Event) string {
	state, exists := sm.states[stateID]
	if !exists {
		return stateID
	}

	if compositeState, ok := state.(CompositeState); ok {
		if initialState := compositeState.InitialState(); initialState != nil {
			return sm.executeCompositeStateEntry(initialState.ID(), event)
		}
	}

	// Complex parallel state activation - activate all regions and their initial states
	if parallelState, ok := state.(ParallelState); ok {
		sm.activeStates[stateID] = true
		for _, region := range parallelState.Regions() {
			if initialState := region.InitialState(); initialState != nil {
				if regionImpl, ok := region.(*RegionImpl); ok {
					regionImpl.currentState = initialState
					finalState := sm.executeCompositeStateEntry(initialState.ID(), event)
					sm.activeStates[finalState] = true
					if regionState, exists := sm.states[finalState]; exists {
						regionState.Enter(sm.context)
					}
				}
			}
		}
	}

	return stateID
}

// findMatchingTransition finds a valid transition for the given event using a deterministic priority order
//
// EVENT ROUTING PRIORITY ORDER:
// =============================
// This method implements a hierarchical event routing mechanism with the following priority order:
//
// 1. ACTIVE REGIONAL STATES (Highest Priority)
//   - Transitions from states currently active within parallel regions
//   - Ensures regional transitions take precedence over parent parallel state transitions
//   - Example: If StateA in Region1 and StateB in Region2 both handle event E, StateA/StateB transitions are checked first
//
// 2. ACTIVE NON-REGIONAL STATES (Fork Execution)
//   - Transitions from states activated by Fork pseudostates
//   - Handles parallel execution where multiple states are simultaneously active
//   - Example: After Fork -> [StateX, StateY], both StateX and StateY can handle events independently
//
// 3. DIRECT PARALLEL STATE PARENTS
//   - Transitions defined at the parallel state level when event originates from a region
//   - Enables events to bubble up from regions to their parent parallel states
//   - Example: Region state handles event by transitioning the entire parallel state
//
// 4. PARENT PARALLEL STATE HIERARCHY (Event Bubbling)
//   - Walks up the parent chain to find parallel states that can handle the event
//   - Supports nested parallel states and complex hierarchies
//   - Example: Event from nested region can bubble up through multiple parallel state levels
//
// 5. CURRENT STATE AND PARENT CHAIN (Traditional HSM)
//   - Standard hierarchical state machine behavior
//   - Checks current state first, then walks up parent hierarchy
//   - Includes parallel state region states as part of the hierarchy traversal
//
// GUARD CONDITION EVALUATION:
// =========================
// - All transitions are evaluated with their guard conditions (if present)
// - Guard conditions receive the current machine context for decision making
// - Only transitions with passing guard conditions are returned
//
// DETERMINISTIC BEHAVIOR:
// ======================
// - The first matching transition found in the priority order is returned
// - This ensures predictable behavior regardless of machine complexity
// - No ambiguity in transition selection for parallel states
//
// PARAMETERS:
// - eventName: The name of the event to find a transition for
// - event: The complete event object containing data and metadata
//
// RETURNS:
// - *Transition: The matching transition (nil if not found)
// - string: The ID of the source state where the transition was found
// - error: Error if no matching transition is found
func (sm *StateMachine) findMatchingTransition(eventName string, event Event) (*Transition, string, error) {
	// Store the original source state for error reporting
	sourceStateID := sm.currentState

	// Debug logging for transition resolution
	// TODO: Consider making this configurable via context or machine configuration
	// fmt.Printf("[DEBUG] findMatchingTransition: searching for event '%s' from source '%s'\n", eventName, sourceStateID)
	// fmt.Printf("[DEBUG] Active states: %v\n", sm.activeStates)

	// ====================================================================
	// PRIORITY 1: ACTIVE REGIONAL STATES (Highest Priority)
	// ====================================================================
	// Check transitions from all active regional states within parallel states
	// This ensures regional transitions are found before parallel state transitions
	// Regional states have highest priority because they represent the most specific context
	for activeStateID := range sm.activeStates {
		if activeStateID == sm.currentState {
			continue // Skip the main current state - we'll handle it in the traditional hierarchy
		}

		// Check if this active state is in a parallel region
		if region := sm.findRegionForState(activeStateID); region != nil {
			// This is a regional state, check its transitions first
			transitions := sm.transitions[activeStateID]
			for _, transition := range transitions {
				if transition.EventName == eventName {
					guardPassed := true
					if transition.Guard != nil {
						result, err := safeEvaluateGuard(transition.Guard, sm.context)
						if err != nil {
							// Guard panicked - skip this transition
							continue
						}
						guardPassed = result
					}
					if guardPassed {
						// fmt.Printf("[DEBUG] PRIORITY 1: Found matching transition from regional state '%s' for event '%s' -> '%s'\n",
						// 	activeStateID, eventName, transition.TargetState)
						return &transition, activeStateID, nil
					}
				}
			}
		}
	}

	// ====================================================================
	// PRIORITY 2: ACTIVE NON-REGIONAL STATES (Fork Execution)
	// ====================================================================
	// Check transitions from all active states (for Fork parallel execution)
	// These are states that were activated by Fork pseudostates and are running in parallel
	for activeStateID := range sm.activeStates {
		transitions := sm.transitions[activeStateID]
		for _, transition := range transitions {
			if transition.EventName == eventName {
				guardPassed := true
				if transition.Guard != nil {
					result, err := safeEvaluateGuard(transition.Guard, sm.context)
					if err != nil {
						// Guard panicked - skip this transition
						continue
					}
					guardPassed = result
				}
				if guardPassed {
					// Check if target is a Join pseudostate - if so, verify this is a valid source
					if targetState, exists := sm.states[transition.TargetState]; exists {
						if pseudoState, ok := targetState.(*PseudoStateImpl); ok && pseudoState.Kind() == Join {
							// This transition leads to a Join - verify the source is valid for at least one combination
							if combinations, hasCombinations := sm.joinConditions[transition.TargetState]; hasCombinations {
								isValidSource := false
								for _, combination := range combinations {
									if slices.Contains(combination, activeStateID) {
										isValidSource = true
										break
									}
								}
								if !isValidSource {
									// This active state is not a valid source for any Join combination - skip this transition
									continue
								}
							}
						}
					}
					// fmt.Printf("[DEBUG] PRIORITY 2: Found matching transition from active state '%s' for event '%s' -> '%s'\n",
					// 	activeStateID, eventName, transition.TargetState)
					return &transition, activeStateID, nil
				}
			}
		}
	}

	// ====================================================================
	// PRIORITY 3: DIRECT PARALLEL STATE PARENTS
	// ====================================================================
	// Check transitions from parallel state parent states when event is from a region
	// This ensures events from regions can bubble up to parent parallel states
	// This handles the case where a region state wants to transition the entire parallel state
	for activeStateID := range sm.activeStates {
		if _, exists := sm.states[activeStateID]; exists {
			// Check if this active state is in a parallel region
			if region := sm.findRegionForState(activeStateID); region != nil {
				parallelStateID := region.ParentState().ID()
				// Check transitions defined at the parallel state level
				if parallelTransitions, hasParallelTransitions := sm.transitions[parallelStateID]; hasParallelTransitions {
					for _, transition := range parallelTransitions {
						if transition.EventName == eventName {
							guardPassed := true
							if transition.Guard != nil {
								result, err := safeEvaluateGuard(transition.Guard, sm.context)
								if err != nil {
									// Guard panicked - skip this transition
									continue
								}
								guardPassed = result
							}
							if guardPassed {
								// fmt.Printf("[DEBUG] PRIORITY 3: Found matching transition from parallel state '%s' for region event '%s' -> '%s'\n",
								// 	parallelStateID, eventName, transition.TargetState)
								return &transition, parallelStateID, nil
							}
						}
					}
				}
			}
		}
	}

	// ====================================================================
	// PRIORITY 4: PARENT PARALLEL STATE HIERARCHY (Event Bubbling)
	// ====================================================================
	// Enhanced event bubbling: check parent parallel states for all active region states
	// This handles complex parallel hierarchies where events need to bubble up through multiple levels
	// For example: Region1 -> ParallelStateA -> ParallelStateB -> Root
	// Events from Region1 can bubble up to ParallelStateA, then ParallelStateB
	for activeStateID := range sm.activeStates {
		if state, exists := sm.states[activeStateID]; exists {
			// Walk up the parent chain to find parallel states
			currentParent := state.Parent()
			for currentParent != nil {
				if currentParent.IsParallel() {
					// Check transitions at this parallel state level
					if parallelTransitions, hasParallelTransitions := sm.transitions[currentParent.ID()]; hasParallelTransitions {
						for _, transition := range parallelTransitions {
							if transition.EventName == eventName {
								guardPassed := true
								if transition.Guard != nil {
									result, err := safeEvaluateGuard(transition.Guard, sm.context)
									if err != nil {
										// Guard panicked - skip this transition
										continue
									}
									guardPassed = result
								}
								if guardPassed {
									// fmt.Printf("[DEBUG] PRIORITY 4: Found matching transition from parent parallel state '%s' for bubbled event '%s' -> '%s'\n",
									// 	currentParent.ID(), eventName, transition.TargetState)
									return &transition, currentParent.ID(), nil
								}
							}
						}
					}
				}
				currentParent = currentParent.Parent()
			}
		}
	}

	// ====================================================================
	// PRIORITY 5: CURRENT STATE AND PARENT CHAIN (Traditional HSM)
	// ====================================================================
	// Traditional hierarchical state machine behavior
	// This is the fallback when no parallel state transitions are found
	currentStateID := sm.currentState
	for currentStateID != "" {
		if state, exists := sm.states[currentStateID]; exists {
			// Skip join pseudostates in normal event processing - they use synchronization mechanism
			// Join pseudostates are handled separately through the join tracking mechanism
			if pseudoState, ok := state.(*PseudoStateImpl); ok && pseudoState.Kind() == Join {
				// Don't process join pseudostates in normal event routing
			} else {
				// Check transitions from the current state in the hierarchy
				transitions := sm.transitions[currentStateID]
				for _, transition := range transitions {
					if transition.EventName == eventName {
						guardPassed := true
						if transition.Guard != nil {
							result, err := safeEvaluateGuard(transition.Guard, sm.context)
							if err != nil {
								// Guard panicked - skip this transition
								continue
							}
							guardPassed = result
						}
						if guardPassed {
							// fmt.Printf("[DEBUG] PRIORITY 5: Found matching transition from hierarchical state '%s' for event '%s' -> '%s'\n",
							// 	currentStateID, eventName, transition.TargetState)
							return &transition, currentStateID, nil
						}
					}
				}
			}
		} else {
			// Handle transitions from states that might not be in the states map
			// This can happen with pseudostates or other special states
			transitions := sm.transitions[currentStateID]
			for _, transition := range transitions {
				if transition.EventName == eventName {
					guardPassed := true
					if transition.Guard != nil {
						result, err := safeEvaluateGuard(transition.Guard, sm.context)
						if err != nil {
							// Guard panicked - skip this transition
							continue
						}
						guardPassed = result
					}
					if guardPassed {
						// fmt.Printf("[DEBUG] PRIORITY 5: Found matching transition from non-cataloged state '%s' for event '%s' -> '%s'\n",
						// 	currentStateID, eventName, transition.TargetState)
						return &transition, currentStateID, nil
					}
				}
			}
		}

		// Check transitions from parallel state region states as part of hierarchy traversal
		// This ensures that region states are also considered in the traditional hierarchy
		if state, exists := sm.states[currentStateID]; exists && state.IsParallel() {
			if parallelState, ok := state.(ParallelState); ok {
				for _, region := range parallelState.Regions() {
					if region.CurrentState() != nil {
						regionStateID := region.CurrentState().ID()
						regionTransitions := sm.transitions[regionStateID]
						for _, transition := range regionTransitions {
							if transition.EventName == eventName {
								guardPassed := true
								if transition.Guard != nil {
									result, err := safeEvaluateGuard(transition.Guard, sm.context)
									if err != nil {
										// Guard panicked - skip this transition
										continue
									}
									guardPassed = result
								}
								if guardPassed {
									// fmt.Printf("[DEBUG] PRIORITY 5: Found matching transition from parallel region state '%s' for event '%s' -> '%s'\n",
									// 	regionStateID, eventName, transition.TargetState)
									return &transition, regionStateID, nil
								}
							}
						}
					}
				}
			}
		}

		// Move up to the parent state in the hierarchy
		if state, exists := sm.states[currentStateID]; exists {
			if state.Parent() != nil {
				currentStateID = state.Parent().ID()
			} else {
				break // Reached the top of the hierarchy
			}
		} else {
			break // State not found, exit the loop
		}
	}

	// ====================================================================
	// NO MATCHING TRANSITION FOUND
	// ====================================================================
	// If we reach here, no matching transition was found in any of the priority levels
	return nil, "", NewNoTransitionError(sourceStateID, event.GetName())
}

// executePseudoState handles the execution logic for pseudostates
func (sm *StateMachine) executePseudoState(stateID string, event Event) (string, error) {
	return sm.executePseudoStateWithSource(stateID, event, "")
}

// executePseudoStateWithSource handles pseudostate execution with explicit source state
func (sm *StateMachine) executePseudoStateWithSource(stateID string, event Event, sourceStateID string) (string, error) {
	state, exists := sm.states[stateID]
	if !exists {
		return "", NewStateNotFoundError(stateID)
	}

	pseudoState, ok := state.(PseudoState)
	if !ok {
		return stateID, nil // Not a pseudostate, return as-is
	}

	pseudoImpl, ok := pseudoState.(*PseudoStateImpl)
	if !ok {
		return stateID, nil // Cannot process this pseudostate type
	}

	switch pseudoState.Kind() {
	case Choice:
		return sm.executeChoicePseudoState(pseudoImpl, event)
	case Junction:
		return sm.executeJunctionPseudoState(pseudoImpl, event)
	case Fork:
		return sm.executeForkPseudoState(pseudoImpl, event)
	case Join:
		return sm.executeJoinPseudoStateWithSource(pseudoImpl, event, sourceStateID)
	case History:
		return sm.executeHistoryPseudoState(pseudoImpl, event, false)
	case DeepHistory:
		return sm.executeHistoryPseudoState(pseudoImpl, event, true)
	default:
		return stateID, nil // Unknown pseudostate, treat as normal state
	}
}

// executeChoicePseudoState processes a choice pseudostate by evaluating conditions
func (sm *StateMachine) executeChoicePseudoState(pseudoState *PseudoStateImpl, event Event) (string, error) {
	if len(pseudoState.choiceConditions) > 0 {
		for _, condition := range pseudoState.choiceConditions {
			guardPassed := true
			if condition.Guard != nil {
				result, err := safeEvaluateGuard(condition.Guard, sm.context)
				if err != nil {
					// Guard panicked - skip this condition
					continue
				}
				guardPassed = result
			}
			if guardPassed {
				if condition.Action != nil {
					_ = safeExecuteAction(condition.Action, sm.context)
				}
				return sm.resolvePseudoStateTarget(condition.Target, event)
			}
		}
	} else {
		transitions, exists := sm.transitions[pseudoState.ID()]
		if exists && len(transitions) > 0 {
			for _, transition := range transitions {
				guardPassed := true
				if transition.Guard != nil {
					result, err := safeEvaluateGuard(transition.Guard, sm.context)
					if err != nil {
						// Guard panicked - skip this transition
						continue
					}
					guardPassed = result
				}
				if guardPassed {
					if transition.Action != nil {
						_ = safeExecuteAction(transition.Action, sm.context)
					}
					return sm.resolvePseudoStateTarget(transition.TargetState, event)
				}
			}
		}
	}

	if pseudoState.defaultTarget != "" {
		return sm.resolvePseudoStateTarget(pseudoState.defaultTarget, event)
	}

	return "", NewTransitionError(ErrCodeTransitionNotAllowed, pseudoState.ID(), "", "", fmt.Sprintf("no valid transition from choice state '%s'", pseudoState.ID()))
}

// executeJunctionPseudoState processes a junction pseudostate by evaluating outgoing transitions
func (sm *StateMachine) executeJunctionPseudoState(pseudoState *PseudoStateImpl, event Event) (string, error) {
	if pseudoState.defaultTarget != "" {
		return sm.resolvePseudoStateTarget(pseudoState.defaultTarget, event)
	}

	transitions, exists := sm.transitions[pseudoState.ID()]
	if exists && len(transitions) > 0 {
		for _, transition := range transitions {
			guardPassed := true
			if transition.Guard != nil {
				result, err := safeEvaluateGuard(transition.Guard, sm.context)
				if err != nil {
					// Guard panicked - skip this transition
					continue
				}
				guardPassed = result
			}
			if guardPassed {
				return sm.resolvePseudoStateTarget(transition.TargetState, event)
			}
		}
	}

	return "", &TransitionError{Code: ErrCodeTransitionNotAllowed, From: pseudoState.ID(), Event: "", Reason: fmt.Sprintf("no valid transition from junction state '%s'", pseudoState.ID())}
}

// executeForkPseudoState processes a fork pseudostate (activates parallel regions)
func (sm *StateMachine) executeForkPseudoState(pseudoState *PseudoStateImpl, event Event) (string, error) {
	// Complex fork execution - activates multiple parallel states simultaneously
	if len(pseudoState.forkTargets) > 0 {
		for i, target := range pseudoState.forkTargets {
			resolvedTarget, err := sm.resolvePseudoStateTarget(target, event)
			if err != nil {
				return "", fmt.Errorf("failed to resolve fork target %s: %w", target, err)
			}

			sm.activeStates[resolvedTarget] = true

			if targetState := sm.states[resolvedTarget]; targetState != nil {
				targetState.Enter(sm.context)
				sm.observers.NotifyStateEnter(resolvedTarget, sm.context)
			}

			if i == 0 {
				regionKey := fmt.Sprintf("fork_%s", pseudoState.ID())
				sm.parallelRegions[regionKey] = pseudoState.forkTargets
			}
		}

		firstTarget, err := sm.resolvePseudoStateTarget(pseudoState.forkTargets[0], event)
		if err != nil {
			return "", err
		}
		return firstTarget, nil
	}

	transitions, exists := sm.transitions[pseudoState.ID()]
	if exists && len(transitions) > 0 {
		var primaryTarget string
		targetStates := make([]string, 0, len(transitions))

		for i, transition := range transitions {
			resolvedTarget, err := sm.resolvePseudoStateTarget(transition.TargetState, event)
			if err != nil {
				return "", fmt.Errorf("failed to resolve transition target %s: %w", transition.TargetState, err)
			}

			targetStates = append(targetStates, resolvedTarget)
			sm.activeStates[resolvedTarget] = true

			if targetState := sm.states[resolvedTarget]; targetState != nil {
				targetState.Enter(sm.context)
				sm.observers.NotifyStateEnter(resolvedTarget, sm.context)
			}

			if i == 0 {
				primaryTarget = resolvedTarget
			}
		}

		regionKey := fmt.Sprintf("fork_%s", pseudoState.ID())
		sm.parallelRegions[regionKey] = targetStates

		return primaryTarget, nil
	}

	return "", NewConfigurationError("ForkState", fmt.Sprintf("no outgoing transitions from fork state '%s'", pseudoState.ID()))
}

// executeJoinPseudoStateWithSource processes a join pseudostate with explicit source state
func (sm *StateMachine) executeJoinPseudoStateWithSource(pseudoState *PseudoStateImpl, event Event, explicitSourceState string) (string, error) {
	// Complex join synchronization - wait for all required source states before proceeding
	joinStateID := pseudoState.ID()

	combinations, hasCombinations := sm.joinConditions[joinStateID]

	if hasCombinations {
		if sm.joinTracking == nil {
			sm.joinTracking = make(map[string]map[string]bool)
		}
		if sm.joinTracking[joinStateID] == nil {
			sm.joinTracking[joinStateID] = make(map[string]bool)
		}

		var fromState string
		if explicitSourceState != "" {
			fromState = explicitSourceState
		} else if smCtx, ok := sm.context.(*StateMachineContext); ok {
			fromState = smCtx.GetSourceState()
		}

		if fromState != "" {
			sm.joinTracking[joinStateID][fromState] = true
			// fmt.Printf("[DEBUG] Join '%s': marked '%s' as arrived. Tracking: %v\n", joinStateID, fromState, sm.joinTracking[joinStateID])
		}

		// Check if any combination is satisfied
		allSourcesReady := false
		var readyCombination []string
		for _, combination := range combinations {
			combinationReady := true
			for _, sourceState := range combination {
				if !sm.joinTracking[joinStateID][sourceState] {
					combinationReady = false
					break
				}
			}
			if combinationReady {
				allSourcesReady = true
				readyCombination = combination
				// fmt.Printf("[DEBUG] Join '%s': combination %v is ready!\n", joinStateID, combination)
				break
			}
		}
		// fmt.Printf("[DEBUG] Join '%s': allSourcesReady = %v\n", joinStateID, allSourcesReady)

		if !allSourcesReady {
			// Join is not ready yet - stay in source state
			// Return the source state so the machine doesn't change currentState
			return fromState, nil
		}

		delete(sm.joinTracking, joinStateID)

		// Remove the states from the ready combination
		for _, sourceState := range readyCombination {
			delete(sm.activeStates, sourceState)
		}

		// Clean up parallel region containing the source states
		for regionKey, regionStates := range sm.parallelRegions {
			containsAllSources := true
			for _, source := range readyCombination {
				found := false
				for _, regionState := range regionStates {
					if regionState == source {
						found = true
						break
					}
				}
				if !found {
					containsAllSources = false
					break
				}
			}

			if containsAllSources {
				delete(sm.parallelRegions, regionKey)
				break
			}
		}
	}

	if pseudoState.joinTarget != "" {
		return sm.resolvePseudoStateTarget(pseudoState.joinTarget, event)
	}

	transitions, exists := sm.transitions[joinStateID]
	if exists && len(transitions) > 0 {
		transition := transitions[0]
		return sm.resolvePseudoStateTarget(transition.TargetState, event)
	}

	return "", NewConfigurationError("JoinState", fmt.Sprintf("no outgoing transitions from join state '%s'", joinStateID))
}

// executeHistoryPseudoState processes a history pseudostate
func (sm *StateMachine) executeHistoryPseudoState(pseudoState *PseudoStateImpl, event Event, deep bool) (string, error) {
	parentID := sm.getHistoryParentID(pseudoState)

	if parentID == "" {
		transitions, exists := sm.transitions[pseudoState.ID()]
		if exists && len(transitions) > 0 {
			transition := transitions[0]
			targetState := transition.TargetState

			if historicalState, exists := sm.stateHistory[targetState]; exists && historicalState != "" {
				if deep {
					return historicalState, nil
				} else {
					return sm.getImmediateSubstate(targetState, historicalState)
				}
			}

			return sm.resolvePseudoStateTarget(targetState, event)
		}
		return sm.useHistoryDefault(pseudoState, event)
	}

	if historicalState, exists := sm.stateHistory[parentID]; exists && historicalState != "" {
		if deep {
			return historicalState, nil
		} else {
			return sm.getImmediateSubstate(parentID, historicalState)
		}
	}

	transitions, exists := sm.transitions[pseudoState.ID()]
	if exists && len(transitions) > 0 {
		transition := transitions[0]
		return sm.resolvePseudoStateTarget(transition.TargetState, event)
	}

	return sm.useHistoryDefault(pseudoState, event)
}

// getHistoryParentID returns the parent state ID of a history pseudostate
func (sm *StateMachine) getHistoryParentID(pseudoState *PseudoStateImpl) string {
	if pseudoState.Parent() != nil {
		return pseudoState.Parent().ID()
	}
	return ""
}

// useHistoryDefault handles the fallback to default history state
func (sm *StateMachine) useHistoryDefault(pseudoState *PseudoStateImpl, event Event) (string, error) {
	if pseudoState.historyDefault != "" {
		return sm.resolvePseudoStateTarget(pseudoState.historyDefault, event)
	}
	return "", &StateError{Code: ErrCodeInvalidState, StateID: pseudoState.ID(), Message: fmt.Sprintf("no history found and no default defined for history state '%s'", pseudoState.ID())}
}

// getImmediateSubstate extracts the immediate substate for shallow history
func (sm *StateMachine) getImmediateSubstate(parentID string, lastState string) (string, error) {
	parts := sm.splitStatePath(lastState)
	if len(parts) > 1 {
		// Find the immediate child of the parentID
		for i, part := range parts {
			if part == parentID && i+1 < len(parts) {
				return parts[i+1], nil
			}
		}
	}
	// If we can't extract a substate, just return the full path
	return lastState, nil
}

// resolvePseudoStateTarget resolves a target state ID, handling nested pseudostates
func (sm *StateMachine) resolvePseudoStateTarget(targetStateID string, event Event) (string, error) {
	// Check if target is also a pseudostate
	if targetState, exists := sm.states[targetStateID]; exists {
		if targetState.IsPseudo() {
			// Recursively process pseudostates
			return sm.executePseudoState(targetStateID, event)
		}
	}

	return targetStateID, nil
}

// SetRegionState sets the state of a specific region in a parallel state
func (sm *StateMachine) SetRegionState(regionID string, stateID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Find the region
	for _, state := range sm.states {
		if parallelState, ok := state.(ParallelState); ok {
			for _, region := range parallelState.Regions() {
				if region.ID() == regionID {
					// Find the target state in the region
					for _, regionState := range region.States() {
						if regionState.ID() == stateID {
							// Set the region's current state
							if regionImpl, ok := region.(*RegionImpl); ok {
								regionImpl.currentState = regionState
								return nil
							}
						}
					}
					return &StateError{Code: ErrCodeStateNotFound, StateID: stateID, Message: fmt.Sprintf("state '%s' not found in region '%s'", stateID, regionID)}
				}
			}
		}
	}

	return NewStateNotFoundError(regionID)
}

// RegionState returns the current state of a specific region
func (sm *StateMachine) RegionState(regionID string) string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Find the region
	for _, state := range sm.states {
		if parallelState, ok := state.(ParallelState); ok {
			for _, region := range parallelState.Regions() {
				if region.ID() == regionID {
					if region.CurrentState() != nil {
						return region.CurrentState().ID()
					}
					return ""
				}
			}
		}
	}

	return ""
}

// GetStateHierarchy returns the full hierarchical path of the current state
func (sm *StateMachine) GetStateHierarchy() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.currentState == "" {
		return []string{}
	}

	state, exists := sm.states[sm.currentState]
	if !exists {
		return []string{sm.currentState}
	}

	hierarchy := []string{sm.currentState}

	// Walk up the parent chain
	for parent := state.Parent(); parent != nil; parent = parent.Parent() {
		hierarchy = append([]string{parent.ID()}, hierarchy...)
	}

	return hierarchy
}

// IsInState checks if the machine is currently in the specified state or any of its substates
func (sm *StateMachine) IsInState(stateID string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Direct match
	if sm.currentState == stateID {
		return true
	}

	// Check if current state is a child of the specified state
	currentState, exists := sm.states[sm.currentState]
	if !exists {
		return false
	}

	// Walk up the parent chain
	for parent := currentState.Parent(); parent != nil; parent = parent.Parent() {
		if parent.ID() == stateID {
			return true
		}
	}

	return false
}

// GetActiveStates returns all currently active states (including parallel regions)
func (sm *StateMachine) GetActiveStates() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	activeStates := []string{}

	// Add current state if set
	if sm.currentState != "" {
		activeStates = append(activeStates, sm.currentState)
	}

	// Add all parallel active states
	for stateID := range sm.activeStates {
		// Avoid duplicates
		found := false
		for _, existing := range activeStates {
			if existing == stateID {
				found = true
				break
			}
		}
		if !found {
			activeStates = append(activeStates, stateID)
		}
	}

	// Legacy support: Check parallel state regions
	if sm.currentState != "" {
		// Get the full hierarchy path to current state
		hierarchy := sm.getStateHierarchy(sm.currentState)

		// For each state in the hierarchy, check if it's a parallel state
		for _, stateID := range hierarchy {
			if state, exists := sm.states[stateID]; exists {
				if parallelState, ok := state.(ParallelState); ok {
					// Add all active region states
					for _, region := range parallelState.Regions() {
						if region.CurrentState() != nil {
							regionState := region.CurrentState().ID()
							// Avoid duplicates
							alreadyAdded := slices.Contains(activeStates, regionState)
							if !alreadyAdded {
								activeStates = append(activeStates, regionState)
							}
						}
					}
				}
			}
		}
	}

	return activeStates
}

// IsStateActive checks if a specific state is currently active
func (sm *StateMachine) IsStateActive(stateID string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	// Check if it's the current state
	if sm.currentState == stateID {
		return true
	}

	// Check if it's in the active states map (parallel execution)
	return sm.activeStates[stateID]
}

// GetParallelRegions returns the current parallel regions and their active states
func (sm *StateMachine) GetParallelRegions() map[string][]string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	regions := make(map[string][]string)
	for key, states := range sm.parallelRegions {
		regionStates := make([]string, len(states))
		copy(regionStates, states)
		regions[key] = regionStates
	}

	return regions
}

// AddObserver adds an observer to the machine
func (sm *StateMachine) AddObserver(observer Observer) {
	sm.observers.AddObserver(observer)
}

// RemoveObserver removes an observer from the machine
func (sm *StateMachine) RemoveObserver(observer Observer) {
	sm.observers.RemoveObserver(observer)
}

// Context returns the machine's context
func (sm *StateMachine) Context() Context {
	return sm.context
}

// WithContext sets the machine's context
func (sm *StateMachine) WithContext(ctx Context) Machine {
	sm.context = ctx
	return sm
}

// MarshalJSON serializes the machine state to JSON
func (sm *StateMachine) MarshalJSON() ([]byte, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	data := map[string]any{
		"currentState": sm.currentState,
		"initialState": sm.initialState,
		"machineState": sm.machineState,
		"contextData":  sm.context.GetAll(),
	}

	return json.Marshal(data)
}

// UnmarshalJSON deserializes the machine state from JSON
func (sm *StateMachine) UnmarshalJSON(data []byte) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	if currentState, ok := state["currentState"].(string); ok {
		sm.currentState = currentState
	}

	if initialState, ok := state["initialState"].(string); ok {
		sm.initialState = initialState
	}

	if machineStateFloat, ok := state["machineState"].(float64); ok {
		sm.machineState = MachineState(int(machineStateFloat))
	}

	if contextData, ok := state["contextData"].(map[string]any); ok {
		for k, v := range contextData {
			sm.context.Set(k, v)
		}
	}

	return nil
}

// splitStatePath splits a hierarchical state path (like "parent.child.grandchild") into segments
func (sm *StateMachine) splitStatePath(statePath string) []string {
	return strings.Split(statePath, ".")
}

// updateStateHistory records the previous state in the history for its composite parent states
func (sm *StateMachine) updateStateHistory(stateID string) {
	if stateID == "" {
		return
	}

	state, exists := sm.states[stateID]
	if !exists {
		return
	}

	currentState := state
	for parent := currentState.Parent(); parent != nil; parent = parent.Parent() {
		if parent.IsComposite() {
			sm.stateHistory[parent.ID()] = stateID
		}
		currentState = parent
	}
}

// isRegionTransition checks if the transition is between states in the same parallel state regions
func (sm *StateMachine) isRegionTransition(sourceStateID, targetStateID string) bool {
	sourceRegion := sm.findRegionForState(sourceStateID)
	targetRegion := sm.findRegionForState(targetStateID)

	if sourceRegion != nil && targetRegion != nil {
		sourceParentID := sourceRegion.ParentState().ID()
		targetParentID := targetRegion.ParentState().ID()
		return sourceParentID == targetParentID
	}

	// Complex region transition logic - check if states are in same parallel context
	if sourceRegion != nil && targetRegion == nil {
		if targetState, exists := sm.states[targetStateID]; exists {
			if targetState.Parent() != nil && targetState.Parent().ID() == sourceRegion.ParentState().ID() {
				return true
			}
		}
		return false
	}

	if sourceRegion == nil && targetRegion != nil {
		if sourceState, exists := sm.states[sourceStateID]; exists {
			if sourceState.Parent() != nil && sourceState.Parent().ID() == targetRegion.ParentState().ID() {
				return true
			}
		}
		return false
	}

	return false
}

// updateRegionStateWithoutCompletionCheck updates the current state of the region without checking for completion
func (sm *StateMachine) updateRegionStateWithoutCompletionCheck(sourceStateID, targetStateID string) {
	targetRegion := sm.findRegionForState(targetStateID)

	if targetRegion == nil {
		sourceRegion := sm.findRegionForState(sourceStateID)
		if sourceRegion != nil {
			targetRegion = sourceRegion
		}
	}

	if targetRegion != nil {
		if targetState, exists := sm.states[targetStateID]; exists {
			if regionImpl, ok := targetRegion.(*RegionImpl); ok {
				regionImpl.currentState = targetState
			}
		}
	}
}

// findRegionForState finds the region that contains the given state
func (sm *StateMachine) findRegionForState(stateID string) Region {
	for _, state := range sm.states {
		if parallelState, ok := state.(ParallelState); ok {
			for _, region := range parallelState.Regions() {
				for _, regionState := range region.States() {
					if regionState.ID() == stateID {
						return region
					}
				}
			}
		}
	}
	return nil
}

// checkParallelStateCompletion checks if all regions of a parallel state have reached final states
// and triggers automatic completion transition if available
func (sm *StateMachine) checkParallelStateCompletion(parallelState ParallelState) {
	if parallelState == nil {
		return
	}

	// Check if all regions have reached final states
	allRegionsComplete := true
	for _, region := range parallelState.Regions() {
		if !sm.isRegionComplete(region) {
			allRegionsComplete = false
			break
		}
	}

	if !allRegionsComplete {
		return
	}

	// All regions are complete, look for completion transition
	completionEventName := "__completion_" + parallelState.ID()

	// First try to find a transition with the specific completion event name
	transitions := sm.transitions[parallelState.ID()]
	for _, transition := range transitions {
		if transition.EventName == completionEventName {
			guardPassed := true
			if transition.Guard != nil {
				result, err := safeEvaluateGuard(transition.Guard, sm.context)
				if err != nil {
					// Guard panicked - skip this transition
					continue
				}
				guardPassed = result
			}
			if guardPassed {
				sm.executeCompletionTransition(parallelState.ID(), transition)
				return
			}
		}
	}

	// If no specific completion event found, try empty event name (OnCompletion pattern)
	for _, transition := range transitions {
		if transition.EventName == "" {
			guardPassed := true
			if transition.Guard != nil {
				result, err := safeEvaluateGuard(transition.Guard, sm.context)
				if err != nil {
					// Guard panicked - skip this transition
					continue
				}
				guardPassed = result
			}
			if guardPassed {
				sm.executeCompletionTransition(parallelState.ID(), transition)
				return
			}
		}
	}
}

// isRegionComplete checks if a region has reached its final state
func (sm *StateMachine) isRegionComplete(region Region) bool {
	if region == nil {
		return false
	}

	// Check if region has a current state
	if region.CurrentState() == nil {
		return false
	}

	// Check if the current state is a final state
	currentStateID := region.CurrentState().ID()
	if currentState, exists := sm.states[currentStateID]; exists {
		return currentState.IsFinal()
	}

	return false
}

// executeCompletionTransition executes an automatic completion transition
func (sm *StateMachine) executeCompletionTransition(sourceStateID string, transition Transition) {
	event := NewEvent("__completion", nil)

	// Exit all regions of the parallel state with proper cleanup
	if sourceState, exists := sm.states[sourceStateID]; exists {
		if parallelState, ok := sourceState.(ParallelState); ok {
			// Exit all region states first
			for _, region := range parallelState.Regions() {
				if region.CurrentState() != nil {
					regionStateID := region.CurrentState().ID()
					region.CurrentState().Exit(sm.context)
					sm.observers.NotifyStateExit(regionStateID, sm.context)
					delete(sm.activeStates, regionStateID)

					// Clear region current state
					if regionImpl, ok := region.(*RegionImpl); ok {
						regionImpl.currentState = nil
					}
				}
			}
		}

		// Exit the parallel state itself
		sourceState.Exit(sm.context)
		sm.observers.NotifyStateExit(sourceStateID, sm.context)
		delete(sm.activeStates, sourceStateID)

		// Clean up parallel regions tracking
		sm.cleanupParallelRegions(sourceStateID)
	}

	// Update context for completion transition
	if smCtx, ok := sm.context.(*StateMachineContext); ok {
		smCtx.updateTransitionInfo(sourceStateID, sm.currentState, transition.TargetState, event)
	}

	// Execute transition action if present
	if transition.Action != nil {
		_ = safeExecuteAction(transition.Action, sm.context)
		sm.observers.NotifyActionExecution("completion_transition", sourceStateID, event, sm.context)
	}

	// Process target state (handle composite states and pseudostates)
	targetState := transition.TargetState
	actualTargetState := sm.executeCompositeStateEntry(targetState, event)

	// Handle pseudostates in the target
	if finalTargetState, err := sm.executePseudoState(actualTargetState, event); err == nil {
		actualTargetState = finalTargetState
		actualTargetState = sm.executeCompositeStateEntry(actualTargetState, event)
	}

	// Update machine state
	previousState := sm.currentState
	sm.currentState = actualTargetState
	sm.activeStates[actualTargetState] = true

	// Update context
	if smCtx, ok := sm.context.(*StateMachineContext); ok {
		smCtx.updateCurrentState(sm.currentState)
	}

	// Notify observers
	sm.observers.NotifyTransition(sourceStateID, actualTargetState, event, sm.context)
	sm.observers.NotifyStateEnter(actualTargetState, sm.context)

	// Execute entry actions for the target state hierarchy
	sm.executeEntryActions(previousState, actualTargetState, event)
}

// cleanupParallelRegions removes tracking for completed parallel state
func (sm *StateMachine) cleanupParallelRegions(parallelStateID string) {
	// Remove from parallel regions tracking
	for regionKey, regionStates := range sm.parallelRegions {
		// Check if this region belongs to the completed parallel state
		for _, stateID := range regionStates {
			if state, exists := sm.states[stateID]; exists {
				// Walk up the parent chain to see if this state belongs to the parallel state
				currentParent := state.Parent()
				for currentParent != nil {
					if currentParent.ID() == parallelStateID {
						delete(sm.parallelRegions, regionKey)
						break
					}
					currentParent = currentParent.Parent()
				}
			}
		}
	}
}
