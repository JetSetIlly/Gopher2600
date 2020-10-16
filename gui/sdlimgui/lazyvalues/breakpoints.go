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

package lazyvalues

import (
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/debugger"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type LazyBreakpoints struct {
	val *LazyValues

	// breakpoints are treated differently to other lazy values. the
	// information is updated on every call to HasBreak() rather than via
	// push() and update()
	breakpoints []atomic.Value // debugger.BreakGroup
}

func newLazyBreakpoints(val *LazyValues) *LazyBreakpoints {
	return &LazyBreakpoints{
		val: val,

		// allocating enough space for every byte in the cartridge space. not worrying
		// about bank sizes or multiple banks
		breakpoints: make([]atomic.Value, memorymap.MemtopCart-memorymap.OriginCart+1),
	}
}

// HasBreak checks to see if disassembly entry has a breakpoint.
func (lz *LazyBreakpoints) HasBreak(e *disassembly.Entry) debugger.BreakGroup {
	i := e.Result.Address & memorymap.CartridgeBits

	lz.val.Dbg.PushRawEvent(func() {
		lz.breakpoints[i].Store(lz.val.Dbg.HasBreak(e))
	})

	if b, ok := lz.breakpoints[i].Load().(debugger.BreakGroup); ok {
		return b
	}

	return debugger.BrkNone
}
