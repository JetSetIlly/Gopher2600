package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay/future"
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

	position  polycounter.Polycounter
	pclk      phaseclock.PhaseClock
	Delay     future.Ticker
	moreHMOVE bool
	hmove     uint8

	// the following attributes are used for information purposes only:

	label       string
	resetPixel  int
	hmovedPixel int

	// ^^^ the above are common to all sprite types ^^^

	color         uint8
	size          uint8
	verticalDelay bool
	enabled       bool
	enabledDelay  bool
	enclockifier  enclockifier
	startEvent    *future.Event
}

func newBallSprite(label string, tv television.Television, hblank, hmoveLatch *bool) *ballSprite {
	bs := ballSprite{
		tv:         tv,
		hblank:     hblank,
		hmoveLatch: hmoveLatch,
		label:      label,
	}

	bs.Delay.Label = label
	bs.enclockifier.size = &bs.size
	bs.enclockifier.pclk = &bs.pclk
	bs.enclockifier.delay = &bs.Delay
	bs.position.Reset()

	return &bs
}

// MachineInfo returns the sprite information in terse format
func (bs ballSprite) MachineInfoTerse() string {
	return bs.String()
}

// MachineInfo returns the sprite information in verbose format
func (bs ballSprite) MachineInfo() string {
	return bs.String()
}

func (bs ballSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation. put the sign bit back to reflect the
	// original value as used in the ROM.
	normalisedHmove := int(bs.hmove) | 0x08

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

	extra := false

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
		extra = true
	}

	if bs.enclockifier.enable {
		// add a comma if we've already noted something else
		if extra {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw (%s)", bs.enclockifier.String()))
		extra = true
	}

	if !bs.enabled {
		if extra {
			s.WriteString(",")
		}
		s.WriteString(" disb")
		extra = true
	}

	if bs.verticalDelay {
		if extra {
			s.WriteString(",")
		}
		s.WriteString(" vdel")
	}
	return s.String()
}

func (bs *ballSprite) tick(motck bool, hmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if hmove {
		bs.moreHMOVE = bs.moreHMOVE && compareHMOVE(hmoveCt, bs.hmove)
	}

	if (hmove && bs.moreHMOVE) || motck {
		// update hmoved pixel value
		if !motck {
			bs.hmovedPixel--

			// adjust for screen boundary
			if bs.hmovedPixel < 0 {
				bs.hmovedPixel += bs.tv.GetSpec().ClocksPerVisible
			}
		}

		bs.pclk.Tick()

		if bs.pclk.Phi2() {
			bs.position.Tick()

			switch bs.position.Count {
			case 39:
				const startDelay = 4
				bs.startEvent = bs.Delay.Schedule(startDelay, bs.enclockifier.start, "START")
			case 40:
				bs.position.Reset()
			}
		}

		// tick future events that are goverened by the sprite
		bs.Delay.Tick()
	}
}

func (bs *ballSprite) prepareForHMOVE() {
	bs.moreHMOVE = true

	if *bs.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the MachineInfo() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		bs.hmovedPixel += 8

		// adjust for screen boundary
		if bs.hmovedPixel > bs.tv.GetSpec().ClocksPerVisible {
			bs.hmovedPixel -= bs.tv.GetSpec().ClocksPerVisible
		}
	}
}

func (bs *ballSprite) resetPosition() {
	// see player sprite resetPosition() for commentary on delay values
	delay := 4
	if *bs.hblank {
		if *bs.hmoveLatch {
			delay = 3
		} else {
			delay = 2
		}
	}

	// drawing of ball sprite must end immediately upon a reset strobe. it will
	// start drawing again after the reset delay period
	bs.enclockifier.drop()
	if bs.startEvent != nil {
		bs.startEvent.Drop()
		bs.startEvent = nil
	}

	bs.Delay.Schedule(delay, func() {
		// end drawing of sprite in case it has started during the delay
		// period. believe it or not, we can get rid of this and pixel output
		// will still be correct (because of how the delayed END signal in the
		// enclockifier works) but debugging information will be confusing if
		// we did this.
		bs.enclockifier.drop()
		if bs.startEvent != nil {
			bs.startEvent.Drop()
			bs.startEvent = nil
		}

		// the pixel at which the sprite has been reset, in relation to the
		// left edge of the screen
		bs.resetPixel, _ = bs.tv.GetState(television.ReqHorizPos)

		if bs.resetPixel >= 0 {
			// resetPixel adjusted by 1 because the tv is not yet in the correct
			// position
			bs.resetPixel++

			// adjust resetPixel for screen boundaries
			if bs.resetPixel > bs.tv.GetSpec().ClocksPerVisible {
				bs.resetPixel -= bs.tv.GetSpec().ClocksPerVisible
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
			//
			// it cannot occur if an HMOVE is not active. sanity check:
			// !!TODO: remove sanity check once we're convinced that this is true
			if !*bs.hmoveLatch {
				panic("sprite reset during HBLANK should not occur without HMOVE")
			}

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
	}, "RESBL")
}

func (bs *ballSprite) pixel() (bool, uint8) {
	if bs.verticalDelay {
		return bs.enabledDelay && bs.enclockifier.enable, bs.color
	}
	return bs.enabled && bs.enclockifier.enable, bs.color
}

func (bs *ballSprite) setEnable(enable bool) {
	bs.enabledDelay = bs.enabled
	bs.enabled = enable
}

func (bs *ballSprite) setVerticalDelay(vdelay bool) {
	bs.verticalDelay = vdelay
}

func (bs *ballSprite) setHmoveValue(value uint8) {
	// see missile sprite for commentary about delay
	//
	bs.Delay.Schedule(1, func() {
		bs.hmove = (value ^ 0x80) >> 4
	}, "HMBL")
}

func (bs *ballSprite) setSize(value uint8) {
	bs.size = value
}

func (bs *ballSprite) setColor(value uint8) {
	bs.color = value
}
