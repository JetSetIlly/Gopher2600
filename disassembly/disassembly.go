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
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/execution"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/cartridge"
	"gopher2600/hardware/memory/memorymap"
	"gopher2600/symbols"
)

// Disassembly represents the annotated disassembly of a 6507 binary
type Disassembly struct {
	cart *cartridge.Cartridge

	// symbols used to format disassembly output
	Symtable *symbols.Table

	// indexed by address. address should be masked with
	// memorymap.AddressMaskCart before access.
	Entries [][memorymap.AddressMaskCart + 1]*Entry

	// the number of each type of entry
	Counts []map[EntryType]int

	// formatting information for all entries found during the flow pass.
	// excluding entries only found during the linear pass because
	// false-positive entries might upset the formatting.
	fields fields

	// static analysis (best effort) of cartridge
	Analysis Analysis
}

// Get returns the disassembly at the specified bank/address.
func (dsm Disassembly) Get(bank int, address uint16) (*Entry, bool) {
	col := dsm.Entries[bank][address&memorymap.AddressMaskCart]
	return col, col != nil
}

// FormatResult returns the formatted representation of an execution result.
// Build string representations with GetField(). Also see Write*() functions for
// less flexible but convenient alternative.
func (dsm Disassembly) FormatResult(result execution.Result) (*Entry, error) {
	return newEntry(result, dsm.Symtable)
}

// FromCartridge initialises a new partial emulation and returns a
// disassembly from the supplied cartridge filename. - useful for one-shot
// disassemblies, like the gopher2600 "disasm" mode
func FromCartridge(cartload cartridgeloader.Loader) (*Disassembly, error) {
	// ignore errors caused by loading of symbols table - we always get a
	// standard symbols table even in the event of an error
	symtable, _ := symbols.ReadSymbolsFile(cartload.Filename)

	cart := cartridge.NewCartridge()

	err := cart.Attach(cartload)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	dsm, err := FromMemory(cart, symtable)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	return dsm, nil
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control.
func FromMemory(cart *cartridge.Cartridge, symtable *symbols.Table) (*Disassembly, error) {
	dsm := &Disassembly{}

	dsm.cart = cart
	dsm.Symtable = symtable
	dsm.Entries = make([][memorymap.AddressMaskCart + 1]*Entry, dsm.cart.NumBanks())

	// exit early if cartridge memory self reports as being ejected
	if dsm.cart.IsEjected() {
		return dsm, nil
	}

	// save cartridge state and defer at end of disassembly. this is necessary
	// because during the disassembly process we may changed mutable parts of
	// the cartridge (eg. extra RAM)
	state := dsm.cart.SaveState()
	defer dsm.cart.RestoreState(state)

	// put cart into its initial state
	dsm.cart.Initialise()

	// create new memory
	mem := &disasmMemory{cart: cart}

	// create a new NoFlowControl CPU to help disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}
	mc.NoFlowControl = true

	// decode pass
	err = dsm.decode(mc)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	// reset
	mc.Reset()
	err = mc.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return nil, err
	}
	dsm.cart.Initialise()

	// flow pass
	err = dsm.flowAnalysis(mc, addresses.Reset, 0)
	if err != nil {
		return nil, errors.New(errors.AnalysisError, err)
	}

	// count entry types
	dsm.countTypes()

	return dsm, nil
}

// count number of each type entry in disassembly
func (dsm *Disassembly) countTypes() {
	dsm.Counts = make([]map[EntryType]int, len(dsm.Entries))
	for b := 0; b < len(dsm.Counts); b++ {
		dsm.Counts[b] = make(map[EntryType]int)
		for _, e := range dsm.Entries[b] {
			if e != nil {
				switch e.Type {
				case EntryTypeAnalysis:
					dsm.Counts[b][EntryTypeAnalysis]++
					fallthrough

				case EntryTypeDecode:
					dsm.Counts[b][EntryTypeDecode]++
					fallthrough

				case EntryTypeNaive:
					dsm.Counts[b][EntryTypeNaive]++
				}
			}
		}
	}
}
