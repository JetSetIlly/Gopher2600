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

import (
	"strings"
)

// ColorSignal is the shortened Chroma-Luminance representation used by the 2600
// internally. The least-significant bit has been masked away.
//
// Expanding the value to actual YIQ values is unecessary at this stage although
// it would arguably be more correct
type ColorSignal uint8

// VideoBlack is the ColorSignal value that indicates no pixel is being output.
// This is only possible because the least-significant bit is not used in the
// colour signal from the 2600 (it has been masked away)
const VideoBlack ColorSignal = 0xff

// Index value to indicate that the signal is invalid
const NoSignal = -1

// SignalAttributes represents the data sent to the television
//
// When reset the Index field should be set to NoSignal and the Color field
// should be set to VideoBlack
type SignalAttributes struct {
	Index  int
	VSync  bool
	VBlank bool
	CBurst bool
	HSync  bool
	Color  ColorSignal
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

type AudioSignalAttributes struct {
	AudioChannel0 uint8
	AudioChannel1 uint8
}
