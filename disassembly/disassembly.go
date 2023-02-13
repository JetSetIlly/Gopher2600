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
	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
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
	Sym symbols.Symbols

	// disasmEntries entries. use BorrowDisasm() for goroutines other than the
	// emulation goroutine
	disasmEntries DisasmEntries

	// critical sectioning to protect disasmEntries
	crit sync.Mutex
}

// DisasmEntries contains the individual disassembled entries of the current ROM.
type DisasmEntries struct {
	// indexed by bank and address. address should be masked with memorymap.CartridgeBits before access
	Entries [][]*Entry
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
		return nil, nil, fmt.Errorf("disassembly: %w", err)
	}

	return dsm, &dsm.Sym, nil
}

// FromCartridge initialises a new partial emulation and returns a disassembly
// from the supplied cartridge filename. Useful for one-shot disassemblies,
// like the gopher2600 "disasm" mode.
func FromCartridge(cartload cartridgeloader.Loader) (*Disassembly, error) {
	var err error

	tv, err := television.NewTelevision("NTSC")
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	vcs, err := hardware.NewVCS(tv, nil)
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	err = vcs.AttachCartridge(cartload, true)
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	dsm, _, err := NewDisassembly(vcs)
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	// ignore errors caused by loading of symbols table - we always get a
	// standard symbols table even in the event of an error
	err = dsm.Sym.ReadDASMSymbolsFile(vcs.Mem.Cart)
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	// do disassembly
	err = dsm.FromMemory()
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	return dsm, nil
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control. Unlike the FromCartridge() function this function
// requires an existing instance of Disassembly.
//
// Disassembly will start/assume the cartridge is in the correct starting bank.
func (dsm *Disassembly) FromMemory() error {
	// unlocking manually before we call the disassmeble() function. this means
	// we have to be careful to manually unlock before returning an error.
	dsm.crit.Lock()

	// create new memory
	copiedBanks, err := dsm.vcs.Mem.Cart.CopyBanks()
	if err != nil {
		dsm.crit.Unlock()
		return fmt.Errorf("disassembly: %w", err)
	}

	startingBank := dsm.vcs.Mem.Cart.GetBank(cpubus.Reset).Number

	mem := newDisasmMemory(startingBank, copiedBanks)
	if mem == nil {
		dsm.crit.Unlock()
		return fmt.Errorf("disassembly: %s", "could not create memory for disassembly")
	}

	// read symbols file
	err = dsm.Sym.ReadDASMSymbolsFile(dsm.vcs.Mem.Cart)
	if err != nil {
		dsm.crit.Unlock()
		return err
	}

	// allocate memory for disassembly. the GUI may find itself trying to
	// iterate through disassembly at the same time as we're doing this.
	dsm.disasmEntries.Entries = make([][]*Entry, dsm.vcs.Mem.Cart.NumBanks())
	for b := 0; b < len(dsm.disasmEntries.Entries); b++ {
		dsm.disasmEntries.Entries[b] = make([]*Entry, memorymap.CartridgeBits+1)
	}

	// exit early if cartridge memory self reports as being ejected
	if dsm.vcs.Mem.Cart.IsEjected() {
		dsm.crit.Unlock()
		return nil
	}

	// create a new NoFlowControl CPU to help disassemble memory
	mc := cpu.NewCPU(nil, mem)
	mc.NoFlowControl = true

	dsm.crit.Unlock()
	// end of critical section

	// disassemble cartridge binary
	return dsm.disassemble(mc, mem)
}

// GetEntryByAddress returns the disassembly entry at the specified
// bank/address. a returned value of nil indicates the entry is not in the
// cartridge; this will usually mean the address is in main VCS RAM.
//
// also returns whether cartridge is currently working from another source
// meaning that the disassembly entry might not be reliable.
func (dsm *Disassembly) GetEntryByAddress(address uint16) *Entry {
	bank := dsm.vcs.Mem.Cart.GetBank(address)

	if bank.NonCart {
		// !!TODO: attempt to decode instructions not in cartridge
		// when implemented, ammend comment for the STEP OVER command
		return nil
	}

	return dsm.disasmEntries.Entries[bank.Number][address&memorymap.CartridgeBits]
}

// ExecutedEntry should be called after execution of a CPU instruction. In many
// instances it behaves the same as FormatResult with an EntryLevel of
// EntryLevelExecuted. Those intances are:
//
// - a coprocessor is executing (and is interfering with what is being executed)
// - the instruction being disassembled was retrieved from a non-Cartridge address
// - the instruction is from an unknown bank
//
// It should not be called if execution.Result is not finalised. This will only
// lead to confusing disassemblies.
//
// ExecutedEntry will update the disassembly corresponding to the result. If
// there is no existing entry, a new entry is added and a messsage is logged
// saying that there has been a "late decoding" - this suggests a flaw in the
// decoding process.
func (dsm *Disassembly) ExecutedEntry(bank mapper.BankInfo, result execution.Result, checkNextAddr bool, nextAddr uint16) *Entry {
	// format if this is not the final cycle of the instruction
	if !result.Final {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	// if co-processor is executing then whatever has been executed by the 6507
	// will not relate to the permanent disassembly. format the result and
	// return
	//
	// (in fact, there's a possible optimisation here where if we know the
	// co-processor/mapper being used we can just return a predefined NOP
	// disassembly)
	if bank.ExecutingCoprocessor {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	// if executed entry is in non-cartridge space then we just format the
	// result and return it. there's nothing else we can really do - there's no
	// point caching it anywhere
	if bank.NonCart {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	// similarly, if bank number is outside the banks we've already decoded
	// then format the result and return
	//
	// I'm not sure when this would apply. maybe it's just a belt-and-braces
	// check. there's no comment to say why this condition was added so leave
	// it for now
	if bank.Number >= len(dsm.disasmEntries.Entries) {
		return dsm.FormatResult(bank, result, EntryLevelExecuted)
	}

	// updating an entry can happen at the same time as iteration which is
	// probably being run from a different goroutine. acknowledge the critical
	// section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// get disassembly entry at address
	idx := result.Address & memorymap.CartridgeBits
	e := dsm.disasmEntries.Entries[bank.Number][idx]

	// opcode is reliable update disasm entry in the normal way
	if e == nil {
		// we're not decoded this bank/address before
		//
		// ideally this wouldn't ever happen but the decode procedure might
		// have missed it because:
		//
		// (a) there's a bug/flaw in the decode procedure
		// (b) the program has taken an unpredictable path
		// (c) the cartridge data wasn't available at the time of disassembly
		//		- which can happen with ACE roms and other modern cartridge types
		e = dsm.FormatResult(bank, result, EntryLevelExecuted)
		dsm.disasmEntries.Entries[bank.Number][idx] = e
	} else {
		// check for opcode reliability. if it's different then return the actual
		// result and not the one in the entries list.
		//
		// this can happen when a ROM with a coprocessor has *just* finished and
		// therefore bank.ExecutingCoProcessor is false.
		//
		// we just accept these are return the formatted result of the actual
		// instruction. we don't update the disassembly
		if e.Result.Defn.OpCode != result.Defn.OpCode {
			return dsm.FormatResult(bank, result, EntryLevelExecuted)
		}

		// we have seen this entry before. update entry to reflect results
		e.updateExecutionEntry(result)
	}

	// bless next entry in case it was missed by the original decoding. there's
	// no guarantee that the bank for the next address will be the same as the
	// current bank, so we have to call the GetBank() function.
	if checkNextAddr {
		bank = dsm.vcs.Mem.Cart.GetBank(nextAddr)
		ne := dsm.disasmEntries.Entries[bank.Number][nextAddr&memorymap.CartridgeBits]
		if ne != nil && ne.Level < EntryLevelBlessed {
			ne.Level = EntryLevelBlessed
		}
	}

	return e
}

// FormatResult creates an Entry for supplied result/bank. It will be assigned
// the specified EntryLevel.
//
// If EntryLevel is EntryLevelExecuted then the disassembly will be updated but
// only if result.Final is true.
func (dsm *Disassembly) FormatResult(bank mapper.BankInfo, result execution.Result, level EntryLevel) *Entry {
	// protect against empty definitions. we shouldn't hit this condition from
	// the disassembly package itself, but it is possible to get it from ad-hoc
	// formatting from GUI interfaces (see CPU window in sdlimgui)
	if result.Defn == nil {
		return &Entry{dsm: dsm}
	}

	e := &Entry{
		dsm:    dsm,
		Result: result,
		Level:  level,
		Bank:   bank.Number,
		Label: Label{
			dsm:     dsm,
			address: result.Address,
			bank:    bank.Number,
		},
		Operand: Operand{
			dsm:    dsm,
			result: result,
			bank:   bank.Number,
		},
	}

	// address of instruction
	e.Address = fmt.Sprintf("$%04x", result.Address)

	// operator of instruction
	e.Operator = result.Defn.Operator.String()

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
			panic("we should not be able to read more bytes than the expected number (expected 3)")
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
			panic("we should not be able to read more bytes than the expected number (expected 2)")
		}
	case 1:
		switch result.ByteCount {
		case 1:
			e.Bytecode = fmt.Sprintf("%02x", result.Defn.OpCode)
		case 0:
			panic("this makes no sense. we must have read at least one byte to know how many bytes to expect")
		default:
			panic("we should not be able to read more bytes than the expected number (expected 1)")
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

	return e
}

// BorrowDisasm will lock the DisasmEntries structure for the durction of the
// supplied function, which will be executed with the disasm structure as an
// argument.
//
// Function will be executed with a nil argument if disassembly is not valid.
//
// Should not be called from the emulation goroutine.
func (dsm *Disassembly) BorrowDisasm(f func(*DisasmEntries)) {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	f(&dsm.disasmEntries)
}
