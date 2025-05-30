package core

import (
	"time"
)

// DeferEventProcessing adds an event to the deferred list
func (ed *EventDeferrer) DeferEvent(event *Event, ctx *Context) {
	deferredEvent := &DeferredEvent{
		Event:    event,
		Context:  ctx,
		Deferred: time.Now(),
	}

	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	ed.events = append(ed.events, deferredEvent)
}

// ProcessDeferredEvents processes all deferred events
func (ed *EventDeferrer) ProcessDeferredEvents(sm *StateMachine) error {
	ed.mutex.Lock()
	deferredEvents := make([]*DeferredEvent, len(ed.events))
	copy(deferredEvents, ed.events)
	ed.events = make([]*DeferredEvent, 0)
	ed.mutex.Unlock()

	for _, deferred := range deferredEvents {
		if err := sm.SendEvent(deferred.Event); err != nil {
			return err
		}
	}

	return nil
}

// GetDeferredEventCount returns the number of deferred events
func (ed *EventDeferrer) GetDeferredEventCount() int {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()
	return len(ed.events)
}

// ClearDeferredEvents removes all deferred events
func (ed *EventDeferrer) ClearDeferredEvents() {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	ed.events = make([]*DeferredEvent, 0)
}

// GetDeferredEvents returns a copy of all deferred events
func (ed *EventDeferrer) GetDeferredEvents() []*DeferredEvent {
	ed.mutex.RLock()
	defer ed.mutex.RUnlock()

	result := make([]*DeferredEvent, len(ed.events))
	copy(result, ed.events)
	return result
}
