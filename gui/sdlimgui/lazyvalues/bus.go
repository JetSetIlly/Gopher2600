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

// LazyBus lazily accesses Mem information from the emulator.
type LazyBus struct {
	val *LazyValues

	addressBus    atomic.Value // uint16
	dataBus       atomic.Value // uint8
	lastCPUWrite  atomic.Value // bool
	dataBusDriven atomic.Value // uint8

	AddressBus    uint16
	DataBus       uint8
	LastCPUWrite  bool
	DataBusDriven uint8
}

func newLazyBus(val *LazyValues) *LazyBus {
	return &LazyBus{val: val}
}

func (lz *LazyBus) push() {
	lz.addressBus.Store(lz.val.vcs.Mem.AddressBus)
	lz.dataBus.Store(lz.val.vcs.Mem.DataBus)
	lz.lastCPUWrite.Store(lz.val.vcs.Mem.LastCPUWrite)
	lz.dataBusDriven.Store(lz.val.vcs.Mem.DataBusDriven)
}

func (lz *LazyBus) update() {
	lz.AddressBus, _ = lz.addressBus.Load().(uint16)
	lz.DataBus, _ = lz.dataBus.Load().(uint8)
	lz.LastCPUWrite, _ = lz.lastCPUWrite.Load().(bool)
	lz.DataBusDriven, _ = lz.dataBusDriven.Load().(uint8)
}
