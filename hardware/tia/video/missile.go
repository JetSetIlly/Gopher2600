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

	enabled            bool
	color              uint8
	size               uint8
	copies             uint8
	enclockifier       enclockifier
	parentPlayer       *playerSprite
	resetToPlayer      bool
	startDrawingEvent  *future.Event
	resetPositionEvent *future.Event

	// stuffedTick notes whether the last tick was as a result of a HMOVE tick.
	// see the pixel() function below for a fuller explanation.
	stuffedTick bool
}

func newMissileSprite(label string, tv television.Television, hblank, hmoveLatch *bool) *missileSprite {
	ms := missileSprite{
		tv:         tv,
		hblank:     hblank,
		hmoveLatch: hmoveLatch,
		label:      label,
	}

	ms.Delay.Label = label
	ms.enclockifier.size = &ms.size
	ms.enclockifier.pclk = &ms.pclk
	ms.enclockifier.delay = &ms.Delay
	ms.position.Reset()
	return &ms

}

// MachineInfo returns the sprite information in terse format
func (ms missileSprite) MachineInfoTerse() string {
	return ms.String()
}

// MachineInfo returns the sprite information in verbose format
func (ms missileSprite) MachineInfo() string {
	return ms.String()
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
	s.WriteString(fmt.Sprintf("> %d >", normalisedHmove))
	s.WriteString(fmt.Sprintf(" %03d", ms.hmovedPixel))
	if ms.moreHMOVE {
		s.WriteString("*]")
	} else {
		s.WriteString("]")
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
		extra = true
	}

	if ms.enclockifier.enable {
		// add a comma if we've already noted something else
		if extra {
			s.WriteString(",")
		}
		s.WriteString(fmt.Sprintf(" drw (%s)", ms.enclockifier.String()))
		extra = true
	}

	if !ms.enabled {
		if extra {
			s.WriteString(",")
		}
		s.WriteString(" disb")
	}

	return s.String()
}

func (ms *missileSprite) rsync(adjustment int) {
	ms.resetPixel -= adjustment
	ms.hmovedPixel -= adjustment
	if ms.resetPixel < 0 {
		ms.resetPixel += ms.tv.GetSpec().ClocksPerVisible
	}
	if ms.hmovedPixel < 0 {
		ms.hmovedPixel += ms.tv.GetSpec().ClocksPerVisible
	}
}

func (ms *missileSprite) tick(motck bool, hmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if hmove {
		ms.moreHMOVE = ms.moreHMOVE && compareHMOVE(hmoveCt, ms.hmove)
	}

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
	if ms.resetToPlayer && ms.parentPlayer.scanCounter.primary && ms.parentPlayer.scanCounter.isMiddle() {
		ms.position.Reset()
		ms.pclk.Reset()
	}

	if (hmove && ms.moreHMOVE) || motck {
		// update hmoved pixel value
		if !motck {
			ms.hmovedPixel--

			// adjust for screen boundary
			if ms.hmovedPixel < 0 {
				ms.hmovedPixel += ms.tv.GetSpec().ClocksPerVisible
			}
		}

		// make a note of why this tick has occurred. see pixel() function
		// below for explanation
		ms.stuffedTick = hmove && ms.moreHMOVE

		ms.pclk.Tick()

		if ms.pclk.Phi2() {
			ms.position.Tick()

			const startDelay = 4
			startEvent := func() {
				ms.enclockifier.start()
				ms.startDrawingEvent = nil
			}

			switch ms.position.Count {
			case 3:
				if ms.copies == 0x01 || ms.copies == 0x03 {
					if ms.resetPositionEvent == nil {
						ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
					}
				}
			case 7:
				if ms.copies == 0x03 || ms.copies == 0x02 || ms.copies == 0x06 {
					if ms.resetPositionEvent == nil {
						ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
					}
				}
			case 15:
				if ms.copies == 0x04 || ms.copies == 0x06 {
					if ms.resetPositionEvent == nil {
						ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
					}
				}
			case 39:
				if ms.resetPositionEvent == nil {
					ms.startDrawingEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
				}
			case 40:
				ms.position.Reset()
			}
		}

		// tick future events that are goverened by the sprite
		ms.Delay.Tick()
	}
}

func (ms *missileSprite) prepareForHMOVE() {
	ms.moreHMOVE = true

	if *ms.hblank {
		// adjust hmovedPixel value. this value is subject to further change so
		// long as moreHMOVE is true. the MachineInfo() function this value is
		// annotated with a "*" to indicate that HMOVE is still in progress
		ms.hmovedPixel += 8

		// adjust for screen boundary
		if ms.hmovedPixel > ms.tv.GetSpec().ClocksPerVisible {
			ms.hmovedPixel -= ms.tv.GetSpec().ClocksPerVisible
		}
	}
}

func (ms *missileSprite) resetPosition() {
	// see player sprite resetPosition() for commentary on delay values
	delay := 4
	if *ms.hblank {
		if *ms.hmoveLatch {
			delay = 3
		} else {
			delay = 2
		}
	}

	// drawing of missile sprite is paused and will resume upon reset
	// completion. compare to ball sprite where drawing is ended and then
	// started under all conditions
	ms.enclockifier.pause()
	if ms.startDrawingEvent != nil {
		ms.startDrawingEvent.Pause()
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
			if ms.resetPixel > ms.tv.GetSpec().ClocksPerVisible {
				ms.resetPixel -= ms.tv.GetSpec().ClocksPerVisible
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
		}

		ms.resetPositionEvent = nil
	}, "RESMx")
}

func (ms *missileSprite) setResetToPlayer(on bool) {
	ms.resetToPlayer = on
}

func (ms *missileSprite) pixel() (bool, uint8) {
	// the missile sprite is drawn if the enclockifier is on. OR, if it will be
	// ON next cycle AND the most recent tick was a result of a HMOVE clock
	// stuff.
	//
	// what's the reason for this? it is fully explained in the AtariAge post
	// "Cosmic Ark Star Field Revisited" by Crsipy, but briefly the
	// exaplanation is this: the extra HMOVE clock causes the "missile logic"
	// to think that the start signal has happened early.
	//
	// in short, the following condition implements the Cosmic Ark starfield.
	px := !ms.resetToPlayer &&
		(ms.enclockifier.enable ||
			(ms.stuffedTick && ms.startDrawingEvent != nil && ms.startDrawingEvent.RemainingCycles == 0))

	return ms.enabled && px, ms.color
}

func (ms *missileSprite) setEnable(enable bool) {
	ms.enabled = enable
}

func (ms *missileSprite) setHmoveValue(value uint8, clearing bool) {
	// see player sprite for details about horizontal movement
	//
	// (the following applies to all sprites but is described here because the
	// effect of scheduling most dramatically applies to the missiles in the
	// cosmic ark starfield.)
	//
	// a delay of 1 on the sprite scheduler, is required for the cosmicark
	// starfield to work correctly. I'm not not entirely sure if this is the
	// correct interpretation or if the timing issue with compareHMOVE should
	// be ironed out somewhere else.

	msg := "HMMx"
	if clearing {
		msg = "HMCLR"
	}

	ms.Delay.Schedule(1, func() {
		ms.hmove = (value ^ 0x80) >> 4
	}, msg)
}

func (ms *missileSprite) setNUSIZ(value uint8) {
	ms.size = (value & 0x30) >> 4
	ms.copies = value & 0x07
}

func (ms *missileSprite) setColor(value uint8) {
	ms.color = value
}
