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

import "github.com/jetsetilly/gopher2600/curated"

type Stub struct{}

// SetFeature implements the GUI interface.
func (s Stub) SetFeature(request FeatureReq, args ...FeatureReqData) error {
	return curated.Errorf(UnsupportedGuiFeature, request)
}

// SetFeatureNoError implements the GUI interface.
func (s Stub) SetFeatureNoError(request FeatureReq, args ...FeatureReqData) {
}

// GetFeature implements the GUI interface.
func (s Stub) GetFeature(request FeatureReq) (FeatureReqData, error) {
	return nil, curated.Errorf(UnsupportedGuiFeature, request)
}
