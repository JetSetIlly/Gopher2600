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
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	ref "github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/symbols"
)

// Disassembly represents the annotated disassembly of a 6507 binary
type Disassembly struct {
	cart *cartridge.Cartridge

	// symbols used to format disassembly output
	Symtable *symbols.Table

	// indexed by address. address should be masked with
	// memorymap.AddressMaskCart before access.
	reference [][ref.AddressMaskCart + 1]*Entry

	// the number of each type of entry. we use this to help prepare
	// disassembly iterations
	counts []map[EntryLevel]int

	// formatting information for all entries found during the flow pass.
	// excluding entries only found during the linear pass because
	// false-positive entries might upset the formatting.
	fields fields
}

// GetEntryByAddress returns the disassembly entry at the specified bank/address.
func (dsm Disassembly) GetEntryByAddress(bank int, address uint16) (*Entry, bool) {
	col := dsm.reference[bank][address&ref.AddressMaskCart]
	return col, col != nil
}

// BlessEntry promotes an entry to the stated EntryLevel
func (dsm *Disassembly) BlessEntry(bank int, address uint16) {
	if bank >= len(dsm.reference) {
		return
	}

	// get entry at address
	e := dsm.reference[bank][address&ref.AddressMaskCart]

	// loop while there are entries to bless, stop on a dead entry
	for e != nil && e.Level != EntryLevelDead && e.Level < EntryLevelBlessed {
		e.Level = EntryLevelBlessed
		address += uint16(e.Result.ByteCount)
		e = dsm.reference[bank][address&ref.AddressMaskCart]
	}
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
	dsm.reference = make([][ref.AddressMaskCart + 1]*Entry, dsm.cart.NumBanks())

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

	return dsm, nil
}
