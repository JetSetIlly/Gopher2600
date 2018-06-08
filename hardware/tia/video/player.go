package video

import (
	"gopher2600/hardware/tia/colorclock"
)

type playerSprite struct {
	*sprite

	color         uint8
	gfxData       uint8
	gfxDataPrev   uint8
	reflection    bool
	verticalDelay bool
}

func newPlayerSprite(label string, colorClock *colorclock.ColorClock) *playerSprite {
	ps := new(playerSprite)
	ps.sprite = newSprite(label, colorClock)
	return ps
}

// nothing to add to sprite type implementation of MachineInfo() and
// MachineInfoTerse()

// tick moves the counters along for the player sprite
func (ps *playerSprite) tick() {
}

// pixel returns the color of the player at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (ps *playerSprite) pixel() (bool, uint8) {
	return false, 0
}

func (ps *playerSprite) scheduleReset(hblank *bool) {
	if *hblank {
		ps.futureReset.schedule(delayResetSpriteDuringHBLANK, true)
	} else {
		ps.futureReset.schedule(delayResetSprite, true)
	}
}
