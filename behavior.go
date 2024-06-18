// Package bt is a minimalist implementation of a behavior tree.
package bt

// State describes the outcome of running a Behavior.
type State int

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

// sequence is a Behavior which is the logical conjunction of child Behavior.
type sequence struct {
	composite
}

// Sequence gets a Behavior with the logical conjunction of child Beheavior.
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

// selection is a Behavior which is the logical disjunction of child Behavior.
type selection struct {
	composite
}

// Selection gets a Behavior with the logical disjunction of child Beheavior.
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
			fallthrough
		case Running:
			return Running
		default:
			return Unknown
		}
	}
	return &decorator{b, repeat}
}
