package fluo

import (
	"context"
	"sync"
)

// Context provides access to data and information during state machine execution
type Context interface {
	context.Context

	Get(key string) (any, bool)
	Set(key string, value any)
	GetAll() map[string]any

	GetMachine() Machine
	GetCurrentState() string
	GetSourceState() string
	GetTargetState() string

	GetCurrentEvent() Event
	GetEventName() string
	GetEventData() any
	GetEventDataAs(target any) bool

	GetPreviousState() string

	WithValue(key string, value any) Context
	Fork() Context
}

// StateMachineContext implements the Context interface
type StateMachineContext struct {
	context.Context
	data          map[string]any
	machine       Machine
	currentState  string
	sourceState   string
	targetState   string
	currentEvent  Event
	previousState string

	mutex sync.RWMutex
}

// NewContext creates a new state machine context
func NewContext(parent context.Context, machine Machine) Context {
	return &StateMachineContext{
		Context:       parent,
		data:          make(map[string]any),
		machine:       machine,
		currentState:  "",
		sourceState:   "",
		targetState:   "",
		currentEvent:  nil,
		previousState: "",
	}
}

// NewSimpleContext creates a simple context for testing
func NewSimpleContext() Context {
	return &StateMachineContext{
		Context:       context.Background(),
		data:          make(map[string]any),
		machine:       nil,
		currentState:  "",
		sourceState:   "",
		targetState:   "",
		currentEvent:  nil,
		previousState: "",
	}
}

// Get retrieves a value from the context
func (ctx *StateMachineContext) Get(key string) (any, bool) {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	value, exists := ctx.data[key]
	return value, exists
}

// Set stores a value in the context
func (ctx *StateMachineContext) Set(key string, value any) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.data[key] = value
}

// GetAll returns all context data
func (ctx *StateMachineContext) GetAll() map[string]any {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	result := make(map[string]any)
	for k, v := range ctx.data {
		result[k] = v
	}
	return result
}

// GetMachine returns the associated state machine
func (ctx *StateMachineContext) GetMachine() Machine {
	return ctx.machine
}

// GetCurrentState returns the current state ID
func (ctx *StateMachineContext) GetCurrentState() string {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.currentState
}

// GetSourceState returns the source state of the current transition
func (ctx *StateMachineContext) GetSourceState() string {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.sourceState
}

// GetTargetState returns the target state of the current transition
func (ctx *StateMachineContext) GetTargetState() string {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.targetState
}

// GetCurrentEvent returns the current event being processed
func (ctx *StateMachineContext) GetCurrentEvent() Event {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.currentEvent
}

// GetEventName returns the name of the current event
func (ctx *StateMachineContext) GetEventName() string {
	if ctx.currentEvent != nil {
		return ctx.currentEvent.GetName()
	}
	return ""
}

// GetEventData returns the data of the current event
func (ctx *StateMachineContext) GetEventData() any {
	if ctx.currentEvent != nil {
		return ctx.currentEvent.GetData()
	}
	return nil
}

// GetEventDataAs attempts to cast event data to the target type
func (ctx *StateMachineContext) GetEventDataAs(target any) bool {
	data := ctx.GetEventData()
	if data == nil {
		return false
	}

	// Use type assertion for common types only
	switch t := target.(type) {
	case *string:
		if str, ok := data.(string); ok {
			*t = str
			return true
		}
	case *int:
		if i, ok := data.(int); ok {
			*t = i
			return true
		}
	case *bool:
		if b, ok := data.(bool); ok {
			*t = b
			return true
		}
	case *float64:
		if f, ok := data.(float64); ok {
			*t = f
			return true
		}
	}

	return false
}

// WithValue creates a new context with an additional key-value pair
func (ctx *StateMachineContext) WithValue(key string, value any) Context {
	newCtx := &StateMachineContext{
		Context:       ctx.Context,
		data:          make(map[string]any),
		machine:       ctx.machine,
		currentState:  ctx.currentState,
		sourceState:   ctx.sourceState,
		targetState:   ctx.targetState,
		currentEvent:  ctx.currentEvent,
		previousState: ctx.previousState,
	}

	// Copy existing data
	ctx.mutex.RLock()
	for k, v := range ctx.data {
		newCtx.data[k] = v
	}
	ctx.mutex.RUnlock()

	// Add new value
	newCtx.data[key] = value

	return newCtx
}

// Fork creates a new context with copied data
func (ctx *StateMachineContext) Fork() Context {
	newCtx := &StateMachineContext{
		Context:       ctx.Context,
		data:          make(map[string]any),
		machine:       ctx.machine,
		currentState:  ctx.currentState,
		sourceState:   ctx.sourceState,
		targetState:   ctx.targetState,
		currentEvent:  ctx.currentEvent,
		previousState: ctx.previousState,
	}

	// Copy existing data
	ctx.mutex.RLock()
	for k, v := range ctx.data {
		newCtx.data[k] = v
	}
	ctx.mutex.RUnlock()

	return newCtx
}

// GetPreviousState returns the previous state
func (ctx *StateMachineContext) GetPreviousState() string {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.previousState
}

// Internal methods for state machine management

// updateTransitionInfo updates the transition-related information in the context
func (ctx *StateMachineContext) updateTransitionInfo(currentState, sourceState, targetState string, event Event) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.currentState = currentState
	ctx.sourceState = sourceState
	ctx.targetState = targetState
	ctx.currentEvent = event
	// Update previous state
	if sourceState != "" && sourceState != ctx.previousState {
		ctx.previousState = sourceState
	}
}

// updateCurrentState updates only the current state
func (ctx *StateMachineContext) updateCurrentState(state string) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.currentState = state
}

// updateCurrentEvent updates only the current event
func (ctx *StateMachineContext) updateCurrentEvent(event Event) {
	ctx.mutex.Lock()
	defer ctx.mutex.Unlock()
	ctx.currentEvent = event
}
