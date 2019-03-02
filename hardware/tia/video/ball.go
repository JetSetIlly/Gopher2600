package video

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/video/future"
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

	// pixelDelayAfterReset is an horrendous hack intended to emulate the
	// effect caused by resetting the ball sprite on the exact pixel a graphics
	// scan starts.  I've tried all manner of other solutions but none seem to
	// work while keeping all other desirable emulation traits intact.
	//
	// if the reset is /scheduled/ on the exact pixel that the ball is sprite
	// is to be drawn then there is a short delay. the ball continues to tick
	// in the normal way but if this delay is still active then the pixel is
	// not drawn.
	pixelDelayAfterReset int
}

func newBallSprite(label string, colorClock *polycounter.Polycounter) *ballSprite {
	bs := new(ballSprite)
	bs.sprite = newSprite(label, colorClock, bs.tick)
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
	if bs.checkForGfxStart(nil) {
		bs.startDrawing()
	} else {
		bs.tickGraphicsScan()
	}

	if bs.pixelDelayAfterReset > 0 {
		bs.pixelDelayAfterReset--
	}
}

// pixel returns the color of the ball at the current time.  returns
// (false, 0) if no pixel is to be seen; and (true, col) if there is
func (bs *ballSprite) pixel() (bool, uint8) {
	// ball should be pixelled if:
	//  o ball is enabled and vertical delay is not enabled
	//  o OR ball was previously enabled and vertical delay is enabled
	//  o AND a reset signal (RESBL) has not recently been triggered
	if (!bs.verticalDelay && bs.enable) || (bs.verticalDelay && bs.enablePrev) {
		if bs.resetFuture == nil || (bs.resetFuture != nil && bs.pixelDelayAfterReset > 0) {
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
	}
	return false, 0
}

func (bs *ballSprite) scheduleReset(onFuture *future.Group) {
	if bs.position.CycleOnNextTick() {
		bs.pixelDelayAfterReset = 3
	} else {
		bs.pixelDelayAfterReset = 0
	}

	bs.resetFuture = onFuture.Schedule(delayResetBall, func() {
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

	onFuture.Schedule(delayEnableBall, func() {
		bs.enable = enable
	}, fmt.Sprintf("%s %s", bs.label, label))
}

func (bs *ballSprite) scheduleVerticalDelay(delay bool, onFuture *future.Group) {
	label := "enabling vertical delay"
	if !delay {
		label = "disabling vertical delay"
	}

	onFuture.Schedule(delayVDELBL, func() {
		bs.verticalDelay = delay
	}, fmt.Sprintf("%s %s", bs.label, label))
}
