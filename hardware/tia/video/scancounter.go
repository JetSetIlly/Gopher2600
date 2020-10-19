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

package video

import (
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
)

// scancounter is the mechanism for outputting player sprite pixels. it is the
// equivalent of the enclockifier type used by the ball and missile sprite.
// scancounter is used only by the player sprite
//
// once a player sprite has reached a START signal during its polycounter
// cycle, the scanCounter is started and is ticked forward every cycle (subject
// to MOTCK, HMOVE and NUSIZ rules).
type scanCounter struct {
	sizeAndCopies *uint8
	pclk          *phaseclock.PhaseClock

	// LatchesSizeAndCopies is used to decide how often to tick the scan
	// counter onto the next pixel (after the additional latching, see below).
	// in most situations we could use the live nusiz value (see above) but
	// resetting the player sprite while the scan counter is active requires
	// some special handling
	LatchedSizeAndCopies uint8

	// number of additional ticks required before drawing begins
	latch int

	// Pixel counts from 7 to -1 for a total of 8 active pixels. we're counting
	// backwards because it is more convenient for the Pixel() function
	Pixel int

	// for the wider player sizes, real ticks are only made every two or every
	// four clocks. 'count' records how many ticks the scanCounter has been on
	// the current pixel value
	count int

	// which copy of the sprite is being drawn. value of zero means the primary
	// copy is being drawn (if enable is true)
	Cpy int
}

func (sc *scanCounter) start() {
	if sc.LatchedSizeAndCopies == 0x05 || sc.LatchedSizeAndCopies == 0x07 {
		sc.latch = 2
	} else {
		sc.latch = 1
	}
}

// IsActive returns true is scan counter is currently outputting pixels.
func (sc scanCounter) IsActive() bool {
	return sc.Pixel != -1
}

// IsLatching returns true if scan counter is about to commence outputting
// pixels.
func (sc scanCounter) IsLatching() bool {
	return sc.latch > 0
}

func (sc *scanCounter) tick() {
	// handle the additional latching
	if sc.latch > 0 {
		sc.latch--
		if sc.latch == 0 {
			sc.count = 0
			sc.Pixel = 7
			sc.LatchedSizeAndCopies = *sc.sizeAndCopies
		}
		return
	}

	tick := true

	// tick pixels differently depending on whether this is the primary copy or
	// the secondary copies. this is all a but magical for my liking but it
	// works and there's some sense to it at least.
	//
	// for the primary copy, we delay the use of the live nusiz value until the
	// correct clock phase. once the nusiz value has been latched then we tick
	// according to how long the scancounter has been on the current pixel
	//
	// for the secondary copies we always use the live nusiz value and a skewed
	// phase-clock. not sure why the skewed clock is required but the effects
	// can clearly be seen with test/test_roms/testSize2Copies_B.bin

	if sc.Cpy == 0 {
		// latch the nusiz value depending on the phase of the player clock
		if *sc.sizeAndCopies == 0x05 {
			if sc.pclk.Phi1() || sc.pclk.Phi2() {
				sc.LatchedSizeAndCopies = *sc.sizeAndCopies
			}
		} else if *sc.sizeAndCopies == 0x07 {
			if sc.pclk.Phi1() {
				sc.LatchedSizeAndCopies = *sc.sizeAndCopies
			}
		} else {
			sc.LatchedSizeAndCopies = *sc.sizeAndCopies
		}

		if sc.LatchedSizeAndCopies == 0x05 {
			if sc.count < 1 {
				tick = false
			}
		} else if sc.LatchedSizeAndCopies == 0x07 {
			if sc.count < 3 {
				tick = false
			}
		}
	} else {
		// timing of ticks for non-primary copies is skewed
		if *sc.sizeAndCopies == 0x05 {
			if !(sc.pclk.LatePhi2() || sc.pclk.LatePhi1()) {
				tick = false
			}
		} else if *sc.sizeAndCopies == 0x07 {
			if !sc.pclk.LatePhi2() {
				tick = false
			}
		}
	}

	if tick {
		if sc.Pixel >= 0 {
			sc.count = 0
			sc.Pixel--

			// default to primary copy whenever we finish drawing. we need this
			// otherwise the above branch, sc.cpy == 0, will not trigger
			// correctly in all instances - if we look at the Player.tick()
			// function we can see why. scanCounter.cpy is set only when the
			// startDrawingEvent has concluded but we need to update the
			// latchedSizeAndCopies value for the primary copy before then.
			if sc.Pixel < 0 {
				sc.Cpy = 0
			}
		}
	} else {
		sc.count++
	}
}
