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

// Package specification contains the definitions, including colour, of the PAL
// and NTSC television protocols supported by the emulation.
package specification

// Rotation indicates the orienation of the television
type Rotation int

// List of valid Rotation values. The values are arranged so that they can be
// thought of as the number of 90 degrees turns to get to that position.
// Alternatively, multiplying the Rotation value by 1.5708 will give the number
// of radians required for the rotation
const (
	NormalRotation Rotation = iota
	LeftRotation
	FlippedRotation
	RightRotation
)

func (r Rotation) String() string {
	switch r {
	case NormalRotation:
		return "normal"
	case LeftRotation:
		return "left"
	case FlippedRotation:
		return "flipped"
	case RightRotation:
		return "right"
	}
	return "unknown rotation"
}
