package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
	"math/bits"
)

type playerSprite struct {
	*sprite

	color         uint8
	gfxData       uint8
	gfxDataPrev   uint8
	size          uint8
	reflected     bool
	verticalDelay bool
	triggerList   []int

	lateStartDraw bool
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
	return fmt.Sprintf("%s gfx: %s %08b", ps.sprite.MachineInfoTerse(), ref, gfxData)
}

// nothing to add to sprite type implementation of MachineInfo() and
// MachineInfoTerse()

// tick moves the counters along for the player sprite
func (ps *playerSprite) tick() {
	// position
	if ps.tickPosition(ps.triggerList) {
		if ps.futureReset.isScheduled() {
			ps.stopDrawing()
			ps.lateStartDraw = true
		} else {
			ps.startDrawing()
		}
	} else {
		// if player.position.tick() has not caused the position counter to
		// cycle then progress draw signal according to color clock phase and
		// nusiz_player_width. for nusiz_player_width and 0b101 and 0b111,
		// pixels are smeared over additional cycles in order to create the
		// double and quadruple sized sprites
		//
		// NOTE: the key difference between player and missile ticking is the
		// absence in the player sprite, of a filter on draw sig ticking when
		// there a position reset is pending
		if ps.size == 0x05 {
			if ps.colorClock.Phase == 0 || ps.colorClock.Phase == 2 {
				ps.tickDrawSig()
			}
		} else if ps.size == 0x07 {
			if ps.colorClock.Phase == 2 {
				ps.tickDrawSig()
			}
		} else {
			ps.tickDrawSig()
		}
	}

	// reset
	if ps.futureReset.tick() {
		ps.resetPosition()
		if ps.lateStartDraw {
			ps.startDrawing()
			ps.lateStartDraw = false
		}
	}
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
}

func (ps *playerSprite) scheduleReflection() {
}
