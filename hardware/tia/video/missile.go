package video

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/video/future"
	"strings"
)

type missileSprite struct {
	*sprite

	// additional sprite information
	color  uint8
	size   uint8
	enable bool

	// the list of color clock states when missile drawing is triggered
	triggerList []int

	// if any of the sprite's draw positions are reached but a reset position
	// signal has been scheduled, then we need to delay the start of the
	// sprite's drawing process. the drawing actually commences when the reset
	// actually takes place (concept shared with player sprite)
	deferDrawStart bool

	// this is a wierd one. if a reset has occured and the missile is about to
	// start drawing on the next tick, then resetTriggeredOnDraw is set to true
	// the deferDrawSig is then set to the opposite value when the draw is
	// supposed to start (concept share with player sprite)
	resetTriggeredOnDraw bool

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
		msg = "[+]"
	} else {
		msg = "[-]"
	}
	return fmt.Sprintf("%s %s", msg, ms.sprite.MachineInfoTerse())
}

// MachineInfo returns the missile sprite information in verbose format
func (ms missileSprite) MachineInfo() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("   color: %d\n", ms.color))
	s.WriteString(fmt.Sprintf("   size: %d\n", ms.size))
	if ms.enable {
		s.WriteString("   enabled: yes")
	} else {
		s.WriteString("   enabled: no")
	}

	return fmt.Sprintf("%s%s", ms.sprite.MachineInfo(), s.String())
}

// tick moves the counters along for the missile sprite
func (ms *missileSprite) tick() {
	// position
	if ms.tickPosition(ms.triggerList) {
		if ms.resetting && !ms.resetTriggeredOnDraw {
			ms.deferDrawStart = true
		} else {
			ms.startDrawing()
		}
	} else {
		// tick graphics scan only if sprite is:
		//	a) not resetting
		//  b) or if it is resetting then only when the position counter is in
		//  a particular configuration
		if !ms.resetting {
			ms.tickGraphicsScan()
		} else {
			// special conditions based on size
			// TODO: investigate special condition
			switch ms.size {
			case 0x0:
				ms.tickGraphicsScan()
			case 0x1:
				ms.tickGraphicsScan()
			case 0x2:
				if ms.position.Phase != 1 {
					ms.tickGraphicsScan()
				}
			case 0x3:
				if ms.position.Phase != 2 {
					ms.tickGraphicsScan()
				}
			}
		}
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

func (ms *missileSprite) scheduleReset(onFutureWrite *future.Group) {
	ms.resetting = true
	ms.resetTriggeredOnDraw = ms.position.CycleOnNextTick()

	onFutureWrite.Schedule(delayResetMissile, func() {
		ms.resetting = false
		ms.resetTriggeredOnDraw = false
		ms.resetPosition()
		if ms.deferDrawStart {
			ms.startDrawing()
			ms.deferDrawStart = false
		}
	}, fmt.Sprintf("%s resetting", ms.label))
}

func (ms *missileSprite) scheduleEnable(enable bool, onFutureWrite *future.Group) {
	label := "enabling missile"
	if !enable {
		label = "disabling missile"
	}
	onFutureWrite.Schedule(delayEnableMissile, func() {
		ms.enable = enable
	}, fmt.Sprintf("%s %s", ms.label, label))
}

func (ms *missileSprite) scheduleResetToPlayer(reset bool, onFutureWrite *future.Group) {
	onFutureWrite.Schedule(delayResetMissileToPlayerPos, func() {
		ms.resetToPlayerPos = reset
	}, fmt.Sprintf("%s resetting to player pos", ms.label))
}
