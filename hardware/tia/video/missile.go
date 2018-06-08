package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
)

type missileSprite struct {
	*sprite

	color        uint8
	size         uint8
	enable       bool
	futureEnable *future

	lateStartDraw bool
}

func newMissileSprite(label string, colorClock *colorclock.ColorClock) *missileSprite {
	ms := new(missileSprite)
	ms.sprite = newSprite(label, colorClock)

	ms.futureEnable = newFuture()
	if ms.futureEnable == nil {
		return nil
	}

	return ms
}

// MachineInfo returns the missile sprite information in terse format
func (ms missileSprite) MachineInfoTerse() string {
	es := ""
	if ms.enable {
		es = "[+] "
	} else {
		es = "[-] "
	}
	return fmt.Sprintf("%s%s", es, ms.sprite.MachineInfoTerse())
}

// MachineInfo returns the missile sprite information in verbose format
func (ms missileSprite) MachineInfo() string {
	es := ""
	if ms.enable {
		es = "\n enabled"
	} else {
		es = "\n disabled"
	}
	if ms.futureEnable.isScheduled() {
		es = fmt.Sprintf("%s [%v in %d cycle", es, ms.futureEnable.payload.(bool), ms.futureEnable.remainingCycles)
		if ms.futureEnable.remainingCycles != 1 {
			es = fmt.Sprintf("%ss]", es)
		} else {
			es = fmt.Sprintf("%s]", es)
		}
	}
	return fmt.Sprintf("%s%s", ms.sprite.MachineInfo(), es)
}

// tick moves the counters along for the missile sprite
func (ms *missileSprite) tick() {
	// position
	if ms.tickPosition(nil) {
		if ms.futureReset.isScheduled() {
			ms.stopDrawing()
			ms.lateStartDraw = true
		} else {
			ms.startDrawing()
		}
	} else {
		if !ms.futureReset.isScheduled() || ms.futureReset.remainingCycles <= 2 {
			ms.tickDrawSig()
		}
	}

	// reset
	if ms.futureReset.tick() {
		ms.resetPosition()
		if ms.lateStartDraw {
			ms.startDrawing()
			ms.lateStartDraw = false
		}
	}

	// enable
	if ms.futureEnable.tick() {
		ms.enable = ms.futureEnable.payload.(bool)
	}
}

// pixel returns the color of the missile at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (ms *missileSprite) pixel() (bool, uint8) {
	if ms.enable {
		switch ms.drawSigCount {
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

func (ms *missileSprite) scheduleReset(hblank *bool) {
	// consume an extra draw sig cycle if the reset is encountered
	// during the first phase of position 000000 or anywhere in 100000. I have no
	// idea why this should be the case but we need to consume an extra
	// tick somewhere for some scenarios and the rule has the desired effect
	// in the examples I've come across so far - found by experimentation
	//
	// TODO: see if we can remove this (what appears to be) special case code for RESM0 and RESM1
	if ms.position.Match(1) || ms.position.MatchBeginning(0) {
		ms.tickDrawSig()
	}

	if *hblank {
		ms.futureReset.schedule(delayResetSpriteDuringHBLANK, true)
	} else {
		ms.futureReset.schedule(delayResetSprite, true)
	}
}

func (ms *missileSprite) scheduleEnable(value uint8) {
	ms.futureEnable.schedule(delayEnableSprite, value&0x20 == 0x20)
}
