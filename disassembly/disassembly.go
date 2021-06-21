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
	"fmt"
	"strings"
	"sync"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/disassembly/coprocessor"
	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

// Disassembly represents the annotated disassembly of a 6507 binary.
type Disassembly struct {
	Prefs *Preferences

	// reference to running hardware. all access to VCS sub-systems to be done
	// via this pointer. this facilitates the rewind system which would
	// otherwise cause stale pointers to misleading information.
	vcs *hardware.VCS

	// symbols used to format disassembly output
	sym symbols.Symbols

	// indexed by bank and address. address should be masked with memorymap.CartridgeBits before access
	entries [][]*Entry

	// any cartridge coprocessor that we find
	Coprocessor *coprocessor.Coprocessor

	// critical sectioning. the iteration functions in particular may be called
	// from a different goroutine. entries in the (disasm array) will likely be
	// updating more or less constantly with ExecuteEntry() so it's important
	// we enforce the critical sections
	//
	// experiments with gochannel driven disassembly service proved too slow
	// for iterating. this is because waiting for the result from any disasm
	// service goroutine is inherently slow.
	//
	// whether a sync.Mutex is the best low level synchronisation method is
	// another question.
	crit sync.Mutex
}

// NewDisassembly is the preferred method of initialisation for the Disassembly
// type.
//
// Also returns a reference to the disassembly's symbol table. This reference
// will never change over the course of the lifetime of the Disassembly type
// itself. ie. the returned reference is safe to use after calls to
// FromMemory() or FromCartrige().
func NewDisassembly(vcs *hardware.VCS) (*Disassembly, *symbols.Symbols, error) {
	dsm := &Disassembly{vcs: vcs}

	var err error

	dsm.Prefs, err = newPreferences(dsm)
	if err != nil {
		return nil, nil, curated.Errorf("disassembly: %v", err)
	}

	return dsm, &dsm.sym, nil
}

// FromCartridge initialises a new partial emulation and returns a disassembly
// from the supplied cartridge filename. Useful for one-shot disassemblies,
// like the gopher2600 "disasm" mode.
func FromCartridge(cartload cartridgeloader.Loader) (*Disassembly, error) {
	var err error

	tv, err := television.NewTelevision("NTSC")
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	err = vcs.AttachCartridge(cartload)
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	dsm, _, err := NewDisassembly(vcs)
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	// ignore errors caused by loading of symbols table - we always get a
	// standard symbols table even in the event of an error
	err = dsm.sym.ReadSymbolsFile(vcs.Mem.Cart)
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	// do disassembly
	err = dsm.FromMemory()
	if err != nil {
		return nil, curated.Errorf("disassembly: %v", err)
	}

	return dsm, nil
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control. Unlike the FromCartridge() function this function
// requires an existing instance of Disassembly
//
// cartridge will finish in its initialised state.
func (dsm *Disassembly) FromMemory() error {
	dsm.crit.Lock()
	// unlocking manually before we call the disassmeble() function. this means
	// we have to be careful to manually unlock before returning an error.

	// read symbols file
	err := dsm.sym.ReadSymbolsFile(dsm.vcs.Mem.Cart)
	if err != nil {
		dsm.crit.Unlock()
		return err
	}

	// allocate memory for disassembly. the GUI may find itself trying to
	// iterate through disassembly at the same time as we're doing this.
	dsm.entries = make([][]*Entry, dsm.vcs.Mem.Cart.NumBanks())
	for b := 0; b < len(dsm.entries); b++ {
		dsm.entries[b] = make([]*Entry, memorymap.CartridgeBits+1)
	}

	// exit early if cartridge memory self reports as being ejected
	if dsm.vcs.Mem.Cart.IsEjected() {
		dsm.crit.Unlock()
		return nil
	}

	// create new memory
	mem := &disasmMemory{}

	// create a new NoFlowControl CPU to help disassemble memory
	mc := cpu.NewCPU(nil, mem)
	mc.NoFlowControl = true

	dsm.crit.Unlock()
	// end of critical section

	// disassemble cartridge binary
	err = dsm.disassemble(mc, mem)
	if err != nil {
		return curated.Errorf("disassembly: %v", err)
	}

	// try added coprocessor disasm support
	dsm.Coprocessor = coprocessor.Add(dsm.vcs, dsm.vcs.Mem.Cart)

	return nil
}

// GetEntryByAddress returns the disassembly entry at the specified
// bank/address. a returned value of nil indicates the entry is not in the
// cartridge; this will usually mean the address is in main VCS RAM.
//
// also returns whether cartridge is currently working from another source
// meaning that the disassembly entry might not be reliable.
func (dsm *Disassembly) GetEntryByAddress(address uint16) (*Entry, bool) {
	bank := dsm.vcs.Mem.Cart.GetBank(address)

	if bank.NonCart {
		// !!TODO: attempt to decode instructions not in cartridge
		return nil, bank.ExecutingCoprocessor
	}

	return dsm.entries[bank.Number][address&memorymap.CartridgeBits], bank.ExecutingCoprocessor
}

// ExecutedEntry creates an Entry from a cpu result that has actually been
// executed. When appropriate, the newly created Entry replaces the previous
// equivalent entry in the disassembly.
//
// If the execution.Result was from an instruction in RAM (cartridge RAM or VCS
// RAM) then the newly created entry is returned but not stored anywhere in the
// Disassembly.
func (dsm *Disassembly) ExecutedEntry(bank mapper.BankInfo, result execution.Result, nextAddr uint16) (*Entry, error) {
	// not touching any result which is not in cartridge space. we are noting
	// execution results from cartridge RAM. the banks.Details field in the
	// disassembly entry notes whether execution was from RAM
	if bank.NonCart {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	if bank.Number >= len(dsm.entries) {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	idx := result.Address & memorymap.CartridgeBits

	// get entry at address
	e := dsm.entries[bank.Number][idx]

	// updating an origin can happen at the same time as iteration which is
	// probably being run from a different goroutine. acknowledge the critical
	// section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// check for opcode reliability. this can happen when it is expected
	// (bank.ExecutingCoProcess is true) or when it is unexpected.
	if bank.ExecutingCoprocessor || e.Result.Defn.OpCode != result.Defn.OpCode {
		// in either instance we want to return the formatted result of the
		// actual execution
		ne, err := dsm.FormatResult(bank, result, EntryLevelExecuted)
		if err != nil {
			return nil, curated.Errorf("disassembly: %v", err)
		}

		return ne, nil
	}

	// opcode is reliable update disasm entry in the normal way
	if e == nil {
		// we're not decoded this bank/address before. note this shouldn't even happen
		var err error
		dsm.entries[bank.Number][idx], err = dsm.FormatResult(bank, result, EntryLevelExecuted)
		if err != nil {
			return nil, curated.Errorf("disassembly: %v", err)
		}
	} else {
		// we have seen this entry before but it's not been executed. update
		// entry to reflect results
		e.updateExecutionEntry(result)
	}

	// bless next entry in case it was missed by the original decoding. there's
	// no guarantee that the bank for the next address will be the same as the
	// current bank, so we have to call the GetBank() function.
	//
	// !!TODO: maybe make sure next entry has been disassembled in it's current form
	bank = dsm.vcs.Mem.Cart.GetBank(nextAddr)
	ne := dsm.entries[bank.Number][nextAddr&memorymap.CartridgeBits]
	if ne.Level < EntryLevelBlessed {
		ne.Level = EntryLevelBlessed
	}

	return e, nil
}

// FormatResult It is the preferred method of initialising for the Entry type.
// It creates a disassembly.Entry based on the bank and result information.
func (dsm *Disassembly) FormatResult(bank mapper.BankInfo, result execution.Result, level EntryLevel) (*Entry, error) {
	// protect against empty definitions. we shouldn't hit this condition from
	// the disassembly package itself, but it is possible to get it from ad-hoc
	// formatting from GUI interfaces (see CPU window in sdlimgui)
	if result.Defn == nil {
		return &Entry{}, nil
	}

	e := &Entry{
		dsm:    dsm,
		Result: result,
		Level:  level,
		Bank:   bank,
		Label: Label{
			dsm:    dsm,
			result: result,
			bank:   bank.Number,
		},
		Operand: Operand{
			dsm:    dsm,
			result: result,
			bank:   bank.Number,
		},
	}

	// address of instruction
	e.Address = fmt.Sprintf("$%04x", result.Address)

	// operator is just a string anyway
	e.Operator = result.Defn.Operator

	// bytecode and operand string is assembled depending on the number of
	// expected bytes (result.Defn.Bytes) and the number of bytes read so far
	// (result.ByteCount).
	//
	// the panics cover situations that should never exists. if result
	// validation is active then the panic situations will have been caught
	// then. if validation is not running then the code could theoretically
	// panic but that's okay, they should have been caught in testing.
	switch result.Defn.Bytes {
	case 3:
		switch result.ByteCount {
		case 3:
			operand := result.InstructionData
			e.Operand.nonSymbolic = fmt.Sprintf("$%04x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x %02x", result.Defn.OpCode, operand&0x00ff, operand&0xff00>>8)
		case 2:
			operand := result.InstructionData
			e.Operand.nonSymbolic = fmt.Sprintf("$??%02x", result.InstructionData)
			e.Bytecode = fmt.Sprintf("%02x %02x ?? ", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand.nonSymbolic = "$????"
			e.Bytecode = fmt.Sprintf("%02x ?? ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number")
		}
	case 2:
		switch result.ByteCount {
		case 2:
			operand := result.InstructionData
			e.Operand.nonSymbolic = fmt.Sprintf("$%02x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand.nonSymbolic = "$??"
			e.Bytecode = fmt.Sprintf("%02x ??", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number")
		}
	case 1:
		switch result.ByteCount {
		case 1:
			e.Bytecode = fmt.Sprintf("%02x", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we shoud not be able to read more bytes than the expected number")
		}
	case 0:
		panic("instructions of zero bytes is not possible")
	default:
		panic("instructions of more than 3 bytes is not possible")
	}
	e.Bytecode = strings.TrimSpace(e.Bytecode)

	// decorate operand with addressing mode indicators. this decorates the
	// non-symbolic operand. we also call the decorate function from the
	// Operand() function when a symbol has been found
	if e.Result.Defn.IsBranch() {
		e.Operand.nonSymbolic = fmt.Sprintf("$%04x", absoluteBranchDestination(e.Result.Address, e.Result.InstructionData))
	} else {
		e.Operand.nonSymbolic = addrModeDecoration(e.Operand.nonSymbolic, e.Result.Defn.AddressingMode)
	}

	// definintion cycles
	if result.Defn.IsBranch() {
		e.DefnCycles = fmt.Sprintf("%d/%d", result.Defn.Cycles, result.Defn.Cycles+1)
	} else {
		e.DefnCycles = fmt.Sprintf("%d", result.Defn.Cycles)
	}

	if level == EntryLevelExecuted {
		e.updateExecutionEntry(result)
	}

	return e, nil
}
