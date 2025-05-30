package flux

import (
	"fmt"
	"sync"
)

// ChoiceState evaluates conditions and selects appropriate transition
type ChoiceState struct {
	*BaseState
	choices       []ChoiceTransition
	defaultTarget State
}

// ChoiceTransition represents a conditional transition option
type ChoiceTransition struct {
	condition GuardCondition
	target    State
}

// NewChoiceState creates a new choice state
func NewChoiceState(name string) *ChoiceState {
	return &ChoiceState{
		BaseState: NewBaseState(name),
		choices:   make([]ChoiceTransition, 0),
	}
}

// AddChoice adds a conditional transition option
func (s *ChoiceState) AddChoice(condition GuardCondition, target State) *ChoiceState {
	s.choices = append(s.choices, ChoiceTransition{
		condition: condition,
		target:    target,
	})
	return s
}

// SetDefaultTarget sets the default target if no conditions match
func (s *ChoiceState) SetDefaultTarget(target State) *ChoiceState {
	s.defaultTarget = target
	return s
}

// Enter evaluates conditions and transitions to appropriate state
func (s *ChoiceState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	for _, choice := range s.choices {
		if choice.condition(ctx) {
			if err := s.Exit(ctx); err != nil {
				return err
			}
			return choice.target.Enter(ctx)
		}
	}

	if s.defaultTarget != nil {
		if err := s.Exit(ctx); err != nil {
			return err
		}
		return s.defaultTarget.Enter(ctx)
	}

	return fmt.Errorf("no valid choice found in choice state %s", s.name)
}

// JunctionState provides static conditional branching
type JunctionState struct {
	*BaseState
	branches      map[string]State
	defaultBranch State
}

// NewJunctionState creates a new junction state
func NewJunctionState(name string) *JunctionState {
	return &JunctionState{
		BaseState: NewBaseState(name),
		branches:  make(map[string]State),
	}
}

// AddBranch adds a named branch to the junction
func (s *JunctionState) AddBranch(branchName string, target State) *JunctionState {
	s.branches[branchName] = target
	return s
}

// SetDefaultBranch sets the default branch
func (s *JunctionState) SetDefaultBranch(target State) *JunctionState {
	s.defaultBranch = target
	return s
}

// SelectBranch transitions to the specified branch
func (s *JunctionState) SelectBranch(branchName string, ctx *Context) error {
	if target, exists := s.branches[branchName]; exists {
		if err := s.Exit(ctx); err != nil {
			return err
		}
		return target.Enter(ctx)
	}

	if s.defaultBranch != nil {
		if err := s.Exit(ctx); err != nil {
			return err
		}
		return s.defaultBranch.Enter(ctx)
	}

	return fmt.Errorf("invalid branch %s in junction state %s", branchName, s.name)
}

// HandleEvent processes branch selection events
func (s *JunctionState) HandleEvent(event *Event, ctx *Context) (State, error) {
	if event.Name == "select_branch" {
		if branchName, ok := event.Data.(string); ok {
			err := s.SelectBranch(branchName, ctx)
			return nil, err
		}
	}
	return s.BaseState.HandleEvent(event, ctx)
}

// ForkState splits execution into multiple parallel regions
type ForkState struct {
	*BaseState
	targets []State
}

// NewForkState creates a new fork state
func NewForkState(name string) *ForkState {
	return &ForkState{
		BaseState: NewBaseState(name),
		targets:   make([]State, 0),
	}
}

// AddTarget adds a target state for parallel execution
func (s *ForkState) AddTarget(target State) *ForkState {
	s.targets = append(s.targets, target)
	return s
}

// Enter activates all target states in parallel
func (s *ForkState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(s.targets))

	for _, target := range s.targets {
		wg.Add(1)
		go func(state State) {
			defer wg.Done()
			if err := state.Enter(ctx); err != nil {
				errors <- err
			}
		}(target)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}

// JoinState waits for multiple parallel states to complete
type JoinState struct {
	*BaseState
	sources   []State
	target    State
	completed map[string]bool
	mutex     sync.Mutex
}

// NewJoinState creates a new join state
func NewJoinState(name string) *JoinState {
	return &JoinState{
		BaseState: NewBaseState(name),
		sources:   make([]State, 0),
		completed: make(map[string]bool),
	}
}

// AddSource adds a source state that must complete
func (s *JoinState) AddSource(source State) *JoinState {
	s.sources = append(s.sources, source)
	s.completed[source.Name()] = false
	return s
}

// SetTarget sets the target state after all sources complete
func (s *JoinState) SetTarget(target State) *JoinState {
	s.target = target
	return s
}

// MarkCompleted marks a source state as completed
func (s *JoinState) MarkCompleted(stateName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.completed[stateName] = true
}

// AreAllCompleted checks if all source states are completed
func (s *JoinState) AreAllCompleted() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, completed := range s.completed {
		if !completed {
			return false
		}
	}
	return true
}

// Enter waits for all sources to complete then transitions to target
func (s *JoinState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	if s.AreAllCompleted() && s.target != nil {
		if err := s.Exit(ctx); err != nil {
			return err
		}
		return s.target.Enter(ctx)
	}

	return nil
}
