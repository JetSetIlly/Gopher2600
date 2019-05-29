package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/polycounter"
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
	ms.sprite = newSprite(label, colorClock, ms.tick)
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
	s.WriteString(fmt.Sprintf("   size: %03b [", ms.size))
	switch ms.size {
	case 0:
		s.WriteString("normal")
	case 1:
		s.WriteString("double")
	case 2:
		s.WriteString("quadruple")
	case 3:
		s.WriteString("double-quad")
	}
	s.WriteString("]\n")
	s.WriteString("   trigger list: ")
	if len(ms.triggerList) > 0 {
		for i := 0; i < len(ms.triggerList); i++ {
			s.WriteString(fmt.Sprintf("%d ", (ms.triggerList[i]*(polycounter.MaxPhase+1))+ms.currentPixel))
		}
		s.WriteString(fmt.Sprintf(" %v\n", ms.triggerList))
	} else {
		s.WriteString("none\n")
	}
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
	if ok, _ := ms.checkForGfxStart(ms.triggerList); ok {
		// if a reset of this sprite is pending then we need to defer the start
		// of the drawing until the reset has occurred. we also need to
		// consider:
		//	* the reset request has not *just* happened (within a video cycle)
		//	* the sprite has not been moved by HMOVE
		//
		// (concept shared with player sprite)
		ms.deferDrawStart = ms.resetFuture != nil &&
			ms.resetFuture.RemainingCycles < ms.resetFuture.InitialCycles &&
			ms.resetPixel == ms.currentPixel

		if !ms.deferDrawStart {
			ms.startDrawing()
		}
	} else {
		// tick graphics scan only if sprite is:
		//	a) not resetting
		//  b) or if it is resetting then only when the position counter is in
		//  a particular configuration
		if ms.resetFuture == nil {
			ms.tickGraphicsScan()
		} else {
			// special conditions based on size
			//
			// note sure about this logic at all. this is what was required for
			// the "missile testcards" to work correctly.
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
// (false, col) if no pixel is to be seen; and (true, col) if there is
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

	return false, ms.color
}

func (ms *missileSprite) scheduleReset(onFutureWrite *future.Group) {
	ms.resetFuture = onFutureWrite.Schedule(delay.ResetMissile, func() {
		ms.resetFuture = nil
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
	onFutureWrite.Schedule(delay.EnableMissile, func() {
		ms.enable = enable
	}, fmt.Sprintf("%s %s", ms.label, label))
}

func (ms *missileSprite) scheduleResetToPlayer(reset bool, onFutureWrite *future.Group) {
	onFutureWrite.Schedule(delay.ResetMissileToPlayerPos, func() {
		ms.resetToPlayerPos = reset
	}, fmt.Sprintf("%s resetting to player pos", ms.label))
}

func (ms *missileSprite) scheduleSetColor(value uint8, onFutureWrite *future.Group) {
	onFutureWrite.Schedule(delay.WritePlayer, func() {
		ms.color = value
	}, fmt.Sprintf("%s color", ms.label))
}

func (ms *missileSprite) scheduleSetNUSIZ(value uint8, onFutureWrite *future.Group) {
	onFutureWrite.Schedule(delay.SetNUSIZ, func() {
		ms.size = (value & 0x30) >> 4
		ms.triggerList = createTriggerList(value & 0x07)
	}, fmt.Sprintf("%s adjusting NUSIZ", ms.label))
}
