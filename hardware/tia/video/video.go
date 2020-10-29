// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package video

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia/delay"
	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
)

// Element is used to record from which video sub-system the pixel
// was generated, taking video priority into account.
type Element int

// List of valid Element Signals.
const (
	ElementBackground Element = iota
	ElementBall
	ElementPlayfield
	ElementPlayer0
	ElementPlayer1
	ElementMissile0
	ElementMissile1
)

func (e Element) String() string {
	switch e {
	case ElementBackground:
		return "Background"
	case ElementBall:
		return "Ball"
	case ElementPlayfield:
		return "Playfield"
	case ElementPlayer0:
		return "Player 0"
	case ElementPlayer1:
		return "Player 1"
	case ElementMissile0:
		return "Missile 0"
	case ElementMissile1:
		return "Missile 1"
	}
	panic("unknown video element")
}

// Video contains all the components of the video sub-system of the VCS TIA chip.
type Video struct {
	// collision matrix
	Collisions *Collisions

	// playfield
	Playfield *Playfield

	// sprite objects
	Player0  *PlayerSprite
	Player1  *PlayerSprite
	Missile0 *MissileSprite
	Missile1 *MissileSprite
	Ball     *BallSprite

	// LastElement records from which TIA video sub-system the most recent
	// pixel was generated, taking priority into account. see Pixel() function
	// for details
	LastElement Element

	// keeping track of whether any sprite element has changed since last call
	// to Pixel(). we use this for some small optimisations
	spriteHasChanged    bool
	lastPlayfieldActive bool
	lastPixelColor      uint8
	Unchanged           bool

	// some register writes require a small latching delay. they never overlap
	// so one event is sufficient
	writing         delay.Event
	writingRegister string
}

// NewVideo is the preferred method of initialisation for the Video sub-system.
//
// The playfield type requires access access to the TIA's phaseclock and
// polyucounter and is used to decide which part of the playfield is to be
// drawn.
//
// The sprites meanwhile require access to the television. This is for
// generating information about the sprites reset position - a debugging only
// requirement but of no performance related consequeunce.
//
// The references to the TIA's HBLANK state and whether HMOVE is latched, are
// required to tune the delays experienced by the various sprite events (eg.
// reset position).
func NewVideo(mem bus.ChipBus, tv signal.TelevisionSprite, pclk *phaseclock.PhaseClock, hsync *polycounter.Polycounter, hblank *bool, hmoveLatch *bool) *Video {
	return &Video{
		Collisions: newCollisions(mem),
		Playfield:  newPlayfield(pclk, hsync),
		Player0:    newPlayerSprite("Player 0", tv, hblank, hmoveLatch),
		Player1:    newPlayerSprite("Player 1", tv, hblank, hmoveLatch),
		Missile0:   newMissileSprite("Missile 0", tv, hblank, hmoveLatch),
		Missile1:   newMissileSprite("Missile 1", tv, hblank, hmoveLatch),
		Ball:       newBallSprite("Ball", tv, hblank, hmoveLatch),
	}
}

// Snapshot creates a copy of the Video sub-system in its current state.
func (vd *Video) Snapshot() *Video {
	n := *vd
	n.Collisions = vd.Collisions.Snapshot()
	n.Playfield = vd.Playfield.Snapshot()
	n.Player0 = vd.Player0.Snapshot()
	n.Player1 = vd.Player1.Snapshot()
	n.Missile0 = vd.Missile0.Snapshot()
	n.Missile1 = vd.Missile1.Snapshot()
	n.Ball = vd.Ball.Snapshot()
	return &n
}

func (vd *Video) Plumb(mem bus.ChipBus, pclk *phaseclock.PhaseClock, hsync *polycounter.Polycounter, hblank *bool, hmoveLatch *bool) {
	vd.Collisions.Plumb(mem)
	vd.Playfield.Plumb(pclk, hsync)
	vd.Player0.Plumb(hblank, hmoveLatch)
	vd.Player1.Plumb(hblank, hmoveLatch)
	vd.Missile0.Plumb(hblank, hmoveLatch)
	vd.Missile1.Plumb(hblank, hmoveLatch)
	vd.Ball.Plumb(hblank, hmoveLatch)
}

// RSYNC adjusts the debugging information of the sprites when an RSYNC is
// triggered.
func (vd *Video) RSYNC(adjustment int) {
	vd.Player0.rsync(adjustment)
	vd.Player1.rsync(adjustment)
	vd.Missile0.rsync(adjustment)
	vd.Missile1.rsync(adjustment)
	vd.Ball.rsync(adjustment)
}

// Tick moves all video elements forward one video cycle. This is the
// conceptual equivalent of the hardware MOTCK line.
func (vd *Video) Tick(visible, hmove bool, hmoveCt uint8) {
	if v, ok := vd.writing.Tick(); ok {
		switch vd.writingRegister {
		case "PF0":
			vd.Playfield.setPF0(v)
		case "PF1":
			vd.Playfield.setPF1(v)
		case "PF2":
			vd.Playfield.setPF2(v)
		case "HMP0":
			vd.Player0.setHmoveValue(v)
		case "HMP1":
			vd.Player1.setHmoveValue(v)
		case "HMM0":
			vd.Missile0.setHmoveValue(v)
		case "HMM1":
			vd.Missile1.setHmoveValue(v)
		case "HMBL":
			vd.Ball.setHmoveValue(v)
		case "HMCLR":
			vd.Player0.clearHmoveValue()
			vd.Player1.clearHmoveValue()
			vd.Missile0.clearHmoveValue()
			vd.Missile1.clearHmoveValue()
			vd.Ball.clearHmoveValue()
		}
	}

	p0 := vd.Player0.tick(visible, hmove, hmoveCt)
	p1 := vd.Player1.tick(visible, hmove, hmoveCt)
	m0 := vd.Missile0.tick(visible, hmove, hmoveCt, vd.Player0.triggerMissileReset())
	m1 := vd.Missile1.tick(visible, hmove, hmoveCt, vd.Player1.triggerMissileReset())
	bl := vd.Ball.tick(visible, hmove, hmoveCt)
	vd.spriteHasChanged = vd.spriteHasChanged || p0 || p1 || m0 || m1 || bl
}

// PrepareSpritesForHMOVE should be called whenever HMOVE is triggered.
func (vd *Video) PrepareSpritesForHMOVE() {
	vd.Player0.prepareForHMOVE()
	vd.Player1.prepareForHMOVE()
	vd.Missile0.prepareForHMOVE()
	vd.Missile1.prepareForHMOVE()
	vd.Ball.prepareForHMOVE()
}

// Pixel returns the color of the pixel at the current clock and also sets the
// collision registers. It will default to returning the background color if no
// sprite or playfield pixel is present.
func (vd *Video) Pixel() uint8 {
	pfa, pfc := vd.Playfield.pixel()

	// optimisation: if nothing has changed since last pixel then return early
	// with the color of the previous pixel. note that we're not optimising
	// based on whether video is on/off (ie. VBLANK/HBLANK)
	if !vd.spriteHasChanged && (!pfa || (pfa && !vd.lastPlayfieldActive)) {
		vd.spriteHasChanged = false
		vd.lastPlayfieldActive = pfa
		vd.Unchanged = true
		return vd.lastPixelColor
	}
	vd.spriteHasChanged = false
	vd.lastPlayfieldActive = pfa
	vd.Unchanged = false

	bgc := vd.Playfield.BackgroundColor
	p0a, p0c, p0k := vd.Player0.pixel()
	p1a, p1c, p1k := vd.Player1.pixel()
	m0a, m0c, m0k := vd.Missile0.pixel()
	m1a, m1c, m1k := vd.Missile1.pixel()
	bla, blc, blk := vd.Ball.pixel()

	// the sprites return a third value which we'll call the collision
	// condition. this condition only applies when detecting collisions with
	// other sprites. it is not used when detecting collisions with the
	// playfield. for playfield collisions we just use the active condition
	// (the first returned value)
	vd.Collisions.tick(p0k, p1k, m0k, m1k, blk, pfa)

	// apply priorities to get pixel color
	var col uint8
	var element Element

	// the interaction of the priority and scoremode bits are a little more
	// complex than at first glance:
	//
	//  o if the priority bit is set then priority ordering applies
	//  o if it is not set but scoremode is set and we're in the left half of
	//		the screen, then priority ordering also applies
	//	o if priority bit is not set but scoremode is set and we're in the
	//		right hand side of the screen then regular ordering applies, except
	//		that playfield has priority over the ball
	//	o phew
	//
	//	that scoremode reorders priority regardless of the priority bit is not
	//	at all obvious but observation proves it to be true. see test.bin ROM
	//
	//	the comment by "supercat" in the discussion "Playfield Score Mode -
	//	effect on ball" on AtariAge proved useful here.
	//
	//	!!TODO: I'm still not 100% sure this is correct. check playfield priorties
	if vd.Playfield.Priority || (vd.Playfield.Scoremode && vd.Playfield.Region == RegionLeft) {
		if pfa { // priority 1
			if vd.Playfield.Scoremode && !vd.Playfield.Priority {
				switch vd.Playfield.Region {
				case RegionLeft:
					col = p0c
				case RegionRight:
					col = p1c
				}
			} else {
				col = pfc
			}

			element = ElementPlayfield
		} else if bla {
			col = blc
			element = ElementBall
		} else if p0a { // priority 2
			col = p0c
			element = ElementPlayer0
		} else if m0a {
			col = m0c
			element = ElementMissile0
		} else if p1a { // priority 3
			col = p1c
			element = ElementPlayer1
		} else if m1a {
			col = m1c
			element = ElementMissile1
		} else {
			col = bgc
			element = ElementBackground
		}
	} else {
		if p0a { // priority 1
			col = p0c
			element = ElementPlayer0
		} else if m0a {
			col = m0c
			element = ElementMissile0
		} else if p1a { // priority 2
			col = p1c
			element = ElementPlayer1
		} else if m1a {
			col = m1c
			element = ElementMissile1
		} else if vd.Playfield.Scoremode && (bla || pfa) {
			// priority 3 (scoremode without priority bit)
			if pfa {
				col = pfc
				switch vd.Playfield.Region {
				case RegionLeft:
					col = p0c
				case RegionRight:
					col = p1c
				}
				element = ElementPlayfield
			} else if bla { // priority 3
				col = blc
				element = ElementBall
			}
		} else {
			// priority 3 (no scoremode or priority bit)
			if bla { // priority 3
				col = blc
				element = ElementBall
			} else if pfa {
				col = pfc
				element = ElementPlayfield
			} else {
				col = bgc
				element = ElementBackground
			}
		}
	}

	vd.LastElement = element
	vd.lastPixelColor = col

	// priority 4
	return col
}

// UpdatePlayfield checks TIA memory for new playfield data. Note that CTRLPF
// is serviced in UpdateSpriteVariations().
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdatePlayfield(data bus.ChipData) bool {
	// homebrew Donkey Kong shows the need for a delay of at least two cycles
	// to write new playfield data
	switch data.Name {
	case "PF0":
		vd.writingRegister = "PF0"
		vd.writing.Schedule(2, data.Value)
	case "PF1":
		vd.writingRegister = "PF1"
		vd.writing.Schedule(2, data.Value)
	case "PF2":
		vd.writingRegister = "PF2"
		vd.writing.Schedule(2, data.Value)
	case "VDELBL":
		vd.spriteHasChanged = true
		vd.Ball.setVerticalDelay(data.Value&0x01 == 0x01)
	default:
		return true
	}

	return false
}

// UpdateSpriteHMOVE checks TIA memory for changes in sprite HMOVE settings.
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdateSpriteHMOVE(data bus.ChipData) bool {
	switch data.Name {
	// horizontal movement values range from -8 to +7 for convenience we
	// convert this to the range 0 to 15. from TIA_HW_Notes.txt:
	//
	// "You may have noticed that the [...] discussion ignores the
	// fact that HMxx values are specified in the range +7 to -8.
	// In an odd twist, this was done purely for the convenience
	// of the programmer! The comparator for D7 in each HMxx latch
	// is wired up in reverse, costing nothing in silicon and
	// effectively inverting this bit so that the value can be
	// treated as a simple 0-15 count for movement left. It might
	// be easier to think of this as having D7 inverted when it
	// is stored in the first place."

	// there is no information about whether response to HMOVE value changes
	// are immediate or take effect after a short delay. experimentation
	// reveals that a delay is required. the reasoning for the value is as
	// below:
	//
	// delay of at least zero (1 additiona cycle) is required. we can see this
	// in the Midnight Magic ROM where the left gutter separator requires it
	//
	// a delay too high (3 or higher) causes the barber pole test ROM to fail
	//
	// (19/01/20) a delay of anything other than 0 or 1, causes Panda Chase to
	// fail.
	//
	// (28/01/20) a delay of anything lower than 1, causes the text in the
	// BASIC ROM to fail
	//
	// the only common value that satisfies all test cases is 1, which equates
	// to a delay of two cycles
	case "HMP0":
		vd.writingRegister = "HMP0"
		vd.writing.Schedule(1, data.Value&0xf0)
	case "HMP1":
		vd.writingRegister = "HMP1"
		vd.writing.Schedule(1, data.Value&0xf0)
	case "HMM0":
		vd.writingRegister = "HMM0"
		vd.writing.Schedule(1, data.Value&0xf0)
	case "HMM1":
		vd.writingRegister = "HMM1"
		vd.writing.Schedule(1, data.Value&0xf0)
	case "HMBL":
		vd.writingRegister = "HMBL"
		vd.writing.Schedule(1, data.Value&0xf0)
	case "HMCLR":
		vd.writingRegister = "HMCLR"
		vd.writing.Schedule(1, 0)

	default:
		return true
	}

	vd.spriteHasChanged = true
	return false
}

// UpdateSpritePositioning checks TIA memory for strobing of reset registers.
//
// Returns true if memory.ChipData has not been serviced.
func (vd *Video) UpdateSpritePositioning(data bus.ChipData) bool {
	switch data.Name {
	// the reset registers *must* be serviced after HSYNC has been ticked.
	// resets are resolved after a short delay, governed by the sprite itself
	case "RESP0":
		vd.Player0.resetPosition()
	case "RESP1":
		vd.Player1.resetPosition()
	case "RESM0":
		vd.Missile0.resetPosition()
	case "RESM1":
		vd.Missile1.resetPosition()
	case "RESBL":
		vd.Ball.resetPosition()
	default:
		return true
	}

	vd.spriteHasChanged = true
	return false
}

// UpdateColor checks TIA memory for changes to color registers.
//
// Returns true if memory.ChipData has not been serviced.
func (vd *Video) UpdateColor(data bus.ChipData) bool {
	switch data.Name {
	case "COLUP0":
		vd.Player0.setColor(data.Value & 0xfe)
		vd.Missile0.setColor(data.Value & 0xfe)
	case "COLUP1":
		vd.Player1.setColor(data.Value & 0xfe)
		vd.Missile1.setColor(data.Value & 0xfe)
	case "COLUBK":
		vd.Playfield.setBackground(data.Value & 0xfe)
	case "COLUPF":
		vd.Playfield.setColor(data.Value & 0xfe)
		vd.Ball.setColor(data.Value & 0xfe)
	default:
		return true
	}

	return false
}

// UpdateSpritePixels checks TIA memory for attribute changes that *must* occur
// after a call to Pixel().
//
// Returns true if memory.ChipData has not been serviced.
func (vd *Video) UpdateSpritePixels(data bus.ChipData) bool {
	// the barnstormer ROM demonstrate perfectly how GRP0 is affected if we
	// alter its state before a call to Pixel().  if we write do alter state
	// before Pixel(), then an unwanted artefact can be seen on scanline 61.
	switch data.Name {
	case "GRP0":
		vd.Player0.setGfxData(data.Value)
		vd.Player1.setOldGfxData()

	case "GRP1":
		vd.Player1.setGfxData(data.Value)
		vd.Player0.setOldGfxData()
		vd.Ball.setEnableDelay()

	case "ENAM0":
		vd.Missile0.setEnable(data.Value&0x02 == 0x02)
	case "ENAM1":
		vd.Missile1.setEnable(data.Value&0x02 == 0x02)
	case "ENABL":
		vd.Ball.setEnable(data.Value&0x02 == 0x02)
	default:
		return true
	}

	vd.spriteHasChanged = true
	return false
}

// UpdateSpriteVariations checks TIA memory for writes to registers that affect
// how sprite pixels are output. Note that CTRLPF is serviced here rather than
// in UpdatePlayfield(), because it affects the ball sprite.
//
// Returns true if memory.ChipData has not been serviced.
func (vd *Video) UpdateSpriteVariations(data bus.ChipData) bool {
	switch data.Name {
	case "CTRLPF":
		vd.Ball.SetCTRLPF(data.Value)
		vd.Playfield.SetCTRLPF(data.Value)
	case "VDELP0":
		vd.Player0.SetVerticalDelay(data.Value&0x01 == 0x01)
	case "VDELP1":
		vd.Player1.SetVerticalDelay(data.Value&0x01 == 0x01)
	case "REFP0":
		vd.Player0.setReflection(data.Value&0x08 == 0x08)
	case "REFP1":
		vd.Player1.setReflection(data.Value&0x08 == 0x08)
	case "RESMP0":
		vd.Missile0.setResetToPlayer(data.Value&0x02 == 0x02)
	case "RESMP1":
		vd.Missile1.setResetToPlayer(data.Value&0x02 == 0x02)
	case "NUSIZ0":
		vd.Player0.setNUSIZ(data.Value)
		vd.Missile0.SetNUSIZ(data.Value)
	case "NUSIZ1":
		vd.Player1.setNUSIZ(data.Value)
		vd.Missile1.SetNUSIZ(data.Value)
	case "CXCLR":
		vd.Collisions.Clear()
	default:
		return true
	}

	vd.spriteHasChanged = true
	return false
}

// UpdateCTRLPF should be called whenever any of the individual components of
// the CTRPF are altered. For example, if Playfield.Reflected is altered, then
// this function should be called so that the CTRLPF value is set to reflect
// the alteration.
//
// This is only of use to debuggers. It's never required in normal operation of
// the emulator.
func (vd *Video) UpdateCTRLPF() {
	vd.Ball.Size &= 0x03
	ctrlpf := vd.Ball.Size << 4

	if vd.Playfield.Reflected {
		ctrlpf |= 0x01
	}
	if vd.Playfield.Scoremode {
		ctrlpf |= 0x02
	}
	if vd.Playfield.Priority {
		ctrlpf |= 0x04
	}

	vd.Playfield.Ctrlpf = ctrlpf
	vd.Ball.Ctrlpf = ctrlpf
	vd.spriteHasChanged = true
}

// UpdateNUSIZ should be called whenever the player/missile size/copies
// information is altered. This function updates the NUSIZ value to reflect the
// changes whilst maintaining the other NUSIZ bits.
//
// This is only of use to debuggers. It's never required in normal operation of
// the emulator.
func (vd *Video) UpdateNUSIZ(num int, fromMissile bool) {
	var nusiz uint8

	if num == 0 {
		if fromMissile {
			vd.Missile0.Copies &= 0x07
			vd.Missile0.Size &= 0x03
			vd.Player0.SizeAndCopies = vd.Missile0.Copies
			nusiz = vd.Missile0.Copies | vd.Missile0.Size<<4
		} else {
			vd.Player0.SizeAndCopies &= 0x07
			vd.Missile0.Copies = vd.Player0.SizeAndCopies
			nusiz = vd.Player0.SizeAndCopies | vd.Missile0.Size<<4
		}
		vd.Player0.Nusiz = nusiz
		vd.Missile0.Nusiz = nusiz
	} else {
		if fromMissile {
			vd.Missile1.Copies &= 0x07
			vd.Missile1.Size &= 0x03
			vd.Player1.SizeAndCopies = vd.Missile1.Copies
			nusiz = vd.Missile1.Copies | vd.Missile1.Size<<4
		} else {
			vd.Player1.SizeAndCopies &= 0x07
			vd.Missile1.Copies = vd.Player1.SizeAndCopies
			nusiz = vd.Player1.SizeAndCopies | vd.Missile1.Size<<4
		}
		vd.Player1.Nusiz = nusiz
		vd.Missile1.Nusiz = nusiz
	}
	vd.spriteHasChanged = true
}
