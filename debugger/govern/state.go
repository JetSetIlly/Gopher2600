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

package govern

// State indicates the emulation's state.
type State int

// List of possible emulation states.
//
// EmulatorStart is the default state and should never be entered once the
// emulator has begun.
//
// Initialising can be used when reinitialising the emulator. for example, when
// a new cartridge is being inserted.
//
// Values are ordered so that order comparisons are meaningful. For example,
// Running is "greater than" Stepping, Paused, etc.
//
// Note that there is a sub-state of the rewinding state that we can potentially
// think of as the "catch-up" state. This occurs in the brief transition period
// between Rewinding and the Running or Pausing state. For simplicity, the
// catch-up loop is part of the Rewinding state
const (
	EmulatorStart State = iota
	Initialising
	Paused
	Stepping
	Rewinding
	Running
	Ending
)

func (s State) String() string {
	switch s {
	case EmulatorStart:
		return "EmulatorStart"
	case Initialising:
		return "Initialising"
	case Paused:
		return "Paused"
	case Stepping:
		return "Stepping"
	case Rewinding:
		return "Rewinding"
	case Running:
		return "Running"
	case Ending:
		return "Ending"
	}

	return ""
}
