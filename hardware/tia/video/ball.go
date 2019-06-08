package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/phaseclock"
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

func newBallSprite(label string, tiaclk *phaseclock.PhaseClock) *ballSprite {
	bs := new(ballSprite)
	bs.sprite = newSprite(label, tiaclk, bs.tick)
	return bs
}

// MachineInfo returns the ball sprite information in terse format
func (bs ballSprite) MachineInfoTerse() string {
	s := strings.Builder{}
	return s.String()
}

// MachineInfo returns the ball sprite information in verbose format
func (bs ballSprite) MachineInfo() string {
	s := strings.Builder{}
	return s.String()
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

func (bs *ballSprite) scheduleReset(onFuture future.Scheduler) {
	bs.resetFuture = onFuture.Schedule(delay.ResetBall, func() {
		bs.resetFuture = nil
		bs.resetPosition()
		bs.startDrawing()
	}, fmt.Sprintf("%s resetting", bs.label))
}

func (bs *ballSprite) scheduleEnable(enable bool, onFuture future.Scheduler) {
	label := "enabling"
	if !enable {
		label = "disabling"
	}

	onFuture.Schedule(delay.EnableBall, func() {
		bs.enable = enable
	}, fmt.Sprintf("%s %s", bs.label, label))
}

func (bs *ballSprite) scheduleVerticalDelay(vdelay bool, onFuture future.Scheduler) {
	label := "enabling vertical delay"
	if !vdelay {
		label = "disabling vertical delay"
	}

	onFuture.Schedule(delay.SetVDELBL, func() {
		bs.verticalDelay = vdelay
	}, fmt.Sprintf("%s %s", bs.label, label))
}
