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
// Paused and Rewinding can have meaningful sub-states
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

// SubState allows more detail for some states. NoSubState indicates that there
// is not more information to impart about the state
type SubState int

// List of possible rewinding sub states
const (
	Normal SubState = iota
	RewindingBackwards
	RewindingForwards
	PausedAtStart
	PausedAtEnd
)

func (s SubState) String() string {
	switch s {
	case RewindingBackwards:
		return "Backwards"
	case RewindingForwards:
		return "Forwards"
	case PausedAtStart:
		return "Paused at start"
	case PausedAtEnd:
		return "Paused at end"
	}
	return ""
}

// StateIntegrity checks whether the combination of state, sub-state makes
// sense. The previous state is also required for a complete check.
//
// Rules:
//
//  1. NoSubState can coexist with any state
//
//  2. PausedAtStart and PausedAtEnd can only be paired with the Paused State
//
//  3. RewindingBackwards and RewindingForwards can only be paired with the
//     Rewinding state
func StateIntegrity(state State, subState SubState) bool {
	if subState == Normal {
		return true
	}
	switch state {
	case Rewinding:
		if subState == RewindingBackwards || subState == RewindingForwards {
			return true
		}
	case Paused:
		if subState == PausedAtEnd || subState == PausedAtStart {
			return true
		}
	}
	return false
}
