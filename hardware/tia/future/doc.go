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

// Pacakge future conceptualises event delays inside the VCS. An event, in this
// context means, something that occurs as a response to a stimulus generated
// by the CPU. For example, when a program causes the CPU to strobe the RESP0
// address, the act of resetting the player's horizontal position does not
// happen immediately. Instead there is a short delay, measured in cycles,
// before the effect of the memory write occurs.
//
// The emulation code is full of events, like RESP0, that from the viewpoint of
// the event itself, occurs in the future. In the future package we use the
// Event type to represent this. The Event type is not instantiated directly.
// Instead, events are scheduled via an instance of the Ticker type.
//
// The Ticker type coordinates scheduled events for those parts of the VCS that
// experience stimulus delays. For instance each player sprite has an instance
// of Ticker. Events are created and registered with Schedule function.  The
// function takes the delay period, a label (useful identifying the event for
// debuggers) and a callback function as arguments. The callback is called
// once the delay period has expired.
//
// The Tick() function of the Ticker type is used to indicate that time has passed.
// The RemainingCycles() function indicates how much more time (or to put
// another way, how many more calls to Tick()) is required before the payload
// is executed. It is up to the users of the package to govern how often and
// when Tick() is called. The other Ticker functions help the governor to fine
// tune the ticks.
//
// To help keep code clean, two other interfaces to the Ticker type are
// provided, the Scheduler and Observer. The Scheduler is used in those places
// where an event is only ever scheduled. The Observer interface meanwhile is
// useful for debuggers.
package future
