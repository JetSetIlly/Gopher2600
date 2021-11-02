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
	// Send a request to set a GUI feature. Returns an error so that it can be
	// shown in the appropriate context (terminal or the log depending on what
	// was being requested)
	SetFeature(request FeatureReq, args ...FeatureReqData) error
}
