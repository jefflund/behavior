// Package bt is a minimalist implementation of a behavior tree.
package bt

// State describes the outcome of running a Behavior.
type State int

func (s State) String() string {
	switch s {
	case Running:
		return "Running"
	case Success:
		return "Success"
	case Failure:
		return "Failure"
	default:
		return "Unknown"
	}
}

// State constants to be used by Behavior.
const (
	Unknown State = iota
	Running
	Success
	Failure
)

// Behavior is a node in a behavior tree.
type Behavior interface {
	Reset()
	Execute() State
}

// Action is a function which acts a Behavior.
type Action func() State

// Reset is a noop.
func (Action) Reset() {}

// Execute calls the underlying function and returns the result.
func (a Action) Execute() State { return a() }

// Conditional is a bool function which acts as a Behavior.
type Conditional func() bool

// Reset is a noop.
func (Conditional) Reset() {}

// Execute calls the function, returning Success if true, or Failure otherwise.
func (c Conditional) Execute() State {
	if c() {
		return Success
	}
	return Failure
}

// composite is the base of a Behavior composed of other Behavior.
type composite struct {
	nodes []Behavior
	index int
}

// Reset moves the index to 0 and resets all child Behavior.
func (c *composite) Reset() {
	c.index = 0
	for _, n := range c.nodes {
		n.Reset()
	}
}

// sequence is a Behavior which is the conjunction of child Behavior.
type sequence struct {
	composite
}

// Sequence gets a Behavior with the conjunction of child Beheavior.
func Sequence(bs ...Behavior) Behavior {
	return &sequence{composite{nodes: bs}}
}

// Execute runs each child Behavior in sequence. It succeeds if all the child
// Behavior suceceed, but immediately fails if any child fails.
func (s *sequence) Execute() State {
	for ; s.index < len(s.nodes); s.index++ {
		switch s.nodes[s.index].Execute() {
		case Running:
			return Running
		case Success:
			continue
		case Failure:
			return Failure
		default:
			return Unknown
		}
	}
	return Success
}

// selection is a Behavior which is the disjunction of child Behavior.
type selection struct {
	composite
}

// Selection gets a Behavior with the disjunction of child Beheavior.
func Selection(bs ...Behavior) Behavior {
	return &selection{composite{nodes: bs}}
}

// Execute runs each child Behavior in sequence. It immediately succeeds if any
// the child Behavior suceceed, but fails if all child Behavior fail.
func (s *selection) Execute() State {
	for ; s.index < len(s.nodes); s.index++ {
		switch s.nodes[s.index].Execute() {
		case Running:
			return Running
		case Success:
			return Success
		case Failure:
			continue
		default:
			return Unknown
		}
	}
	return Failure
}

// pcomposite is the base of a Behavior that runs multiple parallel Behavior.
type pcomposite struct {
	nodes    []Behavior
	complete map[int]bool
}

// Reset resets all child Behavior.
func (c *pcomposite) Reset() {
	c.complete = make(map[int]bool)
	for _, n := range c.nodes {
		n.Reset()
	}
}

// psequence is a Behavior which is the conjunction of parallel child Behavior.
type psequence struct {
	pcomposite
}

// PSequence gets a Behavior with the conjunction of parallel child Beheavior.
func PSequence(bs ...Behavior) Behavior {
	return &psequence{pcomposite{nodes: bs, complete: make(map[int]bool)}}
}

// Execute runs each child behavior in parallel. It succceeds if all the child
// Behavior succeed, but fails if any child fails.
func (s *psequence) Execute() State {
	running := false
	for i, n := range s.nodes {
		if s.complete[i] {
			continue
		}
		switch n.Execute() {
		case Success:
			s.complete[i] = true
		case Running:
			running = true
		case Failure:
			return Failure
		default:
			return Unknown
		}
	}
	if running {
		return Running
	}
	return Success
}

// selection is a Behavior which is the disjunction of parallel child Behavior.
type pselection struct {
	pcomposite
}

// PSelection gets a Behavior with the disjunction of parallel child Beheavior.
func PSelection(bs ...Behavior) Behavior {
	return &pselection{pcomposite{nodes: bs, complete: make(map[int]bool)}}
}

// Execute runs each child behavior in parallel. It succceeds if any the child
// Behavior succeed, but fails if all child Behavior fail.
func (s *pselection) Execute() State {
	running := false
	for i, n := range s.nodes {
		if s.complete[i] {
			continue
		}
		switch n.Execute() {
		case Failure:
			s.complete[i] = true
		case Running:
			running = true
		case Success:
			return Success
		default:
			return Unknown
		}
	}
	if running {
		return Running
	}
	return Failure
}

// decorator is a Behavior which transforms the output of another Behavior.
type decorator struct {
	node      Behavior
	transform func(State) State
}

// Reset resets the underlying Behavior.
func (d *decorator) Reset() {
	d.node.Reset()
}

// Execute runs the underlying Behavior, but returns the transformed State.
func (d *decorator) Execute() State {
	return d.transform(d.node.Execute())
}

// Invert wraps a Behavior to invert Success and Failure.
func Invert(b Behavior) Behavior {
	invert := func(s State) State {
		switch s {
		case Running:
			return Running
		case Success:
			return Failure
		case Failure:
			return Success
		default:
			return Unknown
		}
	}
	return &decorator{b, invert}
}

// Repeat wraps a Behavior to make it run indefinitely.
func Repeat(b Behavior) Behavior {
	repeat := func(s State) State {
		switch s {
		case Success, Failure:
			b.Reset()
			return Running
		case Running:
			return Running
		default:
			return Unknown
		}
	}
	return &decorator{b, repeat}
}

// ForceSuccess wraps a Behavior so Failure instead results in Success.
func ForceSuccess(b Behavior) Behavior {
	force := func(s State) State {
		switch s {
		case Success, Failure:
			return Success
		case Running:
			return Running
		default:
			return Unknown
		}
	}
	return &decorator{b, force}
}

// ForceFailure  wraps a Behavior so Success instead results in Failure.
func ForceFailure(b Behavior) Behavior {
	force := func(s State) State {
		switch s {
		case Success, Failure:
			return Failure
		case Running:
			return Running
		default:
			return Unknown
		}
	}
	return &decorator{b, force}
}

// Until wraps a Behavior so it runs repeatedly until Success.
func Until(b Behavior) Behavior {
	until := func(s State) State {
		switch s {
		case Success:
			return Success
		case Failure:
			b.Reset()
			return Running
		case Running:
			return Running
		default:
			return Unknown
		}
	}
	return &decorator{b, until}
}

// While wraps a Behavior so it runs repeatedly until Failure.
func While(b Behavior) Behavior {
	while := func(s State) State {
		switch s {
		case Failure:
			return Failure
		case Success:
			b.Reset()
			return Running
		case Running:
			return Running
		default:
			return Unknown
		}
	}
	return &decorator{b, while}
}
