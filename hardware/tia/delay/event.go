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

// Event represents an event that will occur in the future
type Event struct {
	initial   int
	remaining int
	paused    bool
	pushed    bool
	payload   func(v interface{})
	arg       interface{}
}

// Schedule an event to occur in the future. The payload function will run
// after delay number of cycles.
func (e *Event) Schedule(delay int, payload func(interface{}), arg interface{}) {
	e.initial = delay + 1
	e.remaining = delay + 1
	e.paused = false
	e.pushed = false
	e.payload = payload
	e.arg = arg
}

// Tick the event forward one cycle. Should be called once per color clock from
// the TIA or TIA controlled subsystem (eg. the player sprite).
func (e *Event) Tick() {
	if e.remaining == 0 || e.paused {
		return
	}

	e.remaining--
	if e.remaining == 0 {
		e.payload(e.arg)
	}
}

// The number of remaining cycles
func (e *Event) Remaining() int {
	return e.remaining - 1
}

// Pause the ticking of the event. Tick() will have no effect. There is no
// Resume() function because it is not needed. However, a paused event can be
// Forced() (missile and player sprites) or Dropped() (ball and player sprites)
func (e *Event) Pause() {
	e.paused = true
}

// Push event so that it starts again. Same as dropping and rescheduling
// although JustStarted() will not return true for a pushed event. Used
// by player, missile and ball sprites
func (e *Event) Push() {
	e.remaining = e.initial
	e.pushed = true
}

// Force an event to run the payload now. Cancels the future event. Used by
// missile and player sprites.
func (e *Event) Force() {
	e.remaining = 0
	e.payload(e.arg)
}

// Drop will cancel a event. Payload will not be run. Used by ball and player
// sprites.
func (e *Event) Drop() {
	e.remaining = 0
}

// JustStarted returns true if Tick() has not yet been called.
func (e *Event) JustStarted() bool {
	return e.initial == e.remaining && !e.pushed
}

// AboutToEnd returns true if the event concludes (and the payload run) on the
// next call to Tikc()
func (e *Event) AboutToEnd() bool {
	return e.remaining == 1
}

// IsActive returns true if the event is "running" (is yet to drop the
// payload). Note that paused event will still report as being active.
func (e *Event) IsActive() bool {
	return e.remaining > 0
}
