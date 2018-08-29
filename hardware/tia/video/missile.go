package video

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
)

type missileSprite struct {
	*sprite

	color       uint8
	size        uint8
	enable      bool
	triggerList []int

	// if any of the sprite's draw positions are reached but a reset position
	// signal has been scheduled, then we need to delay the start of the
	// sprite's drawing process. the drawing actually commences when the reset
	// actually takes place (concept shared with player sprite)
	deferDrawSig bool

	// whether the reset bit is on. from the stella programmer's guide, "as
	// long as [the] control bit is true the missile will remain locked to the
	// centre of its player and the missile graphics will be disabled
	resetToPlayerPos bool

	// the player to which the missile is paired. we use this when resetting
	// the missile position to the player
	parentPlayer *playerSprite
}

func newMissileSprite(label string, colorClock *polycounter.Polycounter) *missileSprite {
	ms := new(missileSprite)
	ms.sprite = newSprite(label, colorClock)
	return ms
}

// MachineInfo returns the missile sprite information in terse format
func (ms missileSprite) MachineInfoTerse() string {
	msg := ""
	if ms.enable {
		msg = "[+] "
	} else {
		msg = "[-] "
	}
	return fmt.Sprintf("%s%s", msg, ms.sprite.MachineInfoTerse())
}

// MachineInfo returns the missile sprite information in verbose format
func (ms missileSprite) MachineInfo() string {
	msg := ""
	if ms.enable {
		msg = "enabled"
	} else {
		msg = "disabled"
	}
	return fmt.Sprintf("%s\n %s", ms.sprite.MachineInfo(), msg)
}

// tick moves the counters along for the missile sprite
func (ms *missileSprite) tick() {
	// position
	if ms.tickPosition(ms.triggerList) {
		if ms.resetting {
			ms.stopDrawing()
			ms.deferDrawSig = true
		} else {
			ms.startDrawing()
		}
	} else {
		ms.tickGraphicsScan()
	}

	// when RESMP bit is on the missile's position is constantly updated
	// ready for when the bit is reset. this doesn't match the behaviour of
	// stella, which seems to only reset the position at the moment the bit is
	// reset. however, it is easier to do it like this and I don't think it has
	// any effect on the emulation.
	if ms.resetToPlayerPos {
		switch ms.parentPlayer.size {
		case 0x05:
			ms.position.Sync(&ms.parentPlayer.position, 9)
		case 0x07:
			ms.position.Sync(&ms.parentPlayer.position, 12)
		default:
			ms.position.Sync(&ms.parentPlayer.position, 5)
		}
	}
}

// pixel returns the color of the missile at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (ms *missileSprite) pixel() (bool, uint8) {
	if ms.enable && !ms.resetToPlayerPos {
		switch ms.graphicsScanCounter {
		case 0:
			return true, ms.color
		case 1:
			if ms.size >= 0x1 {
				return true, ms.color
			}
		case 2, 3:
			if ms.size >= 0x2 {
				return true, ms.color
			}
		case 4, 5, 6, 7:
			if ms.size >= 0x3 {
				return true, ms.color
			}
		}
	}
	return false, 0
}

func (ms *missileSprite) scheduleReset(futureWrite *future) {
	ms.resetting = true

	futureWrite.schedule(delayResetMissile, func() {
		ms.resetting = false
		ms.resetPosition()
		if ms.deferDrawSig {
			ms.startDrawing()
			ms.deferDrawSig = false
		}
	}, fmt.Sprintf("%s resetting", ms.label))
}

func (ms *missileSprite) scheduleEnable(enable bool, futureWrite *future) {
	label := "enabling missile"
	if !enable {
		label = "disabling missile"
	}
	futureWrite.schedule(delayEnableMissile, func() {
		ms.enable = enable
	}, fmt.Sprintf("%s %s", ms.label, label))
}

func (ms *missileSprite) scheduleResetToPlayer(reset bool, futureWrite *future) {
	futureWrite.schedule(delayResetMissileToPlayerPos, func() {
		ms.resetToPlayerPos = reset
	}, fmt.Sprintf("%s resetting to player pos", ms.label))
}
