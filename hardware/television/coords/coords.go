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
// television. A good way to think about them is as a measurement of time. They
// define *when* something happened (this pixel was drawn, this user input was
// received, etc.) relative to the start of the emulation.
//
// They are used throughout the emulation for rewinding, recording/playback and
// many other sub-systems.
package coords

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// FrameIsUndefined is used to indicate that the Frame field of the TelevisionCoords
// struct is to be ignored
const FrameIsUndefined = ^(0)

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
	if c.Frame == FrameIsUndefined {
		return fmt.Sprintf("Scanline: %03d  Clock: %03d", c.Scanline, c.Clock)
	}
	return fmt.Sprintf("Frame: %d  Scanline: %03d  Clock: %03d", c.Frame, c.Scanline, c.Clock)
}

// Equal compares two instances of TelevisionCoords and return true if both are
// equal.
//
// If the Frame field is undefined for either argument then the Frame field is
// ignored for the test.
func Equal(A, B TelevisionCoords) bool {
	if A.Frame == FrameIsUndefined || B.Frame == FrameIsUndefined {
		return A.Scanline == B.Scanline && A.Clock == B.Clock
	}
	return A.Frame == B.Frame && A.Scanline == B.Scanline && A.Clock == B.Clock
}

// GreaterThanOrEqual compares two instances of TelevisionCoords and return
// true if A is greater than or equal to B.
//
// If the Frame field is undefined for either argument then the Frame field is
// ignored for the test.
func GreaterThanOrEqual(A, B TelevisionCoords) bool {
	if A.Frame == FrameIsUndefined || B.Frame == FrameIsUndefined {
		return A.Scanline > B.Scanline || (A.Scanline == B.Scanline && A.Clock >= B.Clock)
	}

	return A.Frame > B.Frame || (A.Frame == B.Frame && A.Scanline > B.Scanline) || (A.Frame == B.Frame && A.Scanline == B.Scanline && A.Clock >= B.Clock)
}

// GreaterThan compares two instances of TelevisionCoords and return true if A
// is greater than to B.
//
// If the Frame field is undefined for either argument then the Frame field is
// ignored for the test.
func GreaterThan(A, B TelevisionCoords) bool {
	if A.Frame == FrameIsUndefined || B.Frame == FrameIsUndefined {
		return A.Scanline > B.Scanline || (A.Scanline == B.Scanline && A.Clock > B.Clock)
	}
	return A.Frame > B.Frame || (A.Frame == B.Frame && A.Scanline > B.Scanline) || (A.Frame == B.Frame && A.Scanline == B.Scanline && A.Clock > B.Clock)
}

// Diff returns the difference between the B and A instances. The
// scanlinesPerFrame value is the number of scanlines in a typical frame for
// the ROM, implying that for best results, the television image should be
// stable.
//
// If the Frame field is undefined for either TelevisionCoords argument then the
// Frame field in the result of the function is also undefined.
func Diff(A, B TelevisionCoords, scanlinesPerFrame int) TelevisionCoords {
	D := TelevisionCoords{
		Frame:    A.Frame - B.Frame,
		Scanline: A.Scanline - B.Scanline,
		Clock:    A.Clock - B.Clock,
	}

	if D.Clock < specification.ClksHBlank {
		D.Scanline--
		D.Clock += specification.ClksScanline
	}

	if D.Scanline < 0 {
		D.Frame--
		D.Scanline += scanlinesPerFrame
	}

	if D.Frame < 0 {
		D.Frame = 0
		D.Scanline = 0
		D.Clock -= specification.ClksScanline
	}

	// if the Frame field in either A or B is undefined then we can set the diff
	// Frame field as undefined alse
	if A.Frame == FrameIsUndefined || B.Frame == FrameIsUndefined {
		D.Frame = FrameIsUndefined
	}

	return D
}

// Sum the the number of clocks in the television coordinates.
//
// If the Frame field is undefined for the TelevisionCoords then the Frame field
// in the result of the function is also undefined.
func Sum(A TelevisionCoords, scanlinesPerFrame int) int {
	if A.Frame == FrameIsUndefined {
		return (A.Scanline * specification.ClksScanline) + A.Clock
	}

	numPerFrame := scanlinesPerFrame * specification.ClksScanline
	return (A.Frame * numPerFrame) + (A.Scanline * specification.ClksScanline) + A.Clock
}
