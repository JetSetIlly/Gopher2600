package video

import (
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/bus"
	"gopher2600/hardware/tia/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/television"
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
	vd.Player0, err = newPlayerSprite("player0", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Player1, err = newPlayerSprite("player1", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Missile0, err = newMissileSprite("missile0", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Missile1, err = newMissileSprite("missile1", tv, hblank, hmoveLatch)
	if err != nil {
		return nil, err
	}
	vd.Ball, err = newBallSprite("ball", tv, hblank, hmoveLatch)
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
func (vd *Video) Pixel() (uint8, television.AltColorSignal) {
	bgc := vd.Playfield.backgroundColor
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

	// apply priorities to get pixel color. the interaction of the priority and
	// scoremode bits are a little more complex than at first glance:
	//
	//  o if the priority bit is set then priority ordering applies
	//  o if it is not set but scoremode is set and we're in the left half of
	//		the screen, then priority ordering also applies
	//	o if priority bit is not set but scoremode is set and we're in the
	//		right hand side of the screen then regular ordering applies, except
	//		that playfield has priority over the ball
	//	o phew
	//
	//	that scoremode reorders priory regardless of the priority bit is not at
	//	all obvious but observation proves it to be true. see test.bin ROM
	//
	//	also the comment by "supercat" in the discussion "Playfield Score Mode
	//	- effect on ball" on AtariAge proved useful here.
	//
	//	!!TODO: I'm still not 100% sure this is correct. check playfield priorties
	priority := vd.Playfield.priority || (vd.Playfield.scoremode && vd.Playfield.region == regionLeft)

	var col uint8
	var altCol television.AltColorSignal

	if priority {
		if pfu { // priority 1
			col = pfc

			// we don't want score mode coloring if priority is on. we can see
			// the effect of this on the top line of "Donkey Kong" on the intro
			// screen of Dietrich's Donkey Kong.
			if vd.Playfield.scoremode && !vd.Playfield.priority {
				switch vd.Playfield.region {
				case regionLeft:
					col = p0c
				case regionRight:
					col = p1c
				}
			}
			altCol = television.AltColPlayfield
		} else if blu {
			col = blc
			altCol = television.AltColBall
		} else if p0u { // priority 2
			col = p0c
			altCol = television.AltColPlayer0
		} else if m0u {
			col = m0c
			altCol = television.AltColMissile0
		} else if p1u { // priority 3
			col = p1c
			altCol = television.AltColPlayer1
		} else if m1u {
			col = m1c
			altCol = television.AltColMissile1
		} else {
			col = bgc
			altCol = television.AltColBackground
		}
	} else {
		if p0u { // priority 1
			col = p0c
			altCol = television.AltColPlayer0
		} else if m0u {
			col = m0c
			altCol = television.AltColMissile0
		} else if p1u { // priority 2
			col = p1c
			altCol = television.AltColPlayer1
		} else if m1u {
			col = m1c
			altCol = television.AltColMissile1
		} else if vd.Playfield.scoremode && (blu || pfu) {
			// priority 3 (scoremode without priority bit)
			if pfu {
				col = pfc
				switch vd.Playfield.region {
				case regionLeft:
					col = p0c
				case regionRight:
					col = p1c
				}
				altCol = television.AltColPlayfield
			} else if blu { // priority 3
				col = blc
				altCol = television.AltColBall
			}
		} else {
			// priority 3 (no scoremode or priority bit)
			if blu { // priority 3
				col = blc
				altCol = television.AltColBall
			} else if pfu {
				col = pfc
				altCol = television.AltColPlayfield
			} else {
				col = bgc
				altCol = television.AltColBackground
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
		tiaDelay.ScheduleWithArg(2, vd.Playfield.setSegment0, data.Value, "PF0")
	case "PF1":
		tiaDelay.ScheduleWithArg(2, vd.Playfield.setSegment1, data.Value, "PF1")
	case "PF2":
		tiaDelay.ScheduleWithArg(2, vd.Playfield.setSegment2, data.Value, "PF2")
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
	// reveals that a delay is required. the reasoning is as below:
	//
	// delay of at least zero (1 additiona cycle) is required. we can see this
	// in the Midnight Magic ROM where the left gutter separator requires it
	//
	// a delay too high (3 or higher) causes the barber pole test ROM to fail
	//
	// not sure what the actual value should be except that it should be
	// somewhere between 0 and 3 (inclusive)
	case "HMP0":
		tiaDelay.ScheduleWithArg(2, vd.Player0.setHmoveValue, data.Value&0xf0, "HMPx")
	case "HMP1":
		tiaDelay.ScheduleWithArg(2, vd.Player1.setHmoveValue, data.Value&0xf0, "HMPx")
	case "HMM0":
		tiaDelay.ScheduleWithArg(2, vd.Missile0.setHmoveValue, data.Value&0xf0, "HMMx")
	case "HMM1":
		tiaDelay.ScheduleWithArg(2, vd.Missile1.setHmoveValue, data.Value&0xf0, "HMMx")
	case "HMBL":
		tiaDelay.ScheduleWithArg(2, vd.Ball.setHmoveValue, data.Value&0xf0, "HMBL")
	case "HMCLR":
		tiaDelay.Schedule(2, vd.Player0.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(2, vd.Player1.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(2, vd.Missile0.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(2, vd.Missile1.clearHmoveValue, "HMCLR")
		tiaDelay.Schedule(2, vd.Ball.clearHmoveValue, "HMCLR")
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
		vd.Ball.setSize((data.Value & 0x30) >> 4)
		vd.Playfield.setControlBits(data.Value)
	case "VDELBL":
		vd.Ball.setVerticalDelay(data.Value&0x01 == 0x01)
	case "VDELP0":
		vd.Player0.setVerticalDelay(data.Value&0x01 == 0x01)
	case "VDELP1":
		vd.Player1.setVerticalDelay(data.Value&0x01 == 0x01)
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
		vd.Missile0.setNUSIZ(data.Value)
	case "NUSIZ1":
		vd.Player1.setNUSIZ(data.Value)
		vd.Missile1.setNUSIZ(data.Value)
	case "CXCLR":
		vd.collisions.clear()
	default:
		return true
	}

	return false
}
