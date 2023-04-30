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

// DisasmEntry implements the CartCoProcDisasmEntry interface.
type DisasmEntry struct {
	// the address value. the formatted value is in the Address field
	Addr uint32

	// snapshot of CPU registers at the result of the instruction
	Registers [NumRegisters]uint32

	// the opcode for the instruction
	Opcode uint16

	// instruction is 32bit and the high opcode
	Is32bit  bool
	OpcodeHi uint16

	// formated strings based for use by disassemblies
	Location string
	Address  string
	Operator string
	Operand  string

	// basic notes about the last execution of the entry
	ExecutionNotes string

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

// DisasmSummary implements the CartCoProcDisasmSummary interface.
type DisasmSummary struct {
	// whether this particular execution was run in immediate mode (ie. no cycle counting)
	ImmediateMode bool

	// count of N, I and S cycles. will be zero if ImmediateMode is true.
	N int
	I int
	S int
}

func (s DisasmSummary) String() string {
	return fmt.Sprintf("N: %d  I: %d  S: %d", s.N, s.I, s.S)
}

// add cycle order information to summary.
func (s *DisasmSummary) add(c cycleOrder) {
	for i := 0; i < c.idx; i++ {
		switch c.queue[i] {
		case N:
			s.N++
		case I:
			s.I++
		case S:
			s.S++
		}
	}
}

// Disassemble a single opcode, returning a new DisasmEntry.
func Disassemble(opcode uint16) DisasmEntry {
	if is32BitThumb2(opcode) {
		return DisasmEntry{
			OpcodeHi: opcode,
			Operand:  "32bit Thumb-2",
		}
	}

	df := decodeThumb(opcode)
	if df == nil {
		return DisasmEntry{
			Opcode:  opcode,
			Operand: "error",
		}
	}

	entry := df(nil, opcode)
	if entry == nil {
		return DisasmEntry{
			Opcode:  opcode,
			Operand: "error",
		}
	}

	entry.Opcode = opcode
	entry.Operator = strings.ToLower(entry.Operator)
	return *entry
}

// converts reglist to a string of register names separated by commas
func reglistToMnemonic(regList uint8, suffix string) string {
	s := strings.Builder{}
	comma := false
	for i := 0; i <= 7; i++ {
		if regList&0x01 == 0x01 {
			if comma {
				s.WriteString(",")
			}
			s.WriteString(fmt.Sprintf("R%d", i))
			comma = true
		}
		regList >>= 1
	}

	// push suffix if one has been specified and adding a comma as required
	if suffix != "" {
		if s.Len() > 0 {
			s.WriteString(",")
		}
		s.WriteString(suffix)
	}

	return s.String()
}
