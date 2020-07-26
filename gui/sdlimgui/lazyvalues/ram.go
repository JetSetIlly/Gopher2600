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

// LazyRAM lazily accesses the RAM area of VCS memory
type LazyRAM struct {
	val *Lazy

	atomicRAM []atomic.Value // []uint8
}

func newLazyRAM(val *Lazy) *LazyRAM {
	return &LazyRAM{
		val:       val,
		atomicRAM: make([]atomic.Value, memorymap.MemtopRAM-memorymap.OriginRAM+1),
	}
}

func (lz *LazyRAM) update() {
	// does not update
}

// Read returns the data at read address
func (lz *LazyRAM) Read(addr uint16) uint8 {
	if !lz.val.active.Load().(bool) || lz.val.Dbg == nil {
		return 0
	}

	lz.val.Dbg.PushRawEvent(func() {
		d, _ := lz.val.Dbg.VCS.Mem.Peek(addr)
		lz.atomicRAM[addr^memorymap.OriginRAM].Store(d)
	})

	d, _ := lz.atomicRAM[addr^memorymap.OriginRAM].Load().(uint8)

	return d
}
