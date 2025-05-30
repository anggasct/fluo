package observers

import (
	"sync"
	"time"

	"github.com/anggasct/fluo/pkg/core"
)

// MetricsObserver collects metrics about state machine execution
type MetricsObserver struct {
	stateVisits      map[string]int
	stateTimeSpent   map[string]time.Duration
	eventCounts      map[string]int
	transitionCounts map[string]int
	errorCount       int
	lastStateEntry   map[string]time.Time
	mutex            sync.RWMutex
}

// NewMetricsObserver creates a new metrics observer
func NewMetricsObserver() *MetricsObserver {
	return &MetricsObserver{
		stateVisits:      make(map[string]int),
		stateTimeSpent:   make(map[string]time.Duration),
		eventCounts:      make(map[string]int),
		transitionCounts: make(map[string]int),
		lastStateEntry:   make(map[string]time.Time),
	}
}

// OnStateEnter records state entry metrics
func (o *MetricsObserver) OnStateEnter(sm *core.StateMachine, state core.State) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	stateName := state.Name()
	o.stateVisits[stateName]++
	o.lastStateEntry[stateName] = time.Now()
}

// OnStateExit records state exit metrics
func (o *MetricsObserver) OnStateExit(sm *core.StateMachine, state core.State) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	stateName := state.Name()
	if entryTime, ok := o.lastStateEntry[stateName]; ok {
		elapsed := time.Since(entryTime)
		o.stateTimeSpent[stateName] += elapsed
		delete(o.lastStateEntry, stateName)
	}
}

// OnTransition records transition metrics
func (o *MetricsObserver) OnTransition(sm *core.StateMachine, from, to core.State, event *core.Event) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Record transition
	var fromName, toName string
	if from != nil {
		fromName = from.Name()
	} else {
		fromName = "nil"
	}

	if to != nil {
		toName = to.Name()
	} else {
		toName = "nil"
	}

	transitionKey := fromName + "->" + toName
	o.transitionCounts[transitionKey]++
}

// OnEventProcessed records event metrics
func (o *MetricsObserver) OnEventProcessed(sm *core.StateMachine, event *core.Event) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.eventCounts[event.Name]++
}

// OnError records error metrics
func (o *MetricsObserver) OnError(sm *core.StateMachine, err error) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.errorCount++
}

// GetStateVisitCounts returns the number of times each state was visited
func (o *MetricsObserver) GetStateVisitCounts() map[string]int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	result := make(map[string]int)
	for state, count := range o.stateVisits {
		result[state] = count
	}
	return result
}

// GetStateTimeSpent returns the time spent in each state
func (o *MetricsObserver) GetStateTimeSpent() map[string]time.Duration {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	result := make(map[string]time.Duration)
	for state, duration := range o.stateTimeSpent {
		result[state] = duration
	}
	return result
}

// GetEventCounts returns the number of times each event was processed
func (o *MetricsObserver) GetEventCounts() map[string]int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	result := make(map[string]int)
	for event, count := range o.eventCounts {
		result[event] = count
	}
	return result
}

// GetTransitionCounts returns the number of times each transition occurred
func (o *MetricsObserver) GetTransitionCounts() map[string]int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	result := make(map[string]int)
	for transition, count := range o.transitionCounts {
		result[transition] = count
	}
	return result
}

// GetErrorCount returns the number of errors
func (o *MetricsObserver) GetErrorCount() int {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	return o.errorCount
}

// Reset resets all metrics
func (o *MetricsObserver) Reset() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.stateVisits = make(map[string]int)
	o.stateTimeSpent = make(map[string]time.Duration)
	o.eventCounts = make(map[string]int)
	o.transitionCounts = make(map[string]int)
	o.errorCount = 0
	o.lastStateEntry = make(map[string]time.Time)
}
