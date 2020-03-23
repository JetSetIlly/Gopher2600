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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package lazyvalues

import (
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
)

// LazyCart lazily accesses cartridge information from the emulator.
type LazyCart struct {
	val *Values

	atomicString   atomic.Value // string
	atomicNumBanks atomic.Value // int
	atomicCurrBank atomic.Value // int
	atomicRAMinfo  atomic.Value // []cartridge.RAMinfo

	String   string
	NumBanks int
	CurrBank int
	RAMinfo  []cartridge.RAMinfo

	// ramInfoRef is used to detect if a new allocation is required
	ramInfoRef *[]cartridge.RAMinfo
}

func newLazyCart(val *Values) *LazyCart {
	return &LazyCart{val: val}
}

func (lz *LazyCart) update() {
	// make a copy of CPU.PCaddr because we will be reading it in a different
	// goroutine (in the PushRawEvent() below) to the one in which it is
	// written (it is written to in the current thread in the LazyCPU.update()
	// function)
	PCaddr := lz.val.CPU.PCaddr

	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicString.Store(lz.val.VCS.Mem.Cart.String())
		lz.atomicNumBanks.Store(lz.val.VCS.Mem.Cart.NumBanks())

		// uses lazy PCaddr value from lazyvalues.CPU
		lz.atomicCurrBank.Store(lz.val.VCS.Mem.Cart.GetBank(PCaddr))

		// CartRAMinfo() returns a slice so we need to copy each
		n := lz.val.VCS.Mem.Cart.GetRAMinfo()
		if lz.ramInfoRef == nil || lz.ramInfoRef != &n {
			lz.ramInfoRef = &n
			m := make([]cartridge.RAMinfo, len(n))
			for i := range n {
				m[i] = n[i]
			}
			lz.atomicRAMinfo.Store(m)
		}
	})
	lz.String, _ = lz.atomicString.Load().(string)
	lz.NumBanks, _ = lz.atomicNumBanks.Load().(int)
	lz.CurrBank, _ = lz.atomicCurrBank.Load().(int)
	lz.RAMinfo, _ = lz.atomicRAMinfo.Load().([]cartridge.RAMinfo)
}
