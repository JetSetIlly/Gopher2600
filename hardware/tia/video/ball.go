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
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/tia/future"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
	"github.com/jetsetilly/gopher2600/television"
)

// BallSizes maps ball size values to descriptions of those sizes
var BallSizes = []string{
	"single size",
	"double size",
	"quad size",
	"double-quad size",
}

// ballSizesBrief maps ball size values to brief descriptions of those sizes
var ballSizesBrief = []string{
	"",
	"2x",
	"4x",
	"8x",
}

type ballSprite struct {
	// see player sprite for detailed commentary on struct attributes

	tv         television.Television
	hblank     *bool
	hmoveLatch *bool

	// ^^^ references to other parts of the VCS ^^^

	position    *polycounter.Polycounter
	pclk        phaseclock.PhaseClock
	Delay       *future.Ticker
	MoreHMOVE   bool
	Hmove       uint8
	lastHmoveCt uint8

	// the following attributes are used for information purposes only:

	label       string
	ResetPixel  int
	HmovedPixel int

	// note whether the last tick was as a result of a HMOVE stuffing tick
	lastTickFromHmove bool

	// ^^^ the above are common to all sprite types ^^^
	//		(see player sprite for commentary)

	//  the ball color should match the color of the playfield foreground.
	//  however, for debugging purposes it is sometimes useful to use different
	//  colors, so this is not a pointer to playfield.ForegroundColor, as you
	//  might expect
	Color uint8

	// for convenience we store the raw CTRLPF register value and the
	// normalised size bits
	Ctrlpf uint8
	Size   uint8

	VerticalDelay      bool
	Enabled            bool
	EnabledDelay       bool
	Enclockifier       enclockifier
	startDrawingEvent  *future.Event
	resetPositionEvent *future.Event
}

func newBallSprite(label string, tv television.Television, hblank, hmoveLatch *bool) (*ballSprite, error) {
	bs := ballSprite{
		tv:         tv,
		hblank:     hblank,
		hmoveLatch: hmoveLatch,
		label:      label,
	}

	var err error

	bs.position, err = polycounter.New(6)
	if err != nil {
		return nil, err
	}

	bs.Delay = future.NewTicker(label)

	bs.Enclockifier.delay = bs.Delay
	bs.Enclockifier.size = &bs.Size

	bs.position.Reset()

	return &bs, nil
}

// Label returns the label for the sprite
func (bs ballSprite) Label() string {
	return bs.label
}

func (bs ballSprite) String() string {
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

	notes := false

	if int(bs.Size) > len(ballSizesBrief) {
		panic("illegal size value for ball")
	}
	sz := ballSizesBrief[bs.Size]
	if len(sz) > 0 {
		s.WriteString(" ")
	}
	s.Write([]byte(sz))

	if bs.MoreHMOVE {
		s.WriteString(" hmoving")
		s.WriteString(fmt.Sprintf(" [%04b]", bs.Hmove))
		notes = true
	}

	if bs.Enclockifier.Active {
		// add a comma if we've already noted something else
		if notes {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw %s", bs.Enclockifier.String()))
		notes = true
	}

	if !bs.Enabled {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" disb")
		notes = true
	}

	if bs.VerticalDelay {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" vdel")
	}
	return s.String()
}

func (bs *ballSprite) rsync(adjustment int) {
	bs.ResetPixel -= adjustment
	bs.HmovedPixel -= adjustment
	if bs.ResetPixel < 0 {
		bs.ResetPixel += television.HorizClksVisible
	}
	if bs.HmovedPixel < 0 {
		bs.HmovedPixel += television.HorizClksVisible
	}
}

func (bs *ballSprite) tick(visible, isHmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if isHmove {
		bs.MoreHMOVE = bs.MoreHMOVE && compareHMOVE(hmoveCt, bs.Hmove)
	}

	bs.lastHmoveCt = hmoveCt

	// early return if nothing to do
	if !(isHmove && bs.MoreHMOVE) && !visible {
		return
	}

	// note whether this text is additional hmove tick. see pixel() function
	// in missile sprite for details
	bs.lastTickFromHmove = isHmove && bs.MoreHMOVE

	// update hmoved pixel value
	if !visible {
		bs.HmovedPixel--

		// adjust for screen boundary
		if bs.HmovedPixel < 0 {
			bs.HmovedPixel += television.HorizClksVisible
		}
	}

	bs.pclk.Tick()

	if bs.pclk.Phi2() {
		bs.position.Tick()

		switch bs.position.Count() {
		case 39:
			const startDelay = 4
			bs.startDrawingEvent = bs.Delay.Schedule(startDelay, bs._futureStartDrawingEvent, "START")
		case 40:
			bs.position.Reset()
		}
	}

	// tick future events that are goverened by the sprite
	bs.Delay.Tick()
}

func (bs *ballSprite) _futureStartDrawingEvent() {
	bs.Enclockifier.start()
	bs.startDrawingEvent = nil
}

func (bs *ballSprite) prepareForHMOVE() {
	bs.MoreHMOVE = true

	if *bs.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the String() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		bs.HmovedPixel += 8

		// adjust for screen boundary
		if bs.HmovedPixel > television.HorizClksVisible {
			bs.HmovedPixel -= television.HorizClksVisible
		}
	}
}

func (bs *ballSprite) resetPosition() {
	// see player sprite resetPosition() for commentary on delay values
	delay := 4
	if *bs.hblank {
		if !*bs.hmoveLatch || bs.lastHmoveCt >= 1 && bs.lastHmoveCt <= 15 {
			delay = 2
		} else {
			delay = 3
		}
	}

	// drawing of ball sprite must end immediately upon a reset strobe. it will
	// start drawing again after the reset delay period
	bs.Enclockifier.drop()
	if bs.startDrawingEvent != nil {
		bs.startDrawingEvent.Drop()
		bs.startDrawingEvent = nil
	}

	// stop any existing reset events. generally, this codepath will not apply
	// because a resetPositionEvent will conculde before being triggere again.
	// but it is possible when using a very quick instruction on the reset register,
	// like a zero page INC, for requests to overlap
	if bs.resetPositionEvent != nil {
		bs.resetPositionEvent.Push()
		return
	}

	bs.resetPositionEvent = bs.Delay.Schedule(delay, bs._futureResetPosition, "RESBL")
}

func (bs *ballSprite) _futureResetPosition() {
	// end drawing of sprite in case it has started during the delay
	// period. believe it or not, we can get rid of this and pixel output
	// will still be correct (because of how the delayed END signal in the
	// enclockifier works) but debugging information will be confusing if
	// we did this.
	bs.Enclockifier.drop()
	if bs.startDrawingEvent != nil {
		bs.startDrawingEvent.Drop()
		bs.startDrawingEvent = nil
	}

	// the pixel at which the sprite has been reset, in relation to the
	// left edge of the screen
	bs.ResetPixel, _ = bs.tv.GetState(television.ReqHorizPos)

	if bs.ResetPixel >= 0 {
		// resetPixel adjusted by 1 because the tv is not yet in the correct
		// position
		bs.ResetPixel++

		// adjust resetPixel for screen boundaries
		if bs.ResetPixel > television.HorizClksVisible {
			bs.ResetPixel -= television.HorizClksVisible
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
	bs.position.Reset()
	bs.pclk.Reset()

	// from TIA_HW_Notes.txt:
	//
	// If you look closely at the START signal for the ball, unlike all
	// the other position counters - the ball reset RESBL does send a START
	// signal for graphics output! This makes the ball incredibly useful
	// since you can trigger it as many times as you like across the same
	// scanline and it will start drawing immediately each time :)
	bs.Enclockifier.start()

	// dump reference to reset event
	bs.resetPositionEvent = nil
}

func (bs *ballSprite) pixel() (active bool, color uint8, collision bool) {
	if !bs.Enabled || (bs.VerticalDelay && !bs.EnabledDelay) {
		return false, bs.Color, *bs.hblank && bs.startDrawingEvent != nil && bs.startDrawingEvent.AboutToEnd()
	}

	// earlyStart condition the same as for missile sprites. see missile
	// pixel() function for details
	earlyStart := bs.lastTickFromHmove && bs.startDrawingEvent != nil && bs.startDrawingEvent.AboutToEnd()

	// the LatePhi1() condition has been added to accomodate a artefact in
	// (on?) "Spike's Peak". On the first screen, there is a break in the path
	// at the base of the mountain (which is correct) but without the
	// LatePhi1() condition there is also a second break later on the path
	// (which I don't believe should be there)
	earlyEnd := !bs.pclk.LatePhi1() && bs.lastTickFromHmove && bs.Enclockifier.aboutToEnd()

	// Well blow me down! moving the cosmic ark star problem solution from the
	// missile implementation and I can now see that I've already solved the
	// first half of the problem for the ball sprite, almost by accident.
	//
	// Commenting and Keeping the original code for amusement.
	//
	// // the ball sprite pixel is drawn under specific conditions
	// px := bs.enclockifier.Active ||
	// 	(bs.lastTickFromHmove && bs.startDrawingEvent != nil && bs.startDrawingEvent.AboutToEnd())

	px := !earlyEnd && (bs.Enclockifier.Active || earlyStart)

	if bs.VerticalDelay {
		return bs.EnabledDelay && px, bs.Color, px || (*bs.hblank && bs.startDrawingEvent != nil && bs.startDrawingEvent.AboutToEnd())

	}

	return px, bs.Color, px || (*bs.hblank && bs.startDrawingEvent != nil && bs.startDrawingEvent.AboutToEnd())
}

// the delayed enable bit is copied from the first when the gfx register for
// player 1 is updated with playerSprite.setGfxData()
func (bs *ballSprite) setEnableDelay() {
	bs.EnabledDelay = bs.Enabled
}

func (bs *ballSprite) setEnable(enable bool) {
	bs.Enabled = enable
}

func (bs *ballSprite) setVerticalDelay(vdelay bool) {
	bs.VerticalDelay = vdelay
}

func (bs *ballSprite) SetCTRLPF(value uint8) {
	bs.Ctrlpf = value
	bs.Size = (value & 0x30) >> 4
}

func (bs *ballSprite) setColor(value uint8) {
	bs.Color = value
}

func (bs *ballSprite) setHmoveValue(v interface{}) {
	bs.Hmove = (v.(uint8) ^ 0x80) >> 4
}

func (bs *ballSprite) clearHmoveValue() {
	bs.Hmove = 0x08
}
