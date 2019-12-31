package video

import (
	"fmt"
	"gopher2600/hardware/tia/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/television"
	"strings"
)

type ballSprite struct {
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

	// ^^^ the above are common to all sprite types ^^^
	//		(see player sprite for commentary)

	color              uint8
	size               uint8
	verticalDelay      bool
	enabled            bool
	enabledDelay       bool
	enclockifier       enclockifier
	startDrawingEvent  *future.Event
	resetPositionEvent *future.Event

	// note whether the last tick was as a result of a HMOVE tick. see the
	// pixel() function in the missile sprite for a detailed explanation
	lastTickFromHmove bool
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

	bs.enclockifier.size = &bs.size
	bs.enclockifier.pclk = &bs.pclk
	bs.enclockifier.delay = bs.Delay
	bs.position.Reset()

	return &bs, nil
}

func (bs ballSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation
	normalisedHmove := int(bs.hmove) - 8
	if normalisedHmove < 0 {
		normalisedHmove = 16 + normalisedHmove
	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s: ", bs.label))
	s.WriteString(fmt.Sprintf("%s %s [%03d ", bs.position, bs.pclk, bs.resetPixel))
	s.WriteString(fmt.Sprintf("> %#1x >", normalisedHmove))
	s.WriteString(fmt.Sprintf(" %03d", bs.hmovedPixel))
	if bs.moreHMOVE {
		s.WriteString("*]")
	} else {
		s.WriteString("]")
	}

	notes := false

	switch bs.size {
	case 0x0:
	case 0x1:
		s.WriteString(" 2x")
	case 0x2:
		s.WriteString(" 4x")
	case 0x3:
		s.WriteString(" 8x")
	}

	if bs.moreHMOVE {
		s.WriteString(" hmoving")
		s.WriteString(fmt.Sprintf(" [%04b]", bs.hmove))
		notes = true
	}

	if bs.enclockifier.enable {
		// add a comma if we've already noted something else
		if notes {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw %s", bs.enclockifier.String()))
		notes = true
	}

	if !bs.enabled {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" disb")
		notes = true
	}

	if bs.verticalDelay {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" vdel")
	}
	return s.String()
}

func (bs *ballSprite) rsync(adjustment int) {
	bs.resetPixel -= adjustment
	bs.hmovedPixel -= adjustment
	if bs.resetPixel < 0 {
		bs.resetPixel += television.HorizClksVisible
	}
	if bs.hmovedPixel < 0 {
		bs.hmovedPixel += television.HorizClksVisible
	}
}

func (bs *ballSprite) tick(visible, isHmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if isHmove {
		bs.moreHMOVE = bs.moreHMOVE && compareHMOVE(hmoveCt, bs.hmove)
	}

	bs.lastHmoveCt = hmoveCt

	// early return if nothing to do
	if !(isHmove && bs.moreHMOVE) && !visible {
		return
	}

	// note whether this text is additional hmove tick. see pixel() function
	// in missile sprite for details
	bs.lastTickFromHmove = isHmove && bs.moreHMOVE

	// update hmoved pixel value
	if !visible {
		bs.hmovedPixel--

		// adjust for screen boundary
		if bs.hmovedPixel < 0 {
			bs.hmovedPixel += television.HorizClksVisible
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
	bs.enclockifier.start()
	bs.startDrawingEvent = nil
}

func (bs *ballSprite) prepareForHMOVE() {
	bs.moreHMOVE = true

	if *bs.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the String() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		bs.hmovedPixel += 8

		// adjust for screen boundary
		if bs.hmovedPixel > television.HorizClksVisible {
			bs.hmovedPixel -= television.HorizClksVisible
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
	bs.enclockifier.drop()
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
	bs.enclockifier.drop()
	if bs.startDrawingEvent != nil {
		bs.startDrawingEvent.Drop()
		bs.startDrawingEvent = nil
	}

	// the pixel at which the sprite has been reset, in relation to the
	// left edge of the screen
	bs.resetPixel, _ = bs.tv.GetState(television.ReqHorizPos)

	if bs.resetPixel >= 0 {
		// resetPixel adjusted by 1 because the tv is not yet in the correct
		// position
		bs.resetPixel++

		// adjust resetPixel for screen boundaries
		if bs.resetPixel > television.HorizClksVisible {
			bs.resetPixel -= television.HorizClksVisible
		}

		// by definition the current pixel is the same as the reset pixel at
		// the moment of reset
		bs.hmovedPixel = bs.resetPixel
	} else {
		// if reset occurs off-screen then force reset pixel to be zero
		bs.resetPixel = 0

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
		bs.hmovedPixel = 7
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
	bs.enclockifier.start()

	// dump reference to reset event
	bs.resetPositionEvent = nil
}

func (bs *ballSprite) pixel() (bool, uint8) {
	if !bs.enabled || (bs.verticalDelay && !bs.enabledDelay) {
		return false, bs.color
	}

	// the ball sprite pixel is drawn under specific conditions
	px := bs.enclockifier.enable ||
		(bs.lastTickFromHmove && bs.startDrawingEvent != nil && bs.startDrawingEvent.AboutToEnd())

	if bs.verticalDelay {
		return bs.enabledDelay && px, bs.color
	}

	return px, bs.color
}

// the delayed enable bit is copied from the first when the gfx register for
// player 1 is updated with playerSprite.setGfxData()
func (bs *ballSprite) setEnableDelay() {
	bs.enabledDelay = bs.enabled
}

func (bs *ballSprite) setEnable(enable bool) {
	bs.enabled = enable
}

func (bs *ballSprite) setVerticalDelay(vdelay bool) {
	bs.verticalDelay = vdelay
}

func (bs *ballSprite) setSize(value uint8) {
	bs.size = value
}

func (bs *ballSprite) setColor(value uint8) {
	bs.color = value
}

func (bs *ballSprite) setHmoveValue(v interface{}) {
	bs.hmove = (v.(uint8) ^ 0x80) >> 4
}

func (bs *ballSprite) clearHmoveValue() {
	bs.hmove = 0x08
}
