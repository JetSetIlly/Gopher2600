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

	// see comment about panic recovery above
	decode := func(opcode uint16) *DisasmEntry {
		defer func() {
			recover()
		}()

		var df decodeFunction

		if arm.state.instruction32bitDecoding {
			arm.state.instruction32bitDecoding = false
			arm.state.instruction32bitResolving = true
			df = arm.decode32bitThumb2(arm.state.instruction32bitOpcodeHi, opcode)
			if df == nil {
				return nil
			}
		} else {
			df = arm.decodeThumb2(opcode)
			if df == nil {
				return nil
			}
		}
		return df()
	}

	for ptr := 0; ptr < len(config.Data); {
		opcode := config.ByteOrder.Uint16(config.Data[ptr:])

		if !arm.state.instruction32bitDecoding && is32BitThumb2(opcode) {
			arm.state.instruction32bitDecoding = true
			arm.state.instruction32bitOpcodeHi = opcode

			// we do no advance instructionPC even though we're shortcircuiting
			// the for loop. the value of instructionPC will be advanced by 4
			// once the full 32bit decoding has completed

			ptr += 2
			continue // for loop
		}

		e := decode(opcode)
		if e != nil {
			arm.completeDisasmEntry(e, opcode, false)
			config.Callback(*e)
		}

		if arm.state.instruction32bitResolving {
			arm.state.instruction32bitResolving = false
			arm.state.instructionPC += 4
		} else {
			arm.state.instructionPC += 2
		}
		ptr += 2
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

// converts shift type value to a suitable mnemonic string
func shiftTypeToMnemonic(typ uint16) string {
	switch typ {
	case 0b00:
		return "LSL"
	case 0b01:
		return "LSR"
	case 0b10:
		return "ASR"
	}
	panic("impossible shift type")
}

// converts reglist to a string of register names separated by commas. does not
// add the surrounding braces
func reglistToMnemonic(regPrefix rune, regList uint8, suffix string) string {
	s := strings.Builder{}
	comma := false
	for i := 0; i <= 7; i++ {
		if regList&0x01 == 0x01 {
			if comma {
				s.WriteString(",")
			}
			s.WriteString(fmt.Sprintf("%c%d", regPrefix, i))
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

// adds the S suffix for a instructions that have an optional 'set flags' stage
func setFlagsMnemonic(setFlags bool) string {
	if setFlags {
		return "S"
	}
	return ""
}

// return branch target as a string. target is specified as an offset, the
// function will apply the offset to the correct PC value
func (arm *ARM) branchTargetOffsetFromPC(offset int64) string {
	return arm.branchTarget(uint32(int64(arm.state.registers[rPC]-2) + offset))
}

// return branch target address as a string
func (arm *ARM) branchTarget(addr uint32) string {
	return fmt.Sprintf("%08x", addr)
}
