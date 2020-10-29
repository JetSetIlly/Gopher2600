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

// Package signal exposes the interface between the VCS and the television
// implementation.
package signal

import "strings"

// ColorSignal represents the signal that is sent from the VCS to the television.
type ColorSignal int

// VideoBlack is the ColorSignal value that indicates no pixel is being output.
const VideoBlack ColorSignal = -1

// SignalAttributes represents the data sent to the television.
type SignalAttributes struct {
	VSync     bool
	VBlank    bool
	CBurst    bool
	HSync     bool
	Pixel     ColorSignal
	AudioData uint8

	// whether the AudioData is valid. should be true only every 114th clock,
	// which equates to 30Khz
	AudioUpdate bool

	// the position on the screen this signal was applied to. added by the
	// television implementation
	HorizPos int
	Scanline int
}

func (a SignalAttributes) String() string {
	s := strings.Builder{}
	if a.VSync {
		s.WriteString("VSYNC ")
	}
	if a.VBlank {
		s.WriteString("VBLANK ")
	}
	if a.CBurst {
		s.WriteString("CBURST ")
	}
	if a.HSync {
		s.WriteString("HSYNC ")
	}
	return s.String()
}

// StateReq is used to identify which television attribute is being asked
// with the GetState() function.
type StateReq int

// List of valid state requests.
const (
	ReqFramenum StateReq = iota
	ReqScanline
	ReqHorizPos
)

// TelevisionTIA exposes only the functions required by the TIA.
type TelevisionTIA interface {
	Signal(SignalAttributes) error
	GetState(StateReq) int
}

// TelevisionSprite exposes only the functions required by the video sprites.
type TelevisionSprite interface {
	GetState(StateReq) int
}
