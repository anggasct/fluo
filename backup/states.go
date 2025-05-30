package flux

import (
	"fmt"
	"sync"
)

// BaseState provides common functionality for all state types
type BaseState struct {
	name        string
	parent      State
	entryAction Action
	exitAction  Action
	doActivity  Action
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

// GetParent returns the parent state
func (s *BaseState) GetParent() State {
	return s.parent
}

// SetParent sets the parent state
func (s *BaseState) SetParent(parent State) {
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

// IsActive returns whether the state is currently active
func (s *BaseState) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isActive
}

// SetActive sets the active status of the state
func (s *BaseState) SetActive(active bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.isActive = active
}

// WithEntryAction sets the entry action for the state
func (s *BaseState) WithEntryAction(action Action) *BaseState {
	s.entryAction = action
	return s
}

// WithExitAction sets the exit action for the state
func (s *BaseState) WithExitAction(action Action) *BaseState {
	s.exitAction = action
	return s
}

// WithDoActivity sets the do activity for the state
func (s *BaseState) WithDoActivity(activity Action) *BaseState {
	s.doActivity = activity
	return s
}

// AddEntryAction adds an entry action (alias for WithEntryAction for compatibility)
func (s *BaseState) AddEntryAction(action Action) {
	s.entryAction = action
}

// AddExitAction adds an exit action (alias for WithExitAction for compatibility)
func (s *BaseState) AddExitAction(action Action) {
	s.exitAction = action
}

// Enter activates the state and executes entry action
func (s *BaseState) Enter(ctx *Context) error {
	s.SetActive(true)
	if s.entryAction != nil {
		return s.entryAction(ctx)
	}
	return nil
}

// Exit deactivates the state and executes exit action
func (s *BaseState) Exit(ctx *Context) error {
	if s.exitAction != nil {
		if err := s.exitAction(ctx); err != nil {
			return err
		}
	}
	s.SetActive(false)
	return nil
}

// HandleEvent processes an event (default implementation returns nil)
func (s *BaseState) HandleEvent(event *Event, ctx *Context) (State, error) {
	return nil, nil
}

// SimpleState represents a basic state without substates
type SimpleState struct {
	*BaseState
}

// NewSimpleState creates a new simple state
func NewSimpleState(name string) *SimpleState {
	return &SimpleState{
		BaseState: NewBaseState(name),
	}
}

// EntryPointState represents an entry point to a composite state
type EntryPointState struct {
	*BaseState
	target State
}

// NewEntryPointState creates a new entry point state
func NewEntryPointState(name string, target State) *EntryPointState {
	return &EntryPointState{
		BaseState: NewBaseState(name),
		target:    target,
	}
}

// Enter transitions to the target state
func (s *EntryPointState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}
	if s.target != nil {
		return s.target.Enter(ctx)
	}
	return nil
}

// ExitPointState represents an exit point from a composite state
type ExitPointState struct {
	*BaseState
	parentTarget State
}

// NewExitPointState creates a new exit point state
func NewExitPointState(name string) *ExitPointState {
	return &ExitPointState{
		BaseState: NewBaseState(name),
	}
}

// SetParentTarget sets the target state in the parent machine
func (s *ExitPointState) SetParentTarget(target State) {
	s.parentTarget = target
}

// Enter processes the exit from composite state
func (s *ExitPointState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}
	if s.parentTarget != nil {
		return s.parentTarget.Enter(ctx)
	}
	return nil
}

// FinalState represents a final state in the state machine
type FinalState struct {
	*BaseState
}

// NewFinalState creates a new final state
func NewFinalState(name string) *FinalState {
	return &FinalState{
		BaseState: NewBaseState(name),
	}
}

// Enter marks the state machine as completed
func (s *FinalState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}
	if ctx.StateMachine != nil {
		ctx.StateMachine.SetCompleted(true)
	}
	return nil
}

// String returns a string representation of the state
func (s *BaseState) String() string {
	return fmt.Sprintf("State(%s)", s.name)
}
