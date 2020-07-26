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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/television"
)

// Renderer implementations accepts ReflectPixel values and associates it in
// some way with the moste recent television signal
type Renderer interface {
	Reflect(LastResult) error
}

// Broker implementations can identify a reflection.Renderer
type Broker interface {
	GetReflectionRenderer() Renderer
}

// LastResult packages together the details of the the last video step. It
// includes the CPU execution result, the bank from which the instruction
// originate, the video element that produced the last video pixel on
// screen; and the raw television signal most recently sent (a PixelRenderer
// only recieves distilled information from the television implementation so
// this is not redundant information).
type LastResult struct {
	CPU          execution.Result
	WSYNC        bool
	Bank         banks.Details
	IsRam        bool
	VideoElement video.Element
	TV           television.SignalAttributes
	Hblank       bool
	Collision    string
	Hmove        Hmove
	Unchanged    bool
}

// Hmove groups the HMOVE reflection information. It's too complex a property
// to distill into a single variable.
type Hmove struct {
	Delay    bool
	DelayCt  int
	Latch    bool
	RippleCt uint8
}

// OverlayList is the list of overlays that should be supported by a
// reflection.Renderer.
var OverlayList = []string{"WSYNC", "Collisions", "HMOVE", "Unchanged"}
