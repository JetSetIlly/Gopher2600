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

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

// LazyCart lazily accesses cartridge information from the emulator.
type LazyCart struct {
	val *LazyValues

	id       atomic.Value // string
	summary  atomic.Value // string
	filename atomic.Value // string
	numBanks atomic.Value // int
	currBank atomic.Value // int

	staticBus atomic.Value // mapper.CartStaticBus
	static    atomic.Value // mapper.CartStatic

	registersBus atomic.Value // mapper.CartRegistersBus
	registers    atomic.Value // mapper.CartRegisters

	ramBus atomic.Value // mapper.CartRAMbus
	ram    atomic.Value // []mapper.CartRAM

	tapeBus   atomic.Value // mapper.CartTapeBus
	tapeState atomic.Value // mapper.CartTapeState

	plusROM         atomic.Value // plusrom.PlusROM
	plusROMAddrInfo atomic.Value // plusrom.AddrInfo
	plusROMNick     atomic.Value // string (from prefs.String.Get())
	plusROMID       atomic.Value // string (from prefs.String.Get())
	plusROMRecvBuff atomic.Value // []uint8
	plusROMSendBuff atomic.Value // []uint8

	ID       string
	Summary  string
	Filename string
	NumBanks int
	CurrBank mapper.BankInfo

	HasStaticBus bool
	StaticBus    mapper.CartStaticBus
	Static       []mapper.CartStatic

	HasRegistersBus bool
	RegistersBus    mapper.CartRegistersBus
	Registers       mapper.CartRegisters

	HasRAMbus bool
	RAMbus    mapper.CartRAMbus
	RAM       []mapper.CartRAM

	HasTapeBus bool
	TapeBus    mapper.CartTapeBus
	TapeState  mapper.CartTapeState

	IsPlusROM       bool
	PlusROMAddrInfo plusrom.AddrInfo
	PlusROMNick     string
	PlusROMID       string
	PlusROMRecvBuff []uint8
	PlusROMSendBuff []uint8
}

func newLazyCart(val *LazyValues) *LazyCart {
	return &LazyCart{val: val}
}

func (lz *LazyCart) push() {
	lz.id.Store(lz.val.Dbg.VCS.Mem.Cart.ID())
	lz.filename.Store(lz.val.Dbg.VCS.Mem.Cart.Filename)
	lz.summary.Store(lz.val.Dbg.VCS.Mem.Cart.MappingSummary())
	lz.numBanks.Store(lz.val.Dbg.VCS.Mem.Cart.NumBanks())
	lz.currBank.Store(lz.val.Dbg.VCS.Mem.Cart.GetBank(lz.val.Dbg.VCS.CPU.PC.Address()))

	sb := lz.val.Dbg.VCS.Mem.Cart.GetStaticBus()
	if sb != nil {
		lz.staticBus.Store(sb)

		// make sure CartStaticBus implementation is meaningful
		a := sb.GetStatic()
		if a != nil {
			lz.static.Store(a)
		}
	}

	rb := lz.val.Dbg.VCS.Mem.Cart.GetRegistersBus()
	if rb != nil {
		lz.registersBus.Store(rb)

		// make sure CartRegistersBus implementation is meaningful
		a := rb.GetRegisters()
		if a != nil {
			lz.registers.Store(a)
		}
	}

	r := lz.val.Dbg.VCS.Mem.Cart.GetRAMbus()
	if r != nil {
		lz.ramBus.Store(r)

		// make sure CartRAMBus implementation is meaningful
		a := r.GetRAM()
		if a != nil {
			lz.ram.Store(a)
		}
	}

	t := lz.val.Dbg.VCS.Mem.Cart.GetTapeBus()
	if t != nil {
		// make sure CartTapeBus implementation is meaningful
		if ok, s := t.GetTapeState(); ok {
			lz.tapeBus.Store(t)
			lz.tapeState.Store(s)
		}
	}

	c := lz.val.Dbg.VCS.Mem.Cart.GetContainer()
	if c != nil {
		if pr, ok := c.(*plusrom.PlusROM); ok {
			lz.plusROM.Store(pr)
			lz.plusROMAddrInfo.Store(pr.CopyAddrInfo())
			lz.plusROMNick.Store(pr.Prefs.Nick.Get())
			lz.plusROMID.Store(pr.Prefs.ID.Get())
			lz.plusROMRecvBuff.Store(pr.CopyRecvBuffer())
			lz.plusROMSendBuff.Store(pr.CopySendBuffer())
		} else {
			lz.plusROM.Store(nil)
		}
	}
}

func (lz *LazyCart) update() {
	lz.ID, _ = lz.id.Load().(string)
	lz.Summary, _ = lz.summary.Load().(string)
	lz.Filename, _ = lz.filename.Load().(string)
	lz.NumBanks, _ = lz.numBanks.Load().(int)
	lz.CurrBank, _ = lz.currBank.Load().(mapper.BankInfo)

	lz.StaticBus, lz.HasStaticBus = lz.staticBus.Load().(mapper.CartStaticBus)
	if lz.HasStaticBus {
		lz.Static, _ = lz.static.Load().([]mapper.CartStatic)

		// a cartridge can implement a static bus but not actually have a
		// static area. this additional test checks for that
		//
		// required for plusrom cartridges
		if lz.Static == nil {
			lz.HasStaticBus = false
		}
	}

	lz.RegistersBus, lz.HasRegistersBus = lz.registersBus.Load().(mapper.CartRegistersBus)
	if lz.HasRegistersBus {
		lz.Registers, _ = lz.registers.Load().(mapper.CartRegisters)

		// a cartridge can implement a registers bus but not actually have any
		// registers. this additional test checks for that
		//
		// required for plusrom cartridges
		if lz.Registers == nil {
			lz.HasRegistersBus = false
		}
	}

	lz.RAMbus, lz.HasRAMbus = lz.ramBus.Load().(mapper.CartRAMbus)
	if lz.HasRAMbus {
		lz.RAM, _ = lz.ram.Load().([]mapper.CartRAM)

		// a cartridge can implement a ram bus but not actually have any ram.
		// this additional test checks for that
		//
		// required for plusrom catridges and atari cartridges without a superchip
		if lz.RAM == nil {
			lz.HasRAMbus = false
		}
	}

	lz.TapeBus, lz.HasTapeBus = lz.tapeBus.Load().(mapper.CartTapeBus)
	if lz.HasTapeBus {
		lz.TapeState, _ = lz.tapeState.Load().(mapper.CartTapeState)
	}

	_, lz.IsPlusROM = lz.plusROM.Load().(*plusrom.PlusROM)
	if lz.IsPlusROM {
		lz.PlusROMAddrInfo, _ = lz.plusROMAddrInfo.Load().(plusrom.AddrInfo)
		lz.PlusROMNick, _ = lz.plusROMNick.Load().(string)
		lz.PlusROMID, _ = lz.plusROMID.Load().(string)
		lz.PlusROMRecvBuff, _ = lz.plusROMRecvBuff.Load().([]uint8)
		lz.PlusROMSendBuff, _ = lz.plusROMSendBuff.Load().([]uint8)
	}
}
