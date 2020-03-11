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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package video

import (
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/bus"
	"gopher2600/hardware/tia/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/television"
	"gopher2600/television/colors"
)

// Video contains all the components of the video sub-system of the VCS TIA chip
type Video struct {
	// collision matrix
	collisions *collisions

	// playfield
	Playfield *playfield

	// sprite objects
	Player0  *playerSprite
	Player1  *playerSprite
	Missile0 *missileSprite
	Missile1 *missileSprite
	Ball     *ballSprite
}

// NewVideo is the preferred method of initialisation for the Video structure.
//
// The playfield type requires access access to the TIA's phaseclock and
// polyucounter and is used to decide which part of the playfield is to be
// drawn.
//
// The sprites meanwhile require access to the television. This is for
// generating information about the sprites reset position - a debugging only
// requirement but of minimal performance related consequeunce.
//
// The references to the TIA's HBLANK state and whether HMOVE is latched, are
// required to tune the delays experienced by the various sprite events (eg.
// reset position).
func NewVideo(mem bus.ChipBus,
	pclk *phaseclock.PhaseClock, hsync *polycounter.Polycounter,
	tv television.Television, hblank, hmoveLatch *bool) (*Video, error) {

	vd := &Video{
		collisions: newCollisions(mem),
		Playfield:  newPlayfield(pclk, hsync),
	}

	var err error

	// sprite objects
	vd.Player0, err = newPlayerSprite("Player 0", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Player1, err = newPlayerSprite("Player 1", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Missile0, err = newMissileSprite("Missile 0", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Missile1, err = newMissileSprite("Missile 1", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Ball, err = newBallSprite("Ball", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}

	// connect player 0 and player 1 to each other
	vd.Player0.otherPlayer = vd.Player1
	vd.Player1.otherPlayer = vd.Player0

	// connect ball to player 1 only - ball sprite's delayed enable set when
	// gfx register of player 1 is written
	vd.Player1.ball = vd.Ball

	// connect missile sprite to its parent player sprite
	vd.Missile0.parentPlayer = vd.Player0
	vd.Missile1.parentPlayer = vd.Player1

	return vd, nil
}

// RSYNC adjusts the debugging information of the sprites when an RSYNC is
// triggered
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
	vd.Player0.tick(visible, hmove, hmoveCt)
	vd.Player1.tick(visible, hmove, hmoveCt)
	vd.Missile0.tick(visible, hmove, hmoveCt)
	vd.Missile1.tick(visible, hmove, hmoveCt)
	vd.Ball.tick(visible, hmove, hmoveCt)
}

// PrepareSpritesForHMOVE should be called whenever HMOVE is triggered
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
func (vd *Video) Pixel() (uint8, colors.AltColor) {
	bgc := vd.Playfield.BackgroundColor
	pfu, pfc := vd.Playfield.pixel()
	p0u, p0c := vd.Player0.pixel()
	p1u, p1c := vd.Player1.pixel()
	m0u, m0c := vd.Missile0.pixel()
	m1u, m1c := vd.Missile1.pixel()
	blu, blc := vd.Ball.pixel()

	if m0u && p1u {
		vd.collisions.cxm0p |= 0x80
		vd.collisions.setMemory(addresses.CXM0P)
	}
	if m0u && p0u {
		vd.collisions.cxm0p |= 0x40
		vd.collisions.setMemory(addresses.CXM0P)
	}

	if m1u && p0u {
		vd.collisions.cxm1p |= 0x80
		vd.collisions.setMemory(addresses.CXM1P)
	}
	if m1u && p1u {
		vd.collisions.cxm1p |= 0x40
		vd.collisions.setMemory(addresses.CXM1P)
	}

	if p0u && pfu {
		vd.collisions.cxp0fb |= 0x80
		vd.collisions.setMemory(addresses.CXP0FB)
	}
	if p0u && blu {
		vd.collisions.cxp0fb |= 0x40
		vd.collisions.setMemory(addresses.CXP0FB)
	}

	if p1u && pfu {
		vd.collisions.cxp1fb |= 0x80
		vd.collisions.setMemory(addresses.CXP1FB)
	}
	if p1u && blu {
		vd.collisions.cxp1fb |= 0x40
		vd.collisions.setMemory(addresses.CXP1FB)
	}

	if m0u && pfu {
		vd.collisions.cxm0fb |= 0x80
		vd.collisions.setMemory(addresses.CXM0FB)
	}
	if m0u && blu {
		vd.collisions.cxm0fb |= 0x40
		vd.collisions.setMemory(addresses.CXM0FB)
	}

	if m1u && pfu {
		vd.collisions.cxm1fb |= 0x80
		vd.collisions.setMemory(addresses.CXM1FB)
	}
	if m1u && blu {
		vd.collisions.cxm1fb |= 0x40
		vd.collisions.setMemory(addresses.CXM1FB)
	}

	if blu && pfu {
		vd.collisions.cxblpf |= 0x80
		vd.collisions.setMemory(addresses.CXBLPF)
	}
	// no bit 6 for CXBLPF

	if p0u && p1u {
		vd.collisions.cxppmm |= 0x80
		vd.collisions.setMemory(addresses.CXPPMM)
	}
	if m0u && m1u {
		vd.collisions.cxppmm |= 0x40
		vd.collisions.setMemory(addresses.CXPPMM)
	}

	// apply priorities to get pixel color
	var col uint8
	var altCol colors.AltColor

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
		if pfu { // priority 1
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

			altCol = colors.AltColPlayfield
		} else if blu {
			col = blc
			altCol = colors.AltColBall
		} else if p0u { // priority 2
			col = p0c
			altCol = colors.AltColPlayer0
		} else if m0u {
			col = m0c
			altCol = colors.AltColMissile0
		} else if p1u { // priority 3
			col = p1c
			altCol = colors.AltColPlayer1
		} else if m1u {
			col = m1c
			altCol = colors.AltColMissile1
		} else {
			col = bgc
			altCol = colors.AltColBackground
		}
	} else {
		if p0u { // priority 1
			col = p0c
			altCol = colors.AltColPlayer0
		} else if m0u {
			col = m0c
			altCol = colors.AltColMissile0
		} else if p1u { // priority 2
			col = p1c
			altCol = colors.AltColPlayer1
		} else if m1u {
			col = m1c
			altCol = colors.AltColMissile1
		} else if vd.Playfield.Scoremode && (blu || pfu) {
			// priority 3 (scoremode without priority bit)
			if pfu {
				col = pfc
				switch vd.Playfield.Region {
				case RegionLeft:
					col = p0c
				case RegionRight:
					col = p1c
				}
				altCol = colors.AltColPlayfield
			} else if blu { // priority 3
				col = blc
				altCol = colors.AltColBall
			}
		} else {
			// priority 3 (no scoremode or priority bit)
			if blu { // priority 3
				col = blc
				altCol = colors.AltColBall
			} else if pfu {
				col = pfc
				altCol = colors.AltColPlayfield
			} else {
				col = bgc
				altCol = colors.AltColBackground
			}
		}

	}

	// priority 4
	return col, altCol
}

// UpdatePlayfield checks TIA memory for new playfield data. Note that CTRLPF
// is serviced in UpdateSpriteVariations().
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdatePlayfield(tiaDelay future.Scheduler, data bus.ChipData) bool {
	// homebrew Donkey Kong shows the need for a delay of at least two cycles
	// to write new playfield data
	switch data.Name {
	case "PF0":
		tiaDelay.ScheduleWithArg(2, vd.Playfield.setPF0, data.Value, "PF0")
	case "PF1":
		tiaDelay.ScheduleWithArg(2, vd.Playfield.setPF1, data.Value, "PF1")
	case "PF2":
		tiaDelay.ScheduleWithArg(2, vd.Playfield.setPF2, data.Value, "PF2")
	default:
		return true
	}

	return false
}

// UpdateSpriteHMOVE checks TIA memory for changes in sprite HMOVE settings.
//
// Returns true if ChipData has *not* been serviced.
func (vd *Video) UpdateSpriteHMOVE(tiaDelay future.Scheduler, data bus.ChipData) bool {
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
		tiaDelay.ScheduleWithArg(1, vd.Player0.setHmoveValue, data.Value&0xf0, "HMPx")
	case "HMP1":
		tiaDelay.ScheduleWithArg(1, vd.Player1.setHmoveValue, data.Value&0xf0, "HMPx")
	case "HMM0":
		tiaDelay.ScheduleWithArg(1, vd.Missile0.setHmoveValue, data.Value&0xf0, "HMMx")
	case "HMM1":
		tiaDelay.ScheduleWithArg(1, vd.Missile1.setHmoveValue, data.Value&0xf0, "HMMx")
	case "HMBL":
		tiaDelay.ScheduleWithArg(1, vd.Ball.setHmoveValue, data.Value&0xf0, "HMBL")
	case "HMCLR":
		tiaDelay.Schedule(1, vd.Player0.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(1, vd.Player1.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(1, vd.Missile0.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(1, vd.Missile1.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(1, vd.Ball.clearHmoveValue, "HMCLR")
	default:
		return true
	}

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
	case "GRP1":
		vd.Player1.setGfxData(data.Value)
	case "ENAM0":
		vd.Missile0.setEnable(data.Value&0x02 == 0x02)
	case "ENAM1":
		vd.Missile1.setEnable(data.Value&0x02 == 0x02)
	case "ENABL":
		vd.Ball.setEnable(data.Value&0x02 == 0x02)
	default:
		return true
	}

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
	case "VDELBL":
		vd.Ball.setVerticalDelay(data.Value&0x01 == 0x01)
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
		vd.collisions.clear()
	default:
		return true
	}

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
}
