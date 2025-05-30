// Package observers provides observers for monitoring state machine events
package observers

import (
	"fmt"
	"sync"

	"github.com/anggasct/fluo/pkg/core"
)

// LogLevel represents the logging level
type LogLevel int

const (
	// LogError logs only errors
	LogError LogLevel = iota
	// LogWarning logs errors and warnings
	LogWarning
	// LogInfo logs errors, warnings, and info
	LogInfo
	// LogDebug logs errors, warnings, info, and debug
	LogDebug
)

// LoggingObserver logs state machine events
type LoggingObserver struct {
	level     LogLevel
	prefix    string
	mutex     sync.RWMutex
	formatter LogFormatter
}

// LogFormatter formats log messages
type LogFormatter func(level LogLevel, format string, args ...interface{}) string

// DefaultLogFormatter provides default log formatting
func DefaultLogFormatter(level LogLevel, format string, args ...interface{}) string {
	levelStr := "INFO"
	switch level {
	case LogError:
		levelStr = "ERROR"
	case LogWarning:
		levelStr = "WARN"
	case LogInfo:
		levelStr = "INFO"
	case LogDebug:
		levelStr = "DEBUG"
	}

	return fmt.Sprintf("[%s] %s", levelStr, fmt.Sprintf(format, args...))
}

// NewLoggingObserver creates a new logging observer
func NewLoggingObserver(level LogLevel, prefix string) *LoggingObserver {
	return &LoggingObserver{
		level:     level,
		prefix:    prefix,
		formatter: DefaultLogFormatter,
	}
}

// SetFormatter sets the log formatter
func (o *LoggingObserver) SetFormatter(formatter LogFormatter) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	o.formatter = formatter
}

// log logs a message at the specified level
func (o *LoggingObserver) log(level LogLevel, format string, args ...interface{}) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	if level <= o.level {
		prefix := ""
		if o.prefix != "" {
			prefix = fmt.Sprintf("[%s] ", o.prefix)
		}

		message := ""
		if o.formatter != nil {
			message = o.formatter(level, format, args...)
		} else {
			message = fmt.Sprintf(format, args...)
		}

		fmt.Printf("%s%s\n", prefix, message)
	}
}

// OnStateEnter logs state entry
func (o *LoggingObserver) OnStateEnter(sm *core.StateMachine, state core.State) {
	o.log(LogInfo, "Entering state: %s", state.Name())
}

// OnStateExit logs state exit
func (o *LoggingObserver) OnStateExit(sm *core.StateMachine, state core.State) {
	o.log(LogInfo, "Exiting state: %s", state.Name())
}

// OnTransition logs transitions
func (o *LoggingObserver) OnTransition(sm *core.StateMachine, from, to core.State, event *core.Event) {
	fromName := "nil"
	if from != nil {
		fromName = from.Name()
	}

	toName := "nil"
	if to != nil {
		toName = to.Name()
	}

	o.log(LogInfo, "Transition: %s -> %s on event: %s", fromName, toName, event.Name)
}

// OnEventProcessed logs events
func (o *LoggingObserver) OnEventProcessed(sm *core.StateMachine, event *core.Event) {
	o.log(LogDebug, "Event processed: %s", event.Name)
}

// OnError logs errors
func (o *LoggingObserver) OnError(sm *core.StateMachine, err error) {
	o.log(LogError, "Error: %v", err)
}
