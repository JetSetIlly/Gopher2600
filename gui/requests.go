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

package gui

// FeatureReq is used to request the setting of a gui attribute
// eg. toggling the overlay
type FeatureReq string

// List of valid feature requests. argument must be of the type specified or
// else the interface{} type conversion will fail and the application will
// probably crash.
//
// Note that, like the name suggests, these are requests, they may or may not
// be satisifed depending other conditions in the GUI. For example, in the
// imgui gui implementation, a capture mouse request will only be honoured if
// the "TV screen" window is active.
const (
	ReqSetVisibility      FeatureReq = "ReqSetVisibility"      // bool
	ReqToggleVisibility   FeatureReq = "ReqToggleVisibility"   // none
	ReqSetVisibleOnStable FeatureReq = "ReqSetVisibleOnStable" // none
	ReqSetPause           FeatureReq = "ReqSetPause"           // bool
	ReqSetMasking         FeatureReq = "ReqSetMasking"         // bool
	ReqToggleMasking      FeatureReq = "ReqToggleMasking"      // none
	ReqSetAltColors       FeatureReq = "ReqSetAltColors"       // bool
	ReqToggleAltColors    FeatureReq = "ReqToggleAltColors"    // none
	ReqSetOverlay         FeatureReq = "ReqSetOverlay"         // bool
	ReqToggleOverlay      FeatureReq = "ReqToggleOverlay"      // none
	ReqSetScale           FeatureReq = "ReqSetScale"           // float
	ReqIncScale           FeatureReq = "ReqIncScale"           // none
	ReqDecScale           FeatureReq = "ReqDecScale"           // none
	ReqSetFpsCap          FeatureReq = "ReqSetFpsCap"          // bool
	ReqAddDebugger        FeatureReq = "ReqAddDebugger"        // *debugger.Debugger
	ReqAddVCS             FeatureReq = "ReqAddVCS"             // *hardware.VCS
	ReqAddDisasm          FeatureReq = "ReqAddDisasm"          // *disassembly.Disassembly
)
