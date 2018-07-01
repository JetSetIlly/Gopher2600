package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
)

type ballSprite struct {
	*sprite

	color         uint8
	size          uint8
	verticalDelay bool
	enable        bool
	enablePrev    bool
	futureEnable  *future
}

func newBallSprite(label string, colorClock *colorclock.ColorClock) *ballSprite {
	bs := new(ballSprite)
	bs.sprite = newSprite(label, colorClock)

	bs.futureEnable = newFuture()
	if bs.futureEnable == nil {
		return nil
	}

	return bs
}

// MachineInfo returns the ball sprite information in terse format
func (bs ballSprite) MachineInfoTerse() string {
	es := ""
	if bs.enable {
		es = "[+] "
	} else {
		es = "[-] "
	}
	return fmt.Sprintf("%s%s", es, bs.sprite.MachineInfoTerse())
}

// MachineInfo returns the ball sprite information in verbose format
func (bs ballSprite) MachineInfo() string {
	es := ""
	if bs.enable {
		es = "\n enabled"
	} else {
		es = "\n disabled"
	}
	if bs.futureEnable.isScheduled() {
		es = fmt.Sprintf("%s [%v in %d cycle", es, bs.futureEnable.payload.(bool), bs.futureEnable.remainingCycles)
		if bs.futureEnable.remainingCycles != 1 {
			es = fmt.Sprintf("%ss]", es)
		} else {
			es = fmt.Sprintf("%s]", es)
		}
	}
	return fmt.Sprintf("%s%s", bs.sprite.MachineInfo(), es)
}

// tick moves the counters along for the ball sprite
func (bs *ballSprite) tick() {
	// position
	if bs.tickPosition(nil) {
		bs.startDrawing()
	} else {
		bs.tickDrawSig()
	}

	// reset
	if bs.futureReset.tick() {
		bs.resetPosition()
		bs.startDrawing()
	}

	// enable
	if bs.futureEnable.tick() {
		bs.enablePrev = bs.enable
		bs.enable = bs.futureEnable.payload.(bool)
	}
}

// pixel returns the color of the ball at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (bs *ballSprite) pixel() (bool, uint8) {
	// ball should be pixelled if:
	//  o ball is enabled and vertical delay is not enabled
	//  o OR ball was previously enabled and vertical delay is enabled
	//  o AND a reset signal (RESBL) has not recently been triggered
	if ((!bs.verticalDelay && bs.enable) || (bs.verticalDelay && bs.enablePrev)) && !bs.futureReset.isScheduled() {
		switch bs.drawSigCount {
		case 0:
			return true, bs.color
		case 1:
			if bs.size >= 0x1 {
				return true, bs.color
			}
		case 2, 3:
			if bs.size >= 0x2 {
				return true, bs.color
			}
		case 4, 5, 6, 7:
			if bs.size == 0x3 {
				return true, bs.color
			}
		}
	}
	return false, 0
}

func (bs *ballSprite) scheduleReset(hblank *bool) {
	if *hblank {
		bs.futureReset.schedule(delayResetBallHBLANK, true)
	} else {
		bs.futureReset.schedule(delayResetBall, true)
	}
}

func (bs *ballSprite) scheduleEnable(value uint8) {
	bs.futureEnable.schedule(delayEnableBall, value&0x02 == 0x02)
}
