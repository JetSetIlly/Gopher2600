// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package video

import (
	"fmt"
	"gopher2600/hardware/tia/phaseclock"
	"gopher2600/hardware/tia/polycounter"
	"strings"
)

type screenRegion int

const (
	regionOffScreen screenRegion = iota
	regionLeft
	regionRight
)

type playfield struct {
	pclk  *phaseclock.PhaseClock
	hsync *polycounter.Polycounter

	// the color for the when playfield is on/off
	foregroundColor uint8
	backgroundColor uint8

	// plafield data is 20bits wide, the second half of the playfield is either
	// a straight repetition of the data or a reflection, depending on the
	// state of the playfield control bits
	data [20]bool

	// the data field is a combination of three segments: pf0, pf1 and pf2.
	// these represent the three registers in VCS memory but we don't actually
	// use then, except in the String() functions
	pf0 uint8
	pf1 uint8
	pf2 uint8

	// playfield properties
	reflected bool
	priority  bool
	scoremode bool

	// region keeps track of which part of the screen we're currently in
	region screenRegion

	// idx is the index into the data field - interpreted depending on
	// screenRegion and reflection settings
	idx int

	// a playfield "pixel" is sustained for the duration 3 video cycles, even
	// if the playfield register is changed. see pixel() function below
	currentPixelIsOn bool
}

func newPlayfield(pclk *phaseclock.PhaseClock, hsync *polycounter.Polycounter) *playfield {
	pf := playfield{pclk: pclk, hsync: hsync}
	return &pf
}

func (pf playfield) String() string {
	s := strings.Builder{}
	s.WriteString("playfield: ")

	s.WriteString(fmt.Sprintf("%04b", pf.pf0>>4))
	s.WriteString(fmt.Sprintf(" %08b", pf.pf1))
	s.WriteString(fmt.Sprintf(" %08b", pf.pf2))

	notes := false

	// sundry playfield information
	if pf.reflected {
		s.WriteString(" refl")
		notes = true
	}
	if pf.scoremode {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" score")
		notes = true
	}
	if pf.priority {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" pri")
	}

	return s.String()
}

func (pf *playfield) pixel() (bool, uint8) {
	// because playfield is closely related to the HSYNC counter there is no
	// separate tick() function

	newPixel := false

	if pf.pclk.Phi2() {
		// RSYNC can monkey with the current hsync value unexpectedly and
		// because of this we need an extra effort to make sure we're in the
		// correct screen region.
		if pf.hsync.Count() >= 37 {
			// just past the centre of the visible screen
			pf.region = regionRight
		} else if pf.hsync.Count() >= 17 {
			// start of visible screen (playfield not affected by HMOVE)
			pf.region = regionLeft
		} else {
			// start of scanline
			pf.region = regionOffScreen
		}

		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. for convenience we're not using a
		// colorclock (tia) delay but simply looking for the hsync.Count 4
		// cycles beyond the trigger point described in the TIA_HW_Notes.txt
		// document.  we believe this has the same effect.
		switch pf.region {
		case 0:
			pf.idx = pf.hsync.Count()
			pf.currentPixelIsOn = false
		case 1:
			pf.idx = pf.hsync.Count() - 17
			newPixel = true
		case 2:
			pf.idx = pf.hsync.Count() - 37
			newPixel = true
		}
	}

	// pixel returns the color of the playfield at the current time.
	// returns (false, 0) if no pixel is to be seen; and (true, col) if there is
	if newPixel && pf.region != regionOffScreen {
		if pf.region == regionLeft || !pf.reflected {
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

func (pf *playfield) setSegment0(v interface{}) {
	pf.pf0 = v.(uint8) & 0xf0
	pf.data[0] = pf.pf0&0x10 == 0x10
	pf.data[1] = pf.pf0&0x20 == 0x20
	pf.data[2] = pf.pf0&0x40 == 0x40
	pf.data[3] = pf.pf0&0x80 == 0x80
}

func (pf *playfield) setSegment1(v interface{}) {
	pf.pf1 = v.(uint8)
	pf.data[4] = pf.pf1&0x80 == 0x80
	pf.data[5] = pf.pf1&0x40 == 0x40
	pf.data[6] = pf.pf1&0x20 == 0x20
	pf.data[7] = pf.pf1&0x10 == 0x10
	pf.data[8] = pf.pf1&0x08 == 0x08
	pf.data[9] = pf.pf1&0x04 == 0x04
	pf.data[10] = pf.pf1&0x02 == 0x02
	pf.data[11] = pf.pf1&0x01 == 0x01
}

func (pf *playfield) setSegment2(v interface{}) {
	pf.pf2 = v.(uint8)
	pf.data[12] = pf.pf2&0x01 == 0x01
	pf.data[13] = pf.pf2&0x02 == 0x02
	pf.data[14] = pf.pf2&0x04 == 0x04
	pf.data[15] = pf.pf2&0x08 == 0x08
	pf.data[16] = pf.pf2&0x10 == 0x10
	pf.data[17] = pf.pf2&0x20 == 0x20
	pf.data[18] = pf.pf2&0x40 == 0x40
	pf.data[19] = pf.pf2&0x80 == 0x80
}

func (pf *playfield) setControlBits(ctrlpf uint8) {
	pf.reflected = ctrlpf&0x01 == 0x01
	pf.scoremode = ctrlpf&0x02 == 0x02
	pf.priority = ctrlpf&0x04 == 0x04
}

func (pf *playfield) setColor(col uint8) {
	pf.foregroundColor = col
}

func (pf *playfield) setBackground(col uint8) {
	pf.backgroundColor = col
}
