package future_test

import (
	"gopher2600/hardware/tia/future"
	"gopher2600/test"
	"testing"
)

func TestFuture_schedulingDelays(t *testing.T) {
	tck := future.NewTicker("test")

	// ticking with no entries
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	// scheduling delay of -1. this means that the payload should run
	// immediately. subsequent calls to Tick() should fail
	tck.Schedule(-1, func() {}, "test event")
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	// scheduling delay of 0. this means that the payload should run on the
	// first Tick(). subsequent ticks should fail
	tck.Schedule(0, func() {}, "test event")
	test.ExpectedSuccess(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	// scheduling delay of 1. this means that the payload should run on the
	// second Tick(). subsequent ticks should fail
	tck.Schedule(1, func() {}, "test event")
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedSuccess(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())

	// scheduling delay of 2. this means that the payload should run on the
	// third Tick(). subsequent ticks should fail
	tck.Schedule(2, func() {}, "test event")
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedSuccess(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
	test.ExpectedFailure(t, tck.Tick())
}
