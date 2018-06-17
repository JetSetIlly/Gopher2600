package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
	"math/bits"
)

type playerSprite struct {
	*sprite

	// player sprite properties in addition to the common sprite properties.
	color         uint8
	gfxData       uint8
	gfxDataPrev   uint8
	size          uint8
	reflected     bool
	verticalDelay bool
	triggerList   []int

	// if any of the sprite's draw positions are reached but a reset position
	// signal has been scheduled, then we need to delay the start of the
	// sprite's drawing process. the drawing actually commences when the reset
	// actually takes place
	//
	// (concept shared with missile sprite)
	deferDrawSig bool

	// the size of the player sprite is implemented by smearing the draw signal
	// over several clock ticks. the draw signal is actually ticked only on
	// certain clock phases, the precise phase is based on when the sprite's
	// position was reset
	resetPhase int

	// if the sprite is currently being drawn when a reset is triggered, we
	// need to delay when the new resetPhase comes into effect. the
	// delayResetPhase flag is set under such circumstances and the
	// resetPhasePrev value used to tick the draw signal until the drawing has
	// completed
	delayResetPhase bool
	resetPhasePrev  int
}

func newPlayerSprite(label string, colorClock *colorclock.ColorClock) *playerSprite {
	ps := new(playerSprite)
	ps.sprite = newSprite(label, colorClock)
	return ps
}

func (ps playerSprite) MachineInfoTerse() string {
	gfxData := ps.gfxData
	if ps.verticalDelay {
		gfxData = ps.gfxDataPrev
	}
	ref := " "
	if ps.reflected {
		ref = "r"
	}

	pix := ps.positionResetPixel + 1
	if ps.size == 0x05 || ps.size == 0x07 {
		pix++
	}

	return fmt.Sprintf("%s (vis: %d) gfx: %s %08b", ps.sprite.MachineInfoTerse(), pix, ref, gfxData)
}

// MachineInfo returns the missile sprite information in verbose format
func (ps playerSprite) MachineInfo() string {
	// TODO: extended MachineInfo() for player sprite
	return fmt.Sprintf("%s%s", ps.sprite.MachineInfo())
}

// tick moves the counters along for the player sprite
func (ps *playerSprite) tick() {
	// position
	if ps.tickPosition(ps.triggerList) {
		if ps.futureReset.isScheduled() {
			ps.stopDrawing()
			ps.deferDrawSig = true
		} else {
			ps.startDrawing()
		}
	} else {
		// if player.position.tick() has not caused the position counter to
		// cycle then progress draw signal according to color clock phase and
		// nusiz_player_width. for nusiz_player_width and 0b101 and 0b111,
		// pixels are smeared over additional cycles in order to create the
		// double and quadruple sized sprites
		if ps.size == 0x05 {
			if ps.delayResetPhase {
				if ps.colorClock.Phase == ps.resetPhasePrev || ps.colorClock.Phase == ps.resetPhasePrev+2 || ps.colorClock.Phase == ps.resetPhasePrev-2 {
					ps.tickDrawSig()
				}
			} else {
				if ps.colorClock.Phase == ps.resetPhase || ps.colorClock.Phase == ps.resetPhase+2 || ps.colorClock.Phase == ps.resetPhase-2 {
					ps.tickDrawSig()
				}
			}
		} else if ps.size == 0x07 {
			if ps.delayResetPhase {
				if ps.colorClock.Phase == ps.resetPhasePrev {
					ps.tickDrawSig()
				}
			} else {
				if ps.colorClock.Phase == ps.resetPhase {
					ps.tickDrawSig()
				}
			}
		} else {
			ps.tickDrawSig()
		}
	}

	// reset
	if ps.futureReset.tick() {
		ps.resetPosition()
		if ps.deferDrawSig {
			ps.startDrawing()
			ps.deferDrawSig = false
		}

		// pixel smearing requires we record the phase on which the actual
		// reset occurs.
		ps.resetPhase = ps.colorClock.Phase
	}

	// turn off the flag that controls which resetPhase value to use when there
	// is no drawing taking place
	ps.delayResetPhase = ps.delayResetPhase && ps.isDrawing()
}

// pixel returns the color of the player at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (ps *playerSprite) pixel() (bool, uint8) {
	// vertical delay
	gfxData := ps.gfxData
	if ps.verticalDelay {
		gfxData = ps.gfxDataPrev
	}

	if ps.reflected {
		gfxData = bits.Reverse8(gfxData)
	}

	// player sprites are unusual in that the first tick of the draw signal is
	// discounted
	if ps.drawSigCount > 0 && ps.drawSigCount <= ps.drawSigMax {
		if gfxData>>(uint8(ps.drawSigMax)-uint8(ps.drawSigCount))&0x01 == 0x01 {
			return true, ps.color
		}
	}

	return false, 0
}

func (ps *playerSprite) scheduleReset(hblank *bool) {
	if *hblank {
		ps.futureReset.schedule(delayResetSpriteDuringHBLANK, true)
	} else {
		ps.futureReset.schedule(delayResetSprite, true)
	}

	// if drawing is currently in progress when reset is scheduled we need to
	// delay the setting of resetPhase until drawing has finished
	if ps.isDrawing() {
		ps.delayResetPhase = true
		ps.resetPhasePrev = ps.resetPhase
	}
}
