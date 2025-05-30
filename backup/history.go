package flux

import (
	"sync"
)

// HistoryType represents the type of history state (shallow or deep)
type HistoryType int

const (
	// ShallowHistory remembers only the direct substate that was active
	ShallowHistory HistoryType = iota
	// DeepHistory remembers the full path to the deepest active substate
	DeepHistory
)

// HistoryState represents a pseudo-state that remembers the previously active state
// in a composite state
type HistoryState struct {
	*BaseState
	historyType  HistoryType
	defaultState State
	lastState    State
	mutex        sync.RWMutex
}

// NewHistoryState creates a new history state with specified type
func NewHistoryState(name string, historyType HistoryType) *HistoryState {
	return &HistoryState{
		BaseState:   NewBaseState(name),
		historyType: historyType,
	}
}

// IsHistory returns true indicating this is a history state
func (s *HistoryState) IsHistory() bool {
	return true
}

// SetDefaultState sets the default target state used when no history exists
func (s *HistoryState) SetDefaultState(state State) *HistoryState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.defaultState = state
	return s
}

// GetDefaultState returns the default state
func (s *HistoryState) GetDefaultState() State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.defaultState
}

// SetLastState records the last active state
func (s *HistoryState) SetLastState(state State) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastState = state
}

// GetLastState returns the last recorded state
func (s *HistoryState) GetLastState() State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lastState
}

// Enter handles entry into the history state by transitioning to
// either the last active state or the default state
func (s *HistoryState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	targetState := s.lastState
	if targetState == nil {
		targetState = s.defaultState
	}
	histType := s.historyType
	s.mutex.RUnlock()

	if targetState == nil {
		return NewHistoryError("No target state available for history state",
			histType.String())
	}

	// Restore history is working

	// Find the parent of this history state (should be a composite state)
	parent := s.GetParent()
	if parent != nil {
		if composite, ok := parent.(*CompositeState); ok {
			// Set the parent as the current state in the state machine if we have a context
			if ctx != nil && ctx.StateMachine != nil {
				ctx.StateMachine.SetCurrentState(composite)
			}

			// Set and activate the target child directly
			composite.mutex.Lock()
			composite.currentChild = targetState
			composite.mutex.Unlock()

			// Now activate the target state
			if err := targetState.Enter(ctx); err != nil {
				return err
			}

			return nil
		}
	}

	// Fallback to direct entry if not in a composite state
	return targetState.Enter(ctx)
}

// String returns a string representation of the history type
func (h HistoryType) String() string {
	switch h {
	case ShallowHistory:
		return "SHALLOW"
	case DeepHistory:
		return "DEEP"
	default:
		return "UNKNOWN"
	}
}
