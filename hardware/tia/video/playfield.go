package video

import (
	"fmt"
	"gopher2600/hardware/tia/colorclock"
)

type playfield struct {
	colorClock *colorclock.ColorClock

	foregroundColor uint8
	backgroundColor uint8

	// plafield data is 20bits wide, the second half of the playfield is either
	// a straight repetition of the data or a reflection, depending on the
	// state of the playfield control bits
	data [20]bool

	// the data field is a combination of three segments: pf0, pf1 and pf2.
	// these represent the three registers in VCS memory but we don't actually
	// use then, except in the MachineInfo() functions
	pf0 uint8
	pf1 uint8
	pf2 uint8

	// playfield properties
	reflected bool
	priority  bool
	scoremode bool

	// there's a slight delay when writing to playfield registers. note that we
	// use the same future instance for all playfield registers. this is okay
	// because the delay is so short there is no chance of another write being
	// scheduled before the previous request has been resolved
	futureWrite *future

	// screenRegion keeps track of which part of the screen we're currently in
	//  0 -> hblank
	//  1 -> left half of screen
	//  2 -> right half of screen
	screenRegion int

	// idx is the index into the data field - interpreted depending on
	// screenRegion and reflection settings
	idx int
}

func newPlayfield(colorClock *colorclock.ColorClock) *playfield {
	pf := new(playfield)
	pf.colorClock = colorClock
	pf.futureWrite = newFuture()
	if pf.futureWrite == nil {
		return nil
	}
	return pf
}

func (pf playfield) MachineInfoTerse() string {
	s := "playfield: "
	for i := 0; i < len(pf.data); i++ {
		if pf.data[i] {
			s += "1"
		} else {
			s += "0"
		}
	}
	return s
}

func (pf playfield) MachineInfo() string {
	return fmt.Sprintf("pf0: %08b\npf1: %08b\npf2: %08b\n%s", pf.pf0, pf.pf1, pf.pf2, pf.MachineInfoTerse())
}

func (pf *playfield) tick() {
	// reset
	if pf.futureWrite.tick() {
		pf.futureWrite.payload.(func())()
	}

	if pf.colorClock.MatchBeginning(17) {
		// start of visible screen (playfield not affected by HMOVE)
		// 101110
		pf.screenRegion = 1
		pf.idx = 0
	} else if pf.colorClock.MatchBeginning(37) {
		// just past the centre of the visible screen
		// 110110
		pf.screenRegion = 2
		pf.idx = 0
	} else if pf.colorClock.MatchBeginning(0) {
		// start of scanline
		// 000000
		pf.screenRegion = 0
	} else if pf.screenRegion != 0 && pf.colorClock.Phase == 0 {
		pf.idx++
	}
}

// pixel returns the color of the playfield at the current time.
// returns (false, 0) if no pixel is to be seen; and (true, col) if there is
func (pf *playfield) pixel() (bool, uint8) {
	if pf.screenRegion != 0 {
		if pf.screenRegion == 1 || !pf.reflected {
			// normal, non-reflected playfield
			if pf.data[pf.idx] {
				return true, pf.foregroundColor
			}
		} else {
			// reflected playfield
			if pf.data[len(pf.data)-pf.idx-1] {
				return true, pf.foregroundColor
			}
		}
	}
	return false, pf.backgroundColor
}

func (pf *playfield) scheduleWrite(segment int, value uint8) {
	var f func()
	switch segment {
	case 0:
		f = func() {
			pf.pf0 = value & 0xf0
			pf.data[0] = pf.pf0&0x10 == 0x10
			pf.data[1] = pf.pf0&0x20 == 0x20
			pf.data[2] = pf.pf0&0x40 == 0x40
			pf.data[3] = pf.pf0&0x80 == 0x80
		}
	case 1:
		f = func() {
			pf.pf1 = value
			pf.data[4] = pf.pf1&0x80 == 0x80
			pf.data[5] = pf.pf1&0x40 == 0x40
			pf.data[6] = pf.pf1&0x20 == 0x20
			pf.data[7] = pf.pf1&0x10 == 0x10
			pf.data[8] = pf.pf1&0x08 == 0x08
			pf.data[9] = pf.pf1&0x04 == 0x04
			pf.data[10] = pf.pf1&0x02 == 0x02
			pf.data[11] = pf.pf1&0x01 == 0x01
		}
	case 2:
		f = func() {
			pf.pf2 = value
			pf.data[12] = pf.pf2&0x01 == 0x01
			pf.data[13] = pf.pf2&0x02 == 0x02
			pf.data[14] = pf.pf2&0x04 == 0x04
			pf.data[15] = pf.pf2&0x08 == 0x08
			pf.data[16] = pf.pf2&0x10 == 0x10
			pf.data[17] = pf.pf2&0x20 == 0x20
			pf.data[18] = pf.pf2&0x40 == 0x40
			pf.data[19] = pf.pf2&0x80 == 0x80
		}
	}

	// delay of 5 video cycles for playfield writes seems correct - 1
	// entire CPU cycle plus one remaining cycle from the current
	// instruction
	//
	// there may be instances when there is more than one remaining video
	// cycle from the current instruction
	//
	// but then again, maybe the delay is 5 video cycles in all instances
	pf.futureWrite.schedule(delayPlayfieldWrite, f)
}
