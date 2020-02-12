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
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/execution"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/cartridge"
	"gopher2600/symbols"
	"strings"
)

const disasmMask = 0x0fff

type bank [disasmMask + 1]*Entry

// Disassembly represents the annotated disassembly of a 6507 binary
type Disassembly struct {
	cart *cartridge.Cartridge

	// discovered/inferred cartridge attributes
	nonCartJmps bool
	interrupts  bool
	forcedRTS   bool

	// symbols used to format disassembly output
	Symtable *symbols.Table

	// Entries is created from two passes. the linear pass which simply decodes
	// every address as though it is an instruction and a flow pass, which only
	// considers addresses that the program counter can hit when the CPU is ran
	// from the reset vector
	Entries []bank

	// formatting information for all entries found during the flow pass.
	// excluding entries only found during the linear pass because
	// false-positive entries might upset the formatting.
	fields fields
}

// Analysis returns a summary of anything interesting found during disassembly.
func (dsm Disassembly) Analysis() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("non-cart JMPs: %v\n", dsm.nonCartJmps))
	s.WriteString(fmt.Sprintf("interrupts: %v\n", dsm.interrupts))
	s.WriteString(fmt.Sprintf("forced RTS: %v\n", dsm.forcedRTS))
	return s.String()
}

// Get returns the disassembly at the specified bank/address
//
// function works best when the address definitely points to a valid
// instruction. This probably means during the execution of a the cartridge
// with proper flow control.
func (dsm Disassembly) Get(bank int, address uint16) (*Entry, bool) {
	col := dsm.Entries[bank][address&disasmMask]
	return col, col != nil
}

// FormatResult is a wrapper for the display.Format() function using the
// current symbol table
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
	dsm.Entries = make([]bank, dsm.cart.NumBanks())

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

	// linear pass
	err = mc.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	err = dsm.linearPass(mc)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	// flow pass
	mc.Reset()
	dsm.cart.Initialise()

	err = mc.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	err = dsm.flowPass(mc, addresses.Reset)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	return dsm, nil
}
