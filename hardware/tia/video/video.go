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

	tiaDelay future.Scheduler
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
// the playfield and sprite objects have access to both tiaClk and hsync.
// in the case of the playfield, they are used to decide which part of the
// playfield is to be drawn. in the case of the the sprite objects, they
// are used only for information purposes - namely the reset and current
// pisel locatoin of the sprites in relation to the hsync counter (or
// screen)
//
// the tiaDelay scheduler is used to queue up sprite reset events and a few
// other events (!!TODO: figuring out what is delayed and how is not yet
// completed)
func NewVideo(tiaClk *phaseclock.PhaseClock,
	hsync *polycounter.Polycounter,
	tiaDelay future.Scheduler,
	mem memory.ChipBus,
	tv television.Television) *Video {

	vd := &Video{tiaDelay: tiaDelay}

	// collision matrix
	vd.collisions = newCollision(mem)

	// playfield
	vd.Playfield = newPlayfield(tiaClk, hsync, tiaDelay)

	// sprite objects
	vd.Player0 = newPlayerSprite("player0", tv)
	if vd.Player0 == nil {
		return nil
	}
	vd.Player1 = newPlayerSprite("player1", tv)
	if vd.Player1 == nil {
		return nil
	}
	vd.Missile0 = newMissileSprite("missile0", tiaClk)
	if vd.Missile0 == nil {
		return nil
	}
	vd.Missile1 = newMissileSprite("missile1", tiaClk)
	if vd.Missile1 == nil {
		return nil
	}
	vd.Ball = newBallSprite("ball", tiaClk)
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

// TickSprites moves all video elements forward one video cycle and is only
// called when motion clock is active
func (vd *Video) TickSprites(visibleScreen bool, hmoveCt uint8) {
	vd.Player0.tick(visibleScreen, hmoveCt)
	vd.Player1.tick(visibleScreen, hmoveCt)
	vd.Missile0.tick()
	vd.Missile1.tick()
	vd.Ball.tick()
}

// PrepareSpritesForHMOVE should be called whenever HMOVE is triggered
func (vd *Video) PrepareSpritesForHMOVE() {
	vd.Player0.prepareForHMOVE()
	vd.Player1.prepareForHMOVE()
	vd.Missile0.prepareForHMOVE()
	vd.Missile1.prepareForHMOVE()
	vd.Ball.prepareForHMOVE()
}

// Resolve returns the color of the pixel at the current clock and also sets the
// collision registers. it will default to returning the background color if no
// sprite or playfield pixel is present.
func (vd *Video) Resolve() (uint8, uint8) {
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
			if vd.Playfield.scoremode == true {
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

// ReadMemory checks the TIA memory for changes to registers that are
// interesting to the video sub-system. all changes happen immediately except
// for those where a "schedule" function is called.
//
// returns true if memory has been serviced
func (vd *Video) ReadMemory(register string, value uint8) bool {
	switch register {
	default:
		return false

	// colour
	case "COLUP0":
		vd.Player0.setColor(value & 0xfe)
		vd.Missile0.scheduleSetColor(value&0xfe, vd.tiaDelay)
	case "COLUP1":
		vd.Player1.setColor(value & 0xfe)
		vd.Missile1.scheduleSetColor(value&0xfe, vd.tiaDelay)

	// playfield / color
	case "COLUBK":
		// vd.onFutureColorClock.Schedule(delay.WritePlayfieldColor, func() {
		vd.Playfield.backgroundColor = value & 0xfe
		// }, "setting COLUBK")
	case "COLUPF":
		// vd.onFutureColorClock.Schedule(delay.WritePlayfieldColor, func() {
		vd.Playfield.foregroundColor = value & 0xfe
		vd.Ball.color = value & 0xfe
		// }, "setting COLUPF")

	// playfield
	case "CTRLPF":
		// !!TODO: write delay?
		vd.Ball.size = (value & 0x30) >> 4
		vd.Playfield.reflected = value&0x01 == 0x01
		vd.Playfield.scoremode = value&0x02 == 0x02
		vd.Playfield.priority = value&0x04 == 0x04
	case "PF0":
		vd.Playfield.scheduleWrite(0, value, vd.tiaDelay)
	case "PF1":
		vd.Playfield.scheduleWrite(1, value, vd.tiaDelay)
	case "PF2":
		vd.Playfield.scheduleWrite(2, value, vd.tiaDelay)

	// ball sprite
	case "ENABL":
		vd.Ball.scheduleEnable(value&0x02 == 0x02, vd.tiaDelay)
	case "RESBL":
		vd.Ball.scheduleReset(vd.tiaDelay)
	case "VDELBL":
		vd.Ball.scheduleVerticalDelay(value&0x01 == 0x01, vd.tiaDelay)

	// player sprites
	case "GRP0":
		vd.Player0.setGfxData(value)
	case "GRP1":
		vd.Player1.setGfxData(value)
	case "RESP0":
		vd.Player0.resetPosition()
	case "RESP1":
		vd.Player1.resetPosition()
	case "VDELP0":
		vd.Player0.setVerticalDelay(value&0x01 == 0x01)
	case "VDELP1":
		vd.Player1.setVerticalDelay(value&0x01 == 0x01)
	case "REFP0":
		vd.Player0.setReflection(value&0x08 == 0x08)
	case "REFP1":
		vd.Player1.setReflection(value&0x08 == 0x08)

	// missile sprites
	case "ENAM0":
		vd.Missile0.scheduleEnable(value&0x02 == 0x02, vd.tiaDelay)
	case "ENAM1":
		vd.Missile1.scheduleEnable(value&0x02 == 0x02, vd.tiaDelay)
	case "RESM0":
		vd.Missile0.scheduleReset(vd.tiaDelay)
	case "RESM1":
		vd.Missile1.scheduleReset(vd.tiaDelay)
	case "RESMP0":
		vd.Missile0.scheduleResetToPlayer(value&0x02 == 0x002, vd.tiaDelay)
	case "RESMP1":
		vd.Missile1.scheduleResetToPlayer(value&0x02 == 0x002, vd.tiaDelay)

	// player & missile sprites
	case "NUSIZ0":
		vd.Player0.setNUSIZ(value)
		vd.Missile0.scheduleSetNUSIZ(value, vd.tiaDelay)
	case "NUSIZ1":
		vd.Player1.setNUSIZ(value)
		vd.Missile1.scheduleSetNUSIZ(value, vd.tiaDelay)

	// clear collisions
	case "CXCLR":
		vd.collisions.clear()

	// horizontal movement
	case "HMCLR":
		vd.Player0.hmove = 0x08
		vd.Player1.hmove = 0x08
		vd.Missile0.horizMovement = 0x08
		vd.Missile1.horizMovement = 0x08
		vd.Ball.horizMovement = 0x08

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
		vd.Player0.setHmoveValue(value & 0xf0)
	case "HMP1":
		vd.Player1.setHmoveValue(value & 0xf0)
	case "HMM0":
		// !!TODO: write delay?
		vd.Missile0.horizMovement = int(value^0x80) >> 4
	case "HMM1":
		// !!TODO: write delay?
		vd.Missile1.horizMovement = int(value^0x80) >> 4
	case "HMBL":
		// !!TODO: write delay?
		vd.Ball.horizMovement = int(value^0x80) >> 4
	}

	return true
}
