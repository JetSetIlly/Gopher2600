package video

import (
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/tia/delay/future"
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

// colors to use for debugging - these are the same colours used by the Stella
// emulator
const (
	debugColBackground = uint8(0x02) // light gray
	debugColBall       = uint8(0xb4) // cyan
	debugColPlayfield  = uint8(0x62) // purple
	debugColPlayer0    = uint8(0x32) // red
	debugColPlayer1    = uint8(0x12) // gold
	debugColMissile0   = uint8(0xf2) // orange
	debugColMissile1   = uint8(0xd2) // green
)

// NewVideo is the preferred method of initialisation for the Video structure
//
// the playfield and sprite objects have access to both pclk and hsync.
// in the case of the playfield, they are used to decide which part of the
// playfield is to be drawn. in the case of the the sprite objects, they
// are used only for information purposes - namely the reset and current
// pisel locatoin of the sprites in relation to the hsync counter (or
// screen)
func NewVideo(pclk *phaseclock.PhaseClock, hsync *polycounter.Polycounter,
	mem memory.ChipBus, tv television.Television,
	hblank, hmoveLatch *bool) *Video {

	vd := &Video{}

	// collision matrix
	vd.collisions = newCollision(mem)

	// playfield
	vd.Playfield = newPlayfield(pclk, hsync)

	// sprite objects
	vd.Player0 = newPlayerSprite("player0", tv, hblank, hmoveLatch)
	if vd.Player0 == nil {
		return nil
	}
	vd.Player1 = newPlayerSprite("player1", tv, hblank, hmoveLatch)
	if vd.Player1 == nil {
		return nil
	}
	vd.Missile0 = newMissileSprite("missile0", tv, hblank, hmoveLatch)
	if vd.Missile0 == nil {
		return nil
	}
	vd.Missile1 = newMissileSprite("missile1", tv, hblank, hmoveLatch)
	if vd.Missile1 == nil {
		return nil
	}
	vd.Ball = newBallSprite("ball", tv, hblank, hmoveLatch)
	if vd.Ball == nil {
		return nil
	}

	// connect player 0 and player 1 to each other
	vd.Player0.otherPlayer = vd.Player1
	vd.Player1.otherPlayer = vd.Player0

	// connect missile sprite to its parent player sprite
	vd.Missile0.parentPlayer = vd.Player0
	vd.Missile1.parentPlayer = vd.Player1

	return vd
}

// Tick moves all video elements forward one video cycle and is only
// called when motion clock is active
func (vd *Video) Tick(motck bool, hmove bool, hmoveCt uint8) {
	vd.Player0.tick(motck, hmove, hmoveCt)
	vd.Player1.tick(motck, hmove, hmoveCt)
	vd.Missile0.tick(motck, hmove, hmoveCt)
	vd.Missile1.tick(motck, hmove, hmoveCt)
	vd.Ball.tick(motck, hmove, hmoveCt)
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
// collision registers. it will default to returning the background color if no
// sprite or playfield pixel is present.
func (vd *Video) Pixel() (uint8, uint8) {
	bgc := vd.Playfield.backgroundColor
	pfu, pfc := vd.Playfield.pixel()
	p0u, p0c := vd.Player0.pixel()
	p1u, p1c := vd.Player1.pixel()
	m0u, m0c := vd.Missile0.pixel()
	m1u, m1c := vd.Missile1.pixel()
	blu, blc := vd.Ball.pixel()

	// collisions
	if m0u && p1u {
		vd.collisions.cxm0p |= 0x80
		vd.collisions.SetMemory(addresses.CXM0P)
	}
	if m0u && p0u {
		vd.collisions.cxm0p |= 0x40
		vd.collisions.SetMemory(addresses.CXM0P)
	}

	if m1u && p0u {
		vd.collisions.cxm1p |= 0x80
		vd.collisions.SetMemory(addresses.CXM1P)
	}
	if m1u && p1u {
		vd.collisions.cxm1p |= 0x40
		vd.collisions.SetMemory(addresses.CXM1P)
	}

	if p0u && pfu {
		vd.collisions.cxp0fb |= 0x80
		vd.collisions.SetMemory(addresses.CXP0FB)
	}
	if p0u && blu {
		vd.collisions.cxp0fb |= 0x40
		vd.collisions.SetMemory(addresses.CXP0FB)
	}

	if p1u && pfu {
		vd.collisions.cxp1fb |= 0x80
		vd.collisions.SetMemory(addresses.CXP1FB)
	}
	if p1u && blu {
		vd.collisions.cxp1fb |= 0x40
		vd.collisions.SetMemory(addresses.CXP1FB)
	}

	if m0u && pfu {
		vd.collisions.cxm0fb |= 0x80
		vd.collisions.SetMemory(addresses.CXM0FB)
	}
	if m0u && blu {
		vd.collisions.cxm0fb |= 0x40
		vd.collisions.SetMemory(addresses.CXM0FB)
	}

	if m1u && pfu {
		vd.collisions.cxm1fb |= 0x80
		vd.collisions.SetMemory(addresses.CXM1FB)
	}
	if m1u && blu {
		vd.collisions.cxm1fb |= 0x40
		vd.collisions.SetMemory(addresses.CXM1FB)
	}

	if blu && pfu {
		vd.collisions.cxblpf |= 0x80
		vd.collisions.SetMemory(addresses.CXBLPF)
	}
	// no bit 6 for CXBLPF

	if p0u && p1u {
		vd.collisions.cxppmm |= 0x80
		vd.collisions.SetMemory(addresses.CXPPMM)
	}
	if m0u && m1u {
		vd.collisions.cxppmm |= 0x40
		vd.collisions.SetMemory(addresses.CXPPMM)
	}

	var col, dcol uint8

	// apply priorities to get pixel color
	if vd.Playfield.priority {
		if pfu { // priority 1
			col = pfc
			dcol = debugColPlayfield
		} else if blu {
			col = blc
			dcol = debugColBall
		} else if p0u { // priority 2
			col = p0c
			dcol = debugColPlayer0
		} else if m0u {
			col = m0c
			dcol = debugColMissile0
		} else if p1u { // priority 3
			col = p1c
			dcol = debugColPlayer1
		} else if m1u {
			col = m1c
			dcol = debugColMissile1
		} else {
			col = bgc
			dcol = debugColBackground
		}

	} else {
		if p0u { // priority 1
			col = p0c
			dcol = debugColPlayer0
		} else if m0u {
			col = m0c
			dcol = debugColMissile0
		} else if p1u { // priority 2
			col = p1c
			dcol = debugColPlayer1
		} else if m1u {
			col = m1c
			dcol = debugColMissile1
		} else if blu { // priority 3
			col = blc
			dcol = debugColBall
		} else if pfu {
			if vd.Playfield.scoremode {
				if vd.Playfield.screenRegion == 2 {
					col = p1c
				} else {
					col = p0c
				}
			} else {
				col = pfc
			}
			dcol = debugColPlayfield
		} else {
			col = bgc
			dcol = debugColBackground
		}
	}

	// priority 4
	return col, dcol
}

// AlterPlayfield checks the TIA memory for changes to playfield data
func (vd *Video) AlterPlayfield(tiaDelay future.Scheduler, data memory.ChipData) {
	switch data.Name {
	case "PF0":
		vd.Playfield.setData(tiaDelay, 0, data.Value)
	case "PF1":
		vd.Playfield.setData(tiaDelay, 1, data.Value)
	case "PF2":
		vd.Playfield.setData(tiaDelay, 2, data.Value)
	}
}

// AlterStateAfterPixel checks the TIA memory for attribute changes that must
// occur after pixel information has been gathered
func (vd *Video) AlterStateAfterPixel(data memory.ChipData) {
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
	}
}

// AlterState checks the TIA memory for changes to sprite attributes. this
// function is intended to be called *after* HSYNC ticking and *before* pixel
// resolution
func (vd *Video) AlterState(data memory.ChipData) {
	switch data.Name {

	// the reset registers *must* be serviced after HSYNC has been ticked.
	//
	// the barnstormer ROM, scanline 61 demonstrates perfectly how GRP0 is affected by
	// this. if we write the new data before the pixel output has been decided then
	// an unwanted artefact is displayed.
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

	// servicing of the following registers could feasibly and safely occur
	// after pixel resolution or even before HSYNC has been ticked. but in the
	// absence of any firm reason to the contrary we'll group them all here
	//
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

	// horizontal movement values range from -8 to +7 for convenience we
	// convert this to the range 0 to 15. From TIA_HW_Notes.txt:
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

	case "HMP0":
		vd.Player0.setHmoveValue(data.Value & 0xf0)
	case "HMP1":
		vd.Player1.setHmoveValue(data.Value & 0xf0)
	case "HMM0":
		vd.Missile0.setHmoveValue(data.Value & 0xf0)
	case "HMM1":
		vd.Missile1.setHmoveValue(data.Value & 0xf0)
	case "HMBL":
		vd.Ball.setHmoveValue(data.Value & 0xf0)

	case "HMCLR":
		vd.Player0.hmove = 0x08
		vd.Player1.hmove = 0x08
		vd.Missile0.hmove = 0x08
		vd.Missile1.hmove = 0x08
		vd.Ball.hmove = 0x08

	}
}
