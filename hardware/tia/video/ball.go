package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/tiaclock"
	"strings"
)

type ballSprite struct {
	*sprite

	// additional sprite information
	color         uint8
	size          uint8
	verticalDelay bool
	enable        bool
	enablePrev    bool
}

func newBallSprite(label string, clk *tiaclock.TIAClock) *ballSprite {
	bs := new(ballSprite)
	bs.sprite = newSprite(label, clk, bs.tick)
	return bs
}

// MachineInfo returns the ball sprite information in terse format
func (bs ballSprite) MachineInfoTerse() string {
	msg := ""
	if bs.enable {
		msg = "[+]"
	} else {
		msg = "[-]"
	}
	return fmt.Sprintf("%s %s", msg, bs.sprite.MachineInfoTerse())
}

// MachineInfo returns the ball sprite information in verbose format
func (bs ballSprite) MachineInfo() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("   color: %d\n", bs.color))
	s.WriteString(fmt.Sprintf("   size: %d\n", bs.size))
	if bs.verticalDelay {
		s.WriteString("   vert delay: yes\n")
		if bs.enablePrev {
			s.WriteString("   enabled: yes")
		} else {
			s.WriteString("   enabled: no")
		}
	} else {
		s.WriteString("   vert delay: no\n")
		if bs.enable {
			s.WriteString("   enabled: yes")
		} else {
			s.WriteString("   enabled: no")
		}
	}

	return fmt.Sprintf("%s%s", bs.sprite.MachineInfo(), s.String())
}

// tick moves the counters along for the ball sprite
func (bs *ballSprite) tick() {
	// position
	if ok, _ := bs.checkForGfxStart(nil); ok {
		bs.startDrawing()
	} else {
		bs.tickGraphicsScan()
	}
}

// pixel returns the color of the ball at the current time.  returns
// (false, col) if no pixel is to be seen; and (true, col) if there is
func (bs *ballSprite) pixel() (bool, uint8) {
	// ball should be pixelled if:
	//  o ball is enabled and vertical delay is not enabled
	//  o OR ball was previously enabled and vertical delay is enabled
	if (!bs.verticalDelay && bs.enable) || (bs.verticalDelay && bs.enablePrev) {
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

	return false, bs.color
}

func (bs *ballSprite) scheduleReset(onFuture *future.Group) {
	bs.resetFuture = onFuture.Schedule(delay.ResetBall, func() {
		bs.resetFuture = nil
		bs.resetPosition()
		bs.startDrawing()
	}, fmt.Sprintf("%s resetting", bs.label))
}

func (bs *ballSprite) scheduleEnable(enable bool, onFuture *future.Group) {
	label := "enabling"
	if !enable {
		label = "disabling"
	}

	onFuture.Schedule(delay.EnableBall, func() {
		bs.enable = enable
	}, fmt.Sprintf("%s %s", bs.label, label))
}

func (bs *ballSprite) scheduleVerticalDelay(vdelay bool, onFuture *future.Group) {
	label := "enabling vertical delay"
	if !vdelay {
		label = "disabling vertical delay"
	}

	onFuture.Schedule(delay.SetVDELBL, func() {
		bs.verticalDelay = vdelay
	}, fmt.Sprintf("%s %s", bs.label, label))
}
