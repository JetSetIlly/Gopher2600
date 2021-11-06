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

// TV is a minimal abstraction of the TV hardware. Exists mainly to avoid a
// circular import to the hardware package.
//
// The only likely implementation of this interface is the
// television.Television type.
type TV interface {
}

// VCS is a minimal abstraction of the VCS hardware. Exists mainly to avoid a
// circular import to the hardware package.
//
// The only likely implementation of this interface is the hardware.VCS type.
type VCS interface {
}

// VCS is a minimal abstraction of the Gopher2600 debugger. Exists mainly to
// avoid a circular import to the debugger package.
//
// The only likely implementation of this interface is the debugger.Debugger
// type.
type Debugger interface {
}

// Emulation defines the public functions required for a GUI implementation
// (and possibly other things) to interface with the underlying emulator.
type Emulation interface {
	TV() TV
	VCS() VCS
	Debugger() Debugger
	UserInput() chan userinput.Event

	// Send a request to set an emulation feature.
	SetFeature(request FeatureReq, args ...FeatureReqData) error

	// Immediate request for the state of the emulation.
	State() State
}

// Mode inidicates the broad features of the emulation. For example, Debugger
// indicates that the emulation is capable or is willing to handle debugging
// features.
type Mode int

// List of defined modes.
const (
	ModeNone Mode = iota
	ModeDebugger
	ModePlay
)

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
const (
	EmulatorStart State = iota
	Initialising
	Paused
	Stepping
	Rewinding
	Running
	Ending
)

// Event describes an event that might occur in the emulation which is outside
// of the scope of the VCS. For example, when the emulation is paused an
// EventPause can be sent to the GUI (see FeatureReq type in the gui package).
type Event int

// List of defined events.
const (
	EventPause Event = iota
	EventRun
	EventRewindBack
	EventRewindFoward
	EventRewindAtStart
	EventRewindAtEnd
	EventScreenshot
)
