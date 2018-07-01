package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
	"gopher2600/hardware/tia/polycounter"
)

// the sprite type is used for those video elements that move about - players,
// missiles and the ball. the VCS doesn't really have anything called a sprite
// but we all know what it means
type sprite struct {
	// label is the name of a particular instance of a sprite (eg. player0 or
	// missile 1)
	label string

	// colorClock references the VCS wide color clock. we only use it to note
	// the Pixel() value of the color clock at the reset point of the sprite.
	colorClock *colorclock.ColorClock

	// all sprites have a slight delay when resetting position
	futureReset *future

	// position of the sprite as a polycounter value - the basic principle
	// behind VCS sprites is to begin drawing of the sprite when position
	// circulates to 0000000
	position polycounter.Polycounter

	// reset position of the sprite -- does not take horizonal movement into
	// account
	positionResetPixel int

	// the draw signal controls which "bit" of the sprite is to be drawn next.
	// generally, the draw signal is activated when the position polycounter
	// matches the colorClock polycounter, but differenct sprite types handle
	// this differently in certain circumstances
	drawSigCount int
	drawSigMax   int
	drawSigOff   int

	// the amount of horizontal movement for the sprite
	horizMovement uint8
}

func newSprite(label string, colorClock *colorclock.ColorClock) *sprite {
	sp := new(sprite)
	if sp == nil {
		return nil
	}

	sp.label = label
	sp.colorClock = colorClock

	sp.futureReset = newFuture()
	if sp.futureReset == nil {
		return nil
	}

	sp.position.SetResetPattern("101101")

	// the direction of count and max is important - don't monkey with it
	// the value is used in Pixel*() functions to determine which pixel to check
	sp.drawSigMax = 8
	sp.drawSigOff = sp.drawSigMax + 1
	sp.drawSigCount = sp.drawSigOff

	return sp
}

// MachineInfoTerse returns the sprite information in terse format
func (sp sprite) MachineInfoTerse() string {
	pos := fmt.Sprintf("pos=%d", sp.positionResetPixel)
	sig := "dsig=-"
	if sp.isDrawing() {
		sig = fmt.Sprintf("dsig=%d", sp.drawSigMax-sp.drawSigCount)
	}
	res := "reset=-"
	if sp.futureReset.isScheduled() {
		res = fmt.Sprintf("reset=%d", sp.futureReset.remainingCycles)
	}
	return fmt.Sprintf("%s: %s %s %s", sp.label, pos, sig, res)
}

// MachineInfo returns the Video information in verbose format
func (sp sprite) MachineInfo() string {
	pos := fmt.Sprintf("reset at pixel %d\nposition: %s", sp.positionResetPixel, sp.position)
	sig := fmt.Sprintf("drawsig: inactive")
	if sp.isDrawing() {
		sig = fmt.Sprintf("drawsig: pixel %d", sp.drawSigMax-sp.drawSigCount+1)
	}
	res := "reset: none scheduled"
	if sp.futureReset.isScheduled() {
		plural := ""
		if sp.futureReset.remainingCycles != 1 {
			plural = "s"
		}
		res = fmt.Sprintf("reset: in %d cycle%s", sp.futureReset.remainingCycles, plural)
	}
	return fmt.Sprintf("%s: %s\n %s\n %s", sp.label, pos, sig, res)
}

func (sp *sprite) resetPosition() {
	sp.position.Reset()

	// note reset position of sprite, in pixels. used to MachineInfo()
	// functions
	sp.positionResetPixel = sp.colorClock.Pixel()
}

func (sp *sprite) tickPosition(triggerList []int) bool {
	if sp.position.Tick(false) {
		return true
	}

	for _, v := range triggerList {
		if v == sp.position.Count && sp.position.Phase == 0 {
			return true
		}
	}

	return false
}

func (sp *sprite) startDrawing() {
	sp.drawSigCount = 0
}

// stopDrawing is used to stop the draw signal prematurely
func (sp *sprite) stopDrawing() {
	sp.drawSigCount = sp.drawSigOff
}

func (sp *sprite) isDrawing() bool {
	return sp.drawSigCount <= sp.drawSigMax
}

func (sp *sprite) tickDrawSig() {
	if sp.isDrawing() {
		sp.drawSigCount++
	}
}
