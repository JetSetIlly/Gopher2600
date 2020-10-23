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

package delay

// Event represents something that will occur in the future.
type Event struct {
	initial   int
	remaining int
	paused    bool
	pushed    bool
	value     uint8
}

// Schedule an event to occur in the future. To keep things simple and easily
// copyable (essential for the rewind system) only one value of type uint8 can
// be stored in an Event.
func (e *Event) Schedule(delay int, value uint8) {
	e.initial = delay + 1
	e.remaining = delay + 1
	e.paused = false
	e.pushed = false
	e.value = value
}

// Tick the event forward one cycle. Should be called once per color clock from
// the TIA or TIA controlled subsystem (eg. the player sprite).
func (e *Event) Tick() (uint8, bool) {
	if e.remaining == 0 || e.paused {
		return 0, false
	}

	e.remaining--
	if e.remaining == 0 {
		return e.value, true
	}

	return 0, false
}

// The number of remaining cycles.
func (e *Event) Remaining() int {
	return e.remaining - 1
}

// Pause the ticking of the event. Tick() will have no effect. There is no
// Resume() function because it is not needed. However, a paused event can be
// Forced() (missile and player sprites) or Dropped() (ball and player sprites).
func (e *Event) Pause() {
	e.paused = true
}

// Push event so that it starts again. Same as dropping and rescheduling
// although JustStarted() will not return true for a pushed event. Used
// by player, missile and ball sprites.
func (e *Event) Push() {
	e.remaining = e.initial
	e.pushed = true
}

// Force an event to end now, returning the value. Used by missile and player sprites.
func (e *Event) Force() uint8 {
	e.remaining = 0
	return e.value
}

// Drop or cancel an event. Used by ball and player sprites.
func (e *Event) Drop() {
	e.remaining = 0
}

// JustStarted returns true if Tick() has not yet been called.
func (e *Event) JustStarted() bool {
	return e.initial == e.remaining && !e.pushed
}

// AboutToEnd returns true if the event expires on the next call to Tick().
func (e *Event) AboutToEnd() bool {
	return e.remaining == 1
}

// IsActive returns true if the event is still active. Paused event will still
// report as being active.
func (e *Event) IsActive() bool {
	return e.remaining > 0
}
