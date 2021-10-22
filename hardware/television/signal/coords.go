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

// Package signal exposes the interface between the VCS and the television
// implementation.
package signal

// TelevisionCoords represents the state of the TV at any moment in time. It
// can be used when all three values need to be stored or passed around.
//
// Zero value for clock field is -specification.ClksHBlank
type TelevisionCoords struct {
	Frame    int
	Scanline int
	Clock    int
}

// GreaterThan compares two instances of TelevisionCoords and return true if
// both are equal.
func (coords TelevisionCoords) Equal(cmp TelevisionCoords) bool {
	return coords.Frame == cmp.Frame && coords.Scanline == cmp.Scanline && coords.Clock == cmp.Clock
}

// GreaterThanOrEqual compares two instances of TelevisionCoords and return true if
// coords is greater than the cmp parameter.
func (coords TelevisionCoords) GreaterThanOrEqual(cmp TelevisionCoords) bool {
	return coords.Frame > cmp.Frame || (coords.Frame == cmp.Frame && coords.Scanline > cmp.Scanline) || (coords.Frame == cmp.Frame && coords.Scanline == cmp.Scanline && coords.Clock >= cmp.Clock)
}
