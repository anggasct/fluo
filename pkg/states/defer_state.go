package states

import (
	"time"

	"github.com/anggasct/fluo/pkg/core"
)

// DeferState is a state that can defer specific events
type DeferState struct {
	*BaseState
	deferrer           *core.EventDeferrer
	deferredEventTypes map[string]bool
}

// NewDeferState creates a new defer state
func NewDeferState(name string) *DeferState {
	return &DeferState{
		BaseState:          NewBaseState(name),
		deferrer:           core.NewEventDeferrer(),
		deferredEventTypes: make(map[string]bool),
	}
}

// AddDeferredEventType adds an event type to be deferred
func (s *DeferState) AddDeferredEventType(eventType string) *DeferState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.deferredEventTypes[eventType] = true
	return s
}

// RemoveDeferredEventType removes an event type from deferring
func (s *DeferState) RemoveDeferredEventType(eventType string) *DeferState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.deferredEventTypes, eventType)
	return s
}

// shouldDeferEvent checks if the event should be deferred
func (s *DeferState) shouldDeferEvent(event *core.Event) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.deferredEventTypes[event.Name]
}

// HandleEvent processes events, deferring some if needed
func (s *DeferState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	// Check if this event should be deferred
	if s.shouldDeferEvent(event) {
		s.deferrer.DeferEvent(event, ctx)
		return nil, nil
	}

	// Let base state handle other events
	return s.BaseState.HandleEvent(event, ctx)
}

// Exit processes any deferred events when leaving the state
func (s *DeferState) Exit(ctx *core.Context) error {
	// Process deferred events
	err := s.deferrer.ProcessDeferredEvents(ctx.StateMachine)
	if err != nil {
		return err
	}

	// Call base exit
	return s.BaseState.Exit(ctx)
}

// GetDeferrer returns the event deferrer
func (s *DeferState) GetDeferrer() *core.EventDeferrer {
	return s.deferrer
}

// TimeoutState represents a state that transitions after a timeout
type TimeoutState struct {
	*BaseState
	timeout      time.Duration
	targetState  core.State
	timeoutEvent string
}

// NewTimeoutState creates a new timeout state
func NewTimeoutState(name string, timeout time.Duration) *TimeoutState {
	return &TimeoutState{
		BaseState:    NewBaseState(name),
		timeout:      timeout,
		timeoutEvent: "TIMEOUT",
	}
}

// SetTargetState sets the state to transition to after timeout
func (s *TimeoutState) SetTargetState(state core.State) *TimeoutState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.targetState = state
	return s
}

// SetTimeoutEvent sets the event to fire on timeout
func (s *TimeoutState) SetTimeoutEvent(event string) *TimeoutState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.timeoutEvent = event
	return s
}

// GetTimeout returns the timeout duration
func (s *TimeoutState) GetTimeout() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.timeout
}

// Enter schedules the timeout and enters the state
func (s *TimeoutState) Enter(ctx *core.Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.RLock()
	timeout := s.timeout
	timeoutEvent := s.timeoutEvent
	s.mutex.RUnlock()

	// Schedule timeout in a goroutine
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Fire timeout event
			ctx.StateMachine.SendEvent(core.NewEvent(timeoutEvent))
		case <-ctx.Done():
			// Context canceled
			return
		}
	}()

	return nil
}

// HandleEvent handles events, including the timeout event
func (s *TimeoutState) HandleEvent(event *core.Event, ctx *core.Context) (core.State, error) {
	s.mutex.RLock()
	timeoutEvent := s.timeoutEvent
	targetState := s.targetState
	s.mutex.RUnlock()

	// Check if this is our timeout event
	if event.Name == timeoutEvent && targetState != nil {
		return targetState, nil
	}

	// Handle other events normally
	return s.BaseState.HandleEvent(event, ctx)
}
