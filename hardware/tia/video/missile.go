package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/tiaclock"
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

func newMissileSprite(label string, clk *tiaclock.TIAClock) *missileSprite {
	ms := new(missileSprite)
	ms.sprite = newSprite(label, clk, ms.tick)
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
			s.WriteString(fmt.Sprintf("%d ", (ms.triggerList[i]*tiaclock.NumStates)+ms.currentPixel))
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
		ms.startDrawing()
	} else {
		ms.tickGraphicsScan()
	}

	if ms.resetToPlayerPos {
		// TODO: this isn't right. needs to consider playersize
		ms.position.Count = ms.parentPlayer.position.Count
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
