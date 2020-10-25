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

	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// LazyRAM lazily accesses the RAM area of VCS memory.
type LazyRAM struct {
	val *LazyValues

	ram atomic.Value // []atomic.Value -> uint8
	RAM []uint8
}

func newLazyRAM(val *LazyValues) *LazyRAM {
	lz := &LazyRAM{
		val: val,
		RAM: make([]uint8, memorymap.MemtopRAM-memorymap.OriginRAM+1),
	}
	lz.ram.Store(make([]atomic.Value, memorymap.MemtopRAM-memorymap.OriginRAM+1))
	return lz
}

func (lz *LazyRAM) push() {
	ram := lz.ram.Load().([]atomic.Value)
	for i := range ram {
		ram[i].Store(lz.val.Dbg.VCS.Mem.RAM.RAM[i])
	}
	lz.ram.Store(ram)
}

func (lz *LazyRAM) update() {
	if ram, ok := lz.ram.Load().([]atomic.Value); ok {
		for i := range ram {
			if v, ok := ram[i].Load().(uint8); ok {
				lz.RAM[i] = v
			}
		}
	}
}
