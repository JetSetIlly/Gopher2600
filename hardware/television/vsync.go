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
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// good test cases for bad VSYNC:
//
//  Lord of the Rings
//	- switching between map screen and play screen
//
//	MsPacman
//	- when bouncing fruit arrives and the player and ghost are at similar
//	vertical position
//
//  Snow White and the Seven Dwarfs
//	- movement in tunnel section
//	- this tests differing VSYNC profile per frame. VSYNC is not actually lost,
//	it just changes during movement

type vsync struct {
	active bool

	// the scanline on which the most recent VSYNC signal started. this is used
	// to populate the FrameInfo VSYNCscanline field
	startScanline int

	// the clock on which the VSYNC was activated
	startClock int

	// number of whole scanlines the VSYNC has been active for. using
	// activeClock as the mark to increase the count
	activeScanlineCount int

	// the ideal scanline at which the "new frame" is triggered. this can be
	// thought of as the number of scanlines between valid VSYNC signals. as
	// such, it is only reset on reception of a valid VSYNC signal
	//
	// that value of this can go way beyond the number of specification.AbsoluteMaxScanlines
	// during the period when there is no VSYNC. it is therefore a good idea to
	// modulo divide this field with AbsoluteMaxScanlines before using it
	//
	// when scanline is equal to flybackScanline then the television is
	// synchronised. see isSynced() function
	scanline int

	// the scanline at which a "new frame" is actually triggered. this will be
	// different than the scanlines field during the synchronisation process causing
	// the screen to visually roll
	flybackScanline int

	// short history of the active field. updated every newFrame(). each bit
	// from LSB to MSB records the active field from most recent to least recent
	//
	// this is likely more than we need but it's simple and it works
	history uint8
}

func (v *vsync) reset() {
	v.active = false
	v.startClock = 0
	v.activeScanlineCount = 0
	v.scanline = 0
	v.flybackScanline = specification.AbsoluteMaxScanlines
	v.startScanline = 0
	v.history = 0
}

func (v vsync) isSynced() bool {
	return v.scanline == v.flybackScanline
}

func (v *vsync) updateHistory() {
	v.history <<= 1
	if v.active {
		v.history |= 0x01
	}
}

func (v *vsync) desync(base int) {
	// move flybackScanline value towards base value. taking into account which
	// value is higher
	if base >= v.flybackScanline {
		v.flybackScanline += (base - v.flybackScanline) * 5 / 100
	} else {
		v.flybackScanline += (v.flybackScanline - base) * 5 / 100
	}

	// do not go past the limits of the TV
	v.flybackScanline = min(v.flybackScanline, specification.AbsoluteMaxScanlines)
}
