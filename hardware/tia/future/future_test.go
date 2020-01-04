package future_test

import (
	"gopher2600/hardware/tia/future"
	"gopher2600/test"
	"testing"
)

func TestFuture_schedulingDelays(t *testing.T) {
	tck := future.NewTicker("test")

	var ev *future.Event

	// ticking with no entries
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	// scheduling delay of -1. this means that the payload should run
	// immediately. subsequent calls to Tick() should fail
	ev = tck.Schedule(-1, func() {}, "test event")
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	// scheduling delay of 0. this means that the payload should run on the
	// first Tick(). subsequent ticks should fail
	ev = tck.Schedule(0, func() {}, "test event")
	test.ExpectedSuccess(t, ev.JustStarted())
	test.ExpectedSuccess(t, ev.AboutToEnd())
	test.ExpectedSuccess(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	// scheduling delay of 1. this means that the payload should run on the
	// second Tick(). subsequent ticks should fail
	ev = tck.Schedule(1, func() {}, "test event")
	test.ExpectedSuccess(t, ev.JustStarted())
	test.ExpectedFailure(t, ev.AboutToEnd())
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedSuccess(t, ev.AboutToEnd())
	test.ExpectedSuccess(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	sentinal := false

	// scheduling delay of 2. this means that the payload should run on the
	// third Tick(). subsequent ticks should fail
	ev = tck.Schedule(2, func() { sentinal = true }, "test event")
	test.ExpectedSuccess(t, ev.JustStarted())
	test.ExpectedFailure(t, ev.AboutToEnd())
	test.ExpectedFailure(t, tck.Tick())
	test.Equate(t, ev.RemainingCycles(), 1)
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedSuccess(t, ev.AboutToEnd())
	test.ExpectedSuccess(t, tck.Tick())

	// for this test we've made sure the payload does something
	test.ExpectedSuccess(t, sentinal)

	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
}

func TestFuture_force(t *testing.T) {
	tck := future.NewTicker("test")

	var ev *future.Event

	sentinal := false

	ev = tck.Schedule(2, func() { sentinal = true }, "test event")
	test.ExpectedSuccess(t, ev.JustStarted())
	test.ExpectedFailure(t, ev.AboutToEnd())
	test.Equate(t, ev.RemainingCycles(), 2)
	ev.Force()
	test.Equate(t, ev.RemainingCycles(), -1)
	test.ExpectedSuccess(t, sentinal)
	test.ExpectedFailure(t, tck.Tick())
}

func TestFuture_drop(t *testing.T) {
	tck := future.NewTicker("test")

	var ev *future.Event

	sentinal := false

	ev = tck.Schedule(2, func() { sentinal = true }, "test event")
	test.ExpectedSuccess(t, ev.JustStarted())
	test.ExpectedFailure(t, ev.AboutToEnd())
	test.Equate(t, ev.RemainingCycles(), 2)
	ev.Drop()
	test.Equate(t, ev.RemainingCycles(), -1)
	test.ExpectedFailure(t, sentinal)
	test.ExpectedFailure(t, tck.Tick())
}

func TestFuture_drop2(t *testing.T) {
	tck := future.NewTicker("test")

	var ev *future.Event

	tck.Schedule(5, func() {}, "test event")
	ev = tck.Schedule(3, func() {}, "test event")
	test.ExpectedFailure(t, tck.Tick())
	test.Equate(t, tck.String(), `test: test event -> 4
test: test event -> 2`)
	ev.Drop()
	test.Equate(t, tck.String(), `test: test event -> 4`)
}
