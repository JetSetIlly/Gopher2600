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

	atomicStaticBus atomic.Value // bus.CartStaticBus
	atomicStatic    atomic.Value // bus.CartStatic

	atomicRegistersBus atomic.Value // bus.CartRegistersBus
	atomicRegisters    atomic.Value // bus.CartRegisters

	atomicRAMbus atomic.Value // bus.CartRAMbus
	atomicRAM    atomic.Value // []bus.CartRAM

	atomicTapeBus   atomic.Value // bus.CartTapeBus
	atomicTapeState atomic.Value // bus.CartTapeState

	ID       string
	Summary  string
	Filename string
	NumBanks int
	CurrBank banks.Details

	HasStaticBus bool
	StaticBus    bus.CartStaticBus
	Static       []bus.CartStatic

	HasRegistersBus bool
	RegistersBus    bus.CartRegistersBus
	Registers       bus.CartRegisters

	HasRAMbus bool
	RAMbus    bus.CartRAMbus
	RAM       []bus.CartRAM

	HasTapeBus bool
	TapeBus    bus.CartTapeBus
	TapeState  bus.CartTapeState
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

		sb := lz.val.Dbg.VCS.Mem.Cart.GetStaticBus()
		if sb != nil {
			lz.atomicStaticBus.Store(sb)
			lz.atomicStatic.Store(sb.GetStatic())
		}

		rb := lz.val.Dbg.VCS.Mem.Cart.GetRegistersBus()
		if rb != nil {
			lz.atomicRegistersBus.Store(rb)
			lz.atomicRegisters.Store(rb.GetRegisters())
		}

		r := lz.val.Dbg.VCS.Mem.Cart.GetRAMbus()
		if r != nil {
			lz.atomicRAMbus.Store(r)
			lz.atomicRAM.Store(r.GetRAM())
		}

		t := lz.val.Dbg.VCS.Mem.Cart.GetTapeBus()
		if t != nil {
			// additional check to see if the tape bus is valid. check boolean
			// result of GetTapeState()
			if ok, s := t.GetTapeState(); ok {
				lz.atomicTapeBus.Store(t)
				lz.atomicTapeState.Store(s)
			}
		}
	})

	lz.ID, _ = lz.atomicID.Load().(string)
	lz.Summary, _ = lz.atomicSummary.Load().(string)
	lz.Filename, _ = lz.atomicFilename.Load().(string)
	lz.NumBanks, _ = lz.atomicNumBanks.Load().(int)
	lz.CurrBank, _ = lz.atomicCurrBank.Load().(banks.Details)

	lz.StaticBus, lz.HasStaticBus = lz.atomicStaticBus.Load().(bus.CartStaticBus)
	if lz.HasStaticBus {
		lz.Static, _ = lz.atomicStatic.Load().([]bus.CartStatic)
	}

	lz.RegistersBus, lz.HasRegistersBus = lz.atomicRegistersBus.Load().(bus.CartRegistersBus)
	if lz.HasRegistersBus {
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

	lz.TapeBus, lz.HasTapeBus = lz.atomicTapeBus.Load().(bus.CartTapeBus)
	if lz.HasTapeBus {
		lz.TapeState, _ = lz.atomicTapeState.Load().(bus.CartTapeState)
	}
}
