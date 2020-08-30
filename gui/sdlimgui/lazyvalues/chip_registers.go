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

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

// LazyChipRegisters lazily accesses chip registere information from the emulator
type LazyChipRegisters struct {
	val *Lazy

	atomicSWACHA atomic.Value // uint8
	atomicSWACHB atomic.Value // uint8
	atomicSWACNT atomic.Value // uint8
	atomicSWBCNT atomic.Value // uint8
	atomicINPT0  atomic.Value // uint8
	atomicINPT1  atomic.Value // uint8
	atomicINPT2  atomic.Value // uint8
	atomicINPT3  atomic.Value // uint8
	atomicINPT4  atomic.Value // uint8
	atomicINPT5  atomic.Value // uint8
	SWACHA       uint8
	SWACNT       uint8
	SWACHB       uint8
	SWBCNT       uint8
	INPT0        uint8
	INPT1        uint8
	INPT2        uint8
	INPT3        uint8
	INPT4        uint8
	INPT5        uint8
}

func newLazyChipRegisters(val *Lazy) *LazyChipRegisters {
	return &LazyChipRegisters{val: val}
}

func (lz *LazyChipRegisters) update() {
	lz.val.Dbg.PushRawEvent(func() {
		v, _ := lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["SWCHA"])
		lz.atomicSWACHA.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["SWACNT"])
		lz.atomicSWACNT.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["SWCHB"])
		lz.atomicSWACHB.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["SWBCNT"])
		lz.atomicSWBCNT.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["INPT0"])
		lz.atomicINPT0.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["INPT1"])
		lz.atomicINPT1.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["INPT2"])
		lz.atomicINPT2.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["INPT3"])
		lz.atomicINPT3.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["INPT4"])
		lz.atomicINPT4.Store(v)
		v, _ = lz.val.Dbg.VCS.Mem.Peek(addresses.ReadAddress["INPT5"])
		lz.atomicINPT5.Store(v)
	})
	lz.SWACHA, _ = lz.atomicSWACHA.Load().(uint8)
	lz.SWACNT, _ = lz.atomicSWACNT.Load().(uint8)
	lz.SWACHB, _ = lz.atomicSWACHB.Load().(uint8)
	lz.SWBCNT, _ = lz.atomicSWBCNT.Load().(uint8)
	lz.INPT0, _ = lz.atomicINPT0.Load().(uint8)
	lz.INPT1, _ = lz.atomicINPT1.Load().(uint8)
	lz.INPT2, _ = lz.atomicINPT2.Load().(uint8)
	lz.INPT3, _ = lz.atomicINPT3.Load().(uint8)
	lz.INPT4, _ = lz.atomicINPT4.Load().(uint8)
	lz.INPT5, _ = lz.atomicINPT5.Load().(uint8)
}
