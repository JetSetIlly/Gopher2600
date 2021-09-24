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

package emulation

import (
	"github.com/jetsetilly/gopher2600/userinput"
)

// State indicates the emulation's state.
type State int

// List of possible emulation states.
const (
	Initialising State = iota
	Running
	Paused
	Stepping
	Rewinding
	Ending
)

// VCS is a minimal abstraction of the VCS hardware. Exists mainly to avoid a
// circular import to the hardware package.
type VCS interface {
}

// VCS is a minimal abstraction of the Gopher2600 debugger. Exists mainly to
// avoid a circular import to the debugger package.
type Debugger interface {
}

// Emulation defines the public functions required for a GUI implementation
// (and possibly other things) to interface with the underlying emulator.
type Emulation interface {
	VCS() VCS
	Debugger() Debugger
	UserInput() chan userinput.Event
	State() State
	Pause(set bool)
}
