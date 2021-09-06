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
type ColorSignal uint8

// VideoBlack is the ColorSignal value that indicates no pixel is being output.
const VideoBlack ColorSignal = 0xff

// SignalAttributes represents the data sent to the television.
type SignalAttributes uint64

// List of bit masks to be used to with the SignalAttribute type.
const (
	VSync       SignalAttributes = 0b0000000000000000000000000000000000000000000001
	VBlank      SignalAttributes = 0b0000000000000000000000000000000000000000000010
	CBurst      SignalAttributes = 0b0000000000000000000000000000000000000000000100
	HSync       SignalAttributes = 0b0000000000000000000000000000000000000000001000
	AudioUpdate SignalAttributes = 0b0000000000000000000000000000000000000000010000
	AudioData   SignalAttributes = 0b0000000000000000000000000111111111111111100000 // 16 bits
	Pixel       SignalAttributes = 0b0000000000000000011111111000000000000000000000 // 8 bits
	Scanline    SignalAttributes = 0b0000000011111111100000000000000000000000000000 // 9 bits
	Clock       SignalAttributes = 0b1111111100000000000000000000000000000000000000 // 8 bits (signed)
)

// List of shift amounts to be used to access the corresponding bits in a
// SignalAttributes value.
const (
	AudioDataShift = 5
	PixelShift     = 21
	ScanlineShift  = 29
	ClockShift     = 38
)

// NoSignal is the null value of the SignalAttributes type.
const NoSignal = 0x00

func (a SignalAttributes) String() string {
	s := strings.Builder{}
	if a&VSync == VSync {
		s.WriteString("VSYNC ")
	}
	if a&VBlank == VBlank {
		s.WriteString("VBLANK ")
	}
	if a&CBurst == CBurst {
		s.WriteString("CBURST ")
	}
	if a&HSync == HSync {
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
	ReqClock
)

// StateAdj is used to specify adjustment scale for the ReqAdjust() function.
type StateAdj int

// List of valid adjustment scales.
const (
	AdjFramenum StateAdj = iota
	AdjScanline
	AdjInstruction
	AdjCPUCycle
	AdjClock
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
