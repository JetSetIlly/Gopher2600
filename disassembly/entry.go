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

	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// EntryLevel describes the level of the Entry.
type EntryLevel int

// List of valid EntryL in increasing reliability.
//
// Decoded entries have been decoded as though every byte point is a valid
// instruction. Blessed entries meanwhile take into consideration the preceding
// instruction and the number of bytes it would have consumed.
//
// Decoded entries are useful in the event of the CPU landing on an address that
// didn't look like an instruction at disassembly time.
//
// Blessed instructions are deemed to be more accurate because they have been
// reached according to the flow of the instructions from the start address.
//
// For normal debugging operations there is no need to use EntryLevelUnused
// outside of the disassembly package. It used for the unusual case where a
// bank is not able to be referenced from the Entry address. See M-Network for
// an example of this, where Bank 7 cannot be mapped to the lower segment.
const (
	EntryLevelUnmappable EntryLevel = iota
	EntryLevelDecoded
	EntryLevelBlessed
	EntryLevelExecuted
)

// Entry is a disassambled instruction. The constituent parts of the
// disassembly. It is a representation of execution.Instruction.
type Entry struct {
	dsm *Disassembly

	// the level of reliability of the information in the Entry
	Level EntryLevel

	// execution.Result does not specify which bank the instruction is from
	// because that information isn't available to the CPU. we note it here if
	// possible.
	Bank mapper.BankInfo

	// the entries below are not defined if Level == EntryLevelUnused

	// copy of the CPU execution. must not be updated except through
	// updateExecutionEntry() function
	Result execution.Result

	// string representations of information in execution.Result. GetField()
	// will apply white-space padding and should be preferred in most
	// instances.
	Label    Label
	Bytecode string
	Address  string
	Mnemonic string
	Operand  Operand

	// formatted cycles information from instructions.Defintion
	DefnCycles string
	Cycles     string

	// information about the most recent execution of the entry
	//
	// should be empty if EntryLevel != EntryLevelExecuted
	ExecutionNotes string
}

// String returns a very basic representation of an Entry. Provided for
// convenience. Probably not of any use except for the simplest of tools.
func (e *Entry) String() string {
	operand, _ := e.Operand.checkString()
	return fmt.Sprintf("%s %s %s", e.Address, e.Mnemonic, operand)
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
		dsm:     dsm,
		Result:  result,
		Level:   level,
		Bank:    bank,
		Label:   Label{dsm: dsm, result: result},
		Operand: Operand{dsm: dsm, result: result},
	}

	// address of instruction
	e.Address = fmt.Sprintf("$%04x", result.Address)

	// mnemonic is just a string anyway
	e.Mnemonic = result.Defn.Mnemonic

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

// some fields in the disassembly entry are updated on every execution.
func (e *Entry) updateExecutionEntry(result execution.Result) {
	e.Result = result

	// update result instance in Label and Operand fields
	e.Label.result = result
	e.Operand.result = result

	// indicate that entry has been executed
	e.Level = EntryLevelExecuted

	// actual cycles
	e.Cycles = fmt.Sprintf("%d", e.Result.Cycles)

	// actual notes
	s := strings.Builder{}

	if e.Result.PageFault {
		s.WriteString("[+1] ")
	}

	if e.Result.Defn.IsBranch() {
		if e.Result.BranchSuccess {
			s.WriteString("branched")
		} else {
			s.WriteString("next")
		}
	}

	if e.Result.CPUBug != "" {
		s.WriteString(e.Result.CPUBug)
		s.WriteString(" ")
	}

	e.ExecutionNotes = strings.TrimSpace(s.String())
}

// add decoration to operand according to the addressing mode of the entry.
// operand taken as an argument because it is called from two different contexts.
func addrModeDecoration(operand string, mode instructions.AddressingMode) string {
	s := operand

	switch mode {
	case instructions.Implied:
	case instructions.Immediate:
		s = fmt.Sprintf("#%s", operand)
	case instructions.Relative:
	case instructions.Absolute:
	case instructions.ZeroPage:
	case instructions.Indirect:
		s = fmt.Sprintf("(%s)", operand)
	case instructions.IndexedIndirect:
		s = fmt.Sprintf("(%s,X)", operand)
	case instructions.IndirectIndexed:
		s = fmt.Sprintf("(%s),Y", operand)
	case instructions.AbsoluteIndexedX:
		s = fmt.Sprintf("%s,X", operand)
	case instructions.AbsoluteIndexedY:
		s = fmt.Sprintf("%s,Y", operand)
	case instructions.ZeroPageIndexedX:
		s = fmt.Sprintf("%s,X", operand)
	case instructions.ZeroPageIndexedY:
		s = fmt.Sprintf("%s,Y", operand)
	}

	return s
}

// absolute branch destination returns the branch operand as the address of the
// branched PC, rather than an offset value.
func absoluteBranchDestination(addr uint16, operand uint16) uint16 {
	// create a mock register with the instruction's address as the initial value
	pc := registers.NewProgramCounter(addr)

	// all 6502 branch instructions are 2 bytes in length
	pc.Add(2)

	// because we're doing 16 bit arithmetic with an 8bit value, we need to
	// make sure the sign bit has been propogated to the more-significant bits
	if operand&0x0080 == 0x0080 {
		operand |= 0xff00
	}

	// add the 2s-complement value to the mock program counter
	pc.Add(operand)

	return pc.Address()
}

// Label implements the Stringer interface. The String() implementation
// returns any address label for the entry. Use GetField() function for
// a white-space padded label.
type Label struct {
	dsm    *Disassembly
	result execution.Result
}

func (l Label) String() string {
	s, _ := l.checkString()
	return s
}

// checkString returns the address label as a symbol (if a symbol is available)
// if a symbol is not available then the the bool return value will be false.
func (l Label) checkString() (string, bool) {
	if l.dsm.Prefs.Symbols.Get().(bool) {
		ma, _ := memorymap.MapAddress(l.result.Address, true)
		if v, ok := l.dsm.Symbols.Label.Entries[ma]; ok {
			return v, true
		}
	}

	return "", false
}

// Operand implements the Stringer interface. The String() implementation
// returns the operand (with symbols if appropriate). Use GetField function for
// white-space padded operand string.
type Operand struct {
	nonSymbolic string
	dsm         *Disassembly
	result      execution.Result
}

func (l Operand) String() string {
	s, _ := l.checkString()
	return s
}

// checkString returns the operand as a symbol (if a symbol is available) if
// a symbol is not available then the the bool return value will be false.
func (l Operand) checkString() (string, bool) {
	if !l.dsm.Prefs.Symbols.Get().(bool) {
		return l.nonSymbolic, false
	}

	s := l.nonSymbolic

	// use symbol for the operand if available/appropriate. we should only do
	// this if operand has been decoded
	if l.result.Final {
		if l.result.Defn.AddressingMode == instructions.Immediate {
			// TODO: immediate symbols

		} else if l.result.ByteCount > 1 {
			// instruction data is only valid if bytecount is 2 or more

			operand := l.result.InstructionData

			switch l.result.Defn.Effect {
			case instructions.Flow:
				if l.result.Defn.IsBranch() {
					operand = absoluteBranchDestination(l.result.Address, operand)

					// look up mock program counter value in symbol table
					if v, ok := l.dsm.Symbols.Label.Entries[operand]; ok {
						s = v
					}
				} else if v, ok := l.dsm.Symbols.Label.Entries[operand]; ok {
					s = addrModeDecoration(v, l.result.Defn.AddressingMode)
				}
			case instructions.Read:
				mappedOperand, _ := memorymap.MapAddress(operand, true)
				if v, ok := l.dsm.Symbols.Read.Entries[mappedOperand]; ok {
					s = addrModeDecoration(v, l.result.Defn.AddressingMode)
				}

			case instructions.Write:
				fallthrough

			case instructions.RMW:
				mappedOperand, _ := memorymap.MapAddress(operand, false)
				if v, ok := l.dsm.Symbols.Write.Entries[mappedOperand]; ok {
					s = addrModeDecoration(v, l.result.Defn.AddressingMode)
				}
			}
		}
	}

	return s, true
}
