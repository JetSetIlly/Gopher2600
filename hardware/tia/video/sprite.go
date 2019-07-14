package video

import (
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"strings"
)

// the sprite type is used for those video elements that move about - players,
// missiles and the ball. the VCS doesn't really have anything called a sprite
// but we all know what it means
type sprite struct {
	// label is the name of a particular instance of a sprite (eg. player0 or
	// missile 1)
	label string

	tiaclk *phaseclock.PhaseClock

	// position of the sprite as a polycounter value - the basic principle
	// behind VCS sprites is to begin drawing of the sprite when position
	// circulates to zero
	position polycounter.Polycounter

	// the draw signal controls which "bit" of the sprite is to be drawn next.
	// generally, the draw signal is activated when the position polycounter
	// matches the hsync polycounter, but differenct sprite types handle
	// this differently in certain circumstances
	graphicsScanCounter int
	graphicsScanMax     int
	graphicsScanOff     int

	// horizontal position of the sprite - may be affected by HMOVE
	resetPixel int

	// horizontal position after hmove has been applied
	currentPixel int

	// the amount of horizontal movement for the sprite
	// -- as set by the 6502 - written into the HMP0/P1/M0/M1/BL register
	// -- normalised into the 0 to 15 range
	horizMovement int
	// -- whether HMOVE is still affecting this sprite
	moreMovementRequired bool

	// each type of sprite has slightly different spriteTick logic which needs
	// to be called from within the HMOVE logic common to all sprite types
	spriteTick func()

	// a note on whether the sprite is about to be reset its position
	resetFuture *future.Event
}

func newSprite(label string, tiaclk *phaseclock.PhaseClock, spriteTick func()) *sprite {
	sp := sprite{label: label, tiaclk: tiaclk, spriteTick: spriteTick}

	// the direction of count and max is important - don't monkey with it
	sp.graphicsScanMax = 8
	sp.graphicsScanOff = sp.graphicsScanMax + 1
	sp.graphicsScanCounter = sp.graphicsScanOff

	return &sp
}

// MachineInfoTerse returns the sprite information in terse format
func (sp sprite) MachineInfoTerse() string {
	s := strings.Builder{}
	return s.String()
}

// MachineInfo returns the Video information in verbose format
func (sp sprite) MachineInfo() string {
	s := strings.Builder{}
	return s.String()
}

func (sp *sprite) resetPosition() {
	sp.position.Reset()

	// note reset position of sprite, in pixels
	sp.resetPixel = -68 + int((sp.position.Count * 4)) + int(*sp.tiaclk)
	sp.currentPixel = sp.resetPixel
}

func (sp *sprite) checkForGfxStart(triggerList []int) (bool, bool) {
	if sp.tiaclk.InPhase() {
		if sp.position.Tick() {
			return true, false
		}

		// check for start positions of additional copies of the sprite
		for _, v := range triggerList {
			if v == int(sp.position.Count) {
				return true, true
			}
		}
	}

	return false, false
}

func (sp *sprite) prepareForHMOVE() {
	// start horizontal movment of this sprite
	sp.moreMovementRequired = true

	// at beginning of hmove sequence, without knowing anything else, the final
	// position of the sprite will be the current position plus 8. the actual
	// value will be reduced depending on what happens during hmove ticking.
	// factors that effect the final position:
	//   o the value in the horizontal movement register (eg. HMP0)
	//   o whether the ticking is occuring during the hblank period
	// both these factors are considered in the resolveHorizMovement() function
	sp.currentPixel += 8
}

func compareBits(a, b uint8) bool {
	// return true if any corresponding bits in the lower nibble are the same.
	// not the same test as a&b!=0. from Towers' TIA_HW_Notes:
	// "When the comparator for a given object detects that none of the 4 bits
	// match the bits in the counter state, it clears this latch"
	return a&0x08 == b&0x08 || a&0x04 == b&0x04 || a&0x02 == b&0x02 || a&0x01 == b&0x01
}

func (sp *sprite) resolveHMOVE(count int) {
	sp.moreMovementRequired = sp.moreMovementRequired && compareBits(uint8(count), uint8(sp.horizMovement))

	if sp.moreMovementRequired {
		// adjust position information
		sp.currentPixel--
		if sp.currentPixel < 0 {
			sp.currentPixel = 159
		}

		// perform an additional tick of the sprite (different sprite types
		// have different tick logic)
		sp.spriteTick()
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
