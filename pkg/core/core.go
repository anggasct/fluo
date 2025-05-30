// Package core provides the central types and interfaces for the Fluo state machine library.
package core

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventPriority defines the priority level of an event
type EventPriority int

const (
	// LowPriority events are processed last when multiple events are pending
	LowPriority EventPriority = iota

	// NormalPriority is the default priority for events
	NormalPriority

	// HighPriority events are processed before normal and low priority events
	HighPriority

	// CriticalPriority events are processed immediately, even interrupting current actions
	CriticalPriority
)

// EventFilter is a function that evaluates whether an event should be processed
type EventFilter func(*Event) bool

// EventHandler defines a function that handles events
type EventHandler func(*Context, *Event) error

// Event represents a state machine event with optional data and metadata
type Event struct {
	Name      string
	Data      interface{}
	Timestamp time.Time
	ID        string
	Priority  EventPriority
	Metadata  map[string]interface{}
}

// NewEvent creates a new event with the given name
func NewEvent(name string) *Event {
	return &Event{
		Name:      name,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Priority:  NormalPriority,
		Metadata:  make(map[string]interface{}),
	}
}

// NewEventWithData creates a new event with name and data
func NewEventWithData(name string, data interface{}) *Event {
	return &Event{
		Name:      name,
		Data:      data,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Priority:  NormalPriority,
		Metadata:  make(map[string]interface{}),
	}
}

// WithPriority sets the priority of the event and returns the event
func (e *Event) WithPriority(priority EventPriority) *Event {
	e.Priority = priority
	return e
}

// WithMetadata adds metadata to the event and returns the event
func (e *Event) WithMetadata(key string, value interface{}) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// GetMetadata retrieves metadata from the event
func (e *Event) GetMetadata(key string) interface{} {
	if e.Metadata == nil {
		return nil
	}
	return e.Metadata[key]
}

// HasMetadata checks if the event has a specific metadata key
func (e *Event) HasMetadata(key string) bool {
	if e.Metadata == nil {
		return false
	}
	_, exists := e.Metadata[key]
	return exists
}

// Clone creates a deep copy of the event
func (e *Event) Clone() *Event {
	metadata := make(map[string]interface{})
	if e.Metadata != nil {
		for k, v := range e.Metadata {
			metadata[k] = v
		}
	}

	return &Event{
		Name:      e.Name,
		Data:      e.Data, // Note: this doesn't deep-copy the data
		Timestamp: e.Timestamp,
		ID:        e.ID,
		Priority:  e.Priority,
		Metadata:  metadata,
	}
}

// Context holds the execution context for state machine operations
type Context struct {
	context.Context
	StateMachine *StateMachine
	Event        *Event
	Data         map[string]interface{}
	mutex        sync.RWMutex
}

// NewContext creates a new context for state machine operations
func NewContext(ctx context.Context, sm *StateMachine) *Context {
	return &Context{
		Context:      ctx,
		StateMachine: sm,
		Data:         make(map[string]interface{}),
	}
}

// GetEvent returns the current event in the context
func (c *Context) GetEvent() *Event {
	return c.Event
}

// SetEvent sets the current event in the context
func (c *Context) SetEvent(event *Event) {
	c.Event = event
}

// Set stores a value in the context data
func (c *Context) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Data[key] = value
}

// Get retrieves a value from the context data
func (c *Context) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	value, exists := c.Data[key]
	return value, exists
}

// GetData retrieves a value from context data with string key
func (c *Context) GetData(key string) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	value, _ := c.Data[key]
	return value
}

// SetData sets a value in context data with string key
func (c *Context) SetData(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Data[key] = value
}

// Clone creates a new context that shares the same StateMachine but with a new data map
func (c *Context) Clone() *Context {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	newCtx := NewContext(c.Context, c.StateMachine)

	// Copy all data from the original context
	for key, value := range c.Data {
		newCtx.SetData(key, value)
	}

	// Copy the current event if present
	if c.Event != nil {
		newCtx.Event = c.Event.Clone()
	}

	return newCtx
}

// GuardCondition evaluates whether a transition should be taken
type GuardCondition func(ctx *Context) bool

// Action performs an operation during state transitions or state activities
type Action func(ctx *Context) error

// State represents a state in the state machine
type State interface {
	Name() string
	Enter(ctx *Context) error
	Exit(ctx *Context) error
	HandleEvent(event *Event, ctx *Context) (State, error)
	IsComposite() bool
	IsParallel() bool
	IsHistory() bool
	GetParent() State
	SetParent(parent State)
}

// Transition represents a transition between states
type Transition struct {
	From     State
	To       State
	Event    string
	Guard    GuardCondition
	Action   Action
	Priority int
}

// NewTransition creates a new transition
func NewTransition(from, to State, event string) *Transition {
	return &Transition{
		From:  from,
		To:    to,
		Event: event,
	}
}

// WithGuard adds a guard condition to the transition
func (t *Transition) WithGuard(guard GuardCondition) *Transition {
	t.Guard = guard
	return t
}

// WithAction adds an action to the transition
func (t *Transition) WithAction(action Action) *Transition {
	t.Action = action
	return t
}

// WithPriority sets the transition priority
func (t *Transition) WithPriority(priority int) *Transition {
	t.Priority = priority
	return t
}

// CanExecute checks if the transition can be executed
func (t *Transition) CanExecute(ctx *Context) bool {
	if t.Guard == nil {
		return true
	}
	return t.Guard(ctx)
}

// Execute performs the transition action
func (t *Transition) Execute(ctx *Context) error {
	if t.Action == nil {
		return nil
	}
	return t.Action(ctx)
}

// StateMachineState represents the current state of the state machine
type StateMachineState int

const (
	// StateStopped indicates the state machine is stopped
	StateStopped StateMachineState = iota
	// StateRunning indicates the state machine is running
	StateRunning
	// StateCompleted indicates the state machine has completed
	StateCompleted
	// StateError indicates the state machine encountered an error
	StateError
)

// StateMachineObserver observes state machine events
type StateMachineObserver interface {
	OnStateEnter(sm *StateMachine, state State)
	OnStateExit(sm *StateMachine, state State)
	OnTransition(sm *StateMachine, from, to State, event *Event)
	OnEventProcessed(sm *StateMachine, event *Event)
	OnError(sm *StateMachine, err error)
}

// DeferredEvent represents an event that has been deferred
type DeferredEvent struct {
	Event    *Event
	Context  *Context
	Deferred time.Time
}

// EventDeferrer manages deferred events
type EventDeferrer struct {
	events []*DeferredEvent
	mutex  sync.RWMutex
}

// NewEventDeferrer creates a new event deferrer
func NewEventDeferrer() *EventDeferrer {
	return &EventDeferrer{
		events: make([]*DeferredEvent, 0),
	}
}
