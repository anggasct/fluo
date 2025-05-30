package states

import (
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// BaseState provides common functionality for all state types
type BaseState struct {
	name        string
	parent      core.State
	entryAction core.Action
	exitAction  core.Action
	doActivity  core.Action
	isActive    bool
	mutex       sync.RWMutex
}

// NewBaseState creates a new base state
func NewBaseState(name string) *BaseState {
	return &BaseState{
		name: name,
	}
}

// Name returns the state name
func (s *BaseState) Name() string {
	return s.name
}

// GetName returns the state name (alias for Name())
func (s *BaseState) GetName() string {
	return s.name
}

// GetParent returns the parent state
func (s *BaseState) GetParent() core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.parent
}

// SetParent sets the parent state
func (s *BaseState) SetParent(parent core.State) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.parent = parent
}

// IsComposite returns false for base state
func (s *BaseState) IsComposite() bool {
	return false
}

// IsParallel returns false for base state
func (s *BaseState) IsParallel() bool {
	return false
}

// IsHistory returns false for base state
func (s *BaseState) IsHistory() bool {
	return false
}

// IsFinal returns false for base state, will be overridden by final states
func (s *BaseState) IsFinal() bool {
	return false
}

// AddEntryAction adds an action to execute when entering the state
func (s *BaseState) AddEntryAction(action core.Action) *BaseState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	originalAction := s.entryAction
	if originalAction == nil {
		s.entryAction = action
	} else {
		s.entryAction = func(ctx *core.Context) error {
			if err := originalAction(ctx); err != nil {
				return err
			}
			return action(ctx)
		}
	}

	return s
}

// AddExitAction adds an action to execute when exiting the state
func (s *BaseState) AddExitAction(action core.Action) *BaseState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	originalAction := s.exitAction
	if originalAction == nil {
		s.exitAction = action
	} else {
		s.exitAction = func(ctx *core.Context) error {
			if err := originalAction(ctx); err != nil {
				return err
			}
			return action(ctx)
		}
	}

	return s
}

// AddDoActivity adds an activity to execute while in the state
func (s *BaseState) AddDoActivity(action core.Action) *BaseState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.doActivity = action
	return s
}

// Enter handles entry into the state
func (s *BaseState) Enter(ctx *core.Context) error {
	s.mutex.Lock()
	s.isActive = true
	action := s.entryAction
	s.mutex.Unlock()

	if action != nil {
		return action(ctx)
	}
	return nil
}

// Exit handles exit from the state
func (s *BaseState) Exit(ctx *core.Context) error {
	s.mutex.Lock()
	s.isActive = false
	action := s.exitAction
	s.mutex.Unlock()

	if action != nil {
		return action(ctx)
	}
	return nil
}

// HandleEvent processes events for the state
func (s *BaseState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	// Base state doesn't handle any events itself
	return nil, nil
}

// IsActive returns whether the state is currently active
func (s *BaseState) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isActive
}

// SimpleState represents a basic state with no internal structure
type SimpleState struct {
	*BaseState
	transitions []*core.Transition
}

// NewSimpleState creates a new simple state
func NewSimpleState(name string) *SimpleState {
	return &SimpleState{
		BaseState:   NewBaseState(name),
		transitions: make([]*core.Transition, 0),
	}
}

// AddTransition adds a transition from this state
func (s *SimpleState) AddTransition(to core.State, event string) *core.Transition {
	transition := core.NewTransition(s, to, event)

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.transitions = append(s.transitions, transition)

	return transition
}

// HandleEvent checks for internal transitions
func (s *SimpleState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, transition := range s.transitions {
		if transition.Event == event.Name && transition.CanExecute(ctx) {
			if err := transition.Execute(ctx); err != nil {
				return nil, err
			}
			return transition.To, nil
		}
	}

	return nil, nil
}

// FinalState represents a terminal state
type FinalState struct {
	*SimpleState
}

// NewFinalState creates a new final state
func NewFinalState(name string) *FinalState {
	return &FinalState{
		SimpleState: NewSimpleState(name),
	}
}

// Enter notifies completion when entering a final state
func (s *FinalState) Enter(ctx *core.Context) error {
	if err := s.SimpleState.Enter(ctx); err != nil {
		return err
	}

	// If there's a parent composite state, notify it of completion
	if parent := s.GetParent(); parent != nil && parent.IsComposite() {
		// Notify completion via context if needed
		// Currently just handled by the parent composite checking final states
	}

	return nil
}

// IsFinal returns true for final state
func (s *FinalState) IsFinal() bool {
	return true
}

// EntryPointState represents an entry point to a composite state
type EntryPointState struct {
	*SimpleState
	target core.State
}

// NewEntryPointState creates a new entry point state
func NewEntryPointState(name string) *EntryPointState {
	return &EntryPointState{
		SimpleState: NewSimpleState(name),
	}
}

// SetTarget sets the target state for this entry point
func (s *EntryPointState) SetTarget(target core.State) *EntryPointState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.target = target
	return s
}

// GetTarget returns the target state
func (s *EntryPointState) GetTarget() core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.target
}

// Enter transfers control to the target state
func (s *EntryPointState) Enter(ctx *core.Context) error {
	if err := s.SimpleState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	target := s.target
	s.mutex.RUnlock()

	if target != nil {
		return target.Enter(ctx)
	}

	return nil
}

// ExitPointState represents an exit point from a composite state
type ExitPointState struct {
	*SimpleState
}

// NewExitPointState creates a new exit point state
func NewExitPointState(name string) *ExitPointState {
	return &ExitPointState{
		SimpleState: NewSimpleState(name),
	}
}
