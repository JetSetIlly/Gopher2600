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

// Mode inidicates the broad features of the emulation. Currently defined to be
// debugger and play.
type Mode int

func (m Mode) String() string {
	switch m {
	case ModeDebugger:
		return "Debugger"
	case ModePlay:
		return "Playmode"
	}

	return ""
}

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
//
// * There is a sub-state of the rewinding state that we can think of as the
// "catch-up" state. This occurs in the brief transition period between
// Rewinding and the Running or Pausing state.
//
// Currently, we handle this state in the CartUpLoop() function of the debugger
// package. There is a good argument to be made for having the catch-up state
// as a distinct State listed below.
const (
	EmulatorStart State = iota
	Initialising
	Paused
	Stepping
	Rewinding
	Running
	Ending
)

// Event is something that happens to change the state of the emulation. For
// example, the user presses the pause while playing  game. This will cause the
// GUI to send an EventPause event to the emulation.
type Event int

// List of defined events.
const (
	EventInitialising Event = iota
	EventPause
	EventRun
	EventRewindBack
	EventRewindFoward
	EventRewindAtStart
	EventRewindAtEnd
	EventScreenshot
	EventMute
	EventUnmute
)

// FeatureReq is used to request the setting of an emulation attribute
// eg. a pause request from the GUI
type FeatureReq string

// FeatureReqData represents the information associated with a FeatureReq. See
// commentary for the defined FeatureReq values for the underlying type.
type FeatureReqData interface{}

// List of valid feature requests. argument must be of the type specified or
// else the interface{} type conversion will fail and the application will
// probably crash.
//
// Note that, like the name suggests, these are requests, they may or may not
// be satisfied depending on other conditions in the GUI.
const (
	// notify gui of the underlying emulation mode.
	ReqSetPause FeatureReq = "ReqSetPause" // bool

	// change emulation mode
	ReqSetMode FeatureReq = "ReqSetMode" // emulation.Mode
)

// Sentinal error returned if emulation does no support requested feature.
const (
	UnsupportedEmulationFeature = "unsupported emulation feature: %v"
)
