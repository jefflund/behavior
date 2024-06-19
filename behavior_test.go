package bt

import (
	"fmt"
	"reflect"
	"testing"
)

func CheckBehavior(name string, t *testing.T, b Behavior, expected []State) {
	actual := make([]State, len(expected))
	for i := 0; i < len(expected); i++ {
		actual[i] = b.Execute()
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s produced incorrect states: %v", name, actual)
	}
}

func Recorded(states ...State) Behavior {
	i := 0
	return Action(func() State {
		result := states[i%len(states)]
		i++
		return result
	})
}

type testBehavior struct {
	base   Behavior
	calls  int
	resets int
}

func (b *testBehavior) Execute() State {
	b.calls++
	return b.base.Execute()
}

func (b *testBehavior) Reset() {
	b.base.Reset()
	b.resets++
}

func TestFunc(t *testing.T) {
	called := false
	b := Func(func() {
		called = true
	})
	if actual := b.Execute(); actual != Success {
		t.Error("Func produce incorrect state:", actual)
	}
	if !called {
		t.Error("Func failed to call func")
	}
}

func TestSequence_Success(t *testing.T) {
	b := Sequence(
		Recorded(Running, Success),
		Recorded(Running, Success),
		Recorded(Success),
	)
	expected := []State{Running, Running, Success}
	CheckBehavior("Sequence (Success)", t, b, expected)
}

func TestSequence_Failure(t *testing.T) {
	b := Sequence(
		Recorded(Running, Success),
		Recorded(Running, Running, Failure),
		Recorded(Success),
	)
	expected := []State{Running, Running, Running, Failure}
	CheckBehavior("Sequence (Failure)", t, b, expected)
}

func TestSequence_Unknown(t *testing.T) {
	b := Sequence(
		Recorded(Running, Success),
		Recorded(Running, Running, Unknown),
		Recorded(Success),
	)
	expected := []State{Running, Running, Running, Unknown}
	CheckBehavior("Sequence (Unknown)", t, b, expected)
}

func TestSelection_Success(t *testing.T) {
	b := Selection(
		Recorded(Running, Failure),
		Recorded(Failure),
		Recorded(Running, Failure),
		Recorded(Success),
		Recorded(Success),
	)
	expected := []State{Running, Running, Success}
	CheckBehavior("Selection (Success)", t, b, expected)
}

func TestSelection_Failure(t *testing.T) {
	b := Selection(
		Recorded(Running, Failure),
		Recorded(Failure),
		Recorded(Running, Failure),
		Recorded(Failure),
		Recorded(Failure),
		Recorded(Running, Failure),
	)
	expected := []State{Running, Running, Running, Failure}
	CheckBehavior("Selection (Failure)", t, b, expected)
}

func TestSelection_Unknown(t *testing.T) {
	b := Selection(
		Recorded(Running, Failure),
		Recorded(Failure),
		Recorded(Running, Failure),
		Recorded(Unknown),
		Recorded(Success),
	)
	expected := []State{Running, Running, Unknown}
	CheckBehavior("Selection (Unknown)", t, b, expected)
}

func TestPSequence_Success(t *testing.T) {
	child := &testBehavior{base: Recorded(Running, Running, Success)}
	b := PSequence(
		child,
		Recorded(Running, Success),
		Recorded(Success),
	)
	expected := []State{Running, Running, Success}
	CheckBehavior("PSequence (Success)", t, b, expected)
	if child.calls != 3 {
		t.Error("PSequence (Success) failed to call child each Execute")
	}
}

func TestPSequence_Failure(t *testing.T) {
	child := &testBehavior{base: Recorded(Running, Running, Success)}
	b := PSequence(
		child,
		Recorded(Running, Failure),
		Recorded(Success),
	)
	expected := []State{Running, Failure}
	CheckBehavior("PSequence (Failure)", t, b, expected)
	if child.calls != 2 {
		t.Error("PSequence (Success) failed to call child each Execute")
	}
}

func TestPSequence_Unknown(t *testing.T) {
	child := &testBehavior{base: Recorded(Running, Running, Success)}
	b := PSequence(
		child,
		Recorded(Running, Unknown),
		Recorded(Success),
	)
	expected := []State{Running, Unknown}
	CheckBehavior("PSequence (Unknown)", t, b, expected)
	if child.calls != 2 {
		t.Error("PSequence (Unknown) failed to call child each Execute")
	}
}

func TestPSelection_Success(t *testing.T) {
	child := &testBehavior{base: Recorded(Running, Running, Success)}
	b := PSelection(
		child,
		Recorded(Running, Success),
		Recorded(Running, Success),
	)
	expected := []State{Running, Success}
	CheckBehavior("PSelection (Success)", t, b, expected)
	if child.calls != 2 {
		t.Error("PSequence (Success) failed to call child each Execute")
	}
}

func TestPSelection_Failure(t *testing.T) {
	child := &testBehavior{base: Recorded(Running, Running, Failure)}
	b := PSelection(
		child,
		Recorded(Failure),
		Recorded(Running, Failure),
	)
	expected := []State{Running, Running, Failure}
	CheckBehavior("PSelection (Failure)", t, b, expected)
	if child.calls != 3 {
		t.Error("PSelection (Failure) failed to call child each Execute")
	}
}

func TestPSelection_Unknown(t *testing.T) {
	child := &testBehavior{base: Recorded(Running, Running, Success)}
	b := PSelection(
		child,
		Recorded(Running, Unknown),
		Recorded(Running, Success),
	)
	expected := []State{Running, Unknown}
	CheckBehavior("PSelection (Unknown)", t, b, expected)
	if child.calls != 2 {
		t.Error("PSequence (Unknown) failed to call child each Execute")
	}
}

func TestConditional(t *testing.T) {
	cases := []struct {
		output   bool
		expected State
	}{
		{true, Success},
		{false, Failure},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("Conditional (%t)", c.output), func(t *testing.T) {
			b := Conditional(func() bool { return c.output })
			if b.Execute() != c.expected {
				t.Errorf("Conditional failed to turn %t into %v", c.output, c.expected)
			}
		})
	}
}

func TestInvert(t *testing.T) {
	b := Invert(Recorded(Running, Failure, Success, Unknown))
	expected := []State{Running, Success, Failure, Unknown}
	CheckBehavior("Invert", t, b, expected)
}

func TestRepeat(t *testing.T) {
	wrapped := &testBehavior{base: Recorded(Running, Failure, Success, Unknown)}
	repeat := Repeat(wrapped)
	expected := []State{Running, Running, Running, Unknown}
	actual := make([]State, len(expected))
	for i := range expected {
		actual[i] = repeat.Execute()
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Error("Repeat produced incorrect states", actual)
	}
	if wrapped.resets != 2 {
		t.Error("Repeat failed to reset wrapped Behavior", wrapped.resets)
	}
}

func TestForceSuccess(t *testing.T) {
	b := ForceSuccess(Recorded(Failure, Running, Success))
	expected := []State{Success, Running, Success}
	CheckBehavior("ForceSuccess", t, b, expected)
}

func TestForceFailure(t *testing.T) {
	b := ForceFailure(Recorded(Failure, Running, Success))
	expected := []State{Failure, Running, Failure}
	CheckBehavior("ForceFailure", t, b, expected)
}

func TestUntil(t *testing.T) {
	b := Until(Recorded(Failure, Running, Failure, Success))
	expected := []State{Running, Running, Running, Success}
	CheckBehavior("Until", t, b, expected)
}

func TestWhile(t *testing.T) {
	b := While(Recorded(Success, Running, Success, Failure))
	expected := []State{Running, Running, Running, Failure}
	CheckBehavior("While", t, b, expected)
}
