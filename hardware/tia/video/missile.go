package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
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
}

func newMissileSprite(label string, colorClock *colorclock.ColorClock) *missileSprite {
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
		if ms.futureReset.isScheduled() {
			ms.stopDrawing()
			ms.deferDrawSig = true
		} else {
			ms.startDrawing()
		}
	} else {
		// tick draw signal only if a position reset is within three cycles of
		// occuring. in effect, this prevents draw signal ticking during the
		// first two cycles of a reset request, unless the reset is scheduled
		// during a HBLANK
		if ms.futureReset.remainingCycles < 4 {
			ms.tickGraphicsScan()
		}
	}

	// reset
	if ms.futureReset.tick() {
		ms.resetPosition()
		if ms.deferDrawSig {
			ms.startDrawing()
			ms.deferDrawSig = false
		}
	}
}

// pixel returns the color of the missile at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (ms *missileSprite) pixel() (bool, uint8) {
	if ms.enable {
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

func (ms *missileSprite) scheduleReset(hblank bool) {
	// consume an extra draw sig cycle if the reset is encountered
	// during the first phase of position 000000 or anywhere in 100000. I have no
	// idea why this should be the case but we need to consume an extra
	// tick somewhere for some scenarios and the rule has the desired effect
	// in the examples I've come across so far - found by experimentation
	//
	// this doesn't smell right TODO: see if we can remove this (what appears
	// to be) special case code for missile sprites
	if ms.position.Match(1) || ms.position.MatchBeginning(0) {
		ms.tickGraphicsScan()
	}

	if !hblank {
		ms.futureReset.schedule(delayResetMissile, true, "resetting")
	} else {
		ms.futureReset.schedule(delayResetMissileHBLANK, true, "resetting")
	}
}

func (ms *missileSprite) scheduleEnable(enable bool, futureWrite *future) {
	label := "enabling"
	if !enable {
		label = "disabling"
	}
	futureWrite.schedule(delayEnableMissile, func() {
		ms.enable = enable
	}, label)
}
