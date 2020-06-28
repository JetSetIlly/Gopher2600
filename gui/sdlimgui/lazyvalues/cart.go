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

	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
)

// LazyCart lazily accesses cartridge information from the emulator
type LazyCart struct {
	val *Lazy

	atomicID       atomic.Value // string
	atomicSummary  atomic.Value // string
	atomicFilename atomic.Value // string
	atomicNumBanks atomic.Value // int
	atomicCurrBank atomic.Value // int

	atomicDebugBus  atomic.Value // bus.CartDebugBus
	atomicStatic    atomic.Value
	atomicRegisters atomic.Value // bus.CartRegisters

	atomicRAMbus atomic.Value // bus.CartRAMbus
	atomicRAM    atomic.Value // []bus.CartRAM

	ID       string
	Summary  string
	Filename string
	NumBanks int
	CurrBank banks.Details

	HasDebugBus bool
	DebugBus    bus.CartDebugBus
	Static      bus.CartStatic
	Registers   bus.CartRegisters

	HasRAMbus bool
	RAMbus    bus.CartRAMbus
	RAM       []bus.CartRAM
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
		lz.atomicFilename.Store(lz.val.Dbg.VCS.Mem.Cart.Filename)
		lz.atomicSummary.Store(lz.val.Dbg.VCS.Mem.Cart.MappingSummary())
		lz.atomicNumBanks.Store(lz.val.Dbg.VCS.Mem.Cart.NumBanks())
		lz.atomicCurrBank.Store(lz.val.Dbg.VCS.Mem.Cart.GetBank(PCaddr))

		b := lz.val.Dbg.VCS.Mem.Cart.GetDebugBus()
		if b != nil {
			lz.atomicDebugBus.Store(b)
			lz.atomicStatic.Store(b.GetStatic())
			lz.atomicRegisters.Store(b.GetRegisters())
		}

		r := lz.val.Dbg.VCS.Mem.Cart.GetRAMbus()
		if r != nil {
			lz.atomicRAMbus.Store(r)
			lz.atomicRAM.Store(r.GetRAM())
		}
	})

	lz.ID, _ = lz.atomicID.Load().(string)
	lz.Summary, _ = lz.atomicSummary.Load().(string)
	lz.Filename, _ = lz.atomicFilename.Load().(string)
	lz.NumBanks, _ = lz.atomicNumBanks.Load().(int)
	lz.CurrBank, _ = lz.atomicCurrBank.Load().(banks.Details)

	lz.DebugBus, lz.HasDebugBus = lz.atomicDebugBus.Load().(bus.CartDebugBus)
	if lz.HasDebugBus {
		lz.Static, _ = lz.atomicStatic.Load().(bus.CartStatic)
		lz.Registers, _ = lz.atomicRegisters.Load().(bus.CartRegisters)
	}

	lz.RAMbus, lz.HasRAMbus = lz.atomicRAMbus.Load().(bus.CartRAMbus)
	if lz.HasRAMbus {
		lz.RAM, _ = lz.atomicRAM.Load().([]bus.CartRAM)

		// as explained in the commentary for the CartRAMbus interface, a
		// cartridge my implement the interface but not actually have any RAM.
		// we check for this here and correct the HasRAMbus boolean accordingly
		if lz.RAM == nil {
			lz.HasRAMbus = false
		}
	}
}
