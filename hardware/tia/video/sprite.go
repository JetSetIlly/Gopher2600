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

	// horizontal position of the sprite - may be affected by HMOVE
	horizPos int

	// horizontal position after hmove has been applied
	hmovedHorizPos int

	// the draw signal controls which "bit" of the sprite is to be drawn next.
	// generally, the draw signal is activated when the position polycounter
	// matches the colorClock polycounter, but differenct sprite types handle
	// this differently in certain circumstances
	graphicsScanCounter int
	graphicsScanMax     int
	graphicsScanOff     int

	// the amount of horizontal movement for the sprite
	// -- as set by the 6502 - normalised into the 0 to 15 range
	// (note that negative numbers indicate movements to the right)
	horizMovement int
	// -- whether HMOVE is still affecting this sprite
	horizMovementLatch bool

	// the tick function that wraps the tickPosition() function
	// - this function is called instead of the local tickPosition() function - the
	// ticker function will calls tickPosition() as appropriate
	tick func()

	// a note on whether the sprite is about to be reset its position
	resetFuture *future.Instance

	// 0 = force reset is off
	// 1 = force reset trigger
	// n = wait for trigger
	forceReset int
	// see comment in tickSpritesForHMOVE()
}

func newSprite(label string, colorClock *polycounter.Polycounter, tick func()) *sprite {
	sp := new(sprite)
	sp.label = label
	sp.colorClock = colorClock
	sp.tick = tick

	sp.position = *polycounter.New6Bit()
	sp.position.SetResetPoint(39) // "101101"

	// the direction of count and max is important - don't monkey with it
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
	s.WriteString(fmt.Sprintf(" pix=%d", sp.horizPos))
	s.WriteString(fmt.Sprintf(" {hm=%d}", sp.horizMovement-8))
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
	s.WriteString(fmt.Sprintf("   reset pos: %s\n", sp.position))
	s.WriteString(fmt.Sprintf("   hmove: %d [%#02x] %04b\n", sp.horizMovement-8, (sp.horizMovement<<4)^0x80, sp.horizMovement))
	s.WriteString(fmt.Sprintf("   pixel: %d\n", sp.horizPos))
	s.WriteString(fmt.Sprintf("   adj pixel: %d", sp.hmovedHorizPos))
	if sp.horizMovementLatch {
		s.WriteString(" *\n")
	} else {
		s.WriteString("\n")
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

func (sp *sprite) resetPosition() {
	sp.position.Reset()

	// note reset position of sprite, in pixels
	sp.horizPos = sp.colorClock.Pixel()
	sp.hmovedHorizPos = sp.horizPos
}

func (sp *sprite) tickPosition(triggerList []int) bool {
	if sp.position.Tick() {
		return true
	}

	// check for start positions of additional copies of the sprite
	for _, v := range triggerList {
		if v == sp.position.Count && sp.position.Phase == 0 {
			return true
		}
	}

	return false
}

func (sp *sprite) PrepareForHMOVE() {
	// start horizontal movment of this sprite
	sp.horizMovementLatch = true
	sp.hmovedHorizPos = sp.horizPos
}

func (sp *sprite) tickSpritesForHMOVE(count int) {
	if sp.horizMovementLatch {
		// bitwise comparison - if no bits match then unset the latch,
		// otherwise continue with the HMOVE for this sprite
		if sp.horizMovement&count == 0 {
			sp.horizMovementLatch = false
		} else {
			// this mental construct is designed to fix a problem in the Keystone
			// Kapers ROM. I don't believe for a moment that this is a perfect
			// solution but it makes sense in the context of that ROM.
			//
			// What seems to be happening in Keystone Kapers ROM is this:
			//
			//	o Ball is reset at end of scanline 95 ($f756); and other scanlines
			//  o HMOVE is tripped at beginning of line 96
			//  o but reset doesn't occur until we resume motion clocks, by which
			//		time HMOVE is finished
			//  o moreover, the game doesn't want the ball to appear at the
			//		beginning of the visible part of the screen; it wants the ball
			//		to appear in the HMOVE gutter on scanlines 97 and 98; so the
			//		move adjustments needs to happen such that the ball really
			//		appears at the end of the scanline
			//  o to cut a long story short, the game needs the ball to have been
			//		reset before the HMOVE has completed on line 96
			//
			// confusing huh?  this delay construct fixes the above issue while not
			// breaking other regression tests. I don't know if this is a generally
			// correct solution or if it's specific to the ball sprite but I'm
			// keeping it in for now.
			if sp.resetFuture != nil {
				if sp.forceReset == 1 {
					sp.resetFuture.Force()
					sp.forceReset = 0
				} else if sp.forceReset == 0 {
					sp.forceReset = delayForceReset
				} else {
					sp.forceReset--
				}
			}

			sp.tick()

			sp.hmovedHorizPos--
			if sp.hmovedHorizPos < 0 {
				sp.hmovedHorizPos = 160
			}
		}
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
