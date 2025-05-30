package flux

import (
	"sync"
)

// ParallelRegion represents an independent state machine region
type ParallelRegion struct {
	name         string
	stateMachine *StateMachine
	isActive     bool
	mutex        sync.RWMutex
}

// NewParallelRegion creates a new parallel region
func NewParallelRegion(name string) *ParallelRegion {
	return &ParallelRegion{
		name:         name,
		stateMachine: NewStateMachine(name + "_sm"),
	}
}

// Name returns the region name
func (r *ParallelRegion) Name() string {
	return r.name
}

// GetStateMachine returns the region's state machine
func (r *ParallelRegion) GetStateMachine() *StateMachine {
	return r.stateMachine
}

// IsActive returns whether the region is active
func (r *ParallelRegion) IsActive() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.isActive
}

// SetActive sets the active status
func (r *ParallelRegion) SetActive(active bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.isActive = active
}

// Start activates the region
func (r *ParallelRegion) Start(ctx *Context) error {
	r.SetActive(true)
	return r.stateMachine.Start()
}

// Stop deactivates the region
func (r *ParallelRegion) Stop(ctx *Context) error {
	if err := r.stateMachine.Stop(); err != nil {
		return err
	}
	r.SetActive(false)
	return nil
}

// SendEvent sends an event to the region's state machine
func (r *ParallelRegion) SendEvent(event *Event) error {
	if r.IsActive() {
		return r.stateMachine.SendEvent(event)
	}
	return nil
}

// IsCompleted returns whether the region has completed
func (r *ParallelRegion) IsCompleted() bool {
	return r.stateMachine.IsCompleted()
}

// ParallelState manages multiple independent state machine regions
type ParallelState struct {
	*BaseState
	regions   map[string]*ParallelRegion
	joinState State
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
func (s *ParallelState) SetJoinState(state State) *ParallelState {
	s.joinState = state
	return s
}

// Enter starts all regions simultaneously
func (s *ParallelState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(s.regions))

	s.mutex.RLock()
	for _, region := range s.regions {
		wg.Add(1)
		go func(r *ParallelRegion) {
			defer wg.Done()
			if err := r.Start(ctx); err != nil {
				errors <- err
			}
		}(region)
	}
	s.mutex.RUnlock()

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return err
		}
	}

	go s.monitorCompletion(ctx)
	return nil
}

// Exit stops all regions
func (s *ParallelState) Exit(ctx *Context) error {
	s.mutex.RLock()
	var wg sync.WaitGroup
	errors := make(chan error, len(s.regions))

	for _, region := range s.regions {
		wg.Add(1)
		go func(r *ParallelRegion) {
			defer wg.Done()
			if err := r.Stop(ctx); err != nil {
				errors <- err
			}
		}(region)
	}
	s.mutex.RUnlock()

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return err
		}
	}

	return s.BaseState.Exit(ctx)
}

// HandleEvent sends event to all active regions
func (s *ParallelState) HandleEvent(event *Event, ctx *Context) (State, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, region := range s.regions {
		if err := region.SendEvent(event); err != nil {
			return nil, err
		}
	}

	return s.BaseState.HandleEvent(event, ctx)
}

// monitorCompletion monitors all regions for completion
func (s *ParallelState) monitorCompletion(ctx *Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if s.areAllRegionsCompleted() && s.joinState != nil {
				if err := s.Exit(ctx); err != nil {
					return
				}
				s.joinState.Enter(ctx)
				return
			}
		}
	}
}

// areAllRegionsCompleted checks if all regions have completed
func (s *ParallelState) areAllRegionsCompleted() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, region := range s.regions {
		if region.IsActive() && !region.IsCompleted() {
			return false
		}
	}
	return true
}

// GetActiveRegions returns all currently active regions
func (s *ParallelState) GetActiveRegions() []*ParallelRegion {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	active := make([]*ParallelRegion, 0)
	for _, region := range s.regions {
		if region.IsActive() {
			active = append(active, region)
		}
	}
	return active
}
