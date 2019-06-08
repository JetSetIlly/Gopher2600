package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/phaseclock"
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

func newMissileSprite(label string, tiaclk *phaseclock.PhaseClock) *missileSprite {
	ms := new(missileSprite)
	ms.sprite = newSprite(label, tiaclk, ms.tick)
	return ms
}

// MachineInfo returns the missile sprite information in terse format
func (ms missileSprite) MachineInfoTerse() string {
	s := strings.Builder{}
	return s.String()
}

// MachineInfo returns the missile sprite information in verbose format
func (ms missileSprite) MachineInfo() string {
	s := strings.Builder{}
	return s.String()
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

func (ms *missileSprite) scheduleReset(onFutureWrite future.Scheduler) {
	ms.resetFuture = onFutureWrite.Schedule(delay.ResetMissile, func() {
		ms.resetFuture = nil
		ms.resetPosition()
		if ms.deferDrawStart {
			ms.startDrawing()
			ms.deferDrawStart = false
		}
	}, fmt.Sprintf("%s resetting", ms.label))
}

func (ms *missileSprite) scheduleEnable(enable bool, onFutureWrite future.Scheduler) {
	label := "enabling missile"
	if !enable {
		label = "disabling missile"
	}
	onFutureWrite.Schedule(delay.EnableMissile, func() {
		ms.enable = enable
	}, fmt.Sprintf("%s %s", ms.label, label))
}

func (ms *missileSprite) scheduleResetToPlayer(reset bool, onFutureWrite future.Scheduler) {
	onFutureWrite.Schedule(delay.ResetMissileToPlayerPos, func() {
		ms.resetToPlayerPos = reset
	}, fmt.Sprintf("%s resetting to player pos", ms.label))
}

func (ms *missileSprite) scheduleSetColor(value uint8, onFutureWrite future.Scheduler) {
	onFutureWrite.Schedule(delay.WritePlayer, func() {
		ms.color = value
	}, fmt.Sprintf("%s color", ms.label))
}

func (ms *missileSprite) scheduleSetNUSIZ(value uint8, onFutureWrite future.Scheduler) {
	onFutureWrite.Schedule(delay.SetNUSIZ, func() {
		ms.size = (value & 0x30) >> 4
		ms.triggerList = createTriggerList(value & 0x07)
	}, fmt.Sprintf("%s adjusting NUSIZ", ms.label))
}
