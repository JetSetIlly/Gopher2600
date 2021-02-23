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

// Info identifies the reflection information that can be ascertained from the
// contents of a VideoStep. Other information can probably be gleaned but these
// are the ones that have been identified. For convenience only.
type Info int

// List of valid Info value.
const (
	WSYNC Info = iota
	Collision
	CXCLR
	HMOVEdelay
	HMOVE
	HMOVElatched
	CoprocessorActive
)

// Renderer implementations display or otherwise process VideoStep values.
type Renderer interface {
	// Mark the start and end of an update event from the television.
	// Reflect() should only be called between calls of UpdatingPixels(true)
	// and UpdatingPixels(false)
	UpdatingPixels(updating bool)

	// Reflect sends a VideoStep instance to the Renderer.
	Reflect(VideoStep) error
}

// Broker implementations can identify a reflection.Renderer.
type Broker interface {
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
	CPU               execution.Result
	Collision         video.Collisions
	Bank              mapper.BankInfo
	TV                signal.SignalAttributes
	Hmove             Hmove
	VideoElement      video.Element
	WSYNC             bool
	IsRAM             bool
	CoprocessorActive bool
	IsHblank          bool
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
