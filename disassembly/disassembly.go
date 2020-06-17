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

package disassembly

import (
	"sync"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/symbols"
)

// Disassembly represents the annotated disassembly of a 6507 binary
type Disassembly struct {
	cart *cartridge.Cartridge

	// symbols used to format disassembly output
	Symtable *symbols.Table

	// indexed by address. address should be masked with
	// memorymap.AddressMaskCart before access.
	reference [][memorymap.AddressMaskCart + 1]*Entry

	// the number of each type of entry. we use this to help prepare
	// disassembly iterations
	counts []map[EntryLevel]int

	// formatting information for all entries found during the flow pass.
	// excluding entries only found during the linear pass because
	// false-positive entries might upset the formatting.
	fields fields

	// critical sectioning
	crit sync.Mutex
}

// GetEntryByAddress returns the disassembly entry at the specified bank/address.
func (dsm *Disassembly) GetEntryByAddress(bank int, address uint16) (*Entry, bool) {
	col := dsm.reference[bank][address&memorymap.AddressMaskCart]
	return col, col != nil
}

// UpdateEntry to more closely resemble the most recent execution.Result
func (dsm *Disassembly) UpdateEntry(bank int, result execution.Result) error {
	if bank >= len(dsm.reference) {
		return nil
	}

	idx := result.Address & memorymap.AddressMaskCart

	// get entry at address
	e := dsm.reference[bank][idx]

	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	if e == nil || e.Result.Defn.OpCode != result.Defn.OpCode {
		var err error
		dsm.reference[bank][idx], err = dsm.FormatResult(bank, result, EntryLevelExecuted)
		if err != nil {
			return errors.New(errors.DisasmError, err)
		}

	} else if e.Level < EntryLevelExecuted || e.UpdateActualOnExecute {
		dsm.counts[bank][e.Level]--
		e.Level = EntryLevelExecuted
		e.Result = result
		e.updateActual()
		dsm.counts[bank][e.Level]++
	}

	return nil
}

// FromCartridge initialises a new partial emulation and returns a disassembly
// from the supplied cartridge filename. Useful for one-shot disassemblies,
// like the gopher2600 "disasm" mode.
func FromCartridge(cartload cartridgeloader.Loader) (*Disassembly, error) {
	dsm := &Disassembly{}

	// ignore errors caused by loading of symbols table - we always get a
	// standard symbols table even in the event of an error
	symtable, _ := symbols.ReadSymbolsFile(cartload.Filename)

	cart := cartridge.NewCartridge()

	err := cart.Attach(cartload)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	err = dsm.FromMemory(cart, symtable)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	return dsm, nil
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control. Unlike the FromCartridge() function this function
// requires an existing instance of Disassembly.
func (dsm *Disassembly) FromMemory(cart *cartridge.Cartridge, symtable *symbols.Table) error {
	dsm.cart = cart
	dsm.Symtable = symtable
	dsm.reference = make([][memorymap.AddressMaskCart + 1]*Entry, dsm.cart.NumBanks())

	// exit early if cartridge memory self reports as being ejected
	if dsm.cart.IsEjected() {
		return nil
	}

	// put cart into its initial state
	dsm.cart.Initialise()

	// create new memory
	mem := &disasmMemory{cart: cart}

	// create a new NoFlowControl CPU to help disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return errors.New(errors.DisasmError, err)
	}
	mc.NoFlowControl = true

	// some cartridge types react when certain registers are read/written. for
	// disassembly purposes we don't want that so we turn on the Passive flag
	// for the duration
	dsm.cart.Passive = true
	defer func() { dsm.cart.Passive = false }()

	// decode pass
	err = dsm.decode(mc)
	if err != nil {
		return errors.New(errors.DisasmError, err)
	}

	// count entry types
	dsm.counts = make([]map[EntryLevel]int, len(dsm.reference))
	for b := 0; b < len(dsm.counts); b++ {
		dsm.counts[b] = make(map[EntryLevel]int)
		for _, e := range dsm.reference[b] {
			if e != nil {
				switch e.Level {
				case EntryLevelDead:
					dsm.counts[b][EntryLevelDead]++

				case EntryLevelDecoded:
					dsm.counts[b][EntryLevelDecoded]++

				case EntryLevelBlessed:
					dsm.counts[b][EntryLevelBlessed]++
				}
			}
		}
	}

	dsm.cart.Initialise()

	return nil
}

// NumBanks returns the number of banks in the disassembly.
func (dsm *Disassembly) NumBanks() int {
	return len(dsm.reference)
}
