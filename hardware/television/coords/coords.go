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

// Package coords represents and can work with television coorindates
//
// Coordinates represent the state of the emulation from the point of the
// television and are used throughout the emulation for rewinding,
// recording/playback and general information.
package coords

import "fmt"

// TelevisionCoords represents the state of the TV at any moment in time. It
// can be used when all three values need to be stored or passed around.
//
// Zero value for clock field is -specification.ClksHBlank
type TelevisionCoords struct {
	Frame    int
	Scanline int
	Clock    int
}

func (c TelevisionCoords) String() string {
	return fmt.Sprintf("Frame: %-4d  Scanline: %-3d    Clock: %-3d",
		c.Frame, c.Scanline, c.Clock)
}

// Equal compares two instances of TelevisionCoords and return true if
// both are equal.
func Equal(A, B TelevisionCoords) bool {
	return A.Frame == B.Frame && A.Scanline == B.Scanline && A.Clock == B.Clock
}

// GreaterThanOrEqual compares two instances of TelevisionCoords and return
// true if A is greater than or equal to B.
func GreaterThanOrEqual(A, B TelevisionCoords) bool {
	return A.Frame > B.Frame || (A.Frame == B.Frame && A.Scanline > B.Scanline) || (A.Frame == B.Frame && A.Scanline == B.Scanline && A.Clock >= B.Clock)
}

// GreaterThan compares two instances of TelevisionCoords and return true if A
// is greater than to B.
func GreaterThan(A, B TelevisionCoords) bool {
	return A.Frame > B.Frame || (A.Frame == B.Frame && A.Scanline > B.Scanline) || (A.Frame == B.Frame && A.Scanline == B.Scanline && A.Clock > B.Clock)
}
