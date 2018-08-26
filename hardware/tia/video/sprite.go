package video

import (
	"fmt"
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
	colorClock *polycounter.Polycounter

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
	graphicsScanCounter int
	graphicsScanMax     int
	graphicsScanOff     int

	// the amount of horizontal movement for the sprite
	horizMovement uint8

	// a note on whether the sprite is about to be reset its position. the
	// actual reset is scheduled by video.futureWrite
	resetting bool
}

func newSprite(label string, colorClock *polycounter.Polycounter) *sprite {
	sp := new(sprite)
	sp.label = label
	sp.colorClock = colorClock

	sp.position.SetResetPattern("101101")

	// the direction of count and max is important - don't monkey with it
	// the value is used in Pixel*() functions to determine which pixel to check
	sp.graphicsScanMax = 8
	sp.graphicsScanOff = sp.graphicsScanMax + 1
	sp.graphicsScanCounter = sp.graphicsScanOff

	return sp
}

// MachineInfoTerse returns the sprite information in terse format
func (sp sprite) MachineInfoTerse() string {
	pos := fmt.Sprintf("pos=%d", sp.positionResetPixel)
	sig := "dsig=-"
	if sp.isDrawing() {
		sig = fmt.Sprintf("dsig=%d", sp.graphicsScanMax-sp.graphicsScanCounter)
	}
	res := "reset=-"
	if sp.resetting {
		res = "reset=+"
	}
	return fmt.Sprintf("%s: %s %s %s", sp.label, pos, sig, res)
}

// MachineInfo returns the Video information in verbose format
func (sp sprite) MachineInfo() string {
	pos := fmt.Sprintf("reset at pixel %d\nposition: %s", sp.positionResetPixel, sp.position)
	sig := fmt.Sprintf("drawing: inactive")
	if sp.isDrawing() {
		sig = fmt.Sprintf("drawing : from pixel %d", sp.graphicsScanMax-sp.graphicsScanCounter+1)
	}
	res := "no reset scheduled"
	if sp.resetting {
		res = "reset scheduled"
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
	sp.graphicsScanCounter = 0
}

// stopDrawing is used to stop the draw signal prematurely
func (sp *sprite) stopDrawing() {
	sp.graphicsScanCounter = sp.graphicsScanOff
}

func (sp *sprite) isDrawing() bool {
	return sp.graphicsScanCounter <= sp.graphicsScanMax
}

func (sp *sprite) tickGraphicsScan() {
	if sp.isDrawing() {
		sp.graphicsScanCounter++
	}
}
