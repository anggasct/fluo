package fluo

import (
	"fmt"
	"strings"
)

// ChoiceCondition represents a condition and target for a choice pseudostate
type ChoiceCondition struct {
	// Guard condition for this choice branch
	Guard GuardFunc
	// Target state for this choice branch
	Target string
	// Action to execute for this specific branch
	Action ActionFunc
}

// MachineBuilder provides the main entry point for building state machines
type MachineBuilder interface {
	State(id string) StateBuilder
	CompositeState(id string) CompositeStateBuilder
	ParallelState(id string) ParallelStateBuilder

	Choice(id string) ChoiceBuilder
	Junction(id string) JunctionBuilder
	Fork(id string) ForkBuilder
	Join(id string) JoinBuilder
	History(id string) HistoryBuilder
	DeepHistory(id string) HistoryBuilder

	Build() MachineDefinition
}

// StateBuilder handles regular atomic state configuration
type StateBuilder interface {
	To(target string) TransitionBuilder
	ToSelf() TransitionBuilder
	ToParent(target string) TransitionBuilder

	OnEntry(action ActionFunc) StateBuilder
	OnExit(action ActionFunc) StateBuilder
	Final() StateBuilder
	Initial() StateBuilder

	State(id string) StateBuilder
	CompositeState(id string) CompositeStateBuilder
	ParallelState(id string) ParallelStateBuilder
	Choice(id string) ChoiceBuilder
	Junction(id string) JunctionBuilder
	Fork(id string) ForkBuilder
	Join(id string) JoinBuilder
	History(id string) HistoryBuilder
	DeepHistory(id string) HistoryBuilder
	Build() MachineDefinition
}

// TransitionBuilder handles transition configuration with inline actions
type TransitionBuilder interface {
	// Event binding
	On(event string) TransitionBuilder
	OnCompletion() TransitionBuilder // Completion transition (automatic when state completes)

	// Conditions
	When(guard GuardFunc) TransitionBuilder
	Unless(guard GuardFunc) TransitionBuilder

	// Actions
	Do(action ActionFunc) TransitionBuilder
	DoIf(condition GuardFunc, action ActionFunc) TransitionBuilder
	DoAsync(action ActionFunc) TransitionBuilder

	// Error handling
	OnError(errorState string) TransitionBuilder
	OnTimeout(timeoutState string) TransitionBuilder

	// Multiple transitions from same state
	To(target string) TransitionBuilder
	ToSelf() TransitionBuilder
	ToParent(target string) TransitionBuilder

	// Navigation back
	State(id string) StateBuilder
	CompositeState(id string) CompositeStateBuilder
	Build() MachineDefinition
}

// CompositeStateBuilder handles hierarchical states
type CompositeStateBuilder interface {
	// Child states
	State(id string) StateBuilder
	CompositeState(id string) CompositeStateBuilder

	// Child pseudostates
	Choice(id string) ChoiceBuilder
	Junction(id string) JunctionBuilder
	Fork(id string) ForkBuilder
	Join(id string) JoinBuilder
	History(id string) HistoryBuilder
	DeepHistory(id string) HistoryBuilder

	// State actions
	OnEntry(action ActionFunc) CompositeStateBuilder
	OnExit(action ActionFunc) CompositeStateBuilder

	// Transitions from this composite state
	To(target string) TransitionBuilder
	ToParent(target string) TransitionBuilder

	// Navigation back to parent
	End() MachineBuilder
	Build() MachineDefinition
}

// ParallelStateBuilder handles parallel regions
type ParallelStateBuilder interface {
	// Regions
	Region(id string) RegionBuilder

	// State actions
	OnEntry(action ActionFunc) ParallelStateBuilder
	OnExit(action ActionFunc) ParallelStateBuilder

	// Transitions from this parallel state
	To(target string) TransitionBuilder
	ToParent(target string) TransitionBuilder

	// Navigation back
	End() MachineBuilder
	Build() MachineDefinition
}

// RegionBuilder handles parallel region configuration
type RegionBuilder interface {
	// States within this region
	State(id string) StateBuilder
	CompositeState(id string) CompositeStateBuilder

	// Pseudostates within this region
	Choice(id string) ChoiceBuilder
	Junction(id string) JunctionBuilder
	Fork(id string) ForkBuilder
	Join(id string) JoinBuilder
	History(id string) HistoryBuilder
	DeepHistory(id string) HistoryBuilder

	// Navigation
	Region(id string) RegionBuilder // Sibling region
	End() ParallelStateBuilder      // Back to parallel state
	Build() MachineDefinition
}

// ChoiceBuilder handles choice pseudostate with conditions
type ChoiceBuilder interface {
	When(condition GuardFunc) ChoiceTransitionBuilder
	Otherwise(target string) ChoiceBuilder
	Do(action ActionFunc) ChoiceBuilder
	OnEntry(action ActionFunc) ChoiceBuilder

	// Navigation back
	State(id string) StateBuilder
	Build() MachineDefinition
}

// ChoiceTransitionBuilder handles conditional transitions from choice
type ChoiceTransitionBuilder interface {
	To(target string) ChoiceBuilder
	Do(action ActionFunc) ChoiceTransitionBuilder
}

// JunctionBuilder handles simple merge points
type JunctionBuilder interface {
	To(target string) JunctionBuilder
	Do(action ActionFunc) JunctionBuilder
	OnEntry(action ActionFunc) JunctionBuilder

	// Navigation back
	State(id string) StateBuilder
	Build() MachineDefinition
}

// ForkBuilder handles splitting to parallel targets
type ForkBuilder interface {
	To(targets ...string) ForkBuilder
	Do(action ActionFunc) ForkBuilder
	OnEntry(action ActionFunc) ForkBuilder

	// Navigation back
	State(id string) StateBuilder
	Build() MachineDefinition
}

// JoinBuilder handles synchronization from multiple sources
type JoinBuilder interface {
	From(sources ...string) JoinBuilder
	To(target string) JoinBuilder
	Do(action ActionFunc) JoinBuilder
	OnEntry(action ActionFunc) JoinBuilder

	// Navigation back
	State(id string) StateBuilder
	Build() MachineDefinition
}

// HistoryBuilder handles history pseudostates
type HistoryBuilder interface {
	Default(target string) HistoryBuilder
	Do(action ActionFunc) HistoryBuilder
	OnEntry(action ActionFunc) HistoryBuilder

	// Navigation back
	State(id string) StateBuilder
	Build() MachineDefinition
}

// Implementation structs

// machineBuilderImpl implements MachineBuilder
type machineBuilderImpl struct {
	machine                  *StateMachine
	initialState             string
	states                   map[string]State
	transitions              []Transition
	built                    bool
	currentTransitionBuilder *transitionBuilderImpl
}

// NewMachine creates a new machine builder with the new fluent API
func NewMachine() MachineBuilder {
	return &machineBuilderImpl{
		machine:     newStateMachine(),
		states:      make(map[string]State),
		transitions: make([]Transition, 0),
	}
}

// NewMachineDefinition creates a new machine definition builder
func NewMachineDefinition() MachineBuilder {
	return NewMachine()
}

// State creates a new atomic state builder
func (mb *machineBuilderImpl) State(id string) StateBuilder {
	mb.saveCurrentTransition()

	var state State
	if existingState, exists := mb.states[id]; exists {
		state = existingState
	} else {
		state = NewAtomicState(id)
		mb.states[id] = state
	}

	return &stateBuilderImpl{
		machineBuilder:     mb,
		currentState:       state,
		stateID:            id,
		pendingTransitions: make([]*transitionBuilderImpl, 0),
	}
}

// CompositeState creates a new composite state builder
func (mb *machineBuilderImpl) CompositeState(id string) CompositeStateBuilder {
	// Create or get existing composite state
	var state CompositeState
	if existingState, exists := mb.states[id]; exists {
		if compositeState, ok := existingState.(CompositeState); ok {
			state = compositeState
		} else {
			// Replace with composite state
			state = NewCompositeState(id)
			mb.states[id] = state
		}
	} else {
		state = NewCompositeState(id)
		mb.states[id] = state
	}

	return &compositeStateBuilderImpl{
		machineBuilder: mb,
		compositeState: state,
		stateID:        id,
	}
}

// ParallelState creates a new parallel state builder
func (mb *machineBuilderImpl) ParallelState(id string) ParallelStateBuilder {
	// Create or get existing parallel state
	var state ParallelState
	if existingState, exists := mb.states[id]; exists {
		if parallelState, ok := existingState.(ParallelState); ok {
			state = parallelState
		} else {
			// Replace with parallel state
			state = NewParallelState(id)
			mb.states[id] = state
		}
	} else {
		state = NewParallelState(id)
		mb.states[id] = state
	}

	return &parallelStateBuilderImpl{
		machineBuilder: mb,
		parallelState:  state,
		stateID:        id,
	}
}

// Choice creates a choice pseudostate builder
func (mb *machineBuilderImpl) Choice(id string) ChoiceBuilder {
	// Save any current transition builder first
	mb.saveCurrentTransition()

	choice := NewPseudoState(id, Choice)
	mb.states[id] = choice

	return &choiceBuilderImpl{
		machineBuilder: mb,
		choiceState:    choice,
	}
}

// Junction creates a junction pseudostate builder
func (mb *machineBuilderImpl) Junction(id string) JunctionBuilder {
	// Save any current transition builder first
	mb.saveCurrentTransition()

	junction := NewPseudoState(id, Junction)
	mb.states[id] = junction

	return &junctionBuilderImpl{
		machineBuilder: mb,
		junctionState:  junction,
	}
}

// Fork creates a fork pseudostate builder
func (mb *machineBuilderImpl) Fork(id string) ForkBuilder {
	// Save any current transition builder first
	mb.saveCurrentTransition()

	fork := NewPseudoState(id, Fork)
	mb.states[id] = fork

	return &forkBuilderImpl{
		machineBuilder: mb,
		forkState:      fork,
	}
}

// Join creates a join pseudostate builder
func (mb *machineBuilderImpl) Join(id string) JoinBuilder {
	// Save any current transition builder first
	mb.saveCurrentTransition()

	join := NewPseudoState(id, Join)
	mb.states[id] = join

	return &joinBuilderImpl{
		machineBuilder: mb,
		joinState:      join,
	}
}

// History creates a shallow history pseudostate builder
func (mb *machineBuilderImpl) History(id string) HistoryBuilder {
	history := NewHistoryState(id, false)
	mb.states[id] = history

	return &historyBuilderImpl{
		machineBuilder: mb,
		historyState:   history,
	}
}

// DeepHistory creates a deep history pseudostate builder
func (mb *machineBuilderImpl) DeepHistory(id string) HistoryBuilder {
	history := NewHistoryState(id, true)
	mb.states[id] = history

	return &historyBuilderImpl{
		machineBuilder: mb,
		historyState:   history,
	}
}

// Build constructs the final machine definition
func (mb *machineBuilderImpl) Build() MachineDefinition {
	// Complex machine building process - validation, state setup, transition wiring, and pseudostate configuration
	mb.saveCurrentTransition()

	if mb.built {
		return &simpleMachineDefinition{
			machine:        mb.machine,
			initialState:   mb.initialState,
			states:         mb.states,
			transitions:    mb.transitions,
			joinConditions: mb.machine.joinConditions,
		}
	}

	if err := mb.validate(); err != nil {
		panic(fmt.Sprintf("Failed to build machine: %v", err))
	}

	mb.machine.initialState = mb.initialState
	mb.machine.currentState = mb.initialState

	if smCtx, ok := mb.machine.context.(*StateMachineContext); ok {
		smCtx.updateCurrentState(mb.machine.currentState)
	}

	mb.machine.states = mb.states

	for _, transition := range mb.transitions {
		sourceState := transition.SourceState
		if mb.machine.transitions[sourceState] == nil {
			mb.machine.transitions[sourceState] = make([]Transition, 0)
		}
		mb.machine.transitions[sourceState] = append(mb.machine.transitions[sourceState], transition)
	}

	for stateID, state := range mb.states {
		if pseudoState, ok := state.(*PseudoStateImpl); ok {
			if pseudoState.Kind() == Join && len(pseudoState.joinSourceCombinations) > 0 {
				mb.machine.joinConditions[stateID] = pseudoState.joinSourceCombinations
			}
		}
	}

	mb.built = true

	return &simpleMachineDefinition{
		machine:        mb.machine,
		initialState:   mb.initialState,
		states:         mb.states,
		transitions:    mb.transitions,
		joinConditions: mb.machine.joinConditions,
	}
}

// validate checks the machine configuration
func (mb *machineBuilderImpl) validate() error {
	if mb.initialState == "" {
		return fmt.Errorf("no initial state defined")
	}

	if _, exists := mb.states[mb.initialState]; !exists {
		return fmt.Errorf("initial state '%s' does not exist", mb.initialState)
	}

	for _, transition := range mb.transitions {
		if _, exists := mb.states[transition.SourceState]; !exists {
			return fmt.Errorf("source state '%s' does not exist for transition", transition.SourceState)
		}
		if _, exists := mb.states[transition.TargetState]; !exists {
			return fmt.Errorf("target state '%s' does not exist for transition", transition.TargetState)
		}
	}

	return nil
}

// addTransition adds a transition to the machine
func (mb *machineBuilderImpl) addTransition(transition Transition) {
	mb.transitions = append(mb.transitions, transition)
}

// saveCurrentTransition saves the current transition builder if any
func (mb *machineBuilderImpl) saveCurrentTransition() {
	if mb.currentTransitionBuilder != nil {
		// Check if this transition has already been added to avoid duplicates
		transition := *mb.currentTransitionBuilder.transition
		alreadyAdded := false

		for _, existingTransition := range mb.transitions {
			if existingTransition.SourceState == transition.SourceState &&
				existingTransition.TargetState == transition.TargetState &&
				existingTransition.EventName == transition.EventName {
				alreadyAdded = true
				break
			}
		}

		if !alreadyAdded {
			mb.addTransition(transition)
		}

		mb.currentTransitionBuilder = nil
	}
}

// StateBuilder implementation

type stateBuilderImpl struct {
	machineBuilder     MachineBuilder
	currentState       State
	stateID            string
	pendingTransitions []*transitionBuilderImpl
	regionContext      *regionContext // Track region context for relative transitions
	compositeParentID  string         // Track composite state parent for relative transitions
}

// regionContext tracks the region context for relative state resolution
type regionContext struct {
	parentStateID string
	regionID      string
	region        Region // Reference to the actual region for setting initial state
}

// To creates a transition to another state
func (sb *stateBuilderImpl) To(target string) TransitionBuilder {
	// Don't save pending transitions here - let the transition builder handle it
	// This prevents duplicate transitions when chaining

	// Resolve relative state names based on context
	resolvedTarget := target

	// If we're in a region context and target is a simple name (no dots),
	// automatically resolve it within the same region
	if sb.regionContext != nil && !strings.Contains(target, ".") && !strings.HasPrefix(target, "../") {
		// Check if the target is a top-level state (like "emergency")
		// by checking if it exists in the machine builder's states
		if mb, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
			if _, exists := mb.states[target]; exists {
				// This is a top-level state, don't prefix with region path
				resolvedTarget = target
			} else {
				// Auto-prefix with region path for simple names within a region
				resolvedTarget = sb.regionContext.parentStateID + "." + sb.regionContext.regionID + "." + target
			}
		}
	} else if sb.compositeParentID != "" && !strings.Contains(target, ".") && !strings.HasPrefix(target, "../") {
		// Check if the target is a top-level state
		if mb, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
			if _, exists := mb.states[target]; exists {
				// This is a top-level state, don't prefix with composite state path
				resolvedTarget = target
			} else {
				// If we're in a composite state context and target is a simple name,
				// resolve it within the same composite state
				resolvedTarget = sb.compositeParentID + "." + target
			}
		}
	} else if strings.HasPrefix(target, "../") {
		// Handle parent navigation - strip the "../" prefix
		parentTarget := target[3:]
		if mb, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
			if _, exists := mb.states[parentTarget]; exists {
				// This is a top-level state, use it directly
				resolvedTarget = parentTarget
			} else {
				// Keep the parent navigation prefix
				resolvedTarget = target
			}
		}
	}

	transition := NewTransition(sb.stateID, resolvedTarget, "")

	transitionBuilder := &transitionBuilderImpl{
		machineBuilder: sb.machineBuilder,
		transition:     transition,
		sourceBuilder:  sb,
	}

	// Set this as the current transition builder on the machine builder
	if mb, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
		// Save any previous transition builder first
		if mb.currentTransitionBuilder != nil {
			mb.addTransition(*mb.currentTransitionBuilder.transition)
		}
		mb.currentTransitionBuilder = transitionBuilder
	}

	return transitionBuilder
}

// ToSelf creates a self-transition
func (sb *stateBuilderImpl) ToSelf() TransitionBuilder {
	return sb.To(sb.stateID)
}

// ToParent creates a transition to parent level (for nested states)
func (sb *stateBuilderImpl) ToParent(target string) TransitionBuilder {
	// For regional states, we want to navigate to the top level
	// Check if the target is a top-level state
	if mb, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
		if _, exists := mb.states[target]; exists {
			// This is a top-level state, use it directly
			return sb.To(target)
		}
	}

	// Add "../" prefix for parent navigation
	return sb.To("../" + target)
}

// OnEntry sets entry action for the state
func (sb *stateBuilderImpl) OnEntry(action ActionFunc) StateBuilder {
	if atomicState, ok := sb.currentState.(*AtomicStateImpl); ok {
		atomicState.WithEntryAction(action)
	}
	return sb
}

// OnExit sets exit action for the state
func (sb *stateBuilderImpl) OnExit(action ActionFunc) StateBuilder {
	if atomicState, ok := sb.currentState.(*AtomicStateImpl); ok {
		atomicState.WithExitAction(action)
	}
	return sb
}

// Final marks this state as final
func (sb *stateBuilderImpl) Final() StateBuilder {
	if atomicState, ok := sb.currentState.(*AtomicStateImpl); ok {
		atomicState.final = true
	}
	return sb
}

func (sb *stateBuilderImpl) Initial() StateBuilder {
	// Mark this state as the initial state
	// Handle different contexts: top-level, composite state, or region
	if sb.regionContext != nil {
		// This is within a region, set as region's initial state
		if machineBuilderImpl, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
			if state, exists := machineBuilderImpl.states[sb.stateID]; exists {
				if regionImpl, ok := sb.regionContext.region.(*RegionImpl); ok {
					regionImpl.WithInitialState(state)
					regionImpl.AddState(state) // Also ensure it's added to the region
				}
			}
		}
	} else if sb.compositeParentID != "" {
		// This is within a composite state, set as composite's initial
		if machineBuilderImpl, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
			if compositeState, exists := machineBuilderImpl.states[sb.compositeParentID]; exists {
				if composite, ok := compositeState.(*CompositeStateImpl); ok {
					// Set as initial state of the composite
					if state, exists := machineBuilderImpl.states[sb.stateID]; exists {
						composite.WithInitialState(state)
					}
				}
			}
		}
	} else {
		// This is a top-level state, set as machine initial
		if machineBuilderImpl, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
			machineBuilderImpl.initialState = sb.stateID
		}
	}
	return sb
}

// Navigation methods - delegate to machine builder
func (sb *stateBuilderImpl) State(id string) StateBuilder {
	// Save any pending transitions from this state builder before switching
	sb.savePendingTransitions()
	return sb.machineBuilder.State(id)
}

func (sb *stateBuilderImpl) CompositeState(id string) CompositeStateBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.CompositeState(id)
}

func (sb *stateBuilderImpl) ParallelState(id string) ParallelStateBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.ParallelState(id)
}

func (sb *stateBuilderImpl) Choice(id string) ChoiceBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.Choice(id)
}

func (sb *stateBuilderImpl) Junction(id string) JunctionBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.Junction(id)
}

func (sb *stateBuilderImpl) Fork(id string) ForkBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.Fork(id)
}

func (sb *stateBuilderImpl) Join(id string) JoinBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.Join(id)
}

func (sb *stateBuilderImpl) History(id string) HistoryBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.History(id)
}

func (sb *stateBuilderImpl) DeepHistory(id string) HistoryBuilder {
	sb.savePendingTransitions()
	return sb.machineBuilder.DeepHistory(id)
}

func (sb *stateBuilderImpl) Build() MachineDefinition {
	sb.savePendingTransitions()
	return sb.machineBuilder.Build()
}

// savePendingTransitions saves all pending transitions to the machine builder
func (sb *stateBuilderImpl) savePendingTransitions() {
	if mb, ok := sb.machineBuilder.(*machineBuilderImpl); ok {
		for _, transitionBuilder := range sb.pendingTransitions {
			mb.addTransition(*transitionBuilder.transition)
		}
	}
	// Clear pending transitions after saving
	sb.pendingTransitions = nil
}

// TransitionBuilder implementation

type transitionBuilderImpl struct {
	machineBuilder MachineBuilder
	transition     *Transition
	sourceBuilder  StateBuilder
}

// On sets the event for this transition
func (tb *transitionBuilderImpl) On(event string) TransitionBuilder {
	tb.transition.EventName = event
	return tb
}

// OnCompletion sets this as a completion transition (automatic when state completes)
func (tb *transitionBuilderImpl) OnCompletion() TransitionBuilder {
	completionEventName := "__completion_" + tb.transition.SourceState
	tb.transition.EventName = completionEventName
	return tb
}

// When adds a guard condition
func (tb *transitionBuilderImpl) When(guard GuardFunc) TransitionBuilder {
	tb.transition.Guard = guard
	return tb
}

// Unless adds a negated guard condition
func (tb *transitionBuilderImpl) Unless(guard GuardFunc) TransitionBuilder {
	tb.transition.Guard = func(ctx Context) bool {
		return !guard(ctx)
	}
	return tb
}

// Do adds an action to this transition (replaces WithTransitionAction)
func (tb *transitionBuilderImpl) Do(action ActionFunc) TransitionBuilder {
	// For now, set single action - can be enhanced to support multiple actions
	tb.transition.Action = action
	return tb
}

// DoIf adds a conditional action
func (tb *transitionBuilderImpl) DoIf(condition GuardFunc, action ActionFunc) TransitionBuilder {
	conditionalAction := func(ctx Context) error {
		if condition(ctx) {
			return action(ctx)
		}
		return nil
	}
	return tb.Do(conditionalAction)
}

// DoAsync adds an async action
func (tb *transitionBuilderImpl) DoAsync(action ActionFunc) TransitionBuilder {
	asyncAction := func(ctx Context) error {
		go func() {
			_ = action(ctx)
		}()
		return nil
	}
	return tb.Do(asyncAction)
}

// OnError sets error transition target
func (tb *transitionBuilderImpl) OnError(errorState string) TransitionBuilder {
	// This can be enhanced to support error handling in transition
	return tb
}

// OnTimeout sets timeout transition target
func (tb *transitionBuilderImpl) OnTimeout(timeoutState string) TransitionBuilder {
	// This can be enhanced to support timeout handling
	return tb
}

// To creates another transition from the same source state
func (tb *transitionBuilderImpl) To(target string) TransitionBuilder {
	// Add current transition to machine
	if mb, ok := tb.machineBuilder.(*machineBuilderImpl); ok {
		mb.addTransition(*tb.transition)
	}

	// Create new transition
	newTransition := NewTransition(tb.transition.SourceState, target, "")
	newTransitionBuilder := &transitionBuilderImpl{
		machineBuilder: tb.machineBuilder,
		transition:     newTransition,
		sourceBuilder:  tb.sourceBuilder,
	}

	// Set this as the current transition builder on the machine builder
	if mb, ok := tb.machineBuilder.(*machineBuilderImpl); ok {
		mb.currentTransitionBuilder = newTransitionBuilder
	}

	return newTransitionBuilder
}

// ToSelf creates a self-transition
func (tb *transitionBuilderImpl) ToSelf() TransitionBuilder {
	return tb.To(tb.transition.SourceState)
}

// ToParent creates a transition to parent level
func (tb *transitionBuilderImpl) ToParent(target string) TransitionBuilder {
	return tb.To("../" + target)
}

// State navigates to a new state definition
func (tb *transitionBuilderImpl) State(id string) StateBuilder {
	// Add current transition before switching
	if mb, ok := tb.machineBuilder.(*machineBuilderImpl); ok {
		mb.addTransition(*tb.transition)
	}
	return tb.machineBuilder.State(id)
}

// CompositeState navigates to composite state definition
func (tb *transitionBuilderImpl) CompositeState(id string) CompositeStateBuilder {
	// Add current transition before switching
	if mb, ok := tb.machineBuilder.(*machineBuilderImpl); ok {
		mb.addTransition(*tb.transition)
	}
	return tb.machineBuilder.CompositeState(id)
}

// Build finalizes the machine
func (tb *transitionBuilderImpl) Build() MachineDefinition {
	// Add current transition before building
	if mb, ok := tb.machineBuilder.(*machineBuilderImpl); ok {
		mb.addTransition(*tb.transition)
	}
	return tb.machineBuilder.Build()
}

// Placeholder implementations for other builders
// These will be implemented as needed

type compositeStateBuilderImpl struct {
	machineBuilder MachineBuilder
	compositeState CompositeState
	stateID        string
}

func (csb *compositeStateBuilderImpl) State(id string) StateBuilder {
	fullID := csb.stateID + "." + id
	stateBuilder := csb.machineBuilder.State(fullID)

	// Add composite state context to enable relative transitions
	if sb, ok := stateBuilder.(*stateBuilderImpl); ok {
		// For composite states, we treat them like a single-level namespace
		sb.compositeParentID = csb.stateID
	}

	return stateBuilder
}

func (csb *compositeStateBuilderImpl) CompositeState(id string) CompositeStateBuilder {
	return csb.machineBuilder.CompositeState(csb.stateID + "." + id)
}

func (csb *compositeStateBuilderImpl) Choice(id string) ChoiceBuilder {
	return csb.machineBuilder.Choice(csb.stateID + "." + id)
}

func (csb *compositeStateBuilderImpl) Junction(id string) JunctionBuilder {
	return csb.machineBuilder.Junction(csb.stateID + "." + id)
}

func (csb *compositeStateBuilderImpl) Fork(id string) ForkBuilder {
	return csb.machineBuilder.Fork(csb.stateID + "." + id)
}

func (csb *compositeStateBuilderImpl) Join(id string) JoinBuilder {
	return csb.machineBuilder.Join(csb.stateID + "." + id)
}

func (csb *compositeStateBuilderImpl) History(id string) HistoryBuilder {
	return csb.machineBuilder.History(csb.stateID + "." + id)
}

func (csb *compositeStateBuilderImpl) DeepHistory(id string) HistoryBuilder {
	return csb.machineBuilder.DeepHistory(csb.stateID + "." + id)
}

func (csb *compositeStateBuilderImpl) OnEntry(action ActionFunc) CompositeStateBuilder {
	// Set entry action on composite state
	return csb
}

func (csb *compositeStateBuilderImpl) OnExit(action ActionFunc) CompositeStateBuilder {
	// Set exit action on composite state
	return csb
}

func (csb *compositeStateBuilderImpl) To(target string) TransitionBuilder {
	transition := NewTransition(csb.stateID, target, "")
	tb := &transitionBuilderImpl{
		machineBuilder: csb.machineBuilder,
		transition:     transition,
		sourceBuilder:  nil, // Not a regular state builder
	}
	if mb, ok := csb.machineBuilder.(*machineBuilderImpl); ok {
		if mb.currentTransitionBuilder != nil {
			mb.addTransition(*mb.currentTransitionBuilder.transition)
		}
		mb.currentTransitionBuilder = tb
	}
	return tb
}

func (csb *compositeStateBuilderImpl) ToParent(target string) TransitionBuilder {
	return csb.To("../" + target)
}

func (csb *compositeStateBuilderImpl) End() MachineBuilder {
	return csb.machineBuilder
}

func (csb *compositeStateBuilderImpl) Build() MachineDefinition {
	return csb.machineBuilder.Build()
}

// Placeholder implementations for other builders
type parallelStateBuilderImpl struct {
	machineBuilder MachineBuilder
	parallelState  ParallelState
	stateID        string
}

func (psb *parallelStateBuilderImpl) Region(id string) RegionBuilder {
	// Create new region within parallel state
	region := NewRegion(id, psb.parallelState)
	psb.parallelState.AddRegion(region)

	return &regionBuilderImpl{
		machineBuilder:  psb.machineBuilder,
		parallelBuilder: psb,
		region:          region,
		regionID:        id,
		parentStateID:   psb.stateID,
	}
}

func (psb *parallelStateBuilderImpl) OnEntry(action ActionFunc) ParallelStateBuilder {
	return psb
}

func (psb *parallelStateBuilderImpl) OnExit(action ActionFunc) ParallelStateBuilder {
	return psb
}

func (psb *parallelStateBuilderImpl) To(target string) TransitionBuilder {
	transition := NewTransition(psb.stateID, target, "")
	tb := &transitionBuilderImpl{
		machineBuilder: psb.machineBuilder,
		transition:     transition,
		sourceBuilder:  nil,
	}
	if mb, ok := psb.machineBuilder.(*machineBuilderImpl); ok {
		if mb.currentTransitionBuilder != nil {
			mb.addTransition(*mb.currentTransitionBuilder.transition)
		}
		mb.currentTransitionBuilder = tb
	}
	return tb
}

func (psb *parallelStateBuilderImpl) ToParent(target string) TransitionBuilder {
	return psb.To("../" + target)
}

func (psb *parallelStateBuilderImpl) End() MachineBuilder {
	return psb.machineBuilder
}

func (psb *parallelStateBuilderImpl) Build() MachineDefinition {
	return psb.machineBuilder.Build()
}

type regionBuilderImpl struct {
	machineBuilder  MachineBuilder
	parallelBuilder ParallelStateBuilder
	region          Region
	regionID        string
	parentStateID   string
}

func (rb *regionBuilderImpl) State(id string) StateBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	stateBuilder := rb.machineBuilder.State(fullID)

	// Add region context to the state builder
	if sb, ok := stateBuilder.(*stateBuilderImpl); ok {
		sb.regionContext = &regionContext{
			parentStateID: rb.parentStateID,
			regionID:      rb.regionID,
			region:        rb.region, // Add reference to the region
		}
	}

	// Add to region
	if regionImpl, ok := rb.region.(*RegionImpl); ok {
		if state, exists := rb.machineBuilder.(*machineBuilderImpl).states[fullID]; exists {
			regionImpl.AddState(state)
		}
	}

	return stateBuilder
}

func (rb *regionBuilderImpl) CompositeState(id string) CompositeStateBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	return rb.machineBuilder.CompositeState(fullID)
}

func (rb *regionBuilderImpl) Choice(id string) ChoiceBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	return rb.machineBuilder.Choice(fullID)
}

func (rb *regionBuilderImpl) Junction(id string) JunctionBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	return rb.machineBuilder.Junction(fullID)
}

func (rb *regionBuilderImpl) Fork(id string) ForkBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	return rb.machineBuilder.Fork(fullID)
}

func (rb *regionBuilderImpl) Join(id string) JoinBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	return rb.machineBuilder.Join(fullID)
}

func (rb *regionBuilderImpl) History(id string) HistoryBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	return rb.machineBuilder.History(fullID)
}

func (rb *regionBuilderImpl) DeepHistory(id string) HistoryBuilder {
	fullID := rb.parentStateID + "." + rb.regionID + "." + id
	return rb.machineBuilder.DeepHistory(fullID)
}

func (rb *regionBuilderImpl) Region(id string) RegionBuilder {
	return rb.parallelBuilder.Region(id)
}

func (rb *regionBuilderImpl) End() ParallelStateBuilder {
	return rb.parallelBuilder
}

func (rb *regionBuilderImpl) Build() MachineDefinition {
	return rb.machineBuilder.Build()
}

// Pseudostate builder implementations
type choiceBuilderImpl struct {
	machineBuilder MachineBuilder
	choiceState    *PseudoStateImpl
}

func (cb *choiceBuilderImpl) When(condition GuardFunc) ChoiceTransitionBuilder {
	return &choiceTransitionBuilderImpl{
		choiceBuilder: cb,
		condition:     condition,
	}
}

func (cb *choiceBuilderImpl) Otherwise(target string) ChoiceBuilder {
	cb.choiceState.SetDefaultTarget(target)
	return cb
}

func (cb *choiceBuilderImpl) Do(action ActionFunc) ChoiceBuilder {
	cb.choiceState.WithEntryAction(action)
	return cb
}

func (cb *choiceBuilderImpl) OnEntry(action ActionFunc) ChoiceBuilder {
	return cb.Do(action)
}

func (cb *choiceBuilderImpl) State(id string) StateBuilder {
	return cb.machineBuilder.State(id)
}

func (cb *choiceBuilderImpl) Build() MachineDefinition {
	return cb.machineBuilder.Build()
}

type choiceTransitionBuilderImpl struct {
	choiceBuilder *choiceBuilderImpl
	condition     GuardFunc
	action        ActionFunc
}

func (ctb *choiceTransitionBuilderImpl) To(target string) ChoiceBuilder {
	ctb.choiceBuilder.choiceState.AddChoiceCondition(ctb.condition, target, ctb.action)
	return ctb.choiceBuilder
}

func (ctb *choiceTransitionBuilderImpl) Do(action ActionFunc) ChoiceTransitionBuilder {
	ctb.action = action
	return ctb
}

// Simple implementations for other pseudostate builders
type junctionBuilderImpl struct {
	machineBuilder MachineBuilder
	junctionState  *PseudoStateImpl
}

func (jb *junctionBuilderImpl) To(target string) JunctionBuilder {
	jb.junctionState.SetDefaultTarget(target)
	return jb
}

func (jb *junctionBuilderImpl) Do(action ActionFunc) JunctionBuilder {
	jb.junctionState.WithEntryAction(action)
	return jb
}

func (jb *junctionBuilderImpl) OnEntry(action ActionFunc) JunctionBuilder {
	return jb.Do(action)
}

func (jb *junctionBuilderImpl) State(id string) StateBuilder {
	return jb.machineBuilder.State(id)
}

func (jb *junctionBuilderImpl) Build() MachineDefinition {
	return jb.machineBuilder.Build()
}

type forkBuilderImpl struct {
	machineBuilder MachineBuilder
	forkState      *PseudoStateImpl
}

func (fb *forkBuilderImpl) To(targets ...string) ForkBuilder {
	fb.forkState.SetForkTargets(targets)
	return fb
}

func (fb *forkBuilderImpl) Do(action ActionFunc) ForkBuilder {
	fb.forkState.WithEntryAction(action)
	return fb
}

func (fb *forkBuilderImpl) OnEntry(action ActionFunc) ForkBuilder {
	return fb.Do(action)
}

func (fb *forkBuilderImpl) State(id string) StateBuilder {
	return fb.machineBuilder.State(id)
}

func (fb *forkBuilderImpl) Build() MachineDefinition {
	return fb.machineBuilder.Build()
}

type joinBuilderImpl struct {
	machineBuilder MachineBuilder
	joinState      *PseudoStateImpl
}

func (jb *joinBuilderImpl) From(sources ...string) JoinBuilder {
	jb.joinState.SetJoinSources(sources)
	return jb
}

func (jb *joinBuilderImpl) To(target string) JoinBuilder {
	jb.joinState.SetJoinTarget(target)
	return jb
}

func (jb *joinBuilderImpl) Do(action ActionFunc) JoinBuilder {
	jb.joinState.WithEntryAction(action)
	return jb
}

func (jb *joinBuilderImpl) OnEntry(action ActionFunc) JoinBuilder {
	return jb.Do(action)
}

func (jb *joinBuilderImpl) State(id string) StateBuilder {
	return jb.machineBuilder.State(id)
}

func (jb *joinBuilderImpl) Build() MachineDefinition {
	return jb.machineBuilder.Build()
}

type historyBuilderImpl struct {
	machineBuilder MachineBuilder
	historyState   *PseudoStateImpl
}

func (hb *historyBuilderImpl) Default(target string) HistoryBuilder {
	hb.historyState.SetHistoryDefault(target)
	return hb
}

func (hb *historyBuilderImpl) Do(action ActionFunc) HistoryBuilder {
	hb.historyState.WithEntryAction(action)
	return hb
}

func (hb *historyBuilderImpl) OnEntry(action ActionFunc) HistoryBuilder {
	return hb.Do(action)
}

func (hb *historyBuilderImpl) State(id string) StateBuilder {
	return hb.machineBuilder.State(id)
}

func (hb *historyBuilderImpl) Build() MachineDefinition {
	return hb.machineBuilder.Build()
}

// simpleMachineDefinition is a simple implementation of MachineDefinition for now
type simpleMachineDefinition struct {
	machine        *StateMachine
	initialState   string
	states         map[string]State
	transitions    []Transition
	joinConditions map[string][][]string
}

// CreateInstance creates a new machine instance
func (smd *simpleMachineDefinition) CreateInstance() Machine {
	newMachine := newStateMachine()
	newMachine.initialState = smd.initialState
	newMachine.currentState = smd.initialState
	newMachine.states = make(map[string]State)
	newMachine.transitions = make(map[string][]Transition)
	newMachine.joinConditions = make(map[string][][]string)
	newMachine.joinTracking = make(map[string]map[string]bool)

	// Copy states
	for id, state := range smd.states {
		newMachine.states[id] = state
	}

	// Copy transitions
	for _, transition := range smd.transitions {
		sourceState := transition.SourceState
		if newMachine.transitions[sourceState] == nil {
			newMachine.transitions[sourceState] = make([]Transition, 0)
		}
		newMachine.transitions[sourceState] = append(newMachine.transitions[sourceState], transition)
	}

	// Copy join conditions (combinations)
	for joinID, combinations := range smd.joinConditions {
		copiedCombinations := make([][]string, len(combinations))
		for i, combination := range combinations {
			copiedCombination := make([]string, len(combination))
			copy(copiedCombination, combination)
			copiedCombinations[i] = copiedCombination
		}
		newMachine.joinConditions[joinID] = copiedCombinations
	}

	return newMachine
}

func (smd *simpleMachineDefinition) Build() MachineDefinition {
	return smd
}

func (smd *simpleMachineDefinition) GetInitialState() string {
	return smd.initialState
}

func (smd *simpleMachineDefinition) GetStates() map[string]State {
	// Return a copy to prevent external modification
	states := make(map[string]State)
	for id, state := range smd.states {
		states[id] = state
	}
	return states
}

func (smd *simpleMachineDefinition) GetTransitions() map[string][]Transition {
	// Convert slice to map format
	transitions := make(map[string][]Transition)
	for _, transition := range smd.transitions {
		sourceState := transition.SourceState
		if transitions[sourceState] == nil {
			transitions[sourceState] = make([]Transition, 0)
		}
		transitions[sourceState] = append(transitions[sourceState], transition)
	}
	return transitions
}
