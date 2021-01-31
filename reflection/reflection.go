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

package reflection

import (
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
)

// Renderer implementations accepts ReflectPixel values and associates it in
// some way with the most recent television signal.
type Renderer interface {
	Reflect(VideoStep) error
}

// IdentifyReflector implementations can identify a reflection.Renderer.
type IdentifyReflector interface {
	GetReflectionRenderer() Renderer
}

// VideoStep packages together the details of the the last video step that
// would otherwise be difficult for a debugger to access.
//
// It includes the CPU execution result, the bank from which the instruction
// originates, the video element that produced the last video pixel on screen;
// among other information.
//
// Note that ordering of the structure is important. There's a saving of about
// 2MB per frame compared to the unoptimal ordering.
type VideoStep struct {
	CPU          execution.Result
	Bank         mapper.BankInfo
	VideoElement video.Element
	TV           signal.SignalAttributes
	Hmove        Hmove
	WSYNC        bool
	IsRAM        bool

	// whether Coprocessor is active
	CoprocessorActive bool

	// whether Hblank is on
	IsHblank bool

	// string representation of collision state for this videostep
	Collision video.Collisions
}

// Hmove groups the HMOVE reflection information. It's too complex a property
// to distil into a single variable.
//
// Ordering of the structure is important.
type Hmove struct {
	DelayCt  int
	Delay    bool
	Latch    bool
	RippleCt uint8
}

// List of valid overlay reflection overlay types.
const (
	WSYNC       = "WSYNC"
	COLLISIONS  = "Collisions"
	HMOVE       = "HMOVE"
	COPROCESSOR = "Coprocessor"
)

// OverlayList is the list of overlays that should be supported by a reflection.Renderer.
var OverlayList = []string{WSYNC, COLLISIONS, HMOVE, COPROCESSOR}
