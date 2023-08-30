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

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

// disassemble according to the current ARM architecture
func (arm *ARM) disassemble(opcode uint16) (DisasmEntry, error) {
	var entry DisasmEntry
	var err error

	switch arm.mmap.ARMArchitecture {
	case architecture.ARM7TDMI:
		entry, err = arm.disassembleARM7TDMI(opcode)
		if err != nil {
			return DisasmEntry{}, err
		}
	case architecture.ARMv7_M:
		entry, err = arm.disassembleARM7vM(opcode)
		if err != nil {
			return DisasmEntry{}, err
		}
	default:
		panic(fmt.Sprintf("unhandled ARM architecture: %s", arm.mmap.ARMArchitecture))
	}

	return entry, nil
}

func (arm *ARM) disassembleARM7TDMI(opcode uint16) (DisasmEntry, error) {
	arm.decodeOnly = true
	defer func() {
		arm.decodeOnly = false
	}()

	df := arm.decodeThumb(opcode)
	if df == nil {
		return DisasmEntry{}, fmt.Errorf("error decoding thumb instruction during disassembly")
	}

	e := df()
	if e == nil {
		return DisasmEntry{}, fmt.Errorf("error executing thumb instruction during disassembly")
	}

	fillDisasmEntry(arm, e, opcode)

	return *e, nil
}

func (arm *ARM) disassembleARM7vM(opcode uint16) (DisasmEntry, error) {
	arm.decodeOnly = true
	defer func() {
		arm.decodeOnly = false
	}()

	var e *DisasmEntry

	if is32BitThumb2(arm.state.function32bitOpcodeHi) {
		df := arm.decodeThumb2(arm.state.function32bitOpcodeHi)
		if df == nil {
			return DisasmEntry{}, fmt.Errorf("error decoding 32bit thumb-2 instruction during disassembly: %04x %04x",
				arm.state.function32bitOpcodeHi, opcode)
		}

		e = df()
		if e == nil {
			e = &DisasmEntry{
				Operand: "32bit instruction",
				Is32bit: true,
			}
		}

	} else {
		df := arm.decodeThumb2(opcode)
		if df == nil {
			return DisasmEntry{}, fmt.Errorf("error decoding 16bit thumb2 instruction during disassembly: %04x", opcode)
		}
		e = df()
		if e == nil {
			return DisasmEntry{}, fmt.Errorf("error executing 16bit thumb2 instruction during disassembly: %04x", opcode)
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
// For disassemblies of executed code see the coprocessor.CartCoProcDisassembler interface
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
			return arm.disassembleARM7vM(opcode)
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

// disasmVerbose provides more detail for the disasm entry
func (arm *ARM) disasmVerbose(entry DisasmEntry) string {
	var s strings.Builder

	s.WriteString(fmt.Sprintf("instruction PC: %08x\n", entry.Addr))
	if entry.Is32bit {
		s.WriteString(fmt.Sprintf("opcode: %04x %04x \n", entry.OpcodeHi, entry.Opcode))
	} else {
		s.WriteString(fmt.Sprintf("opcode: %04x       \n", entry.Opcode))
	}

	// register information for verbose output
	for i, r := range arm.state.registers {
		s.WriteString(fmt.Sprintf("\tR%02d: %08x", i, r))
		if (i+1)%4 == 0 {
			s.WriteString(fmt.Sprintf("\n"))
		}
	}

	return s.String()
}
