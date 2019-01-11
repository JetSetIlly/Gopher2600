package video

import (
	"fmt"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/video/future"
	"strings"
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
	// circulates to zero
	position polycounter.Polycounter

	// pixel of the sprite
	horizPos int

	// adjusted horizontal position
	adjHorizPos int

	// the draw signal controls which "bit" of the sprite is to be drawn next.
	// generally, the draw signal is activated when the position polycounter
	// matches the colorClock polycounter, but differenct sprite types handle
	// this differently in certain circumstances
	graphicsScanCounter int
	graphicsScanMax     int
	graphicsScanOff     int

	// the amount of horizontal movement for the sprite
	horizMovement      uint8
	horizMovementLatch bool

	// the tick function that wraps the tickPosition() function
	// - this function is called instead of the local tickPosition() function - the
	// ticker function will calls tickPosition() as appropriate
	tick func()

	// a note on whether the sprite is about to be reset its position
	resetFuture *future.Instance
}

func newSprite(label string, colorClock *polycounter.Polycounter, tick func()) *sprite {
	sp := new(sprite)
	sp.label = label
	sp.colorClock = colorClock
	sp.tick = tick

	sp.position = *polycounter.New6Bit()
	sp.position.SetResetPoint(39) // "101101"

	// the direction of count and max is important - don't monkey with it
	// the value is used in Pixel*() functions to determine which pixel to check
	sp.graphicsScanMax = 8
	sp.graphicsScanOff = sp.graphicsScanMax + 1
	sp.graphicsScanCounter = sp.graphicsScanOff

	return sp
}

// MachineInfoTerse returns the sprite information in terse format
func (sp sprite) MachineInfoTerse() string {
	s := strings.Builder{}
	s.WriteString(sp.label)
	s.WriteString(": ")
	s.WriteString(sp.position.String())
	if sp.adjHorizPos == sp.horizPos {
		s.WriteString(fmt.Sprintf(" pix=%d", sp.adjHorizPos))
		s.WriteString(fmt.Sprintf(" hm=%d", sp.horizMovement))
	} else {
		s.WriteString(fmt.Sprintf(" {pix=%d", sp.adjHorizPos))
		s.WriteString(fmt.Sprintf(" hm=%d}", sp.horizMovement))
	}
	if sp.isDrawing() {
		s.WriteString(fmt.Sprintf(" drw=%d", sp.graphicsScanMax-sp.graphicsScanCounter))
	} else {
		s.WriteString(" drw=-")
	}
	if sp.resetFuture == nil {
		s.WriteString(" res=-")
	} else {
		s.WriteString(fmt.Sprintf(" res=%d", sp.resetFuture.RemainingCycles))
	}

	return s.String()
}

// MachineInfo returns the Video information in verbose format
func (sp sprite) MachineInfo() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("%s:\n", sp.label))
	s.WriteString(fmt.Sprintf("   position: %s\n", sp.position))
	if sp.adjHorizPos == sp.horizPos {
		s.WriteString(fmt.Sprintf("   pixel: %d\n", sp.adjHorizPos))
	} else {
		s.WriteString(fmt.Sprintf("   pixel: %d {%d}\n", sp.horizPos, sp.adjHorizPos))
	}
	s.WriteString(fmt.Sprintf("   hmove: %d\n", sp.horizMovement))
	if sp.isDrawing() {
		s.WriteString(fmt.Sprintf("   drawing: %d\n", sp.graphicsScanMax-sp.graphicsScanCounter))
	} else {
		s.WriteString("   drawing: inactive\n")
	}
	if sp.resetFuture == nil {
		s.WriteString("   reset: none scheduled\n")
	} else {
		s.WriteString(fmt.Sprintf("   reset: %d cycles\n", sp.resetFuture.RemainingCycles))
	}

	return s.String()
}

// MachineInfoInternal returns low state information about the type
func (sp sprite) MachineInfoInternal() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%04b ", sp.horizMovement))
	if sp.horizMovementLatch {
		s.WriteString("*")
	} else {
		s.WriteString(" ")
	}
	s.WriteString(" ")
	s.WriteString(sp.label)
	return s.String()
}

// newScanline is called at the beginning of a new scanline.
// -- this is only used to reset the adjusted horizontal position value that we
// use to report the horizontal location of the sprite. a bit of a waste
// perhaps.
// -- an alternative would be reset this value based on the position polycounter
// value. but that wouldn't be wholly accurate (information-wise) in all
// instances. that said, the additional function call doesn't impact that much
// on performance, it just feels ugly.
func (sp *sprite) newScanline() {
	// reset adjusted horizontal position value
	sp.adjHorizPos = sp.horizPos
}

func (sp *sprite) resetPosition() {
	sp.position.Reset()

	// note reset position of sprite, in pixels. used in MachineInfo()
	// functions
	if sp.colorClock.Count > 15 {
		sp.horizPos = sp.colorClock.Pixel()
	} else {
		sp.horizPos = 2
	}
	sp.adjHorizPos = sp.horizPos
}

func (sp *sprite) tickPosition(triggerList []int) bool {
	if sp.position.Tick() {
		return true
	}

	for _, v := range triggerList {
		if v == sp.position.Count && sp.position.Phase == 0 {
			return true
		}
	}

	return false
}

func (sp *sprite) tickSpritesForHMOVE(count int) {
	if sp.horizMovementLatch && (sp.horizMovement&uint8(count) != 0) {
		sp.tick()

		// adjust horizontal position
		if sp.horizMovement > 8 {
			if count > 8 {
				sp.adjHorizPos--
				if sp.adjHorizPos < 0 {
					sp.adjHorizPos = 159
				}
			}
		} else if sp.horizMovement < 8 {
			if 8-int(sp.horizMovement)-count >= 0 {
				sp.adjHorizPos++
				if sp.adjHorizPos > 159 {
					sp.adjHorizPos = 0
				}
			}
		}
	} else {
		sp.horizMovementLatch = false
	}
}

func (sp *sprite) startDrawing() {
	sp.graphicsScanCounter = 0
}

func (sp *sprite) isDrawing() bool {
	return sp.graphicsScanCounter <= sp.graphicsScanMax
}

func (sp *sprite) tickGraphicsScan() {
	if sp.isDrawing() {
		sp.graphicsScanCounter++
	}
}
