package future_test

import (
	"gopher2600/hardware/tia/future"
	"testing"
)

func expectFailure(t *testing.T, b bool) {
	t.Helper()
	if b {
		t.Errorf("expected failure")
	}
}

func expectSuccess(t *testing.T, b bool) bool {
	t.Helper()
	if !b {
		t.Errorf("expected success")
		return false
	}

	return true
}

func TestFuture_schedulingDelays(t *testing.T) {
	tck := future.NewTicker("test")

	// ticking with no entries
	expectFailure(t, tck.Tick())
	expectFailure(t, tck.Tick())

	// scheduling delay of -1. this means that the payload should run
	// immediately. subsequent calls to Tick() should fail
	tck.Schedule(-1, func() {}, "test event")
	expectFailure(t, tck.Tick())
	expectFailure(t, tck.Tick())

	// scheduling delay of 0. this means that the payload should run on the
	// first Tick(). subsequent ticks should fail
	tck.Schedule(0, func() {}, "test event")
	expectSuccess(t, tck.Tick())
	expectFailure(t, tck.Tick())
	expectFailure(t, tck.Tick())

	// scheduling delay of 1. this means that the payload should run on the
	// second Tick(). subsequent ticks should fail
	tck.Schedule(1, func() {}, "test event")
	expectFailure(t, tck.Tick())
	expectSuccess(t, tck.Tick())
	expectFailure(t, tck.Tick())
	expectFailure(t, tck.Tick())

	// scheduling delay of 2. this means that the payload should run on the
	// third Tick(). subsequent ticks should fail
	tck.Schedule(2, func() {}, "test event")
	expectFailure(t, tck.Tick())
	expectFailure(t, tck.Tick())
	expectSuccess(t, tck.Tick())
	expectFailure(t, tck.Tick())
	expectFailure(t, tck.Tick())
}
