// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package future_test

import (
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/tia/future"
	"github.com/jetsetilly/gopher2600/test"
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
