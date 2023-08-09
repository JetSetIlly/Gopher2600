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

package arm

import (
	"fmt"
	"strings"
)

// The string that will be in the Operator field in the case of a decoding error
const DisasmEntryErrorOperator = "error:"

// DisasmEntry implements the CartCoProcDisasmEntry interface.
type DisasmEntry struct {
	// the address value. the formatted value is in the Address field
	Addr uint32

	// the opcode for the instruction. in the case of a 32bit instruction, this
	// will be the second word of the opcode
	Opcode uint16

	// instruction is 32bit and the high opcode
	Is32bit  bool
	OpcodeHi uint16

	// -----------

	// formated address for use by disassemblies. more convenient that the Addr
	// field in some contexts
	Address string

	// the operator is the instruction specified by the Opcode field (and
	// OpcodeHi if the instruction is 32bit)
	//
	// the operand is the specific details of the instruction. what registers
	// and what values are used, etc.
	//
	// in the case of an error Operator will contain the DisasmEntryErrorOperator
	// string and the Operand will be the detaled message of the error
	Operator string
	Operand  string

	// -----------

	// the values of the remaining fields are not defined unless the
	// instruction has been executed

	// basic notes about the last execution of the entry
	ExecutionNotes string

	// snapshot of CPU registers at the result of the instruction
	Registers [NumCoreRegisters]uint32

	// basic cycle information
	Cycles         int
	CyclesSequence string

	// cycle details
	MAMCR       int
	BranchTrail BranchTrail
	MergedIS    bool

	// whether this entry was executed in immediate mode. if this field is true
	// then the Cycles and "cycle details" fields will be zero
	ImmediateMode bool
}

// Key implements the CartCoProcDisasmEntry interface.
func (e DisasmEntry) Key() string {
	return e.Address
}

// CSV implements the CartCoProcDisasmEntry interface. Outputs CSV friendly
// entries, albeit seprated by semicolons rather than commas.
func (e DisasmEntry) CSV() string {
	mergedIS := ""
	if e.MergedIS {
		mergedIS = "merged IS"
	}

	return fmt.Sprintf("%s;%s;%s;%d;%s;%s;%s", e.Address, e.Operator, e.Operand, e.Cycles, e.ExecutionNotes, mergedIS, e.CyclesSequence)
}

// String implements the CartCoProcDisasmEntry interface. Returns a very simple
// representation of the disassembly entry.
func (e DisasmEntry) String() string {
	if e.Operator == "" {
		return e.Operand
	}
	return fmt.Sprintf("%s %s", e.Operator, e.Operand)
}

// Size implements the CartCoProcDisasmEntry interface.
func (e DisasmEntry) Size() int {
	if e.Is32bit {
		return 4
	}
	return 2
}

// fillDisasmEntry sets the DisasmEntry fields using information from the emulated ARM
//
// the Is32bit field will remain true if it has previously been set to true
func fillDisasmEntry(arm *ARM, e *DisasmEntry, opcode uint16) {
	e.Addr = arm.state.instructionPC
	e.Opcode = opcode
	e.OpcodeHi = arm.state.function32bitOpcodeHi
	e.Address = fmt.Sprintf("%08x", arm.state.instructionPC)
	e.Operator = strings.ToLower(e.Operator)

	// deciding whether the instruction is 32bit or not needs to cover three
	// 1) when the flag has already been set explicitely
	// 2) when the instruction is only being decoded (static disassembly)
	// 3) when the instruction has actually been executed, or resolved (live disassembly)
	e.Is32bit = e.Is32bit || arm.state.function32bitDecoding || arm.state.function32bitResolving
}
