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

// GUI defines the operations that can be performed on visual user interfaces.
type GUI interface {
	// Send a request to set a GUI feature.
	SetFeature(request FeatureReq, args ...FeatureReqData) error

	// Same as SetFeature() but not waiting for the result. Useful in time
	// critical situations when you are absolutely sure there will be no
	// errors that need handling.
	SetFeatureNoError(request FeatureReq, args ...FeatureReqData)

	// Return current state of GUI feautre.
	GetFeature(request FeatureReq) (FeatureReqData, error)
}

// Sentinal error returned if GUI does no support requested feature.
const (
	UnsupportedGuiFeature = "unsupported gui feature: %v"
)
