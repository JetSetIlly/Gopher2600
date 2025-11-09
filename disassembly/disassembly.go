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
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/logger"
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

	// critical sectioning to protect disasmEntries. the symbols table has it's
	// own critical section
	crit sync.Mutex

	// is true if background disassembly is active
	background atomic.Bool
}

// DisasmEntries contains the individual disassembled entries of the current ROM.
type DisasmEntries struct {
	// indexed by bank and address. address should be masked with memorymap.CartridgeBits before access
	Entries [][]*Entry

	// executed entries in order of execution
	Sequential []*Entry
}

// NewDisassembly is the preferred method of initialisation for the Disassembly
// type.
//
// Also returns a reference to the disassembly's symbol table. This reference
// will never change over the course of the lifetime of the Disassembly type
// itself. ie. the returned reference is safe to use after calls to
// FromMemory() or FromCartridge().
func NewDisassembly(vcs *hardware.VCS) (*Disassembly, *symbols.Symbols, error) {
	dsm := &Disassembly{vcs: vcs}

	var err error

	dsm.Prefs, err = newPreferences(dsm)
	if err != nil {
		return nil, nil, fmt.Errorf("disassembly: %w", err)
	}
	dsm.background.Store(false)

	return dsm, &dsm.Sym, nil
}

const disassemblyLabel = environment.Label("disassembly")

// FromCartridge initialises a new partial emulation and returns a disassembly
// from the supplied cartridge filename. Useful for one-shot disassemblies,
// like the gopher2600 "disasm" mode.
func FromCartridge(cartload cartridgeloader.Loader) (*Disassembly, error) {
	var err error

	tv, err := television.NewTelevision("AUTO")
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	vcs, err := hardware.NewVCS(disassemblyLabel, tv, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("disassembly: %w", err)
	}

	err = vcs.AttachCartridge(cartload, nil)
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

// Background performs a disassembly from memory but in the background
func (dsm *Disassembly) Background(cartload cartridgeloader.Loader) {
	go func() {
		dsm.background.Store(true)
		defer dsm.background.Store(false)
		err := dsm.FromMemory()
		if err != nil {
			logger.Log(dsm.vcs.Env, "disassembly", err.Error())
		}
	}()
}

func (dsm *Disassembly) Reset() {
	dsm.disasmEntries.Sequential = dsm.disasmEntries.Sequential[:0]
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control. Unlike the FromCartridge() function this function
// requires an existing instance of Disassembly.
//
// Disassembly will start/assume the cartridge is in the correct starting bank.
func (dsm *Disassembly) FromMemory() error {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// create new memory
	copiedBanks, err := dsm.vcs.Mem.Cart.CopyBanks()
	if err != nil {
		return fmt.Errorf("disassembly: %w", err)
	}

	startingBank := dsm.vcs.Mem.Cart.GetBank(cpu.Reset).Number

	mem := newDisasmMemory(startingBank, copiedBanks)
	if mem == nil {
		return fmt.Errorf("disassembly: %s", "could not create memory for disassembly")
	}

	// read symbols file
	err = dsm.Sym.ReadDASMSymbolsFile(dsm.vcs.Mem.Cart)
	if err != nil {
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
		return nil
	}

	// create a new NoFlowControl CPU to help disassemble memory
	mc := cpu.NewCPU(mem)
	mc.NoFlowControl = true

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
// ExecutedEntry will update the disassembly corresponding to the result. If
// there is no existing entry, a new entry is added and a messsage is logged
// saying that there has been a "late decoding" - this suggests a flaw in the
// decoding process.
//
// checkNextAddr should be false if the result does no represent a completed
// instruction. in other words, if the instruction has only partially completed
func (dsm *Disassembly) ExecutedEntry(bank mapper.BankInfo, result execution.Result, checkNextAddr bool, nextAddr uint16) *Entry {
	e := dsm.FormatResult(bank, result, EntryLevelExecuted)

	// if co-processor is executing then whatever has been executed by the 6507
	// will not relate to the permanent disassembly. format the result and
	// return
	//
	// (in fact, there's a possible optimisation here where if we know the
	// co-processor/mapper being used we can just return a predefined NOP
	// disassembly)
	if bank.ExecutingCoprocessor {
		return e
	}

	// if executed entry is in non-cartridge space then there's nothing we an do other than updating
	// the sequential disaasembly
	if !bank.NonCart {
		// similarly, if bank number is outside the banks we've already decoded
		// then format the result and return
		//
		// I'm not sure when this would apply. maybe it's just a belt-and-braces
		// check. there's no comment to say why this condition was added so leave
		// it for now
		if bank.Number >= len(dsm.disasmEntries.Entries) {
			return e
		}

		// do not update disassembly if background disassembly is ongoing
		if dsm.background.Load() {
			return e
		}

		// updating an entry can happen at the same time as iteration which is
		// probably being run from a different goroutine. acknowledge the critical
		// section
		dsm.crit.Lock()
		defer dsm.crit.Unlock()

		// add/update entry to disassembly
		idx := result.Address & memorymap.CartridgeBits
		o := dsm.disasmEntries.Entries[bank.Number][idx]
		if o != nil && o.Result.Final {
			e.updateExecutionEntry(result)
		}
		dsm.disasmEntries.Entries[bank.Number][idx] = e

		// bless next entry in case it was missed by the original decoding. there's
		// no guarantee that the bank for the next address will be the same as the
		// current bank, so we have to call the GetBank() function.
		if checkNextAddr && result.Final {
			bank = dsm.vcs.Mem.Cart.GetBank(nextAddr)
			idx := nextAddr & memorymap.CartridgeBits
			ne := dsm.disasmEntries.Entries[bank.Number][idx]
			if ne == nil {
				dsm.disasmEntries.Entries[bank.Number][idx] = dsm.FormatResult(bank, execution.Result{
					Address: nextAddr,
				}, EntryLevelBlessed)
			} else if ne.Level < EntryLevelBlessed {
				ne.Level = EntryLevelBlessed
			}
		}
	}

	// add to sequential list or ammend the last entry as appropriate
	if len(dsm.disasmEntries.Sequential) != 0 {
		last := dsm.disasmEntries.Sequential[len(dsm.disasmEntries.Sequential)-1]

		// the decision whether to ammend of append is more complex than you might expect because we
		// need to consider if the results being worked with is a 'final' result or an interim result

		// when running in the instruction quantum, every result passed to this function will be final
		if result.Final {
			// we don't want to do anything with the sequence if the CPU has moved from the RDY
			// state to the non-RDY state. this prevents STA WSYNC instructions or similar from
			// being repeated while the CPU is not executing
			//
			// looking at the CPU RDY state directly will not work for this. if we only looked the
			// RDY state directly then that will mean a STA WSYNC instruction, for example, will
			// appear to have occurred at the beginning of a scanline
			if result.Rdy || (!result.Rdy && last.Result.Rdy) {
				// if the last result is not final then that means we are in the cycle of clock
				// quantums and we need to replace the last entry with the newer result - the newer
				// result represents the same instruction but it contains more information
				if !last.Result.Final {
					dsm.disasmEntries.Sequential[len(dsm.disasmEntries.Sequential)-1] = e
				} else {
					if last.Result.Address == result.Address &&
						!(result.Defn.IsBranch() || result.Defn.Effect != instructions.Flow) {
						dsm.disasmEntries.Sequential[len(dsm.disasmEntries.Sequential)-1] = e
					} else {
						dsm.disasmEntries.Sequential = append(dsm.disasmEntries.Sequential, e)
					}
				}
			}
		} else if !last.Result.Final {
			// if the previous entry was not final then always replace it with the new entry
			// (this can happen when in the cycle or clock quantums)
			dsm.disasmEntries.Sequential[len(dsm.disasmEntries.Sequential)-1] = e
		} else {
			// if the new result is not final and the last result was final then append the new entry
			// (this can happen when in the cycle or clock quantums)
			dsm.disasmEntries.Sequential = append(dsm.disasmEntries.Sequential, e)
		}
	} else {
		dsm.disasmEntries.Sequential = append(dsm.disasmEntries.Sequential, e)
	}

	const maxLength = 10000
	if len(dsm.disasmEntries.Sequential) > maxLength {
		dsm.disasmEntries.Sequential = dsm.disasmEntries.Sequential[len(dsm.disasmEntries.Sequential)-maxLength:]
	}

	return e
}

// FormatResult creates an Entry for supplied result/bank. It will be assigned
// the specified EntryLevel.
//
// If EntryLevel is EntryLevelExecuted then the disassembly will be updated but
// only if result.Final is true.
func (dsm *Disassembly) FormatResult(bank mapper.BankInfo, result execution.Result, level EntryLevel) *Entry {
	e := &Entry{
		dsm:    dsm,
		Result: result,
		Level:  level,
		Bank:   bank.Number,
		Label: Label{
			dsm:    dsm,
			result: result,
			bank:   bank,
		},
		Operand: Operand{
			dsm:    dsm,
			result: result,
			bank:   bank,
		},
		Coords: dsm.vcs.TV.GetCoords(),
	}

	// address of instruction
	e.Address = fmt.Sprintf("$%04x", result.Address)

	// if definition is nil then set the operator field to ??? and return with no further formatting
	if result.Defn == nil {
		e.Operator = "???"
		return e
	}

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
			e.Operand.partial = fmt.Sprintf("$%04x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x %02x", result.Defn.OpCode, operand&0x00ff, operand&0xff00>>8)
		case 2:
			operand := result.InstructionData
			e.Operand.partial = fmt.Sprintf("$??%02x", result.InstructionData)
			e.Bytecode = fmt.Sprintf("%02x %02x ?? ", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand.partial = "$????"
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
			e.Operand.partial = fmt.Sprintf("$%02x", operand)
			e.Bytecode = fmt.Sprintf("%02x %02x", result.Defn.OpCode, operand&0x00ff)
		case 1:
			e.Operand.partial = "$??"
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
	e.Operand.partial = addrModeDecoration(e.Operand.partial, e.Result.Defn.AddressingMode)

	return e
}

// BorrowDisasm will lock the DisasmEntries structure for the durction of the
// supplied function, which will be executed with the disasm structure as an
// argument.
//
// Function will be executed with a nil argument if disassembly is not valid.
//
// Should not be called from the emulation goroutine.
func (dsm *Disassembly) BorrowDisasm(f func(*DisasmEntries)) bool {
	if dsm.background.Load() {
		return false
	}

	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	f(&dsm.disasmEntries)
	return true
}

// Splice implements the rewinder.Splicer interface
func (dsm *Disassembly) Splice(c coords.TelevisionCoords) {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	for i, e := range dsm.disasmEntries.Sequential {
		if coords.GreaterThan(e.Coords, c) {
			if i == 0 {
				dsm.disasmEntries.Sequential = dsm.disasmEntries.Sequential[:0]
			} else {
				dsm.disasmEntries.Sequential = dsm.disasmEntries.Sequential[:i-1]
			}
			break // for loop
		}
	}
}
