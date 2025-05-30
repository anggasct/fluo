package observers

// NewDefaultLoggingObserver creates a logging observer with default settings (LogInfo level)
func NewDefaultLoggingObserver() *LoggingObserver {
	return NewLoggingObserver(LogInfo, "StateMachine")
}
