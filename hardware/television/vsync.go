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
//
//	RsTennis
//	- always has a valid VSYNC signal but the extent of the frame changes
//	briefly (when the logo bounces on the title screen)
//
//	Tapper
//	- always has a valid VSYNC signal but the extent of the frame changes
//	- fluctuates between 261 and 262

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

	// the scanline the flyback returns to when frame is started as a result of
	// a valid VSYNC signal
	topScanline int
}

func (v *vsync) reset() {
	v.active = false
	v.startScanline = 0
	v.startClock = 0
	v.activeScanlineCount = 0
	v.topScanline = 0
}
