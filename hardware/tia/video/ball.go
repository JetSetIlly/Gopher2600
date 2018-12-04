package video

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
)

type ballSprite struct {
	*sprite

	color         uint8
	size          uint8
	verticalDelay bool
	enable        bool
	enablePrev    bool
}

func newBallSprite(label string, colorClock *polycounter.Polycounter) *ballSprite {
	bs := new(ballSprite)
	bs.sprite = newSprite(label, colorClock)
	return bs
}

// MachineInfo returns the ball sprite information in terse format
func (bs ballSprite) MachineInfoTerse() string {
	msg := ""
	if bs.enable {
		msg = "[+] "
	} else {
		msg = "[-] "
	}
	return fmt.Sprintf("%s%s", msg, bs.sprite.MachineInfoTerse())
}

// MachineInfo returns the ball sprite information in verbose format
func (bs ballSprite) MachineInfo() string {
	msg := ""
	if bs.enable {
		msg = "enabled"
	} else {
		msg = "disabled"
	}
	return fmt.Sprintf("%s\n %s", bs.sprite.MachineInfo(), msg)
}

// tick moves the counters along for the ball sprite
func (bs *ballSprite) tick() {
	// position
	if bs.tickPosition(nil) {
		bs.startDrawing()
	} else {
		bs.tickGraphicsScan()
	}
}

// pixel returns the color of the ball at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (bs *ballSprite) pixel() (bool, uint8) {
	// ball should be pixelled if:
	//  o ball is enabled and vertical delay is not enabled
	//  o OR ball was previously enabled and vertical delay is enabled
	//  o AND a reset signal (RESBL) has not recently been triggered
	if ((!bs.verticalDelay && bs.enable) || (bs.verticalDelay && bs.enablePrev)) && !bs.resetting {
		switch bs.graphicsScanCounter {
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

func (bs *ballSprite) scheduleReset(futureWrite *future) {
	bs.resetting = true
	futureWrite.schedule(delayResetBall, func() {
		bs.resetting = false
		bs.resetPosition()
		bs.startDrawing()
	}, fmt.Sprintf("%s resetting", bs.label))
}

func (bs *ballSprite) scheduleEnable(enable bool, futureWrite *future) {
	label := "enabling"
	if !enable {
		label = "disabling"
	}
	futureWrite.schedule(delayEnableBall, func() {
		bs.enable = enable
	}, fmt.Sprintf("%s %s", bs.label, label))
}
