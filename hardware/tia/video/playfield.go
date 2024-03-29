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
)

// ScreenRegion notes which part of the screen is currently being drawn.
type ScreenRegion int

// List of valid ScreenRegions.
const (
	RegionOffScreen ScreenRegion = iota
	RegionLeft
	RegionRight
)

// the number of color clocks (playfield pixels) per left/right region.
const RegionWidth = 20

// Playfield represnets the static playfield and background, the non-sprite
// areas of the graphical display.
type Playfield struct {
	tia tia

	// the color for the when playfield is on/off
	ForegroundColor uint8
	BackgroundColor uint8

	// RegularData and ReflectedData are updated on every call to the
	// SetPF*() functions
	//
	// Data is (re)pointed to either RegularData or ReflectedData whenever
	// SetPF*() is called and on the screen region boundaries.
	//
	// RegionLeft always uses RegularData and RegionRight uses either
	// RegularDat or ReflectedData depending on the state of the reflected bit
	// at either:
	//	- the start of the region
	//	- when PF bits are changed
	RegularData   []bool
	ReflectedData []bool
	Data          *[]bool

	// knowing what the left and right regions look like at any given time is
	// useful for debugging. for the emulation, the Data field is sufficient.
	LeftData  *[]bool
	RightData *[]bool

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

	// state of playfield "pixel"

	// playfield output color is held for one color-clock, even if the
	// playfield register is changed. we use the colorLatch field to decide
	// what color to use (foreground or background)
	colorLatch bool

	// which color we're using bassed on colorLatch (saves checking every time)
	color uint8

	// the state of colorLatch on the previous color clock
	prevColorLatch bool
}

func newPlayfield(tia tia) *Playfield {
	pf := &Playfield{
		tia:           tia,
		RegularData:   make([]bool, RegionWidth),
		ReflectedData: make([]bool, RegionWidth),
	}
	pf.LeftData = &pf.RegularData
	pf.RightData = &pf.RegularData
	return pf
}

// Snapshot creates a copy of the Video Playfield in its current state.
func (pf *Playfield) Snapshot() *Playfield {
	n := *pf

	n.RegularData = make([]bool, len(pf.RegularData))
	n.ReflectedData = make([]bool, len(pf.ReflectedData))

	copy(n.RegularData, pf.RegularData)
	copy(n.ReflectedData, pf.ReflectedData)

	if pf.Data == &pf.ReflectedData {
		n.Data = &n.ReflectedData
	} else {
		n.Data = &n.RegularData
	}

	n.LeftData = &n.RegularData

	if pf.RightData == &pf.ReflectedData {
		n.RightData = &n.ReflectedData
	} else {
		n.RightData = &n.RegularData
	}

	return &n
}

// Plumb changes into playfield.
func (pf *Playfield) Plumb(tia tia) {
	pf.tia = tia
}

// Label returns an appropriate name for playfield.
func (pf *Playfield) Label() string {
	return "Playfield"
}

func (pf *Playfield) String() string {
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

func (pf *Playfield) tick() bool {
	pf.prevColorLatch = pf.colorLatch

	if *pf.tia.pclk == phaseclock.RisingPhi2 {
		// this switch statement is based on the "Horizontal Sync Counter"
		// table in TIA_HW_Notes.txt. for convenience we're not using a
		// colorclock (tia) delay but simply looking for the hsync.Count 4
		// cycles beyond the trigger point described in the TIA_HW_Notes.txt
		// document.  we believe this has the same effect.
		//
		// further, RSYNC can monkey with the current hsync value unexpectedly
		// and because of this we need an extra effort to make sure we're in
		// the correct screen region.
		switch *pf.tia.hsync {
		case 0:
			// start of scanline
			pf.Region = RegionOffScreen
			pf.latchRegionData()

		case 17:
			// start of visible screen (playfield not affected by HMOVE)
			pf.Region = RegionLeft
			pf.Data = &pf.RegularData
			pf.latchRegionData()

		case 37:
			// just past the centre of the visible screen
			pf.Region = RegionRight
			pf.latchRegionData()
		}

		switch pf.Region {
		case RegionOffScreen:
			pf.colorLatch = false
			pf.color = pf.BackgroundColor
			return pf.colorLatch != pf.prevColorLatch
		case RegionLeft:
			pf.Idx = int(*pf.tia.hsync) - 17
		case RegionRight:
			pf.Idx = int(*pf.tia.hsync) - 37
		}

		// if region is RegionOffScreen then we've returned already

		if pf.Idx >= 0 {
			pf.colorLatch = (*pf.Data)[pf.Idx]
		}
	} else {
		// do nothing if we're in the off screen region
		if pf.Region == RegionOffScreen {
			return false
		}
	}

	// set color according to colorLatch
	if pf.colorLatch {
		pf.color = pf.ForegroundColor
	} else {
		pf.color = pf.BackgroundColor
	}

	return pf.colorLatch != pf.prevColorLatch
}

// called whenever playfield bits change or the screen region changes.
func (pf *Playfield) latchRegionData() {
	pf.LeftData = &pf.RegularData
	if !pf.Reflected {
		pf.RightData = &pf.RegularData
	} else {
		pf.RightData = &pf.ReflectedData
	}

	if pf.Region != RegionRight || !pf.Reflected {
		pf.Data = &pf.RegularData
	} else {
		pf.Data = &pf.ReflectedData
	}
}

// SetPF0 sets the playfield PF0 bits.
func (pf *Playfield) SetPF0(v uint8) {
	pf.PF0 = v & 0xf0
	pf.RegularData[0] = pf.PF0&0x10 == 0x10
	pf.RegularData[1] = pf.PF0&0x20 == 0x20
	pf.RegularData[2] = pf.PF0&0x40 == 0x40
	pf.RegularData[3] = pf.PF0&0x80 == 0x80
	pf.ReflectedData[16] = pf.RegularData[3]
	pf.ReflectedData[17] = pf.RegularData[2]
	pf.ReflectedData[18] = pf.RegularData[1]
	pf.ReflectedData[19] = pf.RegularData[0]
}

// SetPF1 sets the playfield PF1 bits.
func (pf *Playfield) SetPF1(v uint8) {
	pf.PF1 = v
	pf.RegularData[4] = pf.PF1&0x80 == 0x80
	pf.RegularData[5] = pf.PF1&0x40 == 0x40
	pf.RegularData[6] = pf.PF1&0x20 == 0x20
	pf.RegularData[7] = pf.PF1&0x10 == 0x10
	pf.RegularData[8] = pf.PF1&0x08 == 0x08
	pf.RegularData[9] = pf.PF1&0x04 == 0x04
	pf.RegularData[10] = pf.PF1&0x02 == 0x02
	pf.RegularData[11] = pf.PF1&0x01 == 0x01
	pf.ReflectedData[8] = pf.RegularData[11]
	pf.ReflectedData[9] = pf.RegularData[10]
	pf.ReflectedData[10] = pf.RegularData[9]
	pf.ReflectedData[11] = pf.RegularData[8]
	pf.ReflectedData[12] = pf.RegularData[7]
	pf.ReflectedData[13] = pf.RegularData[6]
	pf.ReflectedData[14] = pf.RegularData[5]
	pf.ReflectedData[15] = pf.RegularData[4]
}

// SetPF2 sets the playfield PF2 bits.
func (pf *Playfield) SetPF2(v uint8) {
	pf.PF2 = v
	pf.RegularData[12] = pf.PF2&0x01 == 0x01
	pf.RegularData[13] = pf.PF2&0x02 == 0x02
	pf.RegularData[14] = pf.PF2&0x04 == 0x04
	pf.RegularData[15] = pf.PF2&0x08 == 0x08
	pf.RegularData[16] = pf.PF2&0x10 == 0x10
	pf.RegularData[17] = pf.PF2&0x20 == 0x20
	pf.RegularData[18] = pf.PF2&0x40 == 0x40
	pf.RegularData[19] = pf.PF2&0x80 == 0x80
	pf.ReflectedData[0] = pf.RegularData[19]
	pf.ReflectedData[1] = pf.RegularData[18]
	pf.ReflectedData[2] = pf.RegularData[17]
	pf.ReflectedData[3] = pf.RegularData[16]
	pf.ReflectedData[4] = pf.RegularData[15]
	pf.ReflectedData[5] = pf.RegularData[14]
	pf.ReflectedData[6] = pf.RegularData[13]
	pf.ReflectedData[7] = pf.RegularData[12]
}

func (pf *Playfield) setPF0(v uint8) {
	pf.SetPF0(v)
}

func (pf *Playfield) setPF1(v uint8) {
	pf.SetPF1(v)
}

func (pf *Playfield) setPF2(v uint8) {
	pf.SetPF2(v)
}

func (pf *Playfield) SetCTRLPF(value uint8) {
	pf.Ctrlpf = value
	pf.Scoremode = value&CTRLPFScoremodeMask == CTRLPFScoremodeMask
	pf.Priority = value&CTRLPFPriorityMask == CTRLPFPriorityMask
	pf.Reflected = value&CTRLPFReflectedMask == CTRLPFReflectedMask
}

func (pf *Playfield) setColor(col uint8) {
	pf.ForegroundColor = col
}

func (pf *Playfield) setBackground(col uint8) {
	pf.BackgroundColor = col
}
