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

package television

import (
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// AdjCoords returns a coords.TelevisionCoords with the current coords adjusted
// by the specified amount.
func (tv *Television) AdjCoords(adj Adj, amount int) coords.TelevisionCoords {
	coords := tv.GetCoords()

	switch adj {
	case AdjCycle:
		// adjusting by CPU cycle is the same as adjusting by video cycle
		// accept to say that a CPU cycle is the equivalent of 3 video cycles
		amount *= 3
		fallthrough
	case AdjClock:
		coords.Clock += amount
		if coords.Clock >= specification.ClksScanline {
			coords.Clock -= specification.ClksScanline
			coords.Scanline++
		} else if coords.Clock < -specification.ClksHBlank {
			coords.Clock += specification.ClksScanline
			coords.Scanline--
		}
		if coords.Scanline > tv.state.frameInfo.TotalScanlines {
			coords.Scanline -= tv.state.frameInfo.TotalScanlines
			coords.Frame++
		} else if coords.Scanline < 0 {
			coords.Scanline += tv.state.frameInfo.TotalScanlines
			coords.Frame--
		}
	case AdjScanline:
		coords.Clock = -specification.ClksHBlank
		coords.Scanline += amount
		if coords.Scanline > tv.state.frameInfo.TotalScanlines {
			coords.Scanline -= tv.state.frameInfo.TotalScanlines
			coords.Frame++
		} else if coords.Scanline < 0 {
			coords.Scanline += tv.state.frameInfo.TotalScanlines
			coords.Frame--
		}
	case AdjFrame:
		coords.Clock = -specification.ClksHBlank
		coords.Scanline = 0
		coords.Frame += amount
	}

	// zero values if frame is less than zero
	if coords.Frame < 0 {
		coords.Frame = 0
		coords.Scanline = 0
		coords.Clock = -specification.ClksHBlank
	}

	return coords
}

// Adj is used to specify adjustment scale for the ReqAdjust() function.
type Adj int

// List of valid Adj values.
const (
	AdjFrame Adj = iota
	AdjScanline
	AdjCycle
	AdjClock
)
