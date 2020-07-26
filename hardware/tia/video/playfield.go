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

package video

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/tia/phaseclock"
	"github.com/jetsetilly/gopher2600/hardware/tia/polycounter"
)

// ScreenRegion notes which part of the screen is currently being drawn
type ScreenRegion int

// List of valid ScreenRegions
const (
	RegionOffScreen ScreenRegion = iota
	RegionLeft
	RegionRight
)

type playfield struct {
	pclk  *phaseclock.PhaseClock
	hsync *polycounter.Polycounter

	// the color for the when playfield is on/off
	ForegroundColor uint8
	BackgroundColor uint8

	// plafield Data is 20bits wide, the second half of the playfield is either
	// a straight repetition of the Data or a reflection, depending on the
	// state of the playfield control bits
	Data [20]bool

	// the data field is a combination of three segments: PF0, pf1 and pf2.
	// these represent the three registers in VCS memory but we don't actually
	// use then, except in the String() functions
	PF0 uint8
	PF1 uint8
	PF2 uint8

	// for convenience we store the raw CTRLPF register value and the
	// normalised control bits specific to the playfield
	Ctrlpf    uint8
	Reflected bool
	Priority  bool
	Scoremode bool

	// Region keeps track of which part of the screen we're currently in
	Region ScreenRegion

	// Idx is the index into the data field - interpreted depending on
	// screenRegion and reflection settings
	Idx int

	// a playfield "pixel" is sustained for the duration 3 video cycles, even
	// if the playfield register is changed. see pixel() function below
	currentPixelIsOn bool
}

func newPlayfield(pclk *phaseclock.PhaseClock, hsync *polycounter.Polycounter) *playfield {
	pf := playfield{pclk: pclk, hsync: hsync}
	return &pf
}

func (pf playfield) Label() string {
	return "Playfield"
}

func (pf playfield) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%04b", pf.PF0>>4))
	s.WriteString(fmt.Sprintf(" %08b", pf.PF1))
	s.WriteString(fmt.Sprintf(" %08b", pf.PF2))

	notes := false

	// sundry playfield information
	if pf.Reflected {
		s.WriteString(" refl")
		notes = true
	}
	if pf.Scoremode {
		if notes {
			s.WriteString(",")
		}
		s.WriteString(" score")
		notes = true
	}
	if pf.Priority {
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
			pf.Region = RegionRight
		} else if pf.hsync.Count() >= 17 {
			// start of visible screen (playfield not affected by HMOVE)
			pf.Region = RegionLeft
		} else {
			// start of scanline
			pf.Region = RegionOffScreen
		}

		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. for convenience we're not using a
		// colorclock (tia) delay but simply looking for the hsync.Count 4
		// cycles beyond the trigger point described in the TIA_HW_Notes.txt
		// document.  we believe this has the same effect.
		switch pf.Region {
		case RegionOffScreen:
			pf.Idx = pf.hsync.Count()
			pf.currentPixelIsOn = false
		case RegionLeft:
			pf.Idx = pf.hsync.Count() - 17
			newPixel = true
		case RegionRight:
			pf.Idx = pf.hsync.Count() - 37
			newPixel = true
		}
	}

	// pixel returns the color of the playfield at the current time.
	// returns (false, 0) if no pixel is to be seen; and (true, col) if there is
	if newPixel && pf.Region != RegionOffScreen {
		if pf.Region == RegionLeft || !pf.Reflected {
			// normal, non-reflected playfield
			pf.currentPixelIsOn = pf.Data[pf.Idx]
		} else {
			// reflected playfield
			pf.currentPixelIsOn = pf.Data[len(pf.Data)-pf.Idx-1]
		}
	}

	if pf.currentPixelIsOn {
		return true, pf.ForegroundColor
	}
	return false, pf.BackgroundColor
}

// SetPF0 sets the playfield PF0 bits
func (pf *playfield) SetPF0(v uint8) {
	pf.PF0 = v & 0xf0
	pf.Data[0] = pf.PF0&0x10 == 0x10
	pf.Data[1] = pf.PF0&0x20 == 0x20
	pf.Data[2] = pf.PF0&0x40 == 0x40
	pf.Data[3] = pf.PF0&0x80 == 0x80
}

// SetPF1 sets the playfield PF1 bits
func (pf *playfield) SetPF1(v uint8) {
	pf.PF1 = v
	pf.Data[4] = pf.PF1&0x80 == 0x80
	pf.Data[5] = pf.PF1&0x40 == 0x40
	pf.Data[6] = pf.PF1&0x20 == 0x20
	pf.Data[7] = pf.PF1&0x10 == 0x10
	pf.Data[8] = pf.PF1&0x08 == 0x08
	pf.Data[9] = pf.PF1&0x04 == 0x04
	pf.Data[10] = pf.PF1&0x02 == 0x02
	pf.Data[11] = pf.PF1&0x01 == 0x01
}

// SetPF2 sets the playfield PF2 bits
func (pf *playfield) SetPF2(v uint8) {
	pf.PF2 = v
	pf.Data[12] = pf.PF2&0x01 == 0x01
	pf.Data[13] = pf.PF2&0x02 == 0x02
	pf.Data[14] = pf.PF2&0x04 == 0x04
	pf.Data[15] = pf.PF2&0x08 == 0x08
	pf.Data[16] = pf.PF2&0x10 == 0x10
	pf.Data[17] = pf.PF2&0x20 == 0x20
	pf.Data[18] = pf.PF2&0x40 == 0x40
	pf.Data[19] = pf.PF2&0x80 == 0x80
}

func (pf *playfield) setPF0(v interface{}) {
	pf.SetPF0(v.(uint8))
}

func (pf *playfield) setPF1(v interface{}) {
	pf.SetPF1(v.(uint8))
}

func (pf *playfield) setPF2(v interface{}) {
	pf.SetPF2(v.(uint8))
}

func (pf *playfield) SetCTRLPF(value uint8) {
	pf.Ctrlpf = value
	pf.Reflected = value&0x01 == 0x01
	pf.Scoremode = value&0x02 == 0x02
	pf.Priority = value&0x04 == 0x04
}

func (pf *playfield) setColor(col uint8) {
	pf.ForegroundColor = col
}

func (pf *playfield) setBackground(col uint8) {
	pf.BackgroundColor = col
}
