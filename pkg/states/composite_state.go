package states

import (
	"fmt"
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// CompositeState represents a hierarchical state that contains child states
type CompositeState struct {
	*BaseState
	children     map[string]core.State
	currentChild core.State
	initialChild core.State
	historyState core.State
	transitions  []*core.Transition
	mutex        sync.RWMutex
}

// NewCompositeState creates a new hierarchical composite state
func NewCompositeState(name string) *CompositeState {
	return &CompositeState{
		BaseState: NewBaseState(name),
		children:  make(map[string]core.State),
	}
}

// IsComposite returns true indicating this is a composite state
func (s *CompositeState) IsComposite() bool {
	return true
}

// AddChild adds a child state to this composite state
func (s *CompositeState) AddChild(child core.State) *CompositeState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if child != nil {
		s.children[child.Name()] = child
		child.SetParent(s)
	}

	return s
}

// SetInitialChild sets the initial child state
func (s *CompositeState) SetInitialChild(child core.State) *CompositeState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if child != nil && s.children[child.Name()] != nil {
		s.initialChild = child
	}

	return s
}

// GetCurrentChild returns the currently active child state
func (s *CompositeState) GetCurrentChild() core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.currentChild
}

// GetChild returns a child state by name
func (s *CompositeState) GetChild(name string) core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.children[name]
}

// GetChildren returns all child states
func (s *CompositeState) GetChildren() map[string]core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make(map[string]core.State)
	for k, v := range s.children {
		result[k] = v
	}
	return result
}

// Enter activates the composite state and its initial child
func (s *CompositeState) Enter(ctx *core.Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	historyState := s.historyState
	s.mutex.RUnlock()

	// If we have a history state, enter through that
	if historyState != nil && historyState.IsHistory() {
		return historyState.Enter(ctx)
	}

	// Otherwise use the initial child
	s.mutex.RLock()
	initialChild := s.initialChild
	s.mutex.RUnlock()

	if initialChild == nil {
		return fmt.Errorf("composite state %s has no initial child state", s.Name())
	}

	s.mutex.Lock()
	s.currentChild = initialChild
	s.mutex.Unlock()

	return initialChild.Enter(ctx)
}

// Exit deactivates the current child and the composite state
func (s *CompositeState) Exit(ctx *core.Context) error {
	s.mutex.Lock()
	currentChild := s.currentChild
	s.currentChild = nil
	s.mutex.Unlock()

	// Exit the current child state if one exists
	if currentChild != nil {
		if err := currentChild.Exit(ctx); err != nil {
			return err
		}
	}

	// Exit this state
	return s.BaseState.Exit(ctx)
}

// HandleEvent processes events by delegating to the current child first
func (s *CompositeState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	// Check internal transitions first
	s.mutex.RLock()
	transitions := s.transitions
	s.mutex.RUnlock()

	for _, transition := range transitions {
		if transition.Event == event.Name && transition.CanExecute(ctx) {
			if err := transition.Execute(ctx); err != nil {
				return nil, err
			}
			return transition.To, nil
		}
	}

	// Delegate to current child
	s.mutex.RLock()
	currentChild := s.currentChild
	s.mutex.RUnlock()

	if currentChild == nil {
		return nil, nil
	}

	// Let the child try to handle the event
	nextState, err := currentChild.HandleEvent(event, ctx)
	if err != nil {
		return nil, err
	}

	// If the child handled the event and returned a next state
	if nextState != nil {
		// If the next state is a direct child of this composite, transition to it
		if s.HasChild(nextState.Name()) {
			childResult, err := s.transitionToChild(nextState, ctx)
			if err != nil {
				return nil, err
			}

			// Important: When transitioning between child states,
			// we should return nil to indicate this composite state remains active
			return childResult, nil
		}

		// If the next state is not a child, return it to the parent state machine
		return nextState, nil
	}

	// Event not handled by this state or its children
	return nil, nil
}

// TransitionToChild transitions to a specific child state
func (s *CompositeState) TransitionToChild(childName string, ctx *core.Context) error {
	s.mutex.RLock()
	targetChild := s.children[childName]
	s.mutex.RUnlock()

	if targetChild == nil {
		return fmt.Errorf("child state %s not found in composite state %s", childName, s.Name())
	}

	// Special handling for history states
	if targetChild.IsHistory() {
		return targetChild.Enter(ctx)
	}

	// Normal state entry
	_, err := s.transitionToChild(targetChild, ctx)
	return err
}

// transitionToChild handles internal child state transitions
func (s *CompositeState) transitionToChild(newChild core.State, ctx *core.Context) (core.State, error) {
	// Check if the new state is a child of this composite
	s.mutex.RLock()
	isDirectChild := s.children[newChild.Name()] != nil
	currentChild := s.currentChild
	s.mutex.RUnlock()

	if isDirectChild {
		// Internal transition within this composite
		if currentChild != nil && currentChild != newChild {
			if err := currentChild.Exit(ctx); err != nil {
				return nil, err
			}
		}

		s.mutex.Lock()
		s.currentChild = newChild
		s.mutex.Unlock()

		if err := newChild.Enter(ctx); err != nil {
			return nil, err
		}
		return nil, nil // Stay within this composite
	}

	// External transition - return the new state to parent
	return newChild, nil
}

// HasChild checks if a state is a direct child of this composite
func (s *CompositeState) HasChild(stateName string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.children[stateName] != nil
}

// IsChildActive checks if a specific child is currently active
func (s *CompositeState) IsChildActive(childName string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.currentChild != nil && s.currentChild.Name() == childName
}

// Legacy compatibility methods - these delegate to the new methods

// AddSubstate adds a substate (legacy method, delegates to AddChild)
func (s *CompositeState) AddSubstate(state core.State) *CompositeState {
	return s.AddChild(state)
}

// GetCurrentState returns the current state (legacy method, delegates to GetCurrentChild)
func (s *CompositeState) GetCurrentState() core.State {
	return s.GetCurrentChild()
}

// SetHistoryState sets the history state for this composite state
func (s *CompositeState) SetHistoryState(state core.State) *CompositeState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if state != nil && state.IsHistory() {
		s.historyState = state
	}

	return s
}

// GetHistoryState returns the currently set history state
func (s *CompositeState) GetHistoryState() core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.historyState
}

// AddTransition adds a transition to this composite state
func (s *CompositeState) AddTransition(transition *core.Transition) *CompositeState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.transitions == nil {
		s.transitions = make([]*core.Transition, 0)
	}
	s.transitions = append(s.transitions, transition)
	return s
}
