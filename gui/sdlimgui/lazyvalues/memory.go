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
)

// LazyMem lazily accesses Mem information from the emulator.
type LazyMem struct {
	val *LazyValues

	lastAccessAddress       atomic.Value // uint16
	lastAccessAddressMapped atomic.Value // uint16
	lastAccessData          atomic.Value // uint8
	lastAccessWrite         atomic.Value // bool
	lastAccessMask          atomic.Value // uint8

	LastAccessAddress       uint16
	LastAccessAddressMapped uint16
	LastAccessData          uint8
	LastAccessWrite         bool
	LastAccessMask          uint8
}

func newLazyMem(val *LazyValues) *LazyMem {
	return &LazyMem{val: val}
}

func (lz *LazyMem) push() {
	lz.lastAccessAddress.Store(lz.val.vcs.Mem.LastAccessAddress)
	lz.lastAccessAddressMapped.Store(lz.val.vcs.Mem.LastAccessAddressMapped)
	lz.lastAccessData.Store(lz.val.vcs.Mem.LastAccessData)
	lz.lastAccessWrite.Store(lz.val.vcs.Mem.LastAccessWrite)
	lz.lastAccessMask.Store(lz.val.vcs.Mem.LastAccessMask)
}

func (lz *LazyMem) update() {
	lz.LastAccessAddress, _ = lz.lastAccessAddress.Load().(uint16)
	lz.LastAccessAddressMapped, _ = lz.lastAccessAddressMapped.Load().(uint16)
	lz.LastAccessData, _ = lz.lastAccessData.Load().(uint8)
	lz.LastAccessWrite, _ = lz.lastAccessWrite.Load().(bool)
	lz.LastAccessMask, _ = lz.lastAccessMask.Load().(uint8)
}
