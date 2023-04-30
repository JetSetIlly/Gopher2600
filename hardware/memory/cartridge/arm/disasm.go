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
	"encoding/binary"
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
	Registers [NumRegisters]uint32

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

// fillDisasmEntry sets the DisasmEntry fields using information from the
// emulated ARM. the other fields are not touched so can be set before or after
// calling the function
func fillDisasmEntry(arm *ARM, e *DisasmEntry, opcode uint16) {
	e.Addr = arm.state.instructionPC
	e.Opcode = opcode
	e.Is32bit = arm.state.function32bitDecoding
	e.OpcodeHi = arm.state.function32bitOpcodeHi
	e.Address = fmt.Sprintf("%08x", arm.state.instructionPC)
	e.Operator = strings.ToLower(e.Operator)
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

func (arm *ARM) disassemble(opcode uint16) (DisasmEntry, error) {
	arm.decodeOnly = true
	defer func() {
		arm.decodeOnly = false
	}()

	df := arm.decodeThumb(opcode)
	if df == nil {
		return DisasmEntry{}, fmt.Errorf("error decoding instruction during disassembly")
	}

	e := df(opcode)
	if e == nil {
		return DisasmEntry{}, fmt.Errorf("error decoding instruction during disassembly")
	}

	fillDisasmEntry(arm, e, opcode)

	return *e, nil
}

func (arm *ARM) disassembleThumb2(opcode uint16) (DisasmEntry, error) {
	arm.decodeOnly = true
	defer func() {
		arm.decodeOnly = false
	}()

	var e *DisasmEntry

	if is32BitThumb2(arm.state.function32bitOpcodeHi) {
		df := arm.decodeThumb2(arm.state.function32bitOpcodeHi)
		if df == nil {
			return DisasmEntry{}, fmt.Errorf("error decoding instruction during disassembly")
		}

		e = df(opcode)
		if e == nil {
			e = &DisasmEntry{
				Operand: "32bit instruction",
			}
		}

	} else {
		df := arm.decodeThumb2(opcode)
		if df == nil {
			return DisasmEntry{}, fmt.Errorf("error decoding instruction during disassembly")
		}
		e = df(opcode)
		if e == nil {
			return DisasmEntry{}, fmt.Errorf("error decoding instruction during disassembly")
		}
	}

	fillDisasmEntry(arm, e, opcode)

	return *e, nil
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

// StaticDisassembleConfig is used to set the parameters for a static disassembly
type StaticDisassembleConfig struct {
	Data      []byte
	Origin    uint32
	ByteOrder binary.ByteOrder
	Callback  func(DisasmEntry)
}

// StaticDisassemble is used to statically disassemble a block of memory. It is
// assumed that there is a valid instruction at the start of the block
//
// For disassemblies of executed code see the mapper.CartCoProcDisassembler interface
func StaticDisassemble(config StaticDisassembleConfig) error {
	arm := &ARM{
		state: &ARMState{
			instructionPC: config.Origin,
		},
		byteOrder:  config.ByteOrder,
		decodeOnly: true,
	}

	// because we're disassembling data that may contain non-executable
	// instructions (think of the jump tables between functions, for example) it
	// is likely that we'll encounter a panic raised during instruction
	// decoding
	//
	// panics are useful to have in our implementation because it forces us to
	// notice and to tackle the problem of unimplemented instructions. however,
	// as stated, it is not useful to panic during disassembly
	//
	// it is necessary therefore, to catch panics and to recover()

	for ptr := 0; ptr < len(config.Data); {
		opcode := config.ByteOrder.Uint16(config.Data[ptr:])

		if !arm.state.function32bitDecoding && is32BitThumb2(opcode) {
			arm.state.function32bitOpcodeHi = opcode
			arm.state.function32bitDecoding = true
			ptr += 2
			continue // for loop
		}

		// see comment about panic recovery above
		e, err := func() (e DisasmEntry, err error) {
			defer func() {
				if r := recover(); r != nil {
					// there has been an error but we still want to create a DisasmEntry
					// that can be used in a disassembly output
					var e DisasmEntry
					fillDisasmEntry(arm, &e, opcode)

					// the Operator and Operand fields are used for error information
					e.Operator = DisasmEntryErrorOperator
					e.Operand = fmt.Sprintf("%v", r)
				}
			}()
			return arm.disassembleThumb2(opcode)
		}()

		if err == nil {
			config.Callback(e)
		}

		if arm.state.function32bitDecoding {
			arm.state.instructionPC += 4
		} else {
			arm.state.instructionPC += 2
		}
		ptr += 2

		// reset 32bit fields
		arm.state.function32bitOpcodeHi = 0
		arm.state.function32bitDecoding = false
	}

	return nil
}
