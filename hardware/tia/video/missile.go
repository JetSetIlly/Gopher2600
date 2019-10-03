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

	position    polycounter.Polycounter
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

	// note whether the last tick was as a result of a HMOVE tick. see the
	// pixel() function below for a fuller explanation.
	lastTickFromHmove bool
}

func newMissileSprite(label string, tv television.Television, hblank, hmoveLatch *bool) *missileSprite {
	ms := missileSprite{
		tv:         tv,
		hblank:     hblank,
		hmoveLatch: hmoveLatch,
		label:      label,
	}

	ms.Delay = future.NewTicker(label)

	ms.enclockifier.size = &ms.size
	ms.enclockifier.pclk = &ms.pclk
	ms.enclockifier.delay = ms.Delay
	ms.position.Reset()
	return &ms

}

func (ms missileSprite) String() string {
	// the hmove value as maintained by the sprite type is normalised for
	// for purposes of presentation
	normalisedHmove := int(ms.hmove) - 8
	if normalisedHmove < 0 {
		normalisedHmove = 16 + normalisedHmove
	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s: ", ms.label))
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

	extra := false

	switch ms.size {
	case 0x0:
	case 0x1:
		s.WriteString(" 2x")
		extra = true
	case 0x2:
		s.WriteString(" 4x")
		extra = true
	case 0x3:
		s.WriteString(" 8x")
		extra = true
	}

	if ms.moreHMOVE {
		s.WriteString(" hmoving")
		s.WriteString(fmt.Sprintf(" [%04b]", ms.hmove))
		extra = true
	}

	if ms.enclockifier.enable {
		// add a comma if we've already noted something else
		if extra {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw %s", ms.enclockifier.String()))
		extra = true
	}

	if !ms.enabled {
		if extra {
			s.WriteString(",")
		}
		s.WriteString(" disb")
		extra = true
	}

	if ms.resetToPlayer {
		if extra {
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
		ms.resetPixel += television.ClocksPerVisible
	}
	if ms.hmovedPixel < 0 {
		ms.hmovedPixel += television.ClocksPerVisible
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

	// note whether this text is additional hmove tick. see pixel() function
	// below for explanation
	ms.lastTickFromHmove = isHmove && ms.moreHMOVE

	// update hmoved pixel value
	if !visible {
		ms.hmovedPixel--

		// adjust for screen boundary
		if ms.hmovedPixel < 0 {
			ms.hmovedPixel += television.ClocksPerVisible
		}
	}

	ms.pclk.Tick()

	if ms.pclk.Phi2() {
		ms.position.Tick()

		// start delay is always 4 cycles
		const startDelay = 4

		// which copy of the sprite will we be drawing
		cpy := 0

		startDrawingEvent := func() {
			ms.enclockifier.start()
			ms.enclockifier.cpy = cpy
			ms.startDrawingEvent = nil
		}

		// start drawing if there is no reset or it has just started AND
		// there wasn't a reset event ongoing when the current event
		// started
		startCondition := ms.resetPositionEvent == nil || ms.resetPositionEvent.JustStarted()

		switch ms.position.Count {
		case 3:
			if ms.copies == 0x01 || ms.copies == 0x03 {
				if startCondition {
					ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startDrawingEvent, "START")
					cpy = 1
				}
			}
		case 7:
			if ms.copies == 0x03 || ms.copies == 0x02 || ms.copies == 0x06 {
				if startCondition {
					ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startDrawingEvent, "START")
					if ms.copies == 0x03 {
						cpy = 2
					} else {
						cpy = 1
					}
				}
			}
		case 15:
			if ms.copies == 0x04 || ms.copies == 0x06 {
				if startCondition {
					ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startDrawingEvent, "START")
					if ms.copies == 0x06 {
						cpy = 2
					} else {
						cpy = 1
					}
				}
			}
		case 39:
			if startCondition {
				ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startDrawingEvent, "START")
			}
		case 40:
			ms.position.Reset()
		}
	}

	// tick future events that are goverened by the sprite
	ms.Delay.Tick()
}

func (ms *missileSprite) prepareForHMOVE() {
	ms.moreHMOVE = true

	if *ms.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the String() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		ms.hmovedPixel += 8

		// adjust for screen boundary
		if ms.hmovedPixel > television.ClocksPerVisible {
			ms.hmovedPixel -= television.ClocksPerVisible
		}
	}
}

func (ms *missileSprite) resetPosition() {
	// see player sprite resetPosition() for commentary on delay values
	delay := 4
	if *ms.hblank {
		if !*ms.hmoveLatch {
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
	// but it is possible when using a very quick opcode on the reset register,
	// like a zero page INC, for requests to overlap
	//
	// in the case of the missile sprite, we can see such an occurance in the
	// test.bin test ROM
	if ms.resetPositionEvent != nil {
		ms.resetPositionEvent.Push()
		return
	}

	ms.resetPositionEvent = ms.Delay.Schedule(delay, func() {
		// the pixel at which the sprite has been reset, in relation to the
		// left edge of the screen
		ms.resetPixel, _ = ms.tv.GetState(television.ReqHorizPos)

		if ms.resetPixel >= 0 {
			// resetPixel adjusted by 1 because the tv is not yet in the correct
			// position
			ms.resetPixel++

			// adjust resetPixel for screen boundaries
			if ms.resetPixel > television.ClocksPerVisible {
				ms.resetPixel -= television.ClocksPerVisible
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
	}, "RESMx")
}

func (ms *missileSprite) setResetToPlayer(on bool) {
	ms.resetToPlayer = on
}

func (ms *missileSprite) pixel() (bool, uint8) {
	// the missile sprite has a special state where a stuffed HMOVE clock
	// causes the sprite to this the start signal has happened one cycle early.
	//
	// the condition is fully explained in the AtariAge post "Cosmic Ark Star
	// Field Revisited" by crispy. as suggested by the post title this is the
	// key to implementing the starfield in the Cosmic Ark ROM
	crispy := ms.lastTickFromHmove && ms.startDrawingEvent != nil && ms.startDrawingEvent.AboutToEnd()

	// whether a pixel is output also depends on whether resetToPlayer is off
	px := !ms.resetToPlayer && (ms.enclockifier.enable || crispy)

	return ms.enabled && px, ms.color
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

func (ms *missileSprite) setHmoveValue(tiaDelay future.Scheduler, value uint8, clearing bool) {
	// delay of at least zero (1 additiona cycle) is required. we can see this
	// in the Midnight Magic ROM where the left gutter separator requires it
	//
	// a delay too high (3 or higher) causes the barber pole test ROM to fail
	//
	// not sure what the actual shoudld except that it should be somewhere
	// between 0 and 3 (inclusive)
	tiaDelay.Schedule(2, func() {
		ms.hmove = (value ^ 0x80) >> 4
	}, "HMMx")
}

func (ms *missileSprite) clearHmoveValue(tiaDelay future.Scheduler) {
	// see setHmoveValue() commentary for delay value reasoning
	tiaDelay.Schedule(2, func() {
		ms.hmove = 0x08
	}, "HMCLR")
}
