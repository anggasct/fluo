// Package flux provides a comprehensive finite state machine library for Go
// that supports advanced UML state machine concepts including parallel regions,
// hierarchical states, choice points, and event deferring.
package flux

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Event represents a state machine event with optional data
type Event struct {
	Name      string
	Data      interface{}
	Timestamp time.Time
	ID        string
}

// NewEvent creates a new event with the given name
func NewEvent(name string) *Event {
	return &Event{
		Name:      name,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
	}
}

// NewEventWithData creates a new event with name and data
func NewEventWithData(name string, data interface{}) *Event {
	return &Event{
		Name:      name,
		Data:      data,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
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
