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
	mapping  atomic.Value // string
	filename atomic.Value // string
	numBanks atomic.Value // int
	currBank atomic.Value // int

	staticBus atomic.Value // mapper.CartStaticBus (in container)
	static    atomic.Value // mapper.CartStatic

	registersBus atomic.Value // mapper.CartRegistersBus (in container)
	registers    atomic.Value // mapper.CartRegisters (in container)

	ramBus atomic.Value // mapper.CartRAMbus (in container)
	ram    atomic.Value // []mapper.CartRAM

	tapeBus   atomic.Value // mapper.CartTapeBus (in continer)
	tapeState atomic.Value // mapper.CartTapeState (in container)

	coProcBus atomic.Value // mapper.CartCoProcBus (in container)
	coprocID  atomic.Value // string

	plusROM         atomic.Value // plusrom.PlusROM (in container)
	plusROMAddrInfo atomic.Value // plusrom.AddrInfo
	plusROMNick     atomic.Value // string (from prefs.String.Get())
	plusROMID       atomic.Value // string (from prefs.String.Get())
	plusROMRecvBuff atomic.Value // []uint8
	plusROMSendBuff atomic.Value // []uint8

	ID       string
	Mapping  string
	Filename string
	NumBanks int
	CurrBank mapper.BankInfo

	HasStaticBus bool
	Static       []mapper.CartStatic

	HasRegistersBus bool
	Registers       mapper.CartRegisters

	HasRAMbus bool
	RAM       []mapper.CartRAM

	HasTapeBus bool
	TapeState  mapper.CartTapeState

	HasCoProcBus bool
	CoProcID     string

	IsPlusROM       bool
	PlusROMAddrInfo plusrom.AddrInfo
	PlusROMRecvBuff []uint8
	PlusROMSendBuff []uint8
}

func newLazyCart(val *LazyValues) *LazyCart {
	lz := &LazyCart{val: val}
	lz.staticBus.Store(container{})
	lz.static.Store(container{})
	lz.registersBus.Store(container{})
	lz.registers.Store(container{})
	lz.ramBus.Store(container{})
	lz.ram.Store(container{})
	lz.tapeBus.Store(container{})
	lz.tapeState.Store(container{})
	lz.plusROM.Store(container{})
	lz.coProcBus.Store(container{})
	return lz
}

type container struct {
	v interface{}
}

func (lz *LazyCart) push() {
	lz.id.Store(lz.val.vcs.Mem.Cart.ID())
	lz.filename.Store(lz.val.vcs.Mem.Cart.Filename)
	lz.mapping.Store(lz.val.vcs.Mem.Cart.MappedBanks())
	lz.numBanks.Store(lz.val.vcs.Mem.Cart.NumBanks())
	lz.currBank.Store(lz.val.vcs.Mem.Cart.GetBank(lz.val.vcs.CPU.PC.Address()))

	sb := lz.val.vcs.Mem.Cart.GetStaticBus()
	if sb != nil {
		lz.staticBus.Store(container{v: sb})
		lz.static.Store(container{v: sb.GetStatic()})
	} else {
		lz.staticBus.Store(container{v: nil})
		lz.static.Store(container{v: nil})
	}

	rb := lz.val.vcs.Mem.Cart.GetRegistersBus()
	if rb != nil {
		lz.registersBus.Store(container{v: rb})
		lz.registers.Store(container{v: rb.GetRegisters()})
	} else {
		lz.registersBus.Store(container{v: nil})
		lz.registers.Store(container{v: nil})
	}

	r := lz.val.vcs.Mem.Cart.GetRAMbus()
	if r != nil {
		lz.ramBus.Store(container{v: r})
		lz.ram.Store(container{v: r.GetRAM()})
	} else {
		lz.ramBus.Store(container{v: nil})
		lz.ram.Store(container{v: nil})
	}

	t := lz.val.vcs.Mem.Cart.GetTapeBus()
	if t != nil {
		lz.tapeBus.Store(container{v: t})
		_, state := t.GetTapeState()
		lz.tapeState.Store(container{v: state})
	} else {
		lz.tapeBus.Store(container{v: nil})
		lz.tapeState.Store(container{v: nil})
	}

	c := lz.val.vcs.Mem.Cart.GetContainer()
	if c != nil {
		if pr, ok := c.(*plusrom.PlusROM); ok {
			lz.plusROM.Store(container{v: pr})
			lz.plusROMAddrInfo.Store(pr.CopyAddrInfo())
			lz.plusROMRecvBuff.Store(pr.CopyRecvBuffer())
			lz.plusROMSendBuff.Store(pr.CopySendBuffer())
		} else {
			lz.plusROM.Store(nil)
		}
	} else {
		lz.plusROM.Store(container{v: nil})
	}

	cp := lz.val.vcs.Mem.Cart.GetCoProcBus()
	if cp != nil {
		lz.coProcBus.Store(container{v: cp})
		lz.coprocID.Store(cp.CoProcID())
	} else {
		lz.coProcBus.Store(container{v: nil})
	}
}

func (lz *LazyCart) update() {
	lz.ID, _ = lz.id.Load().(string)
	lz.Mapping, _ = lz.mapping.Load().(string)
	lz.Filename, _ = lz.filename.Load().(string)
	lz.NumBanks, _ = lz.numBanks.Load().(int)
	lz.CurrBank, _ = lz.currBank.Load().(mapper.BankInfo)

	_, lz.HasStaticBus = lz.staticBus.Load().(container).v.(mapper.CartStaticBus)
	if lz.HasStaticBus {
		// a cartridge can implement a static bus but not actually have a
		// static area. this additional test checks for that
		//
		// required for plusrom cartridges
		lz.Static = lz.static.Load().(container).v.([]mapper.CartStatic)
		lz.HasStaticBus = lz.Static != nil
	}

	_, lz.HasRegistersBus = lz.registersBus.Load().(container).v.(mapper.CartRegistersBus)
	if lz.HasRegistersBus {
		// a cartridge can implement a registers bus but not actually have any
		// registers. this additional test checks for that
		//
		// required for plusrom cartridges
		if r, ok := lz.registers.Load().(container).v.(mapper.CartRegisters); ok {
			lz.Registers = r
		} else {
			lz.HasRegistersBus = lz.Registers != nil
		}
	}

	_, lz.HasRAMbus = lz.ramBus.Load().(container).v.(mapper.CartRAMbus)
	if lz.HasRAMbus {
		// a cartridge can implement a ram bus but not actually have any ram.
		// this additional test checks for that
		//
		// required for plusrom catridges and atari cartridges without a superchip
		lz.RAM = lz.ram.Load().(container).v.([]mapper.CartRAM)
		lz.HasRAMbus = lz.RAM != nil
	}

	_, lz.HasTapeBus = lz.tapeBus.Load().(container).v.(mapper.CartTapeBus)
	if lz.HasTapeBus {
		lz.TapeState, _ = lz.tapeState.Load().(container).v.(mapper.CartTapeState)
	}

	_, lz.IsPlusROM = lz.plusROM.Load().(container).v.(*plusrom.PlusROM)
	if lz.IsPlusROM {
		lz.PlusROMAddrInfo, _ = lz.plusROMAddrInfo.Load().(plusrom.AddrInfo)
		lz.PlusROMRecvBuff, _ = lz.plusROMRecvBuff.Load().([]uint8)
		lz.PlusROMSendBuff, _ = lz.plusROMSendBuff.Load().([]uint8)
	}

	_, lz.HasCoProcBus = lz.coProcBus.Load().(container).v.(mapper.CartCoProcBus)
	if lz.HasCoProcBus {
		lz.CoProcID, _ = lz.coprocID.Load().(string)
	}
}
