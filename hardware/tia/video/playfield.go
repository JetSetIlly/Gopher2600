package video

import "fmt"

type playfield struct {
	// plafield data is 20bits wide, the second half of the playfield is either
	// a straight repetition of the data or a reflection, depending on the
	// state of the playfield control bits
	data [20]bool

	// data is a combination of three registers: pf0, pf1 and pf2. how these
	// are combined is not obvious
	pf0 uint8
	pf1 uint8
	pf2 uint8

	tickCount int
	tickPhase int

	// there's a slight delay when writing to playfield registers. note that we
	// use the same delayCounter for all playfield registers. this is okay
	// because the delay is so short there is no chance of another write being
	// requested before the previous request has been resolved
	writeDelay *delayCounter
}

func newPlayfield() *playfield {
	pf := new(playfield)
	pf.writeDelay = newDelayCounter("writing")
	if pf.writeDelay == nil {
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

// map String to MachineInfo
func (pf playfield) String() string {
	return pf.MachineInfo()
}

func (pf *playfield) writePf0(value uint8) {
	pf.pf0 = value & 0xf0
	pf.data[0] = pf.pf0&0x10 == 0x10
	pf.data[1] = pf.pf0&0x20 == 0x20
	pf.data[2] = pf.pf0&0x40 == 0x40
	pf.data[3] = pf.pf0&0x80 == 0x80
}

func (pf *playfield) writePf1(value uint8) {
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

func (pf *playfield) writePf2(value uint8) {
	pf.pf2 = value
	pf.data[12] = pf.pf2&0x01 == 0x01
	pf.data[13] = pf.pf2&0x02 == 0x02
	pf.data[15] = pf.pf2&0x04 == 0x04
	pf.data[15] = pf.pf2&0x08 == 0x08
	pf.data[16] = pf.pf2&0x10 == 0x10
	pf.data[17] = pf.pf2&0x20 == 0x20
	pf.data[18] = pf.pf2&0x40 == 0x40
	pf.data[19] = pf.pf2&0x80 == 0x80
}

// TickPlayfield moves playfield on one video cycle
func (vd *Video) TickPlayfield() {
	// reset
	if vd.Playfield.writeDelay.tick() {
		vd.Playfield.writeDelay.payloadValue.(func())()
	}

	if vd.colorClock.MatchBeginning(17) {
		vd.Playfield.tickPhase = 1
		vd.Playfield.tickCount = 0
	} else if vd.colorClock.MatchBeginning(37) {
		vd.Playfield.tickPhase = 2
		vd.Playfield.tickCount = 0
	} else if vd.colorClock.MatchBeginning(0) {
		vd.Playfield.tickPhase = 0
	} else if vd.Playfield.tickPhase != 0 && vd.colorClock.Phase == 0 {
		vd.Playfield.tickCount++
	}
}

// PixelPlayfield returns the color of the playfield at the current time.
// returns (false, 0) if no pixel is to be seen; and (true, col) if there is
func (vd *Video) PixelPlayfield() (bool, uint8) {
	if vd.Playfield.tickPhase != 0 {
		if vd.Playfield.tickPhase == 1 || !vd.ctrlpfReflection {
			if vd.Playfield.data[vd.Playfield.tickCount] {
				return true, vd.colupf
			}
		} else {
			// TODO: reflected playfield
		}
	}
	return false, 0
}
