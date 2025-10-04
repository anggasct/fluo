package fluo

// Transition represents a state transition
type Transition struct {
	SourceState string
	TargetState string
	EventName   string
	Guard       GuardFunc
	Action      ActionFunc
}

// NewTransition creates a new transition
func NewTransition(sourceState, targetState, eventName string) *Transition {
	return &Transition{
		SourceState: sourceState,
		TargetState: targetState,
		EventName:   eventName,
	}
}

// WithGuard adds a guard condition to the transition
func (t *Transition) WithGuard(guard GuardFunc) *Transition {
	t.Guard = guard
	return t
}

// WithAction adds an action to the transition
func (t *Transition) WithAction(action ActionFunc) *Transition {
	t.Action = action
	return t
}
