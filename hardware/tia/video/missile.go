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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package video

import (
	"fmt"
	"gopher2600/hardware/tia/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/television"
	"strings"
)

type missileSprite struct {
	// see player sprite for detailed commentary on struct attributes

	tv         television.Television
	hblank     *bool
	hmoveLatch *bool

	// ^^^ references to other parts of the VCS ^^^

	position    *polycounter.Polycounter
	pclk        phaseclock.PhaseClock
	Delay       *future.Ticker
	moreHMOVE   bool
	hmove       uint8
	lastHmoveCt uint8

	// the following attributes are used for information purposes only:

	label       string
	resetPixel  int
	hmovedPixel int

	// note whether the last tick was as a result of a HMOVE stuffing tick
	lastTickFromHmove bool

	// ^^^ the above are common to all sprite types ^^^
	//		(see player sprite for commentary)

	enabled bool
	color   uint8

	// for the missile sprite we split the NUSIZx register into size and copies
	size   uint8
	copies uint8

	enclockifier       enclockifier
	parentPlayer       *playerSprite
	resetToPlayer      bool
	startDrawingEvent  *future.Event
	resetPositionEvent *future.Event
}

func newMissileSprite(label string, tv television.Television, hblank, hmoveLatch *bool) (*missileSprite, error) {
	ms := missileSprite{
		tv:         tv,
		hblank:     hblank,
		hmoveLatch: hmoveLatch,
		label:      label,
	}

	var err error

	ms.position, err = polycounter.New(6)
	if err != nil {
		return nil, err
	}

	ms.Delay = future.NewTicker(label)

	ms.enclockifier.size = &ms.size
	ms.enclockifier.pclk = &ms.pclk
	ms.enclockifier.delay = ms.Delay
	ms.position.Reset()
	return &ms, nil

}

// Label returns the label for the sprite
func (ms missileSprite) Label() string {
	return ms.label
}

func (ms missileSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation
	normalisedHmove := int(ms.hmove) - 8
	if normalisedHmove < 0 {
		normalisedHmove = 16 + normalisedHmove
	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s [%03d ", ms.position, ms.pclk, ms.resetPixel))
	s.WriteString(fmt.Sprintf("> %#1x >", normalisedHmove))
	s.WriteString(fmt.Sprintf(" %03d", ms.hmovedPixel))
	if ms.moreHMOVE {
		s.WriteString("*] ")
	} else {
		s.WriteString("] ")
	}

	// interpret nusiz value
	switch ms.copies {
	case 0x0:
		s.WriteString("|")
	case 0x1:
		s.WriteString("|_|")
	case 0x2:
		s.WriteString("|__|")
	case 0x3:
		s.WriteString("|_|_|")
	case 0x4:
		s.WriteString("|___|")
	case 0x6:
		s.WriteString("|__|__|")
	}

	notes := false

	switch ms.size {
	case 0x0:
	case 0x1:
		s.WriteString(" 2x")
		notes = true
	case 0x2:
		s.WriteString(" 4x")
		notes = true
	case 0x3:
		s.WriteString(" 8x")
		notes = true
	}

	if ms.moreHMOVE {
		s.WriteString(" hmoving")
		s.WriteString(fmt.Sprintf(" [%04b]", ms.hmove))
		notes = true
	}

	if ms.enclockifier.enable {
		// add a comma if we've already noted something else
		if notes {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw %s", ms.enclockifier.String()))
		notes = true
	}

	if !ms.enabled {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" disb")
		notes = true
	}

	if ms.resetToPlayer {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" >pl<")
	}

	return s.String()
}

func (ms *missileSprite) rsync(adjustment int) {
	ms.resetPixel -= adjustment
	ms.hmovedPixel -= adjustment
	if ms.resetPixel < 0 {
		ms.resetPixel += television.HorizClksVisible
	}
	if ms.hmovedPixel < 0 {
		ms.hmovedPixel += television.HorizClksVisible
	}
}

func (ms *missileSprite) tick(visible, isHmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if isHmove {
		ms.moreHMOVE = ms.moreHMOVE && compareHMOVE(hmoveCt, ms.hmove)
	}

	ms.lastHmoveCt = hmoveCt

	// reset missile to player position. from TIA_HW_Notes.txt:
	//
	// "The Missile-to-player reset is implemented by resetting the M0 counter
	// when the P0 graphics scan counter is at %100 (in the middle of drawing
	// the player graphics) AND the main copy of P0 is being drawn (ie the
	// missile counter will not be reset when a subsequent copy is drawn, if
	// any). This second condition is generated from a latch outputting [FSTOB]
	// that is reset when the P0 counter wraps around, and set when the START
	// signal is decoded for a 'close', 'medium' or 'far' copy of P0."
	//
	// note: the FSTOB output is the primary flag in the parent player's
	// scancounter
	if ms.resetToPlayer && ms.parentPlayer.scanCounter.cpy == 0 && ms.parentPlayer.scanCounter.isMissileMiddle() {
		ms.position.Reset()
		ms.pclk.Reset()
	}

	// early return if nothing to do
	if !(isHmove && ms.moreHMOVE) && !visible {
		return
	}

	// note whether this is an additional hmove tick. see pixel() function
	// below for explanation
	ms.lastTickFromHmove = isHmove && ms.moreHMOVE

	// update hmoved pixel value
	if !visible {
		ms.hmovedPixel--

		// adjust for screen boundary
		if ms.hmovedPixel < 0 {
			ms.hmovedPixel += television.HorizClksVisible
		}
	}

	ms.pclk.Tick()

	if ms.pclk.Phi2() {
		ms.position.Tick()

		// start drawing if there is no reset or it has just started AND
		// there wasn't a reset event ongoing when the current event
		// started
		if ms.resetPositionEvent == nil || ms.resetPositionEvent.JustStarted() {
			switch ms.position.Count() {
			case 3:
				if ms.copies == 0x01 || ms.copies == 0x03 {
					ms.startDrawingEvent = ms.Delay.ScheduleWithArg(4, ms._futureStartDrawingEvent, 1, "START")
				}
			case 7:
				if ms.copies == 0x03 || ms.copies == 0x02 || ms.copies == 0x06 {
					cpy := 1
					if ms.copies == 0x03 {
						cpy = 2
					}
					ms.startDrawingEvent = ms.Delay.ScheduleWithArg(4, ms._futureStartDrawingEvent, cpy, "START")
				}
			case 15:
				if ms.copies == 0x04 || ms.copies == 0x06 {
					cpy := 1
					if ms.copies == 0x06 {
						cpy = 2
					}
					ms.startDrawingEvent = ms.Delay.ScheduleWithArg(4, ms._futureStartDrawingEvent, cpy, "START")
				}
			case 39:
				ms.startDrawingEvent = ms.Delay.ScheduleWithArg(4, ms._futureStartDrawingEvent, 0, "START")
			case 40:
				ms.position.Reset()
			}
		}
	}

	// tick future events that are goverened by the sprite
	ms.Delay.Tick()
}

func (ms *missileSprite) _futureStartDrawingEvent(v interface{}) {
	ms.enclockifier.start()
	ms.enclockifier.cpy = v.(int)
	ms.startDrawingEvent = nil
}

func (ms *missileSprite) prepareForHMOVE() {
	ms.moreHMOVE = true

	if *ms.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the String() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		ms.hmovedPixel += 8

		// adjust for screen boundary
		if ms.hmovedPixel > television.HorizClksVisible {
			ms.hmovedPixel -= television.HorizClksVisible
		}
	}
}

func (ms *missileSprite) resetPosition() {
	// see player sprite resetPosition() for commentary on delay values
	delay := 4
	if *ms.hblank {
		if !*ms.hmoveLatch || ms.lastHmoveCt >= 1 && ms.lastHmoveCt <= 15 {
			delay = 2
		} else {
			delay = 3
		}
	}

	// drawing of missile sprite is paused and will resume upon reset
	// completion. compare to ball sprite where drawing is ended and then
	// re-started under all conditions
	//
	// important to note we only pause if the draw/start events are not about
	// to end. in other words, if they are not about to end they are allowed to
	// continue naturally while reset event is waiting to conclude
	if !ms.enclockifier.aboutToEnd() {
		ms.enclockifier.pause()
	}
	if ms.startDrawingEvent != nil && !ms.startDrawingEvent.AboutToEnd() {
		ms.startDrawingEvent.Pause()
	}

	// stop any existing reset events. generally, this codepath will not apply
	// because a resetPositionEvent will conclude before being triggered again.
	// but it is possible when using a very quick instruction on the reset register,
	// like a zero page INC, for requests to overlap
	//
	// in the case of the missile sprite, we can see such an occurance in the
	// test.bin test ROM
	if ms.resetPositionEvent != nil {
		ms.resetPositionEvent.Push()
		return
	}

	ms.resetPositionEvent = ms.Delay.Schedule(delay, ms._futureResetPosition, "RESMx")
}

func (ms *missileSprite) _futureResetPosition() {
	// the pixel at which the sprite has been reset, in relation to the
	// left edge of the screen
	ms.resetPixel, _ = ms.tv.GetState(television.ReqHorizPos)

	if ms.resetPixel >= 0 {
		// resetPixel adjusted by 1 because the tv is not yet in the correct
		// position
		ms.resetPixel++

		// adjust resetPixel for screen boundaries
		if ms.resetPixel > television.HorizClksVisible {
			ms.resetPixel -= television.HorizClksVisible
		}

		// by definition the current pixel is the same as the reset pixel at
		// the moment of reset
		ms.hmovedPixel = ms.resetPixel
	} else {
		// if reset occurs off-screen then force reset pixel to be zero
		// (see commentary in ball sprite for detailed reasoning of this
		// branch)
		ms.resetPixel = 0
		ms.hmovedPixel = 7
	}

	// reset both sprite position and clock
	ms.position.Reset()
	ms.pclk.Reset()

	ms.enclockifier.force()
	if ms.startDrawingEvent != nil {
		ms.startDrawingEvent.Force()
		ms.startDrawingEvent = nil
	}

	// dump reference to reset event
	ms.resetPositionEvent = nil
}

func (ms *missileSprite) setResetToPlayer(on bool) {
	ms.resetToPlayer = on
}

func (ms *missileSprite) pixel() (bool, uint8) {
	if !ms.enabled {
		return false, ms.color
	}

	// the missile sprite has a special state where a stuffed HMOVE clock
	// causes the sprite to this the start signal has happened one cycle early.
	earlyStart := ms.lastTickFromHmove && ms.startDrawingEvent != nil && ms.startDrawingEvent.AboutToEnd()

	// similarly in the event of a stuffed HMOVE clock, and when the
	// enclockifier is about to produce its last pixel
	//
	// see ball sprite for explanation for the LatePhi1() condition
	earlyEnd := !ms.pclk.LatePhi1() && ms.lastTickFromHmove && ms.enclockifier.aboutToEnd()

	// both conditions are fully explained in the AtariAge post "Cosmic Ark
	// Star Field Revisited" by crispy. as suggested by the post title this is
	// the key to implementing the starfield in the Cosmic Ark ROM

	// whether a pixel is output also depends on whether resetToPlayer is off
	px := !ms.resetToPlayer && !earlyEnd && (ms.enclockifier.enable || earlyStart)

	return px, ms.color
}

func (ms *missileSprite) setEnable(enable bool) {
	ms.enabled = enable
}

func (ms *missileSprite) setNUSIZ(value uint8) {
	ms.size = (value & 0x30) >> 4
	ms.copies = value & 0x07
}

func (ms *missileSprite) setColor(value uint8) {
	ms.color = value
}

func (ms *missileSprite) setHmoveValue(v interface{}) {
	ms.hmove = (v.(uint8) ^ 0x80) >> 4
}

func (ms *missileSprite) clearHmoveValue() {
	ms.hmove = 0x08
}
