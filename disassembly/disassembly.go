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

package disassembly

import (
	"sync"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/symbols"
)

// Disassembly represents the annotated disassembly of a 6507 binary
type Disassembly struct {
	Prefs *Preferences

	// the cartridge to which the disassembly refers
	cart *cartridge.Cartridge

	// the lowest value to use when formatting address values. changed by the
	// preferences system
	mirrorOrigin uint16

	// symbols used to format disassembly output
	Symtable *symbols.Table

	// indexed by bank and address. address should be masked with memorymap.CartridgeBits before access
	disasm [][]*Entry

	// formatting information for all entries in the disassembly
	fields fields

	// critical sectioning. the iteration functions in particular maybe called
	// from a different goroutine. the entries will likely be updating more or
	// less constantly with UpdateEntry() so it's important we section off
	// access to the disasm array.
	//
	// an alternative and maybe better solution would be to run the disassembly
	// as a service. so UpdateEntry() would be a channel request, iteration
	// would be a channel request, etc.
	//
	// !!TODO: look into turning the disassembly into a channel driven service
	crit sync.Mutex
}

func NewDisassembly() (*Disassembly, error) {
	dsm := &Disassembly{
		mirrorOrigin: memorymap.OriginCart, // default. may be changed by during newPreferences()
	}

	var err error

	dsm.Prefs, err = newPreferences(dsm)
	if err != nil {
		if !curated.Is(err, prefs.NoPrefsFile) {
			return nil, curated.Errorf("disassembly: %v", err)
		}
	}

	return dsm, nil
}

// FromCartridge initialises a new partial emulation and returns a disassembly
// from the supplied cartridge filename. Useful for one-shot disassemblies,
// like the gopher2600 "disasm" mode.
func FromCartridge(cartload cartridgeloader.Loader) (*Disassembly, error) {
	dsm, err := NewDisassembly()
	if err != nil {
		return nil, err
	}

	// ignore errors caused by loading of symbols table - we always get a
	// standard symbols table even in the event of an error
	symtable, _ := symbols.ReadSymbolsFile(cartload.Filename)

	cart := cartridge.NewCartridge()

	err = cart.Attach(cartload)
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	err = dsm.FromMemory(cart, symtable)
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	return dsm, nil
}

// FromMemoryAgain repeats the disassembly using the existing structures
func (dsm *Disassembly) FromMemoryAgain(startAddress ...uint16) error {
	// demote any entry level lower then "executed" to "unused
	dsm.crit.Lock()
	for b := 0; b < len(dsm.disasm); b++ {
		for _, a := range dsm.disasm[b] {
			if a.Level < EntryLevelExecuted {
				a.Level = EntryLevelUnused
			}
		}
	}
	dsm.crit.Unlock()

	// it's important that we don't initiliase the cartridge during the
	// fromMemory() process

	return dsm.fromMemory(startAddress...)
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control. Unlike the FromCartridge() function this function
// requires an existing instance of Disassembly
//
// cartridge will finish in its initialised state
func (dsm *Disassembly) FromMemory(cart *cartridge.Cartridge, symtable *symbols.Table) error {
	dsm.cart = cart

	dsm.Symtable = symtable

	// allocate memory for disassembly. the GUI may find itself trying to
	// iterate through disassembly at the same time as we're doing this.
	dsm.crit.Lock()
	dsm.disasm = make([][]*Entry, dsm.cart.NumBanks())
	for b := 0; b < len(dsm.disasm); b++ {
		dsm.disasm[b] = make([]*Entry, memorymap.CartridgeBits+1)
	}
	dsm.crit.Unlock()

	// exit early if cartridge memory self reports as being ejected
	if dsm.cart.IsEjected() {
		return nil
	}

	// begin and start with the cartridge in its initialised state.
	dsm.cart.Initialise()
	defer dsm.cart.Initialise()

	return dsm.fromMemory()
}

// fromMemory is the underlying function for both FromMemory() and FromMemoryAgain()
func (dsm *Disassembly) fromMemory(startAddress ...uint16) error {
	// create new memory
	mem := &disasmMemory{cart: dsm.cart}

	// create a new NoFlowControl CPU to help disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return curated.Errorf("disassembly: %v", err)
	}
	mc.NoFlowControl = true

	// some cartridge types react when certain registers are read/written. for
	// disassembly purposes we don't want that so we turn on the Passive flag
	// for the duration
	dsm.cart.Passive = true
	defer func() { dsm.cart.Passive = false }()

	// disassemble cartridge binary
	err = dsm.disassemble(mc, mem, startAddress...)
	if err != nil {
		return curated.Errorf("disassembly: %v", err)
	}

	return nil
}

// GetEntryByAddress returns the disassembly entry at the specified
// bank/address. a returned value of nil indicates the entry is not in the
// cartridge; this will usually mean the address is in main VCS RAM
func (dsm *Disassembly) GetEntryByAddress(address uint16) *Entry {
	bank := dsm.cart.GetBank(address)

	if bank.NonCart {
		// !!TODO: attempt to decode instructions not in cartridge
		return nil
	}

	return dsm.disasm[bank.Number][address&memorymap.CartridgeBits]
}

// UpdateEntry to more closely resemble the most recent execution.Result.
//
// If the result is transient (ie. executed from RAM) then nothing is updated
// but a formatted result is returned.
func (dsm *Disassembly) UpdateEntry(bank banks.Details, result execution.Result, nextAddr uint16) (*Entry, error) {
	// not touching any result which is not in cartridge space. we are noting
	// execution results from cartridge RAM. the banks.Details field in the
	// disassembly entry notes whether execution was from RAM
	if bank.NonCart {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	if bank.Number >= len(dsm.disasm) {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	idx := result.Address & memorymap.CartridgeBits

	// get entry at address
	e := dsm.disasm[bank.Number][idx]

	// updating an origin can happen at the same time as iteration which is
	// probably being run from a different goroutine. acknowledge the critical
	// section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	if e == nil || e.Result.Defn.OpCode != result.Defn.OpCode {
		var err error
		dsm.disasm[bank.Number][idx], err = dsm.formatResult(bank, result, EntryLevelExecuted)
		if err != nil {
			return nil, curated.Errorf("disassembly: %v", err)
		}

	} else if e.Level < EntryLevelExecuted || e.UpdateActualOnExecute {
		e.Level = EntryLevelExecuted
		e.Result = result

		// not updating the formatted results. this means that address values
		// will be the same as they were during the disassembly and how they
		// were set by the setCartMirror() function

		e.updateActual()
	}

	// bless next entry in case it was missed by the original decoding. there's
	// no guarantee that the bank for the next address will be the same as the
	// current bank, so we have to call the GetBank() function.
	//
	// !!TODO: maybe make sure next entry has been disassembled in it's current form
	bank = dsm.cart.GetBank(nextAddr)
	ne := dsm.disasm[bank.Number][nextAddr&memorymap.CartridgeBits]
	if ne.Level < EntryLevelBlessed {
		ne.Level = EntryLevelBlessed
	}

	return e, nil
}
