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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
)

// LazyCart lazily accesses cartridge information from the emulator.
type LazyCart struct {
	val *LazyValues

	id        atomic.Value // string
	mapping   atomic.Value // string
	filename  atomic.Value // string
	shortname atomic.Value // string
	numBanks  atomic.Value // int
	currBank  atomic.Value // int

	staticBus atomic.Value // mapper.CartStaticBus (in container)
	static    atomic.Value // mapper.CartStatic

	registersBus atomic.Value // mapper.CartRegistersBus (in container)
	registers    atomic.Value // mapper.CartRegisters (in container)

	ramBus atomic.Value // mapper.CartRAMbus (in container)
	ram    atomic.Value // []mapper.CartRAM

	tapeBus   atomic.Value // mapper.CartTapeBus (in continer)
	tapeState atomic.Value // mapper.CartTapeState (in container)

	coProcBus atomic.Value // coprocessor.CartCoProc (in container)
	coprocID  atomic.Value // string
	coprocPC  atomic.Value // uint32

	plusROM          atomic.Value // plusrom.PlusROM (in container)
	plusROMAddrInfo  atomic.Value // plusrom.AddrInfo
	plusROMSendState atomic.Value // plusrom.SendState

	ID        string
	Mapping   string
	Filename  string
	Shortname string
	NumBanks  int
	CurrBank  mapper.BankInfo

	HasStaticBus bool
	StaticBus    mapper.CartStaticBus
	Static       mapper.CartStatic

	HasRegistersBus bool
	Registers       mapper.CartRegisters

	HasRAMbus bool
	RAM       []mapper.CartRAM

	HasTapeBus bool
	TapeState  mapper.CartTapeState

	HasCoProcBus bool
	CoProcID     string
	CoProcPC     uint32

	IsPlusROM        bool
	PlusROMAddrInfo  plusrom.AddrInfo
	PlusROMSendState plusrom.SendState
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
	lz.shortname.Store(lz.val.vcs.Mem.Cart.ShortName)
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
			lz.plusROMSendState.Store(pr.GetSendState())
		} else {
			lz.plusROM.Store(nil)
		}
	} else {
		lz.plusROM.Store(container{v: nil})
	}

	bus := lz.val.vcs.Mem.Cart.GetCoProcBus()
	if bus != nil {
		lz.coProcBus.Store(container{v: bus})
		lz.coprocID.Store(bus.GetCoProc().ProcessorID())
		pc, _ := bus.GetCoProc().Register(15)
		lz.coprocPC.Store(pc)
	} else {
		lz.coProcBus.Store(container{v: nil})
	}
}

func (lz *LazyCart) update() {
	lz.ID, _ = lz.id.Load().(string)
	lz.Mapping, _ = lz.mapping.Load().(string)
	lz.Filename, _ = lz.filename.Load().(string)
	lz.Shortname, _ = lz.shortname.Load().(string)
	lz.NumBanks, _ = lz.numBanks.Load().(int)
	lz.CurrBank, _ = lz.currBank.Load().(mapper.BankInfo)

	lz.StaticBus, lz.HasStaticBus = lz.staticBus.Load().(container).v.(mapper.CartStaticBus)
	if lz.HasStaticBus {
		// a cartridge can implement a static bus but not actually have a
		// static area. this additional test checks for that
		//
		// required for plusrom cartridges
		if lz.static.Load().(container).v != nil {
			lz.HasStaticBus = true
			lz.Static = lz.static.Load().(container).v.(mapper.CartStatic)
		} else {
			lz.HasStaticBus = false
			lz.Static = nil
		}
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

	// additional test to makes sure the cartridge has a tape that we can use.
	// without this test the GUI will show a tape menu entry and a tape window
	// that will crash because there is no data to draw
	lz.HasTapeBus = lz.HasTapeBus && len(lz.TapeState.Data) > 0

	_, lz.IsPlusROM = lz.plusROM.Load().(container).v.(*plusrom.PlusROM)
	if lz.IsPlusROM {
		lz.PlusROMAddrInfo, _ = lz.plusROMAddrInfo.Load().(plusrom.AddrInfo)
		lz.PlusROMSendState, _ = lz.plusROMSendState.Load().(plusrom.SendState)
	}

	_, lz.HasCoProcBus = lz.coProcBus.Load().(container).v.(coprocessor.CartCoProcBus)
	if lz.HasCoProcBus {
		lz.CoProcID, _ = lz.coprocID.Load().(string)
		lz.CoProcPC, _ = lz.coprocPC.Load().(uint32)
	}
}

func (lz *LazyCart) fastPush() {
	bus := lz.val.vcs.Mem.Cart.GetCoProcBus()
	if bus != nil {
		lz.coProcBus.Store(container{v: bus})
		lz.coprocID.Store(bus.GetCoProc().ProcessorID())
	} else {
		lz.coProcBus.Store(container{v: nil})
	}
}

func (lz *LazyCart) fastUpdate() {
	_, lz.HasCoProcBus = lz.coProcBus.Load().(container).v.(coprocessor.CartCoProcBus)
	if lz.HasCoProcBus {
		lz.CoProcID, _ = lz.coprocID.Load().(string)
	}
}
