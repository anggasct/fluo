package fluo

import (
	"time"
)

// Event represents a trigger for transitions in the state machine
type Event interface {
	GetName() string
	GetData() any
	GetTimestamp() time.Time
	GetMetadata() map[string]any
}

// BaseEvent provides a basic implementation of the Event interface
type BaseEvent struct {
	name      string
	data      any
	timestamp time.Time
	metadata  map[string]any
}

// NewEvent creates a new basic event
func NewEvent(name string, data any) Event {
	return &BaseEvent{
		name:      name,
		data:      data,
		timestamp: time.Now(),
		metadata:  make(map[string]any),
	}
}

// NewEventWithMetadata creates a new event with metadata
func NewEventWithMetadata(name string, data any, metadata map[string]any) Event {
	return &BaseEvent{
		name:      name,
		data:      data,
		timestamp: time.Now(),
		metadata:  metadata,
	}
}

// NewTypedEvent creates a new event with typed data
func NewTypedEvent(name string, data any) Event {
	return NewEvent(name, data)
}

// GetName returns the event name
func (e *BaseEvent) GetName() string {
	return e.name
}

// GetData returns the event data
func (e *BaseEvent) GetData() any {
	return e.data
}

// GetTimestamp returns the event timestamp
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.timestamp
}

// GetMetadata returns the event metadata
func (e *BaseEvent) GetMetadata() map[string]any {
	if e.metadata == nil {
		return make(map[string]any)
	}
	result := make(map[string]any)
	for k, v := range e.metadata {
		result[k] = v
	}
	return result
}

// EventResult represents the result of processing an event
type EventResult struct {
	Processed       bool
	StateChanged    bool
	PreviousState   string
	CurrentState    string
	Error           error
	RejectionReason string
}

// NewEventResult creates a new event result
func NewEventResult(processed, stateChanged bool, prevState, currentState string) *EventResult {
	return &EventResult{
		Processed:     processed,
		StateChanged:  stateChanged,
		PreviousState: prevState,
		CurrentState:  currentState,
	}
}

// WithError adds an error to the event result
func (r *EventResult) WithError(err error) *EventResult {
	r.Error = err
	return r
}

// WithRejection adds a rejection reason to the event result
func (r *EventResult) WithRejection(reason string) *EventResult {
	r.RejectionReason = reason
	r.Processed = false
	return r
}

// Success returns true if the event was processed successfully
func (r *EventResult) Success() bool {
	return r.Processed && r.Error == nil
}
