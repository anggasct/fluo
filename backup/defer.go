package flux

import (
	"container/list"
	"sync"
	"time"
)

// DeferredEvent represents an event that is deferred for later processing
type DeferredEvent struct {
	Event     *Event
	Timestamp time.Time
	Context   *Context
}

// NewDeferredEvent creates a new deferred event
func NewDeferredEvent(event *Event, ctx *Context) *DeferredEvent {
	return &DeferredEvent{
		Event:     event,
		Timestamp: time.Now(),
		Context:   ctx,
	}
}

// EventDeferrer manages deferred events for states
type EventDeferrer struct {
	deferredEvents *list.List
	deferCondition func(event *Event, state State) bool
	mutex          sync.RWMutex
}

// NewEventDeferrer creates a new event deferrer
func NewEventDeferrer() *EventDeferrer {
	return &EventDeferrer{
		deferredEvents: list.New(),
	}
}

// SetDeferCondition sets the condition for deferring events
func (ed *EventDeferrer) SetDeferCondition(condition func(event *Event, state State) bool) {
	ed.deferCondition = condition
}

// ShouldDefer checks if an event should be deferred
func (ed *EventDeferrer) ShouldDefer(event *Event, state State) bool {
	if ed.deferCondition == nil {
		return false
	}
	return ed.deferCondition(event, state)
}

// DeferEvent adds an event to the deferred list
func (ed *EventDeferrer) DeferEvent(event *Event, ctx *Context) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	deferredEvent := NewDeferredEvent(event, ctx)
	ed.deferredEvents.PushBack(deferredEvent)
}

// ProcessDeferredEvents processes all deferred events
func (ed *EventDeferrer) ProcessDeferredEvents(sm *StateMachine) error {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()

	for element := ed.deferredEvents.Front(); element != nil; {
		next := element.Next()
		deferredEvent := element.Value.(*DeferredEvent)

		currentState := sm.CurrentState()
		if !ed.ShouldDefer(deferredEvent.Event, currentState) {
			if err := sm.SendEvent(deferredEvent.Event); err != nil {
				return err
			}
			ed.deferredEvents.Remove(element)
		}

		element = next
	}

	return nil
}

// GetDeferredEventCount returns the number of deferred events
func (ed *EventDeferrer) GetDeferredEventCount() int {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()
	return ed.deferredEvents.Len()
}

// ClearDeferredEvents removes all deferred events
func (ed *EventDeferrer) ClearDeferredEvents() {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	ed.deferredEvents.Init()
}

// GetDeferredEvents returns a copy of all deferred events
func (ed *EventDeferrer) GetDeferredEvents() []*DeferredEvent {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()

	events := make([]*DeferredEvent, 0, ed.deferredEvents.Len())
	for element := ed.deferredEvents.Front(); element != nil; element = element.Next() {
		events = append(events, element.Value.(*DeferredEvent))
	}
	return events
}

// DeferState is a state that can defer specific events
type DeferState struct {
	*BaseState
	deferrer           *EventDeferrer
	deferredEventTypes map[string]bool
}

// NewDeferState creates a new defer state
func NewDeferState(name string) *DeferState {
	state := &DeferState{
		BaseState:          NewBaseState(name),
		deferrer:           NewEventDeferrer(),
		deferredEventTypes: make(map[string]bool),
	}

	state.deferrer.SetDeferCondition(func(event *Event, currentState State) bool {
		if deferState, ok := currentState.(*DeferState); ok {
			return deferState.shouldDeferEvent(event)
		}
		return false
	})

	return state
}

// AddDeferredEventType adds an event type to be deferred
func (s *DeferState) AddDeferredEventType(eventType string) *DeferState {
	s.deferredEventTypes[eventType] = true
	return s
}

// RemoveDeferredEventType removes an event type from deferring
func (s *DeferState) RemoveDeferredEventType(eventType string) *DeferState {
	delete(s.deferredEventTypes, eventType)
	return s
}

// shouldDeferEvent checks if the event should be deferred
func (s *DeferState) shouldDeferEvent(event *Event) bool {
	return s.deferredEventTypes[event.Name]
}

// HandleEvent processes events, deferring some if needed
func (s *DeferState) HandleEvent(event *Event, ctx *Context) (State, error) {
	if s.shouldDeferEvent(event) {
		s.deferrer.DeferEvent(event, ctx)
		return nil, nil
	}

	return s.BaseState.HandleEvent(event, ctx)
}

// Exit processes any deferred events when leaving the state
func (s *DeferState) Exit(ctx *Context) error {
	if ctx.StateMachine != nil {
		if err := s.deferrer.ProcessDeferredEvents(ctx.StateMachine); err != nil {
			return err
		}
	}
	return s.BaseState.Exit(ctx)
}

// GetDeferrer returns the event deferrer
func (s *DeferState) GetDeferrer() *EventDeferrer {
	return s.deferrer
}

// TimeoutState represents a state that transitions after a timeout
type TimeoutState struct {
	*BaseState
	timeout     time.Duration
	targetState State
	timer       *time.Timer
	mutex       sync.Mutex
}

// NewTimeoutState creates a new timeout state
func NewTimeoutState(name string, timeout time.Duration) *TimeoutState {
	return &TimeoutState{
		BaseState: NewBaseState(name),
		timeout:   timeout,
	}
}

// SetTargetState sets the state to transition to after timeout
func (s *TimeoutState) SetTargetState(target State) *TimeoutState {
	s.targetState = target
	return s
}

// Enter starts the timeout timer
func (s *TimeoutState) Enter(ctx *Context) error {
	if err := s.BaseState.Enter(ctx); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.timer != nil {
		s.timer.Stop()
	}

	s.timer = time.AfterFunc(s.timeout, func() {
		if s.targetState != nil && s.IsActive() {
			s.Exit(ctx)
			s.targetState.Enter(ctx)
		}
	})

	return nil
}

// Exit stops the timeout timer
func (s *TimeoutState) Exit(ctx *Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	return s.BaseState.Exit(ctx)
}

// SetTimeout updates the timeout duration
func (s *TimeoutState) SetTimeout(timeout time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.timeout = timeout
}
