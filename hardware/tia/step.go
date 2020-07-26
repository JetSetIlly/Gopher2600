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

package tia

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/television"
)

// Step moves the state of the tia forward one video cycle returns the state of
// the CPU's RDY flag.
func (tia *TIA) Step(readMemory bool) (bool, error) {
	// update debugging information
	tia.videoCycles++

	var memoryData bus.ChipData

	// update memory if required
	if readMemory {
		readMemory, memoryData = tia.mem.ChipRead()
	}

	// make alterations to video state and playfield
	if readMemory {
		readMemory = tia.UpdateTIA(memoryData)
	}
	if readMemory {
		readMemory = tia.Video.UpdatePlayfield(tia.Delay, memoryData)
	}

	// tick phase clock
	tia.pclk.Tick()

	// tick delayed events
	tia.Delay.Tick()

	// tick hsync counter when the Phi2 clock is raised. from TIA_HW_Notes.txt:
	//
	// "This table shows the elapsed number of CLK, CPU cycles, Playfield
	// (PF) bits and Playfield pixels at the start of each counter state
	// (ie when the counter changes to this state on the rising edge of
	// the H@2 clock)."
	//
	// the context of this passage is the Horizontal Sync Counter. It is
	// explicitely saying that the HSYNC counter ticks forward on the rising
	// edge of Phi2.
	if tia.pclk.Phi2() {
		tia.hsync.Tick()

		// hsyncDelay is the number of cycles required before, for example, hblank
		// is reset
		const hsyncDelay = 3

		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. the "key" at the end of that table
		// suggests that (most of) the events are delayed by 4 clocks due to
		// "latching".
		switch tia.hsync.Count() {
		case 57:
			// from TIA_HW_Notes.txt:
			//
			// "The HSync counter resets itself after 57 counts; the decode on
			// HCount=56 performs a reset to 000000 delayed by 4 CLK, so
			// HCount=57 becomes HCount=0. This gives a period of 57 counts
			// or 228 CLK."
			tia.hsync.Reset()

			// from TIA_HW_Notes.txt:
			//
			// "Also of note, the HMOVE latch used to extend the HBlank time
			// is cleared when the HSync Counter wraps around. This fact is
			// exploited by the trick that invloves hitting HMOVE on the 74th
			// CPU cycle of the scanline; the CLK stuffing will still take
			// place during the HBlank and the HSYNC latch will be set just
			// before the counter wraps around."
			tia.HmoveLatch = false

		case 56: // [SHB]
			// allow a new scanline event to occur naturally only when an RSYNC
			// has not been scheduled
			if tia.rsyncEvent == nil {
				tia.Delay.Schedule(hsyncDelay, tia.newScanline, "RESET")
			}

		case 4: // [SHS]
			// start HSYNC. start of new scanline for the television
			// * TIA_HW_Notes.txt does not say there is a 4 clock delay for
			// this. not clear if this is the case.
			//
			// !!TODO: check accuracy of HSync timing
			tia.sig.HSync = true

		case 8: // [RHS]
			// reset HSYNC
			tia.Delay.Schedule(hsyncDelay, tia._futureResetHSYNC, "RHS (TV)")

		case 12: // [RCB]
			// reset color burst
			tia.Delay.Schedule(hsyncDelay, tia._futureResetColorBurst, "RCB (TV)")

		// the two cases below handle the turning off of the hblank flag. from
		// TIA_HW_Notes.txt:
		//
		// "In principle the operation of HMOVE is quite straight-forward; if a
		// HMOVE is initiated immediately after HBlank starts, which is the
		// case when HMOVE is used as documented, the [HMOVE] signal is latched
		// and used to delay the end of the HBlank by exactly 8 CLK, or two
		// counts of the HSync Counter. This is achieved in the TIA by
		// resetting the HB (HBlank) latch on the [LRHB] (Late Reset H-Blank)
		// counter decode rather than the normal [RHB] (Reset H-Blank) decode."

		// in practice we have to careful about when HMOVE has been triggered.
		// the condition below for HSYNC=16 includes a test for an active HMOVE
		// event and whether it is about to be completed. we can see the effect
		// of this in particular in the test ROM "games that do bad thing to
		// HMOVE" at value 14

		case 16: // [RHB]
			// early HBLANK off if hmoveLatch is false
			if !tia.HmoveLatch {
				tia.Delay.Schedule(hsyncDelay, tia._futureResetHBlank, "HRB")
			}

		// ... and "two counts of the HSync Counter" later ...

		case 18:
			// late HBLANK off if hmoveLatch is true
			if tia.HmoveLatch {
				tia.Delay.Schedule(hsyncDelay, tia._futureResetHBlank, "LHRB")
			}
		}
	}

	// alter state of video subsystem. occuring after ticking of TIA clock
	// because some the side effects of some registers require that. in
	// particular, the RESxx registers need to have correct information about
	// the state of HBLANK and the HMOVE latch.
	//
	// to see the effect of this, try moving this function call before the
	// HSYNC tick and see how the ball sprite is rendered incorrectly in
	// Keystone Kapers (this is because the ball is reset on the very last
	// pixel and before HBLANK etc. are in the state they need to be)
	if readMemory {
		readMemory = tia.Video.UpdateSpritePositioning(memoryData)
	}
	if readMemory {
		readMemory = tia.Video.UpdateColor(memoryData)
	}

	// "one extra CLK pulse is sent every 4 CLK" and "on every H@1 signal [...]
	// as an extra 'stuffed' clock signal."
	isHmove := tia.pclk.Phi2()

	// we always call TickSprites but whether or not (and how) the tick
	// actually occurs is left for the sprite object to decide based on the
	// arguments passed here.
	tia.Video.Tick(!tia.Hblank, isHmove, tia.HmoveCt)

	// update hmove counter value
	if isHmove {
		if tia.HmoveCt != 0xff {
			tia.HmoveCt--
		}
	}

	// resolve video pixels
	pixelColor := tia.Video.Pixel()
	if tia.Hblank {
		// if hblank is on then we don't sent the resolved color but the video
		// black signal instead
		tia.sig.Pixel = television.VideoBlack
	} else {
		tia.sig.Pixel = television.ColorSignal(pixelColor)
	}

	if readMemory {
		readMemory = tia.Video.UpdateSpriteHMOVE(tia.Delay, memoryData)
	}
	if readMemory {
		readMemory = tia.Video.UpdateSpriteVariations(memoryData)
	}
	if readMemory {
		readMemory = tia.Video.UpdateSpritePixels(memoryData)
	}
	if readMemory {
		readMemory = tia.Audio.UpdateRegisters(memoryData)
	}

	// copy audio to television signal
	tia.sig.AudioUpdate, tia.sig.AudioData = tia.Audio.Mix()

	// send signal to television
	if err := tia.tv.Signal(tia.sig); err != nil {
		// allow out-of-spec errors for now. this should be optional
		if !errors.Is(err, errors.TVOutOfSpec) {
			return !tia.wsync, err
		}
	}

	return !tia.wsync, nil
}
