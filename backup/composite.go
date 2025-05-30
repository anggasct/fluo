// Package flux provides a comprehensive finite state machine library for Go
// that supports hierarchical states following industry standards.
package flux

import (
	"sync"
)

// CompositeState represents a hierarchical state that contains child states
type CompositeState struct {
	*BaseState
	children     map[string]State
	currentChild State
	initialChild State
	historyState State
	transitions  []*Transition
	mutex        sync.RWMutex
}

// NewCompositeState creates a new hierarchical composite state
func NewCompositeState(name string) *CompositeState {
	return &CompositeState{
		BaseState:   NewBaseState(name),
		children:    make(map[string]State),
		transitions: make([]*Transition, 0),
	}
}

// IsComposite returns true indicating this is a composite state
func (s *CompositeState) IsComposite() bool {
	return true
}

// AddChild adds a child state to this composite state
func (s *CompositeState) AddChild(child State) *CompositeState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.children[child.Name()] = child
	child.SetParent(s)

	// First child automatically becomes the initial child
	if s.initialChild == nil {
		s.initialChild = child
	}

	return s
}

// GetCurrentChild returns the currently active child state
func (s *CompositeState) GetCurrentChild() State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.currentChild
}

// GetChild returns a child state by name
func (s *CompositeState) GetChild(name string) State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.children[name]
}

// GetChildren returns all child states
func (s *CompositeState) GetChildren() map[string]State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	children := make(map[string]State)
	for name, child := range s.children {
		children[name] = child
	}
	return children
}

// Enter activates the composite state and its initial child
func (s *CompositeState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.Lock()
	targetChild := s.initialChild
	s.currentChild = targetChild
	s.mutex.Unlock()

	if targetChild != nil {
		return targetChild.Enter(ctx)
	}

	return nil
}

// Exit deactivates the current child and the composite state
func (s *CompositeState) Exit(ctx *Context) error {
	s.mutex.RLock()
	currentChild := s.currentChild
	historyState := s.historyState
	s.mutex.RUnlock()

	// First record the history state if we have both a current child and history state
	if currentChild != nil && historyState != nil {
		if historyState, ok := historyState.(*HistoryState); ok {
			// Record the last active state before exiting
			historyState.SetLastState(currentChild)
			// For debugging
			// fmt.Printf("Recording history: composite '%s' saving child '%s' in history\n",
			//    s.Name(), currentChild.Name())
		}
	}

	// Now exit the current child if there is one
	if currentChild != nil {
		if err := currentChild.Exit(ctx); err != nil {
			return err
		}
	}

	// Clear the current child but keep history
	s.mutex.Lock()
	s.currentChild = nil
	s.mutex.Unlock()

	// Then exit the base state
	return s.BaseState.Exit(ctx)
}

// HandleEvent processes events by delegating to the current child first
func (s *CompositeState) HandleEvent(event *Event, ctx *Context) (State, error) {
	s.mutex.RLock()
	currentChild := s.currentChild
	transitions := s.transitions
	s.mutex.RUnlock()

	// First try to handle the event in the current child
	if currentChild != nil {
		newState, err := currentChild.HandleEvent(event, ctx)
		if err != nil {
			return nil, err
		}
		if newState != nil {
			// Child handled the event and wants to transition
			return s.transitionToChild(newState, ctx)
		}
	}

	// Check if we have any transitions for this event
	for _, transition := range transitions {
		if transition.From == currentChild && transition.Event == event.Name {
			if transition.CanExecute(ctx) {
				if err := currentChild.Exit(ctx); err != nil {
					return nil, err
				}

				if transition.Action != nil {
					if err := transition.Action(ctx); err != nil {
						return nil, err
					}
				}

				targetChild := transition.To
				// Transition to the new child
				s.mutex.Lock()
				s.currentChild = targetChild
				s.mutex.Unlock()

				if err := targetChild.Enter(ctx); err != nil {
					return nil, err
				}

				return nil, nil
			}
		}
	}

	// If child didn't handle it, delegate to base state
	return s.BaseState.HandleEvent(event, ctx)
}

// TransitionToChild transitions to a specific child state
func (s *CompositeState) TransitionToChild(childName string, ctx *Context) error {
	s.mutex.RLock()
	targetChild := s.children[childName]
	currentChild := s.currentChild
	s.mutex.RUnlock()

	if targetChild == nil {
		return NewStateMachineError("CHILD_NOT_FOUND", "Child state not found: "+childName)
	}

	// Exit current child if any
	if currentChild != nil && currentChild != targetChild {
		// Record the current child in history state if one exists
		if s.historyState != nil {
			if historyState, ok := s.historyState.(*HistoryState); ok {
				// Don't record history state as last state
				if !currentChild.IsHistory() {
					historyState.SetLastState(currentChild)
				}
			}
		}

		// Then exit the current child
		if err := currentChild.Exit(ctx); err != nil {
			return err
		}
	}

	// Set and enter new child
	s.mutex.Lock()
	s.currentChild = targetChild
	s.mutex.Unlock()

	// Special handling for history states
	if targetChild.IsHistory() {
		// Let the history state figure out the real target
		return targetChild.Enter(ctx)
	}

	// Normal state entry
	return targetChild.Enter(ctx)
}

// transitionToChild handles internal child state transitions
func (s *CompositeState) transitionToChild(newChild State, ctx *Context) (State, error) {
	s.mutex.RLock()
	currentChild := s.currentChild
	s.mutex.RUnlock()

	// Check if the new state is a child of this composite
	s.mutex.RLock()
	isDirectChild := s.children[newChild.Name()] != nil
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
func (s *CompositeState) AddSubstate(state State) *CompositeState {
	return s.AddChild(state)
}

// GetCurrentState returns the current state (legacy method, delegates to GetCurrentChild)
func (s *CompositeState) GetCurrentState() State {
	return s.GetCurrentChild()
}

// SetHistoryState sets the history state for this composite state
func (s *CompositeState) SetHistoryState(state State) *CompositeState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if state != nil && state.IsHistory() {
		s.historyState = state
	}

	return s
}

// GetHistoryState returns the currently set history state
func (s *CompositeState) GetHistoryState() State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.historyState
}

// AddTransition adds a transition to this composite state
// This implementation uses the HandleEvent method to check and process transitions
func (s *CompositeState) AddTransition(transition *Transition) *CompositeState {
	// We can't directly access the state machine's transitions array
	// So we'll store them in a map and handle them in HandleEvent
	if s.transitions == nil {
		s.transitions = make([]*Transition, 0)
	}
	s.transitions = append(s.transitions, transition)
	return s
}
