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

// Renderer implementations display or otherwise process VideoStep values.
type Renderer interface {
	// Reflect is used to render a series of ReflectedVideoSteps. The number of
	// entries in the array will always be television.MaxSignalHistory.
	//
	// The layout of the ref array is roughly equivalent to the sig array sent
	// by PixelRenderer.SetPixels(). That is, the first entry always
	// corresponds to the top-left pixel.
	//
	// It's important that implementations forget about and do not reference
	// any previous arrays as otherwise this will cause race errors with the
	// reflection package (which is probably running in a different thread).
	Reflect(ref []ReflectedVideoStep) error
}

// Broker implementations can identify a reflection.Renderer.
type Broker interface {
	GetReflectionRenderer() Renderer
}

// ReflectedVideoStep packages together the details of the the last video step that
// would otherwise be difficult for a debugger to access.
//
// It includes the CPU execution result, the bank from which the instruction
// originates, the video element that produced the last video pixel on screen;
// among other information.
//
// Note that ordering of the structure is important. There's a saving of about
// 2MB per frame compared to the unoptimal ordering.
type ReflectedVideoStep struct {
	CPU               execution.Result
	Collision         video.Collisions
	Bank              mapper.BankInfo
	Signal            signal.SignalAttributes
	Hmove             Hmove
	VideoElement      video.Element
	WSYNC             bool
	IsRAM             bool
	CoprocessorActive bool
	IsHblank          bool
	RSYNCalign        bool
	RSYNCreset        bool
}

// Hmove groups the HMOVE reflection information. It's too complex a property
// to distil into a single variable.
//
// Ordering of the structure is important in order to keep memory usage at a
// minimum.
type Hmove struct {
	DelayCt  int
	Delay    bool
	Latch    bool
	RippleCt uint8
}
