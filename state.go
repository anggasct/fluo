package fluo

// State represents a state in the state machine
type State interface {
	ID() string
	Enter(ctx Context)
	Exit(ctx Context)
	Parent() State
	IsComposite() bool
	IsParallel() bool
	IsPseudo() bool
	IsFinal() bool
}

// AtomicState represents a simple state
type AtomicState interface {
	State
}

// CompositeState represents a state with substates
type CompositeState interface {
	State
	InitialState() State
	Substates() []State
	AddSubstate(state State)
}

// SequentialState represents a sequential composite state
type SequentialState interface {
	CompositeState
}

// ParallelState represents a parallel composite state
type ParallelState interface {
	CompositeState
	Regions() []Region
	AddRegion(region Region)
}

// Region represents a parallel region
type Region interface {
	ID() string
	ParentState() ParallelState
	CurrentState() State
	InitialState() State
	States() []State
	IsComplete() bool    // Check if region has reached final state
	HasFinalState() bool // Check if region contains a final state
}

// PseudoState represents a transient state
type PseudoState interface {
	State
	Kind() PseudoStateKind
}

// PseudoStateKind enumerates the types of pseudostates
type PseudoStateKind int

const (
	// Initial pseudostate marks the starting point
	Initial PseudoStateKind = iota
	// Choice pseudostate for conditional branching
	Choice
	// Junction pseudostate for merge points
	Junction
	// Fork pseudostate for parallel splitting
	Fork
	// Join pseudostate for parallel synchronization
	Join
	// Terminate pseudostate for machine termination
	Terminate
	// History pseudostate for shallow history
	History
	// DeepHistory pseudostate for deep history
	DeepHistory
)

// ActionFunc represents an enhanced action function with error support
type ActionFunc func(ctx Context) error

// GuardFunc represents a guard condition function
type GuardFunc func(ctx Context) bool

// AtomicStateImpl implements the AtomicState interface
type AtomicStateImpl struct {
	id          string
	parent      State
	entryAction ActionFunc
	exitAction  ActionFunc
	final       bool
}

// NewAtomicState creates a new atomic state
func NewAtomicState(id string) *AtomicStateImpl {
	return &AtomicStateImpl{
		id:     id,
		parent: nil,
		final:  false,
	}
}

// NewFinalState creates a new final state
func NewFinalState(id string) *AtomicStateImpl {
	return &AtomicStateImpl{
		id:     id,
		parent: nil,
		final:  true,
	}
}

// ID returns the state identifier
func (s *AtomicStateImpl) ID() string {
	return s.id
}

// Enter executes the entry action
func (s *AtomicStateImpl) Enter(ctx Context) {
	if s.entryAction != nil {
		_ = safeExecuteAction(s.entryAction, ctx)
	}
}

// Exit executes the exit action
func (s *AtomicStateImpl) Exit(ctx Context) {
	if s.exitAction != nil {
		_ = safeExecuteAction(s.exitAction, ctx)
	}
}

// Parent returns the parent state
func (s *AtomicStateImpl) Parent() State {
	return s.parent
}

// IsComposite returns false for atomic states
func (s *AtomicStateImpl) IsComposite() bool {
	return false
}

// IsParallel returns false for atomic states
func (s *AtomicStateImpl) IsParallel() bool {
	return false
}

// IsPseudo returns false for atomic states
func (s *AtomicStateImpl) IsPseudo() bool {
	return false
}

// IsFinal returns whether this is a final state
func (s *AtomicStateImpl) IsFinal() bool {
	return s.final
}

// WithEntryAction sets the entry action for the state
func (s *AtomicStateImpl) WithEntryAction(action ActionFunc) *AtomicStateImpl {
	s.entryAction = action
	return s
}

// WithExitAction sets the exit action for the state
func (s *AtomicStateImpl) WithExitAction(action ActionFunc) *AtomicStateImpl {
	s.exitAction = action
	return s
}

// WithParent sets the parent state
func (s *AtomicStateImpl) WithParent(parent State) *AtomicStateImpl {
	s.parent = parent
	return s
}

// CompositeStateImpl implements the CompositeState interface
type CompositeStateImpl struct {
	AtomicStateImpl
	initialState State
	substates    []State
	substateMap  map[string]State
	transitions  []Transition // Store transitions defined within this composite state
}

// NewCompositeState creates a new composite state
func NewCompositeState(id string) *CompositeStateImpl {
	return &CompositeStateImpl{
		AtomicStateImpl: *NewAtomicState(id),
		substates:       make([]State, 0),
		substateMap:     make(map[string]State),
		transitions:     make([]Transition, 0),
	}
}

// IsComposite returns true for composite states
func (s *CompositeStateImpl) IsComposite() bool {
	return true
}

// InitialState returns the initial substate
func (s *CompositeStateImpl) InitialState() State {
	return s.initialState
}

// Substates returns all substates
func (s *CompositeStateImpl) Substates() []State {
	return s.substates
}

// AddSubstate adds a substate to the composite state
func (s *CompositeStateImpl) AddSubstate(state State) {
	s.substates = append(s.substates, state)
	s.substateMap[state.ID()] = state

	// Set parent relationship for atomic states
	if atomicState, ok := state.(*AtomicStateImpl); ok {
		atomicState.WithParent(s)
	}

	// Set parent relationship for composite states
	if compositeState, ok := state.(*CompositeStateImpl); ok {
		compositeState.WithParent(s)
	}

	// Set parent relationship for parallel states
	if parallelState, ok := state.(*ParallelStateImpl); ok {
		parallelState.WithParent(s)
	}
}

// WithInitialState sets the initial substate
func (s *CompositeStateImpl) WithInitialState(state State) *CompositeStateImpl {
	s.initialState = state
	return s
}

// WithParent sets the parent state
func (s *CompositeStateImpl) WithParent(parent State) *CompositeStateImpl {
	s.parent = parent
	return s
}

// SequentialStateImpl implements the SequentialState interface
type SequentialStateImpl struct {
	CompositeStateImpl
}

// NewSequentialState creates a new sequential composite state
func NewSequentialState(id string) *SequentialStateImpl {
	return &SequentialStateImpl{
		CompositeStateImpl: *NewCompositeState(id),
	}
}

// ParallelStateImpl implements the ParallelState interface
type ParallelStateImpl struct {
	CompositeStateImpl
	regions []Region
}

// NewParallelState creates a new parallel composite state
func NewParallelState(id string) *ParallelStateImpl {
	return &ParallelStateImpl{
		CompositeStateImpl: *NewCompositeState(id),
		regions:            make([]Region, 0),
	}
}

// IsParallel returns true for parallel states
func (s *ParallelStateImpl) IsParallel() bool {
	return true
}

// Regions returns all parallel regions
func (s *ParallelStateImpl) Regions() []Region {
	return s.regions
}

// AddRegion adds a parallel region
func (s *ParallelStateImpl) AddRegion(region Region) {
	s.regions = append(s.regions, region)
}

// RegionImpl implements the Region interface
type RegionImpl struct {
	id           string
	parentState  ParallelState
	currentState State
	initialState State
	states       []State
	stateMap     map[string]State
}

// NewRegion creates a new parallel region
func NewRegion(id string, parentState ParallelState) *RegionImpl {
	return &RegionImpl{
		id:          id,
		parentState: parentState,
		states:      make([]State, 0),
		stateMap:    make(map[string]State),
	}
}

// ID returns the region identifier
func (r *RegionImpl) ID() string {
	return r.id
}

// ParentState returns the parent parallel state
func (r *RegionImpl) ParentState() ParallelState {
	return r.parentState
}

// CurrentState returns the current state in this region
func (r *RegionImpl) CurrentState() State {
	return r.currentState
}

// InitialState returns the initial state of this region
func (r *RegionImpl) InitialState() State {
	return r.initialState
}

// States returns all states in this region
func (r *RegionImpl) States() []State {
	return r.states
}

// AddState adds a state to this region
func (r *RegionImpl) AddState(state State) {
	r.states = append(r.states, state)
	r.stateMap[state.ID()] = state
}

// WithInitialState sets the initial state for this region
func (r *RegionImpl) WithInitialState(state State) *RegionImpl {
	r.initialState = state
	return r
}

// IsComplete checks if this region has reached a final state
func (r *RegionImpl) IsComplete() bool {
	if r.currentState == nil {
		return false
	}
	return r.currentState.IsFinal()
}

// HasFinalState checks if this region contains at least one final state
func (r *RegionImpl) HasFinalState() bool {
	for _, state := range r.states {
		if state.IsFinal() {
			return true
		}
	}
	return false
}

// PseudoStateImpl implements the PseudoState interface
type PseudoStateImpl struct {
	AtomicStateImpl
	kind                   PseudoStateKind
	choiceConditions       []ChoiceCondition // For Choice pseudostates
	defaultTarget          string            // Default target for Choice/Junction (state ID)
	forkTargets            []string          // Target states for Fork (state IDs)
	joinSourceCombinations [][]string        // Source state combinations for Join (each element is one valid combination)
	joinTarget             string            // Target state for Join (state ID)
	historyDefault         string            // Default state for History pseudostates (state ID)
	historyType            PseudoStateKind   // History or DeepHistory
}

// NewPseudoState creates a new pseudostate
func NewPseudoState(id string, kind PseudoStateKind) *PseudoStateImpl {
	return &PseudoStateImpl{
		AtomicStateImpl: *NewAtomicState(id),
		kind:            kind,
	}
}

// NewHistoryState creates a new history pseudostate
func NewHistoryState(id string, deep bool) *PseudoStateImpl {
	kind := History
	if deep {
		kind = DeepHistory
	}
	return &PseudoStateImpl{
		AtomicStateImpl: *NewAtomicState(id),
		kind:            kind,
		historyType:     kind,
	}
}

// IsPseudo returns true for pseudostates
func (s *PseudoStateImpl) IsPseudo() bool {
	return true
}

// Kind returns the pseudostate kind
func (s *PseudoStateImpl) Kind() PseudoStateKind {
	return s.kind
}

// Configuration methods for PseudoStateImpl to support builder API

// AddChoiceCondition adds a condition for Choice pseudostates
func (s *PseudoStateImpl) AddChoiceCondition(guard GuardFunc, target string, action ActionFunc) {
	s.choiceConditions = append(s.choiceConditions, ChoiceCondition{
		Guard:  guard,
		Target: target,
		Action: action,
	})
}

// SetDefaultTarget sets the default target for Choice/Junction pseudostates
func (s *PseudoStateImpl) SetDefaultTarget(target string) {
	s.defaultTarget = target
}

// SetForkTargets sets the target states for Fork pseudostates
func (s *PseudoStateImpl) SetForkTargets(targets []string) {
	s.forkTargets = targets
}

// AddForkTarget adds a target state for Fork pseudostates
func (s *PseudoStateImpl) AddForkTarget(target string) {
	s.forkTargets = append(s.forkTargets, target)
}

// SetJoinSources adds a combination of source states for Join pseudostates
// Each call to this method represents one valid source combination
func (s *PseudoStateImpl) SetJoinSources(sources []string) {
	// Add this combination to the list
	combination := make([]string, len(sources))
	copy(combination, sources)
	s.joinSourceCombinations = append(s.joinSourceCombinations, combination)
}

// SetJoinTarget sets the target state for Join pseudostates
func (s *PseudoStateImpl) SetJoinTarget(target string) {
	s.joinTarget = target
}

// SetHistoryDefault sets the default target for History pseudostates
func (s *PseudoStateImpl) SetHistoryDefault(target string) {
	s.historyDefault = target
}
