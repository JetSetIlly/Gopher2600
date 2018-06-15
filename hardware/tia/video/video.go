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

	// horizontal movement
	hmp0 uint8
	hmp1 uint8
	hmm0 uint8
	hmm1 uint8
	hmbl uint8
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

	// horizontal movment
	vd.hmp0 = 0x08
	vd.hmp1 = 0x08
	vd.hmm0 = 0x08
	vd.hmm1 = 0x08
	vd.hmbl = 0x08

	return vd
}

// Tick moves all video elements on one video cycle
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

	if vd.hmp0 >= uint8(count) {
		vd.Player0.tick()
	}
	if vd.hmp1 >= uint8(count) {
		vd.Player1.tick()
	}
	if vd.hmm0 >= uint8(count) {
		vd.Missile0.tick()
	}
	if vd.hmm1 >= uint8(count) {
		vd.Missile1.tick()
	}
	if vd.hmbl >= uint8(count) {
		vd.Ball.tick()
	}
}

// GetPixel returns the color of the pixel at the current time. it will default
// to returning the background color if no sprite or playfield pixel is present
// - it need not be called therefore during VBLANK or HBLANK
func (vd Video) GetPixel() uint8 {
	if vd.Playfield.priority {
		// priority 1
		if use, c := vd.Playfield.pixel(); use {
			return c
		}
		if use, c := vd.Ball.pixel(); use {
			return c
		}

		// priority 2
		if use, c := vd.Player1.pixel(); use {
			return c
		}
		if use, c := vd.Missile1.pixel(); use {
			return c
		}

		// priority 3
		if use, c := vd.Player0.pixel(); use {
			return c
		}
		if use, c := vd.Missile0.pixel(); use {
			return c
		}
	} else {
		// priority 1
		if use, c := vd.Player0.pixel(); use {
			return c
		}
		if use, c := vd.Missile0.pixel(); use {
			return c
		}

		// priority 2
		if use, c := vd.Player1.pixel(); use {
			return c
		}
		if use, c := vd.Missile1.pixel(); use {
			return c
		}

		// priority 3
		if use, c := vd.Playfield.pixel(); use {
			return c
		}
		if use, c := vd.Ball.pixel(); use {
			return c
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
		vd.Player0.reflected = value&0x04 == 0x04
	case "REFP1":
		vd.Player1.reflected = value&0x04 == 0x04
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
		vd.Player0.gfxDataPrev = vd.Player0.gfxData
		vd.Player0.gfxData = value
	case "GRP1":
		vd.Player1.gfxDataPrev = vd.Player1.gfxData
		vd.Player1.gfxData = value
	case "ENAM0":
		vd.Missile0.scheduleEnable(value)
	case "ENAM1":
		vd.Missile1.scheduleEnable(value)
	case "ENABL":
		vd.Ball.scheduleEnable(value)
	case "HMP0":
		vd.hmp0 = (value ^ 0x80) >> 4
	case "HMP1":
		vd.hmp1 = (value ^ 0x80) >> 4
	case "HMM0":
		vd.hmm0 = (value ^ 0x80) >> 4
	case "HMM1":
		vd.hmm1 = (value ^ 0x80) >> 4
	case "HMBL":
		vd.hmbl = (value ^ 0x80) >> 4
	case "VDELP0":
		vd.Player0.verticalDelay = value&0x01 == 0x01
	case "VDELP1":
		vd.Player1.verticalDelay = value&0x01 == 0x01
	case "VDELBL":
		vd.Ball.verticalDelay = value&0x01 == 0x01
	case "RESMP0":
	case "RESMP1":
	case "HMCLR":
		vd.hmp0 = 0x08
		vd.hmp1 = 0x08
		vd.hmm0 = 0x08
		vd.hmm1 = 0x08
		vd.hmbl = 0x08
	case "CXCLR":
	}

	return false
}
