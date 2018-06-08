package video

import (
	"gopher2600/hardware/tia/colorclock"
)

// Video contains all the components of the video sub-system of the VCS TIA chip
type Video struct {
	colorClock *colorclock.ColorClock
	hblank     *bool

	// sprite objects
	player0  *sprite
	player1  *sprite
	missile0 *sprite
	missile1 *sprite
	Ball     *sprite

	// colors
	colup0 uint8
	colup1 uint8
	colupf uint8
	colubk uint8

	// TODO: player sprite data

	// playfield
	Playfield *playfield

	// playfield control
	// -- including ball size
	ctrlpfReflection bool
	ctrlpfPriority   bool
	ctrlpfScoremode  bool
	ctrlpfBallSize   uint8

	// TODO: player/missile number & spacing
	// TODO: trigger lists
	// TODO: missile/ball size

	// player reflection
	refp0 bool
	refp1 bool

	// missile/ball enabling
	enam0      bool
	enam1      bool
	enabl      bool
	enam0Delay *delayCounter
	enam1Delay *delayCounter
	enablDelay *delayCounter
	enam0Prev  bool
	enam1Prev  bool
	enablPrev  bool

	// vertical delay
	vdelp0 bool
	vdelp1 bool
	vdelbl bool

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
	vd.Playfield = newPlayfield()

	// missile/ball enabling
	vd.enam0Delay = newDelayCounter("(dis/en)abling")
	if vd.enam0Delay == nil {
		return nil
	}
	vd.enam1Delay = newDelayCounter("(dis/en)abling")
	if vd.enam1Delay == nil {
		return nil
	}
	vd.enablDelay = newDelayCounter("(dis/en)abling")
	if vd.enablDelay == nil {
		return nil
	}

	// sprite objects
	vd.player0 = newSprite("player0", nil)
	if vd.player0 == nil {
		return nil
	}
	vd.player1 = newSprite("player1", nil)
	if vd.player1 == nil {
		return nil
	}
	vd.missile0 = newSprite("missile0", &vd.enam0)
	if vd.missile0 == nil {
		return nil
	}
	vd.missile1 = newSprite("missile1", &vd.enam1)
	if vd.missile1 == nil {
		return nil
	}
	vd.Ball = newSprite("ball", &vd.enabl)
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

// MachineInfoTerse returns the Video information in terse format
func (vd Video) MachineInfoTerse() string {
	return ""
}

// MachineInfo returns the Video information in verbose format
func (vd Video) MachineInfo() string {
	return ""
}

// map String to MachineInfo
func (vd Video) String() string {
	return vd.MachineInfo()
}

// TickSprites moves sprite elements on one video cycle
func (vd *Video) TickSprites() {
	// TODO: tick other sprites
	vd.TickBall()
}

// TickSpritesForHMOVE moves sprite elements if horiz movement value is in range
func (vd *Video) TickSpritesForHMOVE(count int) {
	if count == 0 {
		return
	}

	if vd.hmp0 >= uint8(count) {
	}
	if vd.hmp1 >= uint8(count) {
	}
	if vd.hmm0 >= uint8(count) {
	}
	if vd.hmm1 >= uint8(count) {
	}
	if vd.hmbl >= uint8(count) {
		vd.TickBall()
	}
}

// GetPixel returns the color of the pixel at the current time. it will default
// to returning background color if no sprite or playfield pixel is present -
// it should not be called therefore unless a VCS pixel is to be displayed
func (vd Video) GetPixel() uint8 {
	col := vd.colubk

	// TODO: complete pixel ordering
	if vd.ctrlpfPriority {
		// player 1
		// missile 1
		// player 0
		// missile 0
		use, c := vd.PixelPlayfield()
		if use {
			col = c
		}
		use, c = vd.PixelBall()
		if use {
			col = c
		}

	} else {
		use, c := vd.PixelBall()
		if use {
			col = c
		}
		use, c = vd.PixelPlayfield()
		if use {
			col = c
		}
		// player 1
		// missile 1
		// player 0
		// missile 0
	}

	return col
}

// ReadVideoMemory checks the TIA memory for changes to registers that are
// interesting to the video sub-system
func (vd *Video) ReadVideoMemory(register string, value uint8) bool {
	switch register {
	case "NUSIZ0":
	case "NUSIZ1":
	case "COLUP0":
		vd.colup0 = value & 0xfe
	case "COLUP1":
		vd.colup1 = value & 0xfe
	case "COLUPF":
		vd.colupf = value & 0xfe
	case "COLUBK":
		vd.colubk = value & 0xfe
	case "CTRLPF":
		vd.ctrlpfBallSize = (value & 0x30) >> 4
		vd.ctrlpfReflection = value&0x01 == 0x01
		vd.ctrlpfScoremode = value&0x02 == 0x02
		vd.ctrlpfPriority = value&0x04 == 0x04
	case "REFP0":
		vd.refp0 = value&0x40 == 0x40
	case "REFP1":
		vd.refp1 = value&0x40 == 0x40

		// delay of 5 video cycles for playfield writes seems correct - 1
		// entire CPU cycle plus one remaining cycle from the current
		// instruction
		//
		// there may be instances when there is more than one remaining video
		// cycle from the current instruction
		//
		// but then again, maybe the delay is 5 video cycles in all instances
	case "PF0":
		vd.Playfield.writeDelay.start(5, func() { vd.Playfield.writePf0(value) })
	case "PF1":
		vd.Playfield.writeDelay.start(5, func() { vd.Playfield.writePf1(value) })
	case "PF2":
		vd.Playfield.writeDelay.start(5, func() { vd.Playfield.writePf2(value) })

	case "RESP0":
	case "RESP1":
	case "RESM0":
	case "RESM1":
	case "RESBL":
		if *vd.hblank {
			vd.Ball.resetDelay.start(2, true)
		} else {
			vd.Ball.resetDelay.start(4, true)
		}
	case "GRP0":
	case "GRP1":
	case "ENAM0":
	case "ENAM1":
	case "ENABL":
		vd.enablDelay.start(1, value&0x20 == 0x20)
	case "HMP0":
	case "HMP1":
	case "HMM0":
	case "HMM1":
	case "HMBL":
		vd.hmbl = (value ^ 0x80) >> 4
	case "VDELP0":
		vd.vdelp0 = value&0x01 == 0x01
	case "VDELP1":
		vd.vdelp1 = value&0x01 == 0x01
	case "VDELBL":
		vd.vdelbl = value&0x01 == 0x01
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
