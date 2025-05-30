package builders

import (
	"github.com/anggasct/fluo/pkg/core"
	"github.com/anggasct/fluo/pkg/states"
)

// SubmachineBuilder helps build submachine states with fluent API
type SubmachineBuilder struct {
	state *states.SubmachineState
}

// NewSubmachineBuilder creates a new submachine builder
func NewSubmachineBuilder(id string) *SubmachineBuilder {
	return &SubmachineBuilder{
		state: states.NewSubmachineState(id),
	}
}

// WithSubmachine sets the embedded state machine
func (b *SubmachineBuilder) WithSubmachine(sm *core.StateMachine) *SubmachineBuilder {
	b.state.SetSubmachine(sm)
	return b
}

// WithEntryConnector sets the entry connector
func (b *SubmachineBuilder) WithEntryConnector(connector *states.SubmachineConnector) *SubmachineBuilder {
	b.state.SetEntryConnector(connector)
	return b
}

// WithExitConnector sets the exit connector
func (b *SubmachineBuilder) WithExitConnector(connector *states.SubmachineConnector) *SubmachineBuilder {
	b.state.SetExitConnector(connector)
	return b
}

// WithCompletionEvent sets the completion event
func (b *SubmachineBuilder) WithCompletionEvent(event string) *SubmachineBuilder {
	b.state.SetCompletionEvent(event)
	return b
}

// WithErrorEvent sets the error event
func (b *SubmachineBuilder) WithErrorEvent(event string) *SubmachineBuilder {
	b.state.SetErrorEvent(event)
	return b
}

// WithStateMapping adds a state mapping
func (b *SubmachineBuilder) WithStateMapping(submachineState, parentState string) *SubmachineBuilder {
	b.state.AddStateMapping(submachineState, parentState)
	return b
}

// WithEventForwarding enables/disables event forwarding
func (b *SubmachineBuilder) WithEventForwarding(enabled bool) *SubmachineBuilder {
	b.state.SetEventForwarding(enabled)
	return b
}

// WithEntryAction adds an entry action
func (b *SubmachineBuilder) WithEntryAction(action core.Action) *SubmachineBuilder {
	b.state.AddEntryAction(action)
	return b
}

// WithExitAction adds an exit action
func (b *SubmachineBuilder) WithExitAction(action core.Action) *SubmachineBuilder {
	b.state.AddExitAction(action)
	return b
}

// Build creates the submachine state
func (b *SubmachineBuilder) Build() *states.SubmachineState {
	return b.state
}
