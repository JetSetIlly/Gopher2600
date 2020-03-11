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
	"gopher2600/hardware/memory/cartridge"
	"sync/atomic"
)

// LazyCart lazily accesses cartridge information from the emulator.
type LazyCart struct {
	val *Values

	atomicString   atomic.Value // string
	atomicNumBanks atomic.Value // int
	atomicRAMinfo  atomic.Value // []cartridge.RAMinfo

	String   string
	NumBanks int
	RAMinfo  []cartridge.RAMinfo

	// ramInfoRef is used to detect if a new allocation is required
	ramInfoRef *[]cartridge.RAMinfo

	// getBank from cartridge requires an address. used GetBank() function
	atomicGetBank atomic.Value // int
}

func newLazyCart(val *Values) *LazyCart {
	return &LazyCart{val: val}
}

func (lz *LazyCart) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicString.Store(lz.val.VCS.Mem.Cart.String())
		lz.atomicNumBanks.Store(lz.val.VCS.Mem.Cart.NumBanks)
	})
	lz.String, _ = lz.atomicString.Load().(string)
	lz.NumBanks, _ = lz.atomicNumBanks.Load().(int)

	// CartRAMinfo() returns a slice so we need to copy each
	lz.val.Dbg.PushRawEvent(func() {
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
	lz.RAMinfo, _ = lz.atomicRAMinfo.Load().([]cartridge.RAMinfo)
}

// GetBank returns the cartridge bank associated with the address
func (lz *LazyCart) GetBank(pcAddr uint16) int {
	if lz.val.Dbg == nil {
		return 0
	}

	lz.val.Dbg.PushRawEvent(func() { lz.atomicGetBank.Store(lz.val.VCS.Mem.Cart.GetBank(pcAddr)) })
	c, _ := lz.atomicGetBank.Load().(int)
	return c
}
