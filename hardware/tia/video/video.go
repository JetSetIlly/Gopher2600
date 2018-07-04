package video

import (
	"gopher2600/hardware/tia/colorclock"
)

// Video contains all the components of the video sub-system of the VCS TIA chip
type Video struct {
	colorClock *colorclock.ColorClock
	hblank     *bool

	// playfield
	Playfield *playfield

	// sprite objects
	Player0  *playerSprite
	Player1  *playerSprite
	Missile0 *missileSprite
	Missile1 *missileSprite
	Ball     *ballSprite

	// collision matrix
	Collisions collisions
}

// New is the preferred method of initialisation for the Video structure
func New(colorClock *colorclock.ColorClock, hblank *bool) *Video {
	vd := new(Video)
	if vd == nil {
		return nil
	}

	vd.colorClock = colorClock
	vd.hblank = hblank

	// playfield
	vd.Playfield = newPlayfield(vd.colorClock)

	// sprite objects
	vd.Player0 = newPlayerSprite("player0", vd.colorClock)
	if vd.Player0 == nil {
		return nil
	}
	vd.Player1 = newPlayerSprite("player1", vd.colorClock)
	if vd.Player1 == nil {
		return nil
	}
	vd.Missile0 = newMissileSprite("missile0", vd.colorClock)
	if vd.Missile0 == nil {
		return nil
	}
	vd.Missile1 = newMissileSprite("missile1", vd.colorClock)
	if vd.Missile1 == nil {
		return nil
	}
	vd.Ball = newBallSprite("ball", vd.colorClock)
	if vd.Ball == nil {
		return nil
	}

	// connect player 0 and player 1 to each other (via the vertical delay bit)
	vd.Player0.gfxDataDelay = &vd.Player1.gfxDataPrev
	vd.Player1.gfxDataDelay = &vd.Player0.gfxDataPrev

	return vd
}

// Tick moves all video elements forward one video cycle
func (vd *Video) Tick() {
	vd.Playfield.tick()
	if !*vd.hblank {
		vd.Player0.tick()
		vd.Player1.tick()
		vd.Missile0.tick()
		vd.Missile1.tick()
		vd.Ball.tick()
	}
}

// TickSpritesForHMOVE moves sprite elements if horiz movement value is in range
func (vd *Video) TickSpritesForHMOVE(count int) {
	if count == 0 {
		return
	}

	if vd.Player0.horizMovement >= uint8(count) {
		vd.Player0.tick()
	}
	if vd.Player1.horizMovement >= uint8(count) {
		vd.Player1.tick()
	}
	if vd.Missile0.horizMovement >= uint8(count) {
		vd.Missile0.tick()
	}
	if vd.Missile1.horizMovement >= uint8(count) {
		vd.Missile1.tick()
	}
	if vd.Ball.horizMovement >= uint8(count) {
		vd.Ball.tick()
	}
}

// Pixel returns the color of the pixel at the current time. it will default
// to returning the background color if no sprite or playfield pixel is
// present. it also sets the collision registers
// - it need not be called therefore during VBLANK or HBLANK
func (vd *Video) Pixel() uint8 {
	pfu, pfc := vd.Playfield.pixel()
	blu, blc := vd.Ball.pixel()
	p0u, p0c := vd.Player0.pixel()
	p1u, p1c := vd.Player1.pixel()
	m0u, m0c := vd.Missile0.pixel()
	m1u, m1c := vd.Missile1.pixel()

	// collisions
	if m0u && p1u {
		vd.Collisions.cxm0p |= 0x80
	}
	if m0u && p0u {
		vd.Collisions.cxm0p |= 0x40
	}

	if m1u && p0u {
		vd.Collisions.cxm1p |= 0x80
	}
	if m1u && p1u {
		vd.Collisions.cxm1p |= 0x40
	}

	if p0u && pfu {
		vd.Collisions.cxp0fb |= 0x80
	}
	if p0u && blu {
		vd.Collisions.cxp0fb |= 0x40
	}

	if p1u && pfu {
		vd.Collisions.cxp1fb |= 0x80
	}
	if p1u && blu {
		vd.Collisions.cxp1fb |= 0x40
	}

	if m0u && pfu {
		vd.Collisions.cxm0fb |= 0x80
	}
	if m0u && blu {
		vd.Collisions.cxm0fb |= 0x40
	}

	if m1u && pfu {
		vd.Collisions.cxm1fb |= 0x80
	}
	if m1u && blu {
		vd.Collisions.cxm1fb |= 0x40
	}

	if blu && pfu {
		vd.Collisions.cxblpf |= 0x80
	}
	// no bit 6 for CXBLPF

	if p0u && p1u {
		vd.Collisions.cxppmm |= 0x80
	}
	if m0u && m1u {
		vd.Collisions.cxppmm |= 0x40
	}

	// apply priorities to get pixel color
	if vd.Playfield.priority {
		// priority 1
		if pfu {
			return pfc
		}
		if blu {
			return blc
		}

		// priority 2
		if p1u {
			return p1c
		}
		if m1u {
			return m1c
		}

		// priority 3
		if p0u {
			return p0c
		}
		if m0u {
			return m0c
		}
	} else {
		// priority 1
		if p0u {
			return p0c
		}
		if m0u {
			return m0c
		}

		// priority 2
		if p1u {
			return p1c
		}
		if m1u {
			return m1c
		}

		// priority 3
		if pfu {
			return pfc
		}
		if blu {
			return blc
		}
	}

	// priority 4
	return vd.Playfield.backgroundColor
}

func createTriggerList(playerSize uint8) []int {
	var triggerList []int
	switch playerSize {
	case 0x0, 0x05, 0x07:
		// empty trigger list
	case 0x01:
		triggerList = []int{4} // 111100
	case 0x02:
		triggerList = []int{8} // 110111
	case 0x03:
		triggerList = []int{4, 8} // 111100, 110111
	case 0x04:
		triggerList = []int{4} // 110111
	case 0x06:
		triggerList = []int{8, 16} // 110111, 011100
	}
	return triggerList
}

// ReadVideoMemory checks the TIA memory for changes to registers that are
// interesting to the video sub-system. all changes happen immediately except
// for those where a "schedule" function is called.
func (vd *Video) ReadVideoMemory(register string, value uint8) bool {
	switch register {
	case "NUSIZ0":
		vd.Missile0.size = (value & 0x30) >> 4
		vd.Player0.size = value & 0x07
		vd.Player0.triggerList = createTriggerList(vd.Player0.size)
		vd.Missile0.triggerList = vd.Player0.triggerList
	case "NUSIZ1":
		vd.Missile1.size = (value & 0x30) >> 4
		vd.Player1.size = value & 0x07
		vd.Player1.triggerList = createTriggerList(vd.Player1.size)
		vd.Missile1.triggerList = vd.Player1.triggerList
	case "COLUP0":
		vd.Player0.color = value & 0xfe
		vd.Missile0.color = value & 0xfe
	case "COLUP1":
		vd.Player1.color = value & 0xfe
		vd.Missile1.color = value & 0xfe
	case "COLUPF":
		vd.Playfield.foregroundColor = value & 0xfe
		vd.Ball.color = value & 0xfe
	case "COLUBK":
		vd.Playfield.backgroundColor = value & 0xfe
	case "CTRLPF":
		vd.Ball.size = (value & 0x30) >> 4
		vd.Playfield.reflected = value&0x01 == 0x01
		vd.Playfield.scoremode = value&0x02 == 0x02
		vd.Playfield.priority = value&0x04 == 0x04
	case "REFP0":
		vd.Player0.reflected = value&0x08 == 0x08
	case "REFP1":
		vd.Player1.reflected = value&0x08 == 0x08
	case "PF0":
		vd.Playfield.scheduleWrite(0, value)
	case "PF1":
		vd.Playfield.scheduleWrite(1, value)
	case "PF2":
		vd.Playfield.scheduleWrite(2, value)
	case "RESP0":
		vd.Player0.scheduleReset(vd.hblank)
	case "RESP1":
		vd.Player1.scheduleReset(vd.hblank)
	case "RESM0":
		vd.Missile0.scheduleReset(vd.hblank)
	case "RESM1":
		vd.Missile1.scheduleReset(vd.hblank)
	case "RESBL":
		vd.Ball.scheduleReset(vd.hblank)
	case "GRP0":
		vd.Player0.gfxDataPrev = vd.Player1.gfxData
		vd.Player0.gfxData = value
	case "GRP1":
		vd.Player1.gfxDataPrev = vd.Player0.gfxData
		vd.Player1.gfxData = value
	case "ENAM0":
		vd.Missile0.scheduleEnable(value)
	case "ENAM1":
		vd.Missile1.scheduleEnable(value)
	case "ENABL":
		vd.Ball.scheduleEnable(value)
	case "HMP0":
		vd.Player0.horizMovement = (value ^ 0x80) >> 4
	case "HMP1":
		vd.Player1.horizMovement = (value ^ 0x80) >> 4
	case "HMM0":
		vd.Missile0.horizMovement = (value ^ 0x80) >> 4
	case "HMM1":
		vd.Missile1.horizMovement = (value ^ 0x80) >> 4
	case "HMBL":
		vd.Ball.horizMovement = (value ^ 0x80) >> 4
	case "VDELP0":
		vd.Player0.verticalDelay = value&0x01 == 0x01
	case "VDELP1":
		vd.Player1.verticalDelay = value&0x01 == 0x01
	case "VDELBL":
		vd.Ball.verticalDelay = value&0x01 == 0x01
	case "RESMP0":
	case "RESMP1":
	case "HMCLR":
		vd.Player0.horizMovement = 0x08
		vd.Player1.horizMovement = 0x08
		vd.Missile0.horizMovement = 0x08
		vd.Missile1.horizMovement = 0x08
		vd.Ball.horizMovement = 0x08
	case "CXCLR":
		vd.Collisions.clear()
	}

	return false
}
