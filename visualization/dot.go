package visualization

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/anggasct/fluo"
)

// DOTGenerator generates Graphviz DOT format representations of state machines
type DOTGenerator struct {
	machineDefinition fluo.MachineDefinition
	options           DOTOptions
}

// DOTOptions configures the DOT generation
type DOTOptions struct {
	ShowGuardConditions bool
	ShowActions         bool
	ShowPseudostates    bool
	CompactMode         bool
	RankDirection       string // "TB", "LR", "BT", "RL"
	NodeShape           string
	TransitionStyle     string
	CompositeStateStyle string
	ParallelStateStyle  string
	PseudostateStyle    string
}

// DefaultDOTOptions returns sensible default options for DOT generation
func DefaultDOTOptions() DOTOptions {
	return DOTOptions{
		ShowGuardConditions: true,
		ShowActions:         true,
		ShowPseudostates:    true,
		CompactMode:         false,
		RankDirection:       "TB",
		NodeShape:           "box",
		TransitionStyle:     "solid",
		CompositeStateStyle: "rounded,filled",
		ParallelStateStyle:  "rounded,filled",
		PseudostateStyle:    "circle",
	}
}

// NewDOTGenerator creates a new DOT generator for the given machine definition
func NewDOTGenerator(machineDefinition fluo.MachineDefinition, options ...DOTOptions) *DOTGenerator {
	opts := DefaultDOTOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	return &DOTGenerator{
		machineDefinition: machineDefinition,
		options:           opts,
	}
}

// Generate creates a DOT representation of the state machine
func (g *DOTGenerator) Generate() (string, error) {
	var dot strings.Builder

	// DOT header
	dot.WriteString("digraph StateMachine {\n")
	dot.WriteString(fmt.Sprintf("  rankdir=%s;\n", g.options.RankDirection))
	dot.WriteString("  node [shape=box];\n")
	dot.WriteString("  edge [fontsize=10];\n\n")

	// Generate states
	if err := g.generateStates(&dot); err != nil {
		return "", fmt.Errorf("failed to generate states: %w", err)
	}

	// Generate transitions
	if err := g.generateTransitions(&dot); err != nil {
		return "", fmt.Errorf("failed to generate transitions: %w", err)
	}

	// DOT footer
	dot.WriteString("}\n")

	return dot.String(), nil
}

// generateStates generates DOT nodes for all states
func (g *DOTGenerator) generateStates(dot *strings.Builder) error {
	states := g.machineDefinition.GetStates()
	initialState := g.machineDefinition.GetInitialState()

	dot.WriteString("  // States\n")

	for stateID, state := range states {
		if err := g.generateStateNode(dot, stateID, state, stateID == initialState); err != nil {
			return err
		}
	}

	return nil
}

// generateStateNode generates a DOT node for a single state
func (g *DOTGenerator) generateStateNode(dot *strings.Builder, stateID string, state fluo.State, isInitial bool) error {
	// Determine node style based on state type
	style := g.options.NodeShape
	fillColor := "lightblue"
	label := stateID

	if isInitial {
		fillColor = "lightgreen"
		label += "\\n(initial)"
	}

	if state.IsFinal() {
		style = "doublecircle"
		fillColor = "lightcoral"
	} else if state.IsPseudo() {
		if pseudoState, ok := state.(fluo.PseudoState); ok {
			style = g.options.PseudostateStyle
			label = fmt.Sprintf("%s\\n[%s]", stateID, g.getPseudostateKindName(pseudoState.Kind()))
			fillColor = "lightyellow"
		}
	} else if state.IsComposite() {
		// Split composite state style into shape and other attributes
		parts := strings.Split(g.options.CompositeStateStyle, ",")
		if len(parts) > 0 {
			style = parts[0] // Use first part as shape
		}
		fillColor = "lightcyan"
		if state.IsParallel() {
			style = g.options.ParallelStateStyle
			fillColor = "lavender"
		}
	}

	dot.WriteString(fmt.Sprintf("  \"%s\" [shape=%s style=\"filled\" fillcolor=%s label=\"%s\"];\n",
		stateID, style, fillColor, label))

	return nil
}

// generateTransitions generates DOT edges for all transitions
func (g *DOTGenerator) generateTransitions(dot *strings.Builder) error {
	transitions := g.machineDefinition.GetTransitions()

	dot.WriteString("  // Transitions\n")

	for from, toStates := range transitions {
		for _, to := range toStates {
			// Access the TargetState field from the Transition struct
			dot.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", from, to.TargetState))
		}
	}

	return nil
}

// getPseudostateKindName returns a human-readable name for pseudostate kinds
func (g *DOTGenerator) getPseudostateKindName(kind fluo.PseudoStateKind) string {
	switch kind {
	case fluo.Initial:
		return "Initial"
	case fluo.Choice:
		return "Choice"
	case fluo.Junction:
		return "Junction"
	case fluo.Fork:
		return "Fork"
	case fluo.Join:
		return "Join"
	case fluo.Terminate:
		return "Terminate"
	case fluo.History:
		return "History"
	case fluo.DeepHistory:
		return "Deep History"
	default:
		return "Unknown"
	}
}

// GenerateToFile writes the DOT representation to a file
func (g *DOTGenerator) GenerateToFile(filename string) error {
	content, err := g.Generate()
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(content), 0644)
}

// SVGGenerator generates SVG representations by calling Graphviz
type SVGGenerator struct {
	dotGenerator *DOTGenerator
}

// NewSVGGenerator creates a new SVG generator
func NewSVGGenerator(machineDefinition fluo.MachineDefinition, options ...DOTOptions) *SVGGenerator {
	return &SVGGenerator{
		dotGenerator: NewDOTGenerator(machineDefinition, options...),
	}
}

// Generate creates an SVG representation of the state machine
func (g *SVGGenerator) Generate() (string, error) {
	dotContent, err := g.dotGenerator.Generate()
	if err != nil {
		return "", err
	}

	// Use Graphviz dot command to convert DOT to SVG
	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = strings.NewReader(dotContent)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute dot command: %w (make sure Graphviz is installed)", err)
	}

	return out.String(), nil
}

// GenerateSVG creates an SVG representation of the state machine
// This is a convenience method on DOTGenerator for compatibility
func (g *DOTGenerator) GenerateSVG() (string, error) {
	svgGen := &SVGGenerator{dotGenerator: g}
	return svgGen.Generate()
}
