package video

import (
	"fmt"
	"gopher2600/hardware/tia/delay"
	"gopher2600/hardware/tia/delay/future"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"strings"
)

type playfield struct {
	tiaClk *phaseclock.PhaseClock
	hsync  *polycounter.Polycounter

	// tiaDelay is not currently used
	tiaDelay future.Scheduler

	// the color for the when playfield is on/off
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

	// screenRegion keeps track of which part of the screen we're currently in
	//  0 -> hblank
	//  1 -> left half of screen
	//  2 -> right half of screen
	screenRegion int

	// idx is the index into the data field - interpreted depending on
	// screenRegion and reflection settings
	idx int

	// a playfield "pixel" is sustained for the duration 3 video cycles, even
	// if the playfield register is changed. see pixel() function below
	currentPixelIsOn bool
}

func newPlayfield(tiaClk *phaseclock.PhaseClock, hsync *polycounter.Polycounter, tiaDelay future.Scheduler) *playfield {
	pf := playfield{tiaClk: tiaClk, hsync: hsync, tiaDelay: tiaDelay}
	return &pf
}

func (pf playfield) MachineInfoTerse() string {
	s := strings.Builder{}
	s.WriteString("playfield: ")

	// playfield bits - first half
	for i := 0; i < len(pf.data); i++ {
		if pf.data[i] {
			s.WriteString("1")
		} else {
			s.WriteString("0")
		}
	}

	// playfield bits - second half
	for i := len(pf.data) - 1; i >= 0; i-- {
		if pf.data[i] {
			s.WriteString("1")
		} else {
			s.WriteString("0")
		}
	}

	// sundry playfield information
	if pf.reflected {
		s.WriteString(" reflected")
	}
	if pf.scoremode {
		s.WriteString(" scoremode")
	}
	if pf.priority {
		s.WriteString(" priority")
	}

	return s.String()
}

func (pf playfield) MachineInfo() string {
	s := strings.Builder{}
	s.WriteString("playfield: ")

	// prepare a line to point to the current playfield bit; or a suitable
	// message to indcate no playfield output
	idxPointer := ""
	switch pf.screenRegion {
	case 0:
		idxPointer = "no playfield during hblank period"
	case 1:
		idxPointer = fmt.Sprintf("%s^", strings.Repeat(" ", len(s.String())+pf.idx))
	case 2:
		idxPointer = fmt.Sprintf("%s^", strings.Repeat(" ", len(s.String())+pf.idx+len(pf.data)))
	}

	// playfield bits - first half
	for i := 0; i < len(pf.data); i++ {
		if pf.data[i] {
			s.WriteString("1")
		} else {
			s.WriteString("0")
		}
	}
	// playfield bits - second half
	for i := len(pf.data) - 1; i >= 0; i-- {
		if pf.data[i] {
			s.WriteString("1")
		} else {
			s.WriteString("0")
		}
	}

	// output the pointer line we prepared earlier
	s.WriteString(fmt.Sprintf("\n%s", idxPointer))

	// sundry playfield information
	s.WriteString(fmt.Sprintf("\n   pf0: %08b\n   pf1: %08b\n   pf2: %08b", pf.pf0, pf.pf1, pf.pf2))
	s.WriteString(fmt.Sprintf("\n   fg color: %d", pf.foregroundColor))
	s.WriteString(fmt.Sprintf("\n   bg color: %d", pf.backgroundColor))
	s.WriteString(fmt.Sprintf("\n   reflected: %v\n   scoremode: %v\n   priority %v\n", pf.reflected, pf.scoremode, pf.priority))

	return s.String()
}

func (pf *playfield) pixel() (bool, uint8) {
	// because playfield is closely related to the HSYNC counter there is no
	// separate tick() function

	newPixel := false

	if pf.tiaClk.InPhase() {
		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. for convenience we're not using a
		// colorclock (tia) delay but simply looking for the hsync.Count 4
		// cycles beyond the trigger point described in the TIA_HW_Notes.txt
		// document.  we believe this has the same effect.
		switch pf.hsync.Count {
		case 17: // [RHB]
			// start of visible screen (playfield not affected by HMOVE)
			pf.screenRegion = 1
			pf.idx = 0
			newPixel = true
		case 37: // [CNT]
			// just past the centre of the visible screen
			pf.screenRegion = 2
			pf.idx = 0
			newPixel = true
		case 0:
			// start of scanline
			pf.screenRegion = 0
			pf.idx = 0
			newPixel = true
		default:
			pf.idx++
			newPixel = true
		}
	}

	// pixel returns the color of the playfield at the current time.
	// returns (false, 0) if no pixel is to be seen; and (true, col) if there is
	if newPixel && pf.screenRegion != 0 {
		if pf.screenRegion == 1 || !pf.reflected {
			// normal, non-reflected playfield
			pf.currentPixelIsOn = pf.data[pf.idx]
		} else {
			// reflected playfield
			pf.currentPixelIsOn = pf.data[len(pf.data)-pf.idx-1]
		}
	}

	if pf.currentPixelIsOn {
		return true, pf.foregroundColor
	}
	return false, pf.backgroundColor
}

func (pf *playfield) scheduleWrite(segment int, value uint8, futureWrite future.Scheduler) {
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

	futureWrite.Schedule(delay.WritePlayfield, f, "writing")
}
