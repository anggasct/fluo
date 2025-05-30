package states

import (
	"context"
	"fmt"
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// ParallelRegion represents an independent region within a parallel state
type ParallelRegion struct {
	name          string
	stateMachine  *core.StateMachine
	isActive      bool
	isCompleted   bool
	eventHandlers map[string]core.Action
	mutex         sync.RWMutex
}

// NewParallelRegion creates a new parallel region
func NewParallelRegion(name string, stateMachine *core.StateMachine) *ParallelRegion {
	return &ParallelRegion{
		name:          name,
		stateMachine:  stateMachine,
		eventHandlers: make(map[string]core.Action),
	}
}

// Name returns the region name
func (r *ParallelRegion) Name() string {
	return r.name
}

// GetStateMachine returns the region's state machine
func (r *ParallelRegion) GetStateMachine() *core.StateMachine {
	return r.stateMachine
}

// Start starts the region
func (r *ParallelRegion) Start(ctx *core.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.isActive = true
	r.isCompleted = false

	if r.stateMachine != nil {
		return r.stateMachine.Start(ctx)
	}

	return nil
}

// Stop stops the region
func (r *ParallelRegion) Stop(ctx context.Context) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.isActive = false

	if r.stateMachine != nil {
		r.stateMachine.Stop(ctx)
	}
}

// IsActive returns whether the region is active
func (r *ParallelRegion) IsActive() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isActive
}

// SetCompleted marks the region as completed
func (r *ParallelRegion) SetCompleted(completed bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.isCompleted = completed
}

// AddEventHandler adds an event handler for the region
func (r *ParallelRegion) AddEventHandler(eventName string, handler core.Action) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.eventHandlers[eventName] = handler
}

// HandleEvent handles an event in the region
func (r *ParallelRegion) HandleEvent(event *core.Event, ctx *core.Context) error {
	r.mutex.RLock()
	handler := r.eventHandlers[event.Name]
	r.mutex.RUnlock()

	if handler != nil {
		return handler(ctx)
	}

	return nil
}

// SendEvent sends an event to the region's state machine
func (r *ParallelRegion) SendEvent(event *core.Event) error {
	r.mutex.RLock()
	stateMachine := r.stateMachine
	isActive := r.isActive
	r.mutex.RUnlock()

	if !isActive {
		return fmt.Errorf("region %s is not active", r.name)
	}

	if stateMachine != nil {
		return stateMachine.SendEvent(event)
	}

	return nil
}

// IsCompleted returns whether the region has completed
func (r *ParallelRegion) IsCompleted() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Check completion flag first
	if r.isCompleted {
		return true
	}

	// If we have a state machine, check if it's completed
	if r.stateMachine != nil {
		return r.stateMachine.IsCompleted()
	}

	return false
}

// ParallelState manages multiple independent state machine regions
type ParallelState struct {
	*BaseState
	regions   map[string]*ParallelRegion
	joinState core.State
	mutex     sync.RWMutex
}

// NewParallelState creates a new parallel state
func NewParallelState(name string) *ParallelState {
	return &ParallelState{
		BaseState: NewBaseState(name),
		regions:   make(map[string]*ParallelRegion),
	}
}

// IsParallel returns true for parallel states
func (s *ParallelState) IsParallel() bool {
	return true
}

// AddRegion adds an independent state machine region
func (s *ParallelState) AddRegion(region *ParallelRegion) *ParallelState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.regions[region.Name()] = region
	return s
}

// GetRegion returns a region by name
func (s *ParallelState) GetRegion(name string) *ParallelRegion {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.regions[name]
}

// SetJoinState sets the state to transition to when all regions complete
func (s *ParallelState) SetJoinState(state core.State) *ParallelState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.joinState = state
	return s
}

// Enter starts all regions
func (s *ParallelState) Enter(ctx *core.Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	regions := make([]*ParallelRegion, 0, len(s.regions))
	for _, region := range s.regions {
		regions = append(regions, region)
	}
	s.mutex.RUnlock()

	// Start all regions
	for _, region := range regions {
		// Create a context clone for each region to ensure proper isolation
		regionCtx := ctx.Clone()
		if err := region.Start(regionCtx); err != nil {
			return err
		}
	}

	return nil
}

// Exit stops all regions
func (s *ParallelState) Exit(ctx *core.Context) error {
	s.mutex.RLock()
	regions := make([]*ParallelRegion, 0, len(s.regions))
	for _, region := range s.regions {
		regions = append(regions, region)
	}
	s.mutex.RUnlock()

	// Stop all regions
	for _, region := range regions {
		region.Stop(ctx.Context)
	}

	return s.BaseState.Exit(ctx)
}

// HandleEvent dispatches events to all regions
func (s *ParallelState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	s.mutex.RLock()
	regions := make([]*ParallelRegion, 0, len(s.regions))
	joinState := s.joinState
	for _, region := range s.regions {
		regions = append(regions, region)
	}
	s.mutex.RUnlock()

	// Distribute event to all regions
	for _, region := range regions {
		if region.IsActive() {
			if err := region.HandleEvent(event, ctx); err != nil {
				return nil, err
			}

			if err := region.SendEvent(event); err != nil {
				// Non-critical error, just log it
				// log.Printf("Error sending event to region %s: %v", region.Name(), err)
			}
		}
	}

	// Check if all regions are completed
	allCompleted := true
	for _, region := range regions {
		if !region.IsCompleted() {
			allCompleted = false
			break
		}
	}

	// If all regions are completed and we have a join state, transition to it
	if allCompleted && joinState != nil {
		return joinState, nil
	}

	return nil, nil
}

// AreAllRegionsCompleted checks if all regions are completed
func (s *ParallelState) AreAllRegionsCompleted() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, region := range s.regions {
		if !region.IsCompleted() {
			return false
		}
	}

	return true
}

// GetAllRegions returns all regions in this parallel state
func (s *ParallelState) GetAllRegions() map[string]*ParallelRegion {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Create a copy to avoid concurrent map access issues
	regions := make(map[string]*ParallelRegion)
	for name, region := range s.regions {
		regions[name] = region
	}
	return regions
}
