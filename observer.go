package fluo

import "fmt"

// Observer represents an entity that observes state machine lifecycle
type Observer interface {
	// Required methods

	// OnTransition is called when a state transition occurs
	OnTransition(from string, to string, event Event, ctx Context)

	// OnStateEnter is called when entering a new state
	OnStateEnter(state string, ctx Context)
}

// ExtendedObserver provides additional optional observation methods
type ExtendedObserver interface {
	Observer

	// OnStateExit is called when exiting a state
	OnStateExit(state string, ctx Context)

	// OnGuardEvaluation is called when a guard condition is evaluated
	OnGuardEvaluation(from string, to string, event Event, result bool, ctx Context)

	// OnEventRejected is called when an event is rejected (no valid transition)
	OnEventRejected(event Event, reason string, ctx Context)

	// OnError is called when an error occurs during processing
	OnError(err error, ctx Context)

	// OnActionExecution is called when an action is executed
	OnActionExecution(actionType string, state string, event Event, ctx Context)

	// OnMachineStarted is called when the state machine starts
	OnMachineStarted(ctx Context)

	// OnMachineStopped is called when the state machine stops
	OnMachineStopped(ctx Context)
}

// BaseObserver provides a default implementation with no-op methods
type BaseObserver struct{}

// OnTransition implements the required Observer method
func (o *BaseObserver) OnTransition(from string, to string, event Event, ctx Context) {
	// Default implementation - no operation
}

// OnStateEnter implements the required Observer method
func (o *BaseObserver) OnStateEnter(state string, ctx Context) {
	// Default implementation - no operation
}

// OnStateExit implements the optional ExtendedObserver method
func (o *BaseObserver) OnStateExit(state string, ctx Context) {
	// Default implementation - no operation
}

// OnGuardEvaluation implements the optional ExtendedObserver method
func (o *BaseObserver) OnGuardEvaluation(from string, to string, event Event, result bool, ctx Context) {
	// Default implementation - no operation
}

// OnEventRejected implements the optional ExtendedObserver method
func (o *BaseObserver) OnEventRejected(event Event, reason string, ctx Context) {
	// Default implementation - no operation
}

// OnError implements the optional ExtendedObserver method
func (o *BaseObserver) OnError(err error, ctx Context) {
	// Default implementation - no operation
}

// OnActionExecution implements the optional ExtendedObserver method
func (o *BaseObserver) OnActionExecution(actionType string, state string, event Event, ctx Context) {
	// Default implementation - no operation
}

// OnMachineStarted implements the optional ExtendedObserver method
func (o *BaseObserver) OnMachineStarted(ctx Context) {
	// Default implementation - no operation
}

// OnMachineStopped implements the optional ExtendedObserver method
func (o *BaseObserver) OnMachineStopped(ctx Context) {
	// Default implementation - no operation
}

// ObserverManager manages a collection of observers
type ObserverManager struct {
	observers []Observer
}

// NewObserverManager creates a new observer manager
func NewObserverManager() *ObserverManager {
	return &ObserverManager{
		observers: make([]Observer, 0),
	}
}

// AddObserver adds an observer to the manager
func (om *ObserverManager) AddObserver(observer Observer) {
	om.observers = append(om.observers, observer)
}

// RemoveObserver removes an observer from the manager
func (om *ObserverManager) RemoveObserver(observer Observer) {
	for i, obs := range om.observers {
		if obs == observer {
			om.observers = append(om.observers[:i], om.observers[i+1:]...)
			break
		}
	}
}

// NotifyTransition notifies all observers of a state transition
func (om *ObserverManager) NotifyTransition(from string, to string, event Event, ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Observer panicked - log it if there's an error observer but don't crash
					if extObs, ok := observer.(ExtendedObserver); ok {
						// Try to notify about the observer error, but catch any panic from that too
						func() {
							defer func() { recover() }()
							extObs.OnError(fmt.Errorf("observer panic in OnTransition: %v", r), ctx)
						}()
					}
				}
			}()
			observer.OnTransition(from, to, event, ctx)
		}()
	}
}

// NotifyStateEnter notifies all observers of state entry
func (om *ObserverManager) NotifyStateEnter(state string, ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if extObs, ok := observer.(ExtendedObserver); ok {
						func() {
							defer func() { recover() }()
							extObs.OnError(fmt.Errorf("observer panic in OnStateEnter: %v", r), ctx)
						}()
					}
				}
			}()
			observer.OnStateEnter(state, ctx)
		}()
	}
}

// NotifyStateExit notifies all observers of state exit
func (om *ObserverManager) NotifyStateExit(state string, ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		if extObs, ok := observer.(ExtendedObserver); ok {
			func() {
				defer func() {
					if r := recover(); r != nil {
						func() {
							defer func() { recover() }()
							extObs.OnError(fmt.Errorf("observer panic in OnStateExit: %v", r), ctx)
						}()
					}
				}()
				extObs.OnStateExit(state, ctx)
			}()
		}
	}
}

// NotifyGuardEvaluation notifies all observers of guard evaluation
func (om *ObserverManager) NotifyGuardEvaluation(from string, to string, event Event, result bool, ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		if extObs, ok := observer.(ExtendedObserver); ok {
			extObs.OnGuardEvaluation(from, to, event, result, ctx)
		}
	}
}

// NotifyEventRejected notifies all observers of event rejection
func (om *ObserverManager) NotifyEventRejected(event Event, reason string, ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		if extObs, ok := observer.(ExtendedObserver); ok {
			extObs.OnEventRejected(event, reason, ctx)
		}
	}
}

// NotifyError notifies all observers of errors
func (om *ObserverManager) NotifyError(err error, ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		if extObs, ok := observer.(ExtendedObserver); ok {
			extObs.OnError(err, ctx)
		}
	}
}

// NotifyActionExecution notifies all observers of action execution
func (om *ObserverManager) NotifyActionExecution(actionType string, state string, event Event, ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		if extObs, ok := observer.(ExtendedObserver); ok {
			extObs.OnActionExecution(actionType, state, event, ctx)
		}
	}
}

// NotifyMachineStarted notifies all observers that the machine has started
func (om *ObserverManager) NotifyMachineStarted(ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		if extObs, ok := observer.(ExtendedObserver); ok {
			extObs.OnMachineStarted(ctx)
		}
	}
}

// NotifyMachineStopped notifies all observers that the machine has stopped
func (om *ObserverManager) NotifyMachineStopped(ctx Context) {
	observers := make([]Observer, len(om.observers))
	copy(observers, om.observers)

	for _, observer := range observers {
		if extObs, ok := observer.(ExtendedObserver); ok {
			extObs.OnMachineStopped(ctx)
		}
	}
}
