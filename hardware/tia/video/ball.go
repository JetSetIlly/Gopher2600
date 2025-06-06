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
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/delay"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
)

// BallSizes maps ball size values to descriptions of those sizes.
var BallSizes = []string{
	"single size",
	"double size",
	"quad size",
	"double-quad size",
}

// ballSizesBrief maps ball size values to brief descriptions of those sizes.
var ballSizesBrief = []string{
	"",
	"2x",
	"4x",
	"8x",
}

// BallSprite represents the moveable ball sprite in the VCS graphical display.
type BallSprite struct {
	tia tia

	// ^^^ references to other parts of the VCS ^^^

	position    polycounter.Polycounter
	pclk        phaseclock.PhaseClock
	MoreHMOVE   bool
	Hmove       uint8
	lastHmoveCt uint8

	// the following attributes are used for information purposes only:

	label       string
	ResetPixel  int
	HmovedPixel int

	// note whether the last tick was as a result of a HMOVE stuffing tick
	// which left MoreHMOVE in a true state
	lastTickFromHmove bool

	// ^^^ the above are common to all sprite types ^^^
	// (see player sprite for commentary)

	//  the ball color should match the color of the playfield foreground.
	//  however, for debugging purposes it is sometimes useful to use different
	//  colors, so this is not a pointer to playfield.ForegroundColor, as you
	//  might expect
	Color uint8

	// for convenience we store the raw CTRLPF register value and the
	// normalised size bits
	Ctrlpf uint8
	Size   uint8

	VerticalDelay bool
	Enabled       bool
	EnabledDelay  bool

	// position reset and enclockifier start events are both delayed by a small
	// number of cycles
	futureReset delay.Event
	futureStart delay.Event

	// outputting of pixels is handled by the ball/missile enclockifier
	Enclockifier enclockifier

	// state of player "pixel"
	pixelOn        bool
	pixelCollision bool
}

func newBallSprite(label string, tia tia) *BallSprite {
	bs := &BallSprite{
		tia:   tia,
		label: label,
	}

	bs.Enclockifier.size = &bs.Size
	bs.position = 0

	return bs
}

// Snapshot creates a copy of the ball in its current state.
func (bs *BallSprite) Snapshot() *BallSprite {
	n := *bs
	return &n
}

// Plumb changes into ball sprite.
func (bs *BallSprite) Plumb(tia tia) {
	bs.tia = tia
	bs.Enclockifier.size = &bs.Size
}

// Label returns an appropriate name for the sprite.
func (bs *BallSprite) Label() string {
	return bs.label
}

func (bs *BallSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation
	normalisedHmove := int(bs.Hmove) - 8
	if normalisedHmove < 0 {
		normalisedHmove = 16 + normalisedHmove
	}

	s := strings.Builder{}
	s.WriteString(bs.label)
	s.WriteString(": ")
	s.WriteString(fmt.Sprintf("%s %s [%03d ", bs.position, bs.pclk, bs.ResetPixel))
	s.WriteString(fmt.Sprintf("> %#1x >", normalisedHmove))
	s.WriteString(fmt.Sprintf(" %03d", bs.HmovedPixel))
	if bs.MoreHMOVE {
		s.WriteString("*]")
	} else {
		s.WriteString("]")
	}

	if int(bs.Size) > len(ballSizesBrief) {
		panic("illegal size value for ball")
	}
	sz := ballSizesBrief[bs.Size]
	if len(sz) > 0 {
		s.WriteString(" ")
	}
	s.Write([]byte(sz))
	s.WriteString(",")

	if bs.MoreHMOVE {
		s.WriteString(fmt.Sprintf(" hmoving [%04b],", bs.Hmove))
	}

	if bs.Enclockifier.Active {
		s.WriteString(fmt.Sprintf(" drw %s,", bs.Enclockifier.String()))
	}

	if !bs.Enabled {
		s.WriteString(" disb,")
	}

	if bs.VerticalDelay {
		s.WriteString(" vdel,")
	}

	return strings.TrimSuffix(s.String(), ",")
}

func (bs *BallSprite) rsync(adjustment int) {
	bs.ResetPixel -= adjustment
	bs.HmovedPixel -= adjustment
	if bs.ResetPixel < 0 {
		bs.ResetPixel += specification.ClksVisible
	}
	if bs.HmovedPixel < 0 {
		bs.HmovedPixel += specification.ClksVisible
	}
}

func (bs *BallSprite) tickHBLANK() bool {
	// check to see if there is more movement required for this sprite
	bs.MoreHMOVE = bs.MoreHMOVE && compareHMOVE(bs.tia.hmove.Ripple, bs.Hmove)
	if !bs.MoreHMOVE {
		return false
	}

	// update hmoved pixel value & adjust for screen boundary
	bs.HmovedPixel--
	if bs.HmovedPixel < 0 {
		bs.HmovedPixel += specification.ClksVisible
	}

	bs.lastTickFromHmove = bs.MoreHMOVE

	return bs.tick()
}

func (bs *BallSprite) tickHMOVE() bool {
	// check to see if there is more movement required for this sprite
	bs.MoreHMOVE = bs.MoreHMOVE && compareHMOVE(bs.tia.hmove.Ripple, bs.Hmove)

	// cancel motion clock if necessary
	if bs.MoreHMOVE && bs.tia.env.Prefs.Revision.Live.LostMOTCK.Load().(bool) {
		return false
	}

	bs.lastTickFromHmove = bs.MoreHMOVE

	return bs.tick()
}

// returns true if pixel has changed.
func (bs *BallSprite) tick() bool {
	bs.lastHmoveCt = bs.tia.hmove.Ripple

	bs.pclk++
	if bs.pclk >= phaseclock.NumStates {
		bs.pclk = 0
	}

	if bs.pclk == phaseclock.RisingPhi2 {
		bs.position++
		if bs.position >= polycounter.LenTable6Bit {
			bs.position = 0
		}

		switch bs.position {
		case 39:
			bs.futureStart.Schedule(4, 0)
		case 40:
			bs.position = polycounter.ResetValue
		}
	}

	bs.Enclockifier.tick()

	// tick delayed events
	bs.futureReset.Tick(bs._futureResetPosition)
	bs.futureStart.Tick(bs._futureStartDrawingEvent)

	return true
}

func (bs *BallSprite) _futureStartDrawingEvent(_ uint8) {
	bs.Enclockifier.start()
}

func (bs *BallSprite) prepareForHMOVE() {
	bs.MoreHMOVE = true

	if *bs.tia.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the String() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		bs.HmovedPixel += 8

		// adjust for screen boundary
		if bs.HmovedPixel > specification.ClksVisible {
			bs.HmovedPixel -= specification.ClksVisible
		}
	}
}

func (bs *BallSprite) resetPosition() {
	// see player sprite resetPosition() for commentary on delay values
	delay := 4
	if *bs.tia.hblank {
		// the difference between a delay of 2 and 3 seems to be whether the
		// HMOVE is currently rippling. I'm not sure why the test should be
		// "greater than zero" rather than "greater than or equal to zero"
		//
		// good tests for identifying the condition so far are the "reset_during_hmove/ball" roms
		// and also Activision's Seaquest. The scuba divers are drawn with the ball sprite in
		// that game. a delay of 2 causes the diver to just be visible on the right of the
		// screen (in the wings, as it were)
		if bs.tia.hmove.Ripple > 0 {
			delay = 2
		} else {
			delay = 3
		}
	}

	// drawing of ball sprite must end immediately upon a reset strobe. it will
	// start drawing again after the reset delay period
	bs.Enclockifier.drop()
	if bs.futureStart.IsActive() {
		bs.futureStart.Drop()
	}

	// stop any existing reset events. generally, this codepath will not apply
	// because a resetPositionEvent will conculde before being triggere again.
	// but it is possible when using a very quick instruction on the reset register,
	// like a zero page INC, for requests to overlap
	if bs.futureReset.IsActive() {
		bs.futureReset.Push()
		return
	}

	bs.futureReset.Schedule(delay, 0)
}

func (bs *BallSprite) _futureResetPosition(_ uint8) {
	// end drawing of sprite in case it has started during the delay
	// period. believe it or not, we can get rid of this and pixel output
	// will still be correct (because of how the delayed END signal in the
	// enclockifier works) but debugging information will be confusing if
	// we did this.
	bs.Enclockifier.drop()
	if bs.futureStart.IsActive() {
		bs.futureStart.Drop()
	}

	// the pixel at which the sprite has been reset, in relation to the
	// left edge of the screen
	bs.ResetPixel = bs.tia.tv.GetCoords().Clock

	if bs.ResetPixel >= 0 {
		// resetPixel adjusted by 1 because the tv is not yet in the correct
		// position
		bs.ResetPixel++

		// adjust resetPixel for screen boundaries
		if bs.ResetPixel > specification.ClksVisible {
			bs.ResetPixel -= specification.ClksVisible
		}

		// by definition the current pixel is the same as the reset pixel at
		// the moment of reset
		bs.HmovedPixel = bs.ResetPixel
	} else {
		// if reset occurs off-screen then force reset pixel to be zero
		bs.ResetPixel = 0

		// a reset of this kind happens when the reset register has been
		// strobed but not completed before the HBLANK period, and a HMOVE
		// forces the reset to occur.

		// setting hmovedPixel below: I'm not sure about the value of 7 at
		// all; but I couldn't figure out how to derive it algorithmically.
		//
		// observation of Keystone Kapers suggests that it's okay
		// (scanlines being 62 and 97 two slightly different scenarios
		// where the value is correct)
		//
		// also a very rough test ROM tries a couple of things to the same
		// effect: test/my_test_roms/ball/late_reset.bin
		bs.HmovedPixel = 7
	}

	// reset both sprite position and clock
	bs.position = polycounter.ResetValue
	bs.pclk = phaseclock.ResetValue

	// from TIA_HW_Notes.txt:
	//
	// If you look closely at the START signal for the ball, unlike all
	// the other position counters - the ball reset RESBL does send a START
	// signal for graphics output! This makes the ball incredibly useful
	// since you can trigger it as many times as you like across the same
	// scanline and it will start drawing immediately each time :)
	bs.Enclockifier.start()
}

func (bs *BallSprite) pixel() {
	if (!bs.EnabledDelay && bs.VerticalDelay) || (!bs.Enabled && !bs.VerticalDelay) {
		bs.pixelOn = false
		bs.pixelCollision = false
		return
	}

	// earlyStart condition the same as for missile sprites. see missile
	// pixel() function for details
	earlyStart := bs.lastTickFromHmove && bs.futureStart.AboutToEnd()

	// the LatePhi1() condition has been added to accommodate a artefact in
	// (on?) "Spike's Peak". On the first screen, there is a break in the path
	// at the base of the mountain (which is correct) but without the
	// LatePhi1() condition there is also a second break later on the path
	// (which I don't believe should be there)
	earlyEnd := bs.pclk != phaseclock.FallingPhi1 && bs.lastTickFromHmove && bs.Enclockifier.aboutToEnd()

	// Well blow me down! moving the cosmic ark star problem solution from the
	// missile implementation and I can now see that I've already solved the
	// first half of the problem for the ball sprite, almost by accident.
	//
	// Commenting and Keeping the original code for amusement.
	//
	// // the ball sprite pixel is drawn under specific conditions
	// px := bs.enclockifier.Active ||
	//	(bs.lastTickFromHmove && bs.startDrawingEvent != nil && bs.startDrawingEvent.AboutToEnd())

	bs.pixelOn = !earlyEnd && (bs.Enclockifier.Active || earlyStart)

	if bs.VerticalDelay {
		bs.pixelCollision = bs.pixelOn || (*bs.tia.hblank && bs.EnabledDelay && bs.futureStart.AboutToEnd())
	}

	bs.pixelCollision = bs.pixelOn || (*bs.tia.hblank && bs.Enabled && bs.futureStart.AboutToEnd())

	// reset lastTickFromHmove flag
	bs.lastTickFromHmove = false
}

// the delayed enable bit is set when the gfx register for player 1 is updated.
func (bs *BallSprite) setEnableDelay() {
	bs.EnabledDelay = bs.Enabled
}

func (bs *BallSprite) setEnable(enable bool) {
	bs.Enabled = enable
}

func (bs *BallSprite) setVerticalDelay(vdelay bool) {
	bs.VerticalDelay = vdelay
}

func (bs *BallSprite) SetCTRLPF(value uint8) {
	bs.Ctrlpf = value
	bs.Size = (value & 0x30) >> 4
}

func (bs *BallSprite) setColor(value uint8) {
	bs.Color = value
}

func (bs *BallSprite) setHmoveValue(v uint8) {
	bs.Hmove = (v ^ 0x80) >> 4
}

func (bs *BallSprite) clearHmoveValue() {
	bs.Hmove = 0x08
}
