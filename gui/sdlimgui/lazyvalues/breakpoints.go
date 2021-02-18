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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

type LazyBreakpoints struct {
	val *LazyValues

	updateForBank atomic.Value // int
	updateStart   atomic.Value // uint16
	updateEnd     atomic.Value // uint16

	// breakpoints are treated differently to other lazy values. the
	// information is updated on every call to HasBreak() rather than via
	// push() and update()
	breakpoints []atomic.Value // debugger.BreakGroup
}

func newLazyBreakpoints(val *LazyValues) *LazyBreakpoints {
	lz := &LazyBreakpoints{
		val: val,

		// allocating enough space for every byte in the cartridge space. not worrying
		// about bank sizes or multiple banks
		breakpoints: make([]atomic.Value, memorymap.MemtopCart-memorymap.OriginCart+1),
	}

	lz.updateForBank.Store(0)
	lz.updateStart.Store(uint16(0))
	lz.updateEnd.Store(uint16(0))

	return lz
}

func (lz *LazyBreakpoints) push() {
	// ask debugger about breakpoints on the bank/addresses specified by SetUpdateList()
	b := lz.updateForBank.Load().(int)
	s := lz.updateStart.Load().(uint16)
	e := lz.updateEnd.Load().(uint16)
	for i := s; i <= e; i++ {
		// index is counting just the cartridge bits but HasBreak() expects a
		// real cartridge address. add the OriginCart bits to the index
		e := lz.val.Dbg.HasBreak(i|memorymap.OriginCart, b)
		lz.breakpoints[i].Store(e)
	}
}

func (lz *LazyBreakpoints) update() {
}

// SetUpdateList defines the range of addresses that will be checked by the
// HasBreak() function. The actual checking is performed during the normal
// lazyvalues.Refresh() process and defining a narrow a list as possible speeds
// up the process considerably.
func (lz *LazyBreakpoints) SetUpdateList(bank int, start uint16, end uint16) {
	lz.updateForBank.Store(bank)

	// storing just the cartridge bits and not the bits that indicate the
	// cartridge mirror. this is because it's possible for start value to
	// numerically come after the end value, which means the for() loop above
	// will fail early.
	lz.updateStart.Store(start & memorymap.CartridgeBits)
	lz.updateEnd.Store(end & memorymap.CartridgeBits)
}

// HasBreak checks to see if disassembly entry has a breakpoint.
func (lz *LazyBreakpoints) HasBreak(addr uint16) debugger.BreakGroup {
	i := addr & memorymap.CartridgeBits

	if b, ok := lz.breakpoints[i].Load().(debugger.BreakGroup); ok {
		return b
	}

	return debugger.BrkNone
}
