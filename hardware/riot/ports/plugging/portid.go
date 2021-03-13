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

package plugging

// PortID differentiates the different ports in the VCS into which peripherals
// can be plugged.
type PortID string

// List of defined PortIDs.
//
// We could potentially extend this to support a future Quadtari implementation.
const (
	Unplugged   PortID = "Unplugged"
	LeftPlayer  PortID = "Left"
	RightPlayer PortID = "Right"
	Panel       PortID = "Panel"
)
