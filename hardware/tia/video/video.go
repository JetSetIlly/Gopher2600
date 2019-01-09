package video

import (
	"gopher2600/hardware/tia/polycounter"
	"gopher2600/hardware/tia/video/future"
)

// Video contains all the components of the video sub-system of the VCS TIA chip
type Video struct {
	colorClock *polycounter.Polycounter

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

	// there's a slight delay when changing the state of video objects. we're
	// using two future instances to emulate what happens in the 2600. the
	// first is OnFutureColorClock, which *ticks* every video cycle. we use this for
	// writing playfield bits, player bits and enable flags for missiles and
	// the ball.
	//
	// the second future instance is FutureMotionClock. this is for those
	// writes that only occur during the "motion clock", resetting sprite
	// positions
	OnFutureColorClock  future.Group
	OnFutureMotionClock future.Group
}

// NewVideo is the preferred method of initialisation for the Video structure
func NewVideo(colorClock *polycounter.Polycounter) *Video {
	vd := new(Video)
	vd.colorClock = colorClock

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
	vd.Player0.gfxDataOther = &vd.Player1.gfxData
	vd.Player1.gfxDataOther = &vd.Player0.gfxData

	// connect missile sprite to its parent player sprite
	vd.Missile0.parentPlayer = vd.Player0
	vd.Missile1.parentPlayer = vd.Player1

	return vd
}

// TickFutures is called *every* video clock
func (vd *Video) TickFutures(sprites bool) {
	// resolve delayed write operations
	vd.OnFutureColorClock.Tick()

	if sprites {
		vd.OnFutureMotionClock.Tick()
	}
}

// TickPlayfield is called *every* video clock
func (vd *Video) TickPlayfield() {
	// tick playfield forward
	vd.Playfield.tick()
}

// TickSprites moves all video elements forward one video cycle and is only
// called when motion clock is active
func (vd *Video) TickSprites() {
	vd.Player0.tick()
	vd.Player1.tick()
	vd.Missile0.tick()
	vd.Missile1.tick()
	vd.Ball.tick()
}

// NewScanline is called at beginning of every scanline
func (vd *Video) NewScanline() {
	vd.Player0.newScanline()
	vd.Player1.newScanline()
	vd.Missile0.newScanline()
	vd.Missile1.newScanline()
	vd.Ball.newScanline()
}

// PrepareSpritesForHMOVE should be called whenever HMOVE is triggered
func (vd *Video) PrepareSpritesForHMOVE() {
	vd.Player0.horizMovementLatch = true
	vd.Player1.horizMovementLatch = true
	vd.Missile0.horizMovementLatch = true
	vd.Missile1.horizMovementLatch = true
	vd.Ball.horizMovementLatch = true
}

// TickSpritesForHMOVE is only called when HMOVE is active
func (vd *Video) TickSpritesForHMOVE(count int) {
	vd.Player0.tickSpritesForHMOVE(count)
	vd.Player1.tickSpritesForHMOVE(count)
	vd.Missile0.tickSpritesForHMOVE(count)
	vd.Missile1.tickSpritesForHMOVE(count)
	vd.Ball.tickSpritesForHMOVE(count)
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
		if p0u {
			return p0c
		}
		if m0u {
			return m0c
		}

		// priority 3
		if p1u {
			return p1c
		}
		if m1u {
			return m1c
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
		if blu {
			return blc
		}
		if pfu {
			return pfc
		}
	}

	// priority 4
	return vd.Playfield.backgroundColor
}

func createTriggerList(playerSize uint8) []int {
	var triggerList []int
	switch playerSize {
	case 0x0, 0x05, 0x07:
		// empty trigger list - single sprite of varying widths
	case 0x01:
		triggerList = []int{4}
	case 0x02:
		triggerList = []int{8}
	case 0x03:
		triggerList = []int{4, 8}
	case 0x04:
		triggerList = []int{16}
	case 0x06:
		triggerList = []int{8, 16}
	}
	return triggerList
}

// ReadVideoMemory checks the TIA memory for changes to registers that are
// interesting to the video sub-system. all changes happen immediately except
// for those where a "schedule" function is called.
func (vd *Video) ReadVideoMemory(register string, value uint8) bool {
	switch register {
	case "NUSIZ0":
		// TODO: write delay?
		vd.Missile0.size = (value & 0x30) >> 4
		vd.Player0.size = value & 0x07
		vd.Player0.triggerList = createTriggerList(vd.Player0.size)
		vd.Missile0.triggerList = vd.Player0.triggerList
	case "NUSIZ1":
		// TODO: write delay?
		vd.Missile1.size = (value & 0x30) >> 4
		vd.Player1.size = value & 0x07
		vd.Player1.triggerList = createTriggerList(vd.Player1.size)
		vd.Missile1.triggerList = vd.Player1.triggerList
	case "COLUP0":
		// TODO: write delay?
		vd.Player0.color = value & 0xfe
		vd.Missile0.color = value & 0xfe
	case "COLUP1":
		// TODO: write delay?
		vd.Player1.color = value & 0xfe
		vd.Missile1.color = value & 0xfe
	case "COLUPF":
		// TODO: write delay?
		vd.Playfield.foregroundColor = value & 0xfe
		vd.Ball.color = value & 0xfe
	case "COLUBK":
		// this delay works and fixes a graphical issue with the "Keystone
		// Kapers" rom. I'm not entirely sure this is the correct fix however.
		// and I'm definitely now sure about the delay time.
		vd.OnFutureColorClock.Schedule(delayWritePlayfield, func() {
			vd.Playfield.backgroundColor = value & 0xfe
		}, "setting COLUBK")
	case "CTRLPF":
		// TODO: write delay?
		vd.Ball.size = (value & 0x30) >> 4
		vd.Playfield.reflected = value&0x01 == 0x01
		vd.Playfield.scoremode = value&0x02 == 0x02
		vd.Playfield.priority = value&0x04 == 0x04
	case "REFP0":
		// TODO: write delay?
		vd.Player0.reflected = value&0x08 == 0x08
	case "REFP1":
		// TODO: write delay?
		vd.Player1.reflected = value&0x08 == 0x08
	case "PF0":
		vd.Playfield.scheduleWrite(0, value, &vd.OnFutureColorClock)
	case "PF1":
		vd.Playfield.scheduleWrite(1, value, &vd.OnFutureColorClock)
	case "PF2":
		vd.Playfield.scheduleWrite(2, value, &vd.OnFutureColorClock)
	case "RESP0":
		vd.Player0.scheduleReset(&vd.OnFutureMotionClock)
	case "RESP1":
		vd.Player1.scheduleReset(&vd.OnFutureMotionClock)
	case "RESM0":
		vd.Missile0.scheduleReset(&vd.OnFutureMotionClock)
	case "RESM1":
		vd.Missile1.scheduleReset(&vd.OnFutureMotionClock)
	case "RESBL":
		vd.Ball.scheduleReset(&vd.OnFutureMotionClock)
	case "GRP0":
		vd.Player0.scheduleWrite(value, &vd.OnFutureColorClock)
	case "GRP1":
		vd.Player1.scheduleWrite(value, &vd.OnFutureColorClock)
	case "ENAM0":
		vd.Missile0.scheduleEnable(value&0x02 == 0x02, &vd.OnFutureColorClock)
	case "ENAM1":
		vd.Missile1.scheduleEnable(value&0x02 == 0x02, &vd.OnFutureColorClock)
	case "ENABL":
		vd.Ball.scheduleEnable(value&0x02 == 0x02, &vd.OnFutureColorClock)
	case "VDELP0":
		vd.Player0.scheduleVerticalDelay(value&0x01 == 0x01, &vd.OnFutureMotionClock)
	case "VDELP1":
		vd.Player1.scheduleVerticalDelay(value&0x01 == 0x01, &vd.OnFutureMotionClock)
	case "VDELBL":
		vd.Ball.scheduleVerticalDelay(value&0x01 == 0x01, &vd.OnFutureMotionClock)
	case "RESMP0":
		vd.Missile0.scheduleResetToPlayer(value&0x02 == 0x002, &vd.OnFutureColorClock)
	case "RESMP1":
		vd.Missile1.scheduleResetToPlayer(value&0x02 == 0x002, &vd.OnFutureColorClock)
	case "CXCLR":
		vd.Collisions.clear()

		// horizontal movement values range from -8 to +7
		// for convenience we convert this to the range 0 to 15
		//
		// TODO: write delay?
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

	case "HMCLR":
		// note that HMCLR does not reset the horizontal movement latches in
		// the sprite object (TIA_HW_Notes)
		vd.Player0.horizMovement = 0x08
		vd.Player1.horizMovement = 0x08
		vd.Missile0.horizMovement = 0x08
		vd.Missile1.horizMovement = 0x08
		vd.Ball.horizMovement = 0x08

	}

	return false
}
