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

	addressBus              atomic.Value // uint16
	lastAccessAddressMapped atomic.Value // uint16
	lastAccessData          atomic.Value // uint8
	lastAccessWrite         atomic.Value // bool
	lastAccessMask          atomic.Value // uint8

	AddressBus      uint16
	DataBus         uint8
	LastAccessWrite bool
	LastAccessMask  uint8
}

func newLazyMem(val *LazyValues) *LazyMem {
	return &LazyMem{val: val}
}

func (lz *LazyMem) push() {
	lz.addressBus.Store(lz.val.vcs.Mem.AddressBus)
	lz.lastAccessData.Store(lz.val.vcs.Mem.DataBus)
	lz.lastAccessWrite.Store(lz.val.vcs.Mem.LastCPUWrite)
	lz.lastAccessMask.Store(lz.val.vcs.Mem.DataBusDriven)
}

func (lz *LazyMem) update() {
	lz.AddressBus, _ = lz.addressBus.Load().(uint16)
	lz.DataBus, _ = lz.lastAccessData.Load().(uint8)
	lz.LastAccessWrite, _ = lz.lastAccessWrite.Load().(bool)
	lz.LastAccessMask, _ = lz.lastAccessMask.Load().(uint8)
}
