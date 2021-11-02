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
