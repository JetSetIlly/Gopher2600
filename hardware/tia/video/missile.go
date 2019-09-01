package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay/future"
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

	enabled       bool
	color         uint8
	size          uint8
	copies        uint8
	enclockifier  enclockifier
	parentPlayer  *playerSprite
	resetToPlayer bool
	startEvent    *future.Event
	resetEvent    *future.Event
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
	// for purposes of presentation. put the sign bit back to reflect the
	// original value as used in the ROM.
	normalisedHmove := int(ms.hmove) | 0x08

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s %s [%03d ", ms.position, ms.pclk, ms.resetPixel))
	s.WriteString(fmt.Sprintf("> %#1x >", normalisedHmove))
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

func (ms *missileSprite) tick(motck bool, hmove bool, hmoveCt uint8) {
	// check to see if there is more movement required for this sprite
	if hmove {
		ms.moreHMOVE = ms.moreHMOVE && compareHMOVE(hmoveCt, ms.hmove)
	}

	// update missile location depending on whether resetToPlayer flag is on
	if ms.resetToPlayer {
		ms.position.Count = ms.parentPlayer.position.Count
		ms.pclk.Sync(ms.parentPlayer.pclk)
		// this isn't exactly accuracte but it'll do for now
		// !!TODO: improve accuracy of reset missile to player
	} else {
		if (hmove && ms.moreHMOVE) || motck {
			// update hmoved pixel value
			if !motck {
				ms.hmovedPixel--

				// adjust for screen boundary
				if ms.hmovedPixel < 0 {
					ms.hmovedPixel += ms.tv.GetSpec().ClocksPerVisible
				}
			}

			ms.pclk.Tick()

			if ms.pclk.Phi2() {
				ms.position.Tick()

				const startDelay = 4
				startEvent := func() {
					ms.enclockifier.start()
					ms.startEvent = nil
				}

				switch ms.position.Count {
				case 3:
					if ms.copies == 0x01 || ms.copies == 0x03 {
						if ms.resetEvent == nil {
							ms.startEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
						}
					}
				case 7:
					if ms.copies == 0x03 || ms.copies == 0x02 || ms.copies == 0x06 {
						if ms.resetEvent == nil {
							ms.startEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
						}
					}
				case 15:
					if ms.copies == 0x04 || ms.copies == 0x06 {
						if ms.resetEvent == nil {
							ms.startEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
						}
					}
				case 39:
					if ms.resetEvent == nil {
						ms.startEvent = ms.Delay.Schedule(startDelay, startEvent, "START")
					}
				case 40:
					ms.position.Reset()
				}
			}

			// tick future events that are goverened by the sprite
			ms.Delay.Tick()
		}
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
	if ms.startEvent != nil {
		ms.startEvent.Pause()
	}

	ms.resetEvent = ms.Delay.Schedule(delay, func() {
		// the pixel at which the sprite has been reset, in relation to the
		// left edge of the screen
		ms.resetPixel, _ = ms.tv.GetState(television.ReqHorizPos)

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

		// reset both sprite position and clock
		ms.position.Reset()
		ms.pclk.Reset()

		ms.enclockifier.force()
		if ms.startEvent != nil {
			ms.startEvent.Force()
		}

		ms.resetEvent = nil
	}, "RESMx")
}

func (ms *missileSprite) setResetToPlayer(on bool) {
	ms.resetToPlayer = on
}

func (ms *missileSprite) pixel() (bool, uint8) {
	return ms.enabled && ms.enclockifier.enable, ms.color
}

func (ms *missileSprite) setEnable(enable bool) {
	ms.enabled = enable
}

func (ms *missileSprite) setHmoveValue(value uint8) {
	ms.hmove = (value ^ 0x80) >> 4
}

func (ms *missileSprite) setNUSIZ(value uint8) {
	ms.size = (value & 0x30) >> 4
	ms.copies = value & 0x07
}

func (ms *missileSprite) setColor(value uint8) {
	ms.color = value
}
