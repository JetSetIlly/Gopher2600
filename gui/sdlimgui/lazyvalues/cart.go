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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// LazyCart lazily accesses cartridge information from the emulator
type LazyCart struct {
	val *Lazy

	atomicID         atomic.Value // string
	atomicSummary    atomic.Value // string
	atomicFilename   atomic.Value // string
	atomicNumBanks   atomic.Value // int
	atomicCurrBank   atomic.Value // int
	atomicRAMdetails atomic.Value // []memorymap.SubArea

	atomicStaticArea        atomic.Value // cartridge.StaticArea
	atomicStaticAreaPresent atomic.Value // bool

	ID         string
	Summary    string
	Filename   string
	NumBanks   int
	CurrBank   int
	RAMdetails []memorymap.SubArea

	// StaticArea is an interface to the cartridge mapper. interface functions
	// need to be called through PushRawEvent()
	StaticArea        cartridge.StaticArea
	StaticAreaPresent bool
}

func newLazyCart(val *Lazy) *LazyCart {
	return &LazyCart{val: val}
}

func (lz *LazyCart) update() {
	// make a copy of CPU.PCaddr because we will be reading it in a different
	// goroutine (in the PushRawEvent() below) to the one in which it is
	// written (it is written to in the current thread in the LazyCPU.update()
	// function)
	PCaddr := lz.val.CPU.PCaddr

	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicID.Store(lz.val.Dbg.VCS.Mem.Cart.ID())
		lz.atomicSummary.Store(lz.val.Dbg.VCS.Mem.Cart.String())
		lz.atomicFilename.Store(lz.val.Dbg.VCS.Mem.Cart.Filename)
		lz.atomicNumBanks.Store(lz.val.Dbg.VCS.Mem.Cart.NumBanks())
		lz.atomicRAMdetails.Store(lz.val.Dbg.VCS.Mem.Cart.GetRAM())
		lz.atomicCurrBank.Store(lz.val.Dbg.VCS.Mem.Cart.GetBank(PCaddr))

		sa := lz.val.Dbg.VCS.Mem.Cart.GetStaticArea()
		if sa != nil {
			lz.atomicStaticAreaPresent.Store(true)
			lz.atomicStaticArea.Store(sa)
		} else {
			lz.atomicStaticAreaPresent.Store(false)
		}
	})

	lz.ID, _ = lz.atomicID.Load().(string)
	lz.Summary, _ = lz.atomicSummary.Load().(string)
	lz.Filename, _ = lz.atomicFilename.Load().(string)
	lz.NumBanks, _ = lz.atomicNumBanks.Load().(int)
	lz.CurrBank, _ = lz.atomicCurrBank.Load().(int)
	lz.RAMdetails, _ = lz.atomicRAMdetails.Load().([]memorymap.SubArea)

	lz.StaticAreaPresent, _ = lz.atomicStaticAreaPresent.Load().(bool)
	if lz.StaticAreaPresent {
		lz.StaticArea, _ = lz.atomicStaticArea.Load().(cartridge.StaticArea)
	}
}
