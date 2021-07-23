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

package gui

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"

// FeatureReq is used to request the setting of a gui attribute
// eg. toggling the overlay.
type FeatureReq string

// FeatureReqData represents the information associated with a FeatureReq. See
// commentary for the defined FeatureReq values for the underlying type.
type FeatureReqData interface{}

// EmulationState indicates to the GUI that the emulatoin is in a particular
// state.
//
// Note that these should be set for all application types. The GUI state will
// start in StateInitialising and a playmode for example, should set
// StateRunning as soon as the emulation begins (with vcs.Run).
type EmulationState int

// List of valid emulation states.
const (
	StateInitialising EmulationState = iota
	StatePaused
	StateRunning
	StateStepping
	StateRewinding
	StateEnding
)

// PlusROMFirstInstallation is used to pass information to the GUI as part of
// the request.
type PlusROMFirstInstallation struct {
	Finish chan error
	Cart   *plusrom.PlusROM
}

// List of valid feature requests. argument must be of the type specified or
// else the interface{} type conversion will fail and the application will
// probably crash.
//
// Note that, like the name suggests, these are requests, they may or may not
// be satisfied depending other conditions in the GUI.
const (
	// ReqSetPlaymode is called whenever the playmode loop is entered.
	//
	// first argument is a pointer to the VCS, second argument is an instance
	// of received for gui events.
	ReqSetPlaymode FeatureReq = "ReqSetPlaymode" // playmode.Playmode

	// ReqSetDebugmode is called whenever the playmode loop is entered.
	//
	// first argument is a pointer to the debugger, second argument is an
	// instance of received for gui events.
	ReqSetDebugmode FeatureReq = "ReqSetDebugmode" // *debugger.Debugger, chan userinput.Event

	// notify GUI of emulation state. the GUI should use this to alter how
	// infomration, particularly the display of the PixelRenderer.
	ReqState FeatureReq = "ReqState" // EmulationState

	// whether gui should try to sync with the monitro refresh rate. not all
	// gui modes have to obey this but for presentation/play modes it's a good
	// idea to have it set.
	ReqMonitorSync FeatureReq = "ReqMonitorSync" // bool

	// whether the gui is visible or not. results in an error if requested in
	// playmode.
	ReqSetVisibility FeatureReq = "ReqSetVisibility" // bool

	// put gui output into full-screen mode (ie. no window border and content
	// the size of the monitor).
	ReqFullScreen FeatureReq = "ReqFullScreen" // bool

	// special request for PlusROM cartridges.
	ReqPlusROMFirstInstallation FeatureReq = "ReqPlusROMFirstInstallation" // PlusROMFirstInstallation

	// controller has changed for one of the ports. the string is a description
	// of the controller.
	ReqControllerChange FeatureReq = "ReqControllerChange" // plugging.PortID, string
)
