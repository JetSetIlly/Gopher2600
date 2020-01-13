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
type FeatureReq int

// list of valid feature requests. argument must be of the type specified or
// else the interface{} type conversion will fail and the application will
// probably crash
const (
	ReqSetVisibility      FeatureReq = iota // bool
	ReqToggleVisibility                     // none
	ReqSetVisibleOnStable                   // none
	ReqSetPause                             // bool
	ReqSetMasking                           // bool
	ReqToggleMasking                        // none
	ReqSetAltColors                         // bool
	ReqToggleAltColors                      // none
	ReqSetOverlay                           // bool
	ReqToggleOverlay                        // none
	ReqSetScale                             // float
	ReqIncScale                             // none
	ReqDecScale                             // none
	ReqSetFpsCap                            // bool
)

func (r FeatureReq) String() string {
	switch r {
	case ReqSetVisibility:
		return "ReqSetVisibility"
	case ReqToggleVisibility:
		return "ReqToggleVisibility"
	case ReqSetVisibleOnStable:
		return "ReqSetVisibleOnStable"
	case ReqSetPause:
		return "ReqSetPause"
	case ReqSetMasking:
		return "ReqSetMasking"
	case ReqToggleMasking:
		return "ReqToggleMasking"
	case ReqSetAltColors:
		return "ReqSetAltColors"
	case ReqToggleAltColors:
		return "ReqToggleAltColors"
	case ReqSetOverlay:
		return "ReqSetOverlay"
	case ReqToggleOverlay:
		return "ReqToggleOverlay"
	case ReqSetScale:
		return "ReqSetScale"
	case ReqIncScale:
		return "ReqIncScale"
	case ReqDecScale:
		return "ReqDecScale"
	case ReqSetFpsCap:
		return "ReqSetFpsCap"
	}

	return "unknown request"
}
