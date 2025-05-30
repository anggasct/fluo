package builders

// NewBuilder provides backward compatibility with older versions of the library
// It's simply an alias for NewStateMachineBuilder
func NewBuilder(name string) *StateMachineBuilder {
	return NewStateMachineBuilder(name)
}
