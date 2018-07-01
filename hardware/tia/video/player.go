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
	gfxDataDelay  *uint8
	size          uint8
	reflected     bool
	verticalDelay bool
	triggerList   []int

	// if any of the sprite's draw positions are reached but a reset position
	// signal has been scheduled, then we need to delay the start of the
	// sprite's drawing process. the drawing actually commences when the reset
	// actually takes place (concept shared with missile sprite)
	deferDrawSig bool
}

func newPlayerSprite(label string, colorClock *colorclock.ColorClock) *playerSprite {
	ps := new(playerSprite)
	ps.sprite = newSprite(label, colorClock)
	return ps
}

func (ps playerSprite) MachineInfoTerse() string {
	gfxData := ps.gfxData
	vdel := ""
	if ps.verticalDelay {
		gfxData = *ps.gfxDataDelay
		vdel = " v"
	}
	ref := " "
	if ps.reflected {
		ref = "r"
	}

	// NOTE that because of the delay in starting pixel output with player
	// sprites we are adding one to our reported pixel start position (with
	// additional pixels for the larger player sizes)
	visPix := ps.positionResetPixel + 1
	if ps.size == 0x05 || ps.size == 0x07 {
		visPix++
	}

	return fmt.Sprintf("%s (vis: %d, hm: %d) gfx: %s %08b%s", ps.sprite.MachineInfoTerse(), visPix, ps.horizMovement-8, ref, gfxData, vdel)
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
		if ps.deferDrawSig {
			ps.startDrawing()
			ps.deferDrawSig = false
		}
	}
}

// pixel returns the color of the player at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (ps *playerSprite) pixel() (bool, uint8) {
	// vertical delay
	gfxData := ps.gfxData
	if ps.verticalDelay {
		gfxData = *ps.gfxDataDelay
	}

	// reflection
	if ps.reflected {
		gfxData = bits.Reverse8(gfxData)
	}

	// player sprites are unusual in that the first tick of the draw signal is
	// discounted
	// NOTE: we are not drawing a pixel on drawSigCount of 0, like we would
	// with the ball and player sprites. rather than introduce a new 'future'
	// instance we simply start outputting pixels one drawSigCount (or one
	// clock) later
	if ps.drawSigCount > 0 && ps.drawSigCount <= ps.drawSigMax {
		if gfxData>>(uint8(ps.drawSigMax)-uint8(ps.drawSigCount))&0x01 == 0x01 {
			return true, ps.color
		}
	}

	return false, 0
}

func (ps *playerSprite) scheduleReset(hblank *bool) {
	if *hblank {
		ps.futureReset.schedule(delayResetPlayerHBLANK, true)
	} else {
		ps.futureReset.schedule(delayResetPlayer, true)
	}
}
