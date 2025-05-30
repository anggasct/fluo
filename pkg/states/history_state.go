package states

import (
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// HistoryType represents the type of history state (shallow or deep)
type HistoryType int

const (
	// ShallowHistory remembers only the direct substate that was active
	ShallowHistory HistoryType = iota
	// DeepHistory remembers the full path to the deepest active substate
	DeepHistory
)

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

// HistoryState represents a pseudo-state that remembers the previously active state
// in a composite state
type HistoryState struct {
	*BaseState
	historyType  HistoryType
	defaultState core.State
	lastState    core.State
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
func (s *HistoryState) SetDefaultState(state core.State) *HistoryState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.defaultState = state
	return s
}

// GetDefaultState returns the default state
func (s *HistoryState) GetDefaultState() core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.defaultState
}

// GetHistoryType returns the history type
func (s *HistoryState) GetHistoryType() HistoryType {
	return s.historyType
}

// RecordState records a state in history
func (s *HistoryState) RecordState(state core.State) {
	if state == nil {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastState = state
}

// GetLastState returns the last recorded state
func (s *HistoryState) GetLastState() core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.lastState
}

// Clear clears the history
func (s *HistoryState) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.lastState = nil
}

// Enter handles entry to history state by transitioning to the remembered state
func (s *HistoryState) Enter(ctx *core.Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	lastState := s.lastState
	defaultState := s.defaultState
	s.mutex.RUnlock()

	var targetState core.State

	// If there's no history, use default state
	if lastState == nil {
		targetState = defaultState
	} else {
		// Otherwise use last state
		targetState = lastState
	}

	// If no target state is available, we're done
	if targetState == nil {
		return nil
	}

	// If this is deep history, find the deepest active state
	if s.historyType == DeepHistory {
		for targetState.IsComposite() {
			if composite, ok := targetState.(*CompositeState); ok {
				if child := composite.GetCurrentChild(); child != nil {
					targetState = child
				} else {
					break
				}
			} else {
				break
			}
		}
	}

	// Enter the target state
	return targetState.Enter(ctx)
}

// Exit is a no-op for history states
func (s *HistoryState) Exit(ctx *core.Context) error {
	// No special exit behavior needed
	return s.BaseState.Exit(ctx)
}

// HandleEvent delegates to parent composite state
func (s *HistoryState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	// History states don't handle events directly
	return nil, nil
}
