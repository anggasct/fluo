package states

import (
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// ChoiceState represents a state that dynamically chooses between outgoing transitions
type ChoiceState struct {
	*BaseState
	choices []*ChoiceOption
	mutex   sync.RWMutex
}

// ChoiceOption represents an option in a choice state
type ChoiceOption struct {
	Guard    core.GuardCondition
	Target   core.State
	Priority int
}

// NewChoiceState creates a new choice state
func NewChoiceState(name string) *ChoiceState {
	return &ChoiceState{
		BaseState: NewBaseState(name),
		choices:   make([]*ChoiceOption, 0),
	}
}

// AddChoice adds a choice option
func (s *ChoiceState) AddChoice(guard core.GuardCondition, target core.State) *ChoiceState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.choices = append(s.choices, &ChoiceOption{
		Guard:  guard,
		Target: target,
	})

	return s
}

// AddChoiceWithPriority adds a choice option with priority
func (s *ChoiceState) AddChoiceWithPriority(guard core.GuardCondition, target core.State, priority int) *ChoiceState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.choices = append(s.choices, &ChoiceOption{
		Guard:    guard,
		Target:   target,
		Priority: priority,
	})

	return s
}

// AddElseChoice adds a default choice option with no guard
func (s *ChoiceState) AddElseChoice(target core.State) *ChoiceState {
	return s.AddChoice(func(ctx *core.Context) bool { return true }, target)
}

// Enter evaluates the choices and transitions to the first valid one
func (s *ChoiceState) Enter(ctx *core.Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	choices := make([]*ChoiceOption, len(s.choices))
	copy(choices, s.choices)
	s.mutex.RUnlock()

	// Sort by priority (higher first)
	for i := 0; i < len(choices)-1; i++ {
		for j := 0; j < len(choices)-i-1; j++ {
			if choices[j].Priority < choices[j+1].Priority {
				choices[j], choices[j+1] = choices[j+1], choices[j]
			}
		}
	}

	// Find the first choice that evaluates to true
	var targetState core.State
	for _, choice := range choices {
		if choice.Guard == nil || choice.Guard(ctx) {
			targetState = choice.Target
			break
		}
	}

	// If we found a target, enter it
	if targetState != nil {
		return targetState.Enter(ctx)
	}

	// If no target was found, we simply remain active
	// This is usually an error condition in UML state machines
	return nil
}

// HandleEvent delegates to the selected state
func (s *ChoiceState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	// Choice states don't handle events directly
	return nil, nil
}

// JunctionState represents a static junction pseudo-state
type JunctionState struct {
	*BaseState
	paths []*JunctionPath
	mutex sync.RWMutex
}

// JunctionPath represents a path through a junction
type JunctionPath struct {
	From  core.State
	To    core.State
	Guard core.GuardCondition
}

// NewJunctionState creates a new junction state
func NewJunctionState(name string) *JunctionState {
	return &JunctionState{
		BaseState: NewBaseState(name),
		paths:     make([]*JunctionPath, 0),
	}
}

// AddPath adds a path through the junction
func (s *JunctionState) AddPath(from, to core.State, guard core.GuardCondition) *JunctionState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.paths = append(s.paths, &JunctionPath{
		From:  from,
		To:    to,
		Guard: guard,
	})

	return s
}

// FindPath finds an applicable path through the junction
func (s *JunctionState) FindPath(from core.State, ctx *core.Context) core.State {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, path := range s.paths {
		if path.From == from && (path.Guard == nil || path.Guard(ctx)) {
			return path.To
		}
	}

	return nil
}
