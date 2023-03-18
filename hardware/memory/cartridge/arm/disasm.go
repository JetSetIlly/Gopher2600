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

	// second opcode for 32bit instructions
	Is32bit  bool
	OpcodeLo uint16

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

// String returns a very simple representation of the disassembly entry.
func (e DisasmEntry) String() string {
	if e.Operator == "" {
		return e.Operand
	}
	return fmt.Sprintf("%s %s", e.Operator, e.Operand)
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

// Disassemble a single opcode. True is returned if the instruction is 32bit
// and False if it is 16bit.
func Disassemble(opcode uint16) (DisasmEntry, bool) {
	if is32BitThumb2(opcode) {
		return DisasmEntry{
			Opcode:  opcode,
			Operand: "32bit Thumb-2",
		}, true
	}
	return disassemble(opcode), false
}

func disassemble(opcode uint16) DisasmEntry {
	var f func(opcode uint16) DisasmEntry

	if opcode&0xf000 == 0xf000 {
		// format 19 - Long branch with link
		f = disasmLongBranchWithLink
	} else if opcode&0xf000 == 0xe000 {
		// format 18 - Unconditional branch
		f = disasmUnconditionalBranch
	} else if opcode&0xff00 == 0xdf00 {
		// format 17 - Software interrupt"
		f = disasmSoftwareInterrupt
	} else if opcode&0xf000 == 0xd000 {
		// format 16 - Conditional branch
		f = disasmConditionalBranch
	} else if opcode&0xf000 == 0xc000 {
		// format 15 - Multiple load/store
		f = disasmMultipleLoadStore
	} else if opcode&0xf600 == 0xb400 {
		// format 14 - Push/pop registers
		f = disasmPushPopRegisters
	} else if opcode&0xff00 == 0xb000 {
		// format 13 - Add offset to stack pointer
		f = disasmAddOffsetToSP
	} else if opcode&0xf000 == 0xa000 {
		// format 12 - Load address
		f = disasmLoadAddress
	} else if opcode&0xf000 == 0x9000 {
		// format 11 - SP-relative load/store
		f = disasmSPRelativeLoadStore
	} else if opcode&0xf000 == 0x8000 {
		// format 10 - Load/store halfword
		f = disasmLoadStoreHalfword
	} else if opcode&0xe000 == 0x6000 {
		// format 9 - Load/store with immediate offset
		f = disasmLoadStoreWithImmOffset
	} else if opcode&0xf200 == 0x5200 {
		// format 8 - Load/store sign-extended byte/halfword
		f = disasmLoadStoreSignExtendedByteHalford
	} else if opcode&0xf200 == 0x5000 {
		// format 7 - Load/store with register offset
		f = disasmLoadStoreWithRegisterOffset
	} else if opcode&0xf800 == 0x4800 {
		// format 6 - PC-relative load
		f = disasmPCrelativeLoad
	} else if opcode&0xfc00 == 0x4400 {
		// format 5 - Hi register operations/branch exchange
		f = disasmHiRegisterOps
	} else if opcode&0xfc00 == 0x4000 {
		// format 4 - ALU operations
		f = disasmALUoperations
	} else if opcode&0xe000 == 0x2000 {
		// format 3 - Move/compare/add/subtract immediate
		f = disasmMovCmpAddSubImm
	} else if opcode&0xf800 == 0x1800 {
		// format 2 - Add/subtract
		f = disasmAddSubtract
	} else if opcode&0xe000 == 0x0000 {
		// format 1 - Move shifted register
		f = disasmMoveShiftedRegister
	} else {
		// 16bit thumb-2 instruction
		var operand string
		if opcode&0xff00 == 0xbf00 {
			if opcode&0xff0f == 0xbf00 {
				operand = "16bit Thumb-2"
			} else {
				operand = "16bit Thumb-2"
			}
		} else {
			if opcode&0xff00 == 0xbe00 {
				operand = "16bit Thumb-2"
			} else if opcode&0xff00 == 0xba00 {
				operand = "16bit Thumb-2"
			} else if opcode&0xffe8 == 0xb668 {
				operand = "16bit Thumb-2"
			} else if opcode&0xffe8 == 0xb660 {
				operand = "16bit Thumb-2"
			} else if opcode&0xfff0 == 0xb650 {
				operand = "16bit Thumb-2"
			} else if opcode&0xfff0 == 0xb640 {
				operand = "16bit Thumb-2"
			} else if opcode&0xf600 == 0xb400 {
				operand = "16bit Thumb-2"
			} else if opcode&0xf500 == 0xb100 {
				operand = "16bit Thumb-2"
			} else if opcode&0xff00 == 0xb200 {
				operand = "16bit Thumb-2"
			} else if opcode&0xff00 == 0xb000 {
				operand = "16bit Thumb-2"
			}
		}

		return DisasmEntry{
			Opcode:  opcode,
			Operand: operand,
		}
	}

	e := f(opcode)
	e.Operator = strings.ToLower(e.Operator)
	return e
}

func disasmMoveShiftedRegister(opcode uint16) DisasmEntry {
	// format 1 - Move shifted register

	var entry DisasmEntry
	entry.Opcode = opcode

	op := (opcode & 0x1800) >> 11
	shift := (opcode & 0x7c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	switch op {
	case 0b00:
		entry.Operator = "LSL"
		entry.Operand = fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, shift)
	case 0b01:
		entry.Operator = "LSR"
		entry.Operand = fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, shift)
	case 0b10:
		entry.Operator = "ASR"
		entry.Operand = fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, shift)
	case 0x11:
		panic("illegal instruction")
	}

	return entry
}

func disasmAddSubtract(opcode uint16) DisasmEntry {
	// format 2 - Add/subtract

	var entry DisasmEntry
	entry.Opcode = opcode

	immediate := opcode&0x0400 == 0x0400
	subtract := opcode&0x0200 == 0x0200
	imm := uint32((opcode & 0x01c0) >> 6)
	srcReg := (opcode & 0x038) >> 3
	destReg := opcode & 0x07

	if subtract {
		entry.Operator = "SUB"
		if immediate {
			entry.Operand = fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, imm)
		} else {
			entry.Operand = fmt.Sprintf("R%d, R%d, R%d ", destReg, srcReg, imm)
		}
	} else {
		entry.Operator = "ADD"
		if immediate {
			entry.Operand = fmt.Sprintf("R%d, R%d, #$%02x ", destReg, srcReg, imm)
		} else {
			entry.Operand = fmt.Sprintf("R%d, R%d, R%d ", destReg, srcReg, imm)
		}
	}

	return entry
}

// "The instructions in this group perform operations between a Lo register and
// an 8-bit immediate value".
func disasmMovCmpAddSubImm(opcode uint16) DisasmEntry {
	// format 3 - Move/compare/add/subtract immediate

	var entry DisasmEntry
	entry.Opcode = opcode

	op := (opcode & 0x1800) >> 11
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode & 0x00ff)

	switch op {
	case 0b00:
		entry.Operator = "MOV"
		entry.Operand = fmt.Sprintf("R%d, #$%02x ", destReg, imm)
	case 0b01:
		entry.Operator = "CMP"
		entry.Operand = fmt.Sprintf("R%d, #$%02x ", destReg, imm)
	case 0b10:
		entry.Operator = "ADD"
		entry.Operand = fmt.Sprintf("R%d, #$%02x ", destReg, imm)
	case 0b11:
		entry.Operator = "SUB"
		entry.Operand = fmt.Sprintf("R%d, #$%02x ", destReg, imm)
	}

	return entry
}

// "The following instructions perform ALU operations on a Lo register pair".
func disasmALUoperations(opcode uint16) DisasmEntry {
	// format 4 - ALU operations

	var entry DisasmEntry
	entry.Opcode = opcode

	op := (opcode & 0x03c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	switch op {
	case 0b0000:
		entry.Operator = "AND"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b0001:
		entry.Operator = "EOR"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b0010:
		entry.Operator = "LSL"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b0011:
		entry.Operator = "LSR"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b0100:
		entry.Operator = "ASR"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b0101:
		entry.Operator = "ADC"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b0110:
		entry.Operator = "SBC"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b0111:
		entry.Operator = "ROR"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1000:
		entry.Operator = "TST"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1001:
		entry.Operator = "NEG"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1010:
		entry.Operator = "CMP"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1011:
		entry.Operator = "CMN"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1100:
		entry.Operator = "ORR"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1101:
		entry.Operator = "MUL"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1110:
		entry.Operator = "BIC"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	case 0b1111:
		entry.Operator = "MVN"
		entry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
	default:
		panic(fmt.Sprintf("unimplemented ALU operation (%04b)", op))
	}

	return entry
}

func disasmHiRegisterOps(opcode uint16) DisasmEntry {
	// format 5 - Hi register operations/branch exchange

	var entry DisasmEntry
	entry.Opcode = opcode

	op := (opcode & 0x300) >> 8
	hi1 := opcode&0x80 == 0x80
	hi2 := opcode&0x40 == 0x40
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// labels used to decoraate operands indicating Hi/Lo register usage
	destLabel := "R"
	srcLabel := "R"
	if hi1 {
		destReg += 8
		destLabel = "H"
	}
	if hi2 {
		srcReg += 8
		srcLabel = "H"
	}

	switch op {
	case 0b00:
		entry.Operator = "ADD"
		entry.Operand = fmt.Sprintf("%s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
	case 0b01:
		entry.Operator = "CMP"
		entry.Operand = fmt.Sprintf("%s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
	case 0b10:
		entry.Operator = "MOV"
		entry.Operand = fmt.Sprintf("%s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
	case 0b11:
		// called BLX in ARMv7-M architecture
		entry.Operator = "BX"
		entry.Operand = fmt.Sprintf("%s%d ", srcLabel, srcReg)
	}

	return entry
}

func disasmPCrelativeLoad(opcode uint16) DisasmEntry {
	var entry DisasmEntry
	entry.Opcode = opcode

	// format 6 - PC-relative load
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode&0x00ff) << 2

	entry.Operator = "LDR"
	entry.Operand = fmt.Sprintf("R%d, [PC, #$%02x] ", destReg, imm)

	return entry
}

func disasmLoadStoreWithRegisterOffset(opcode uint16) DisasmEntry {
	// format 7 - Load/store with register offset

	var entry DisasmEntry
	entry.Opcode = opcode

	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x0400 == 0x0400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	if load {
		if byteTransfer {
			entry.Operator = "LDRB"
			entry.Operand = fmt.Sprintf("R%d, [R%d, R%d]", reg, baseReg, offsetReg)
			return entry
		}

		entry.Operator = "LDR"
		entry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		return entry
	}

	if byteTransfer {
		entry.Operator = "STRB"
		entry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		return entry
	}

	entry.Operator = "STR"
	entry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
	return entry
}

func disasmLoadStoreSignExtendedByteHalford(opcode uint16) DisasmEntry {
	// format 8 - Load/store sign-extended byte/halfword

	var entry DisasmEntry
	entry.Opcode = opcode

	hi := opcode&0x0800 == 0x800
	sign := opcode&0x0400 == 0x400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	if sign {
		if hi {
			// load sign-extended halfword
			entry.Operator = "LDSH"
			entry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
			return entry
		}

		// load sign-extended byte
		entry.Operator = "LDSB"
		entry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)

		return entry
	}

	if hi {
		// load halfword
		entry.Operator = "LDRH"
		entry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)

		return entry
	}

	// store halfword
	entry.Operator = "STRH"
	entry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)

	return entry
}

func disasmLoadStoreWithImmOffset(opcode uint16) DisasmEntry {
	// format 9 - Load/store with immediate offset

	var entry DisasmEntry
	entry.Opcode = opcode

	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x1000 == 0x1000
	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "For word accesses (B = 0), the value specified by #Imm is a full 7-bit address, but must
	// be word-aligned (ie with bits 1:0 set to 0), since the assembler places #Imm >> 2 in
	// the Offset5 field." -- ARM7TDMI Data Sheet
	if !byteTransfer {
		offset <<= 2
	}

	if load {
		if byteTransfer {
			entry.Operator = "LDRB"
			entry.Operand = fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset)

			return entry
		}

		entry.Operator = "LDR"
		entry.Operand = fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset)

		return entry
	}

	// store
	if byteTransfer {
		entry.Operator = "STRB"
		entry.Operand = fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset)

		return entry
	}

	entry.Operator = "STR"
	entry.Operand = fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset)

	return entry
}

func disasmLoadStoreHalfword(opcode uint16) DisasmEntry {
	// format 10 - Load/store halfword

	var entry DisasmEntry
	entry.Opcode = opcode

	load := opcode&0x0800 == 0x0800
	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "#Imm is a full 6-bit address but must be halfword-aligned (ie with bit 0 set to 0) since
	// the assembler places #Imm >> 1 in the Offset5 field." -- ARM7TDMI Data Sheet
	offset <<= 1

	if load {
		entry.Operator = "LDRH"
		entry.Operand = fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset)

		return entry
	}

	entry.Operator = "STRH"
	entry.Operand = fmt.Sprintf("R%d, [R%d, #$%02x] ", reg, baseReg, offset)

	return entry
}

func disasmSPRelativeLoadStore(opcode uint16) DisasmEntry {
	// format 11 - SP-relative load/store

	var entry DisasmEntry
	entry.Opcode = opcode

	load := opcode&0x0800 == 0x0800
	reg := (opcode & 0x07ff) >> 8
	offset := uint32(opcode & 0xff)

	// The offset supplied in #Imm is a full 10-bit address, but must always be word-aligned
	// (ie bits 1:0 set to 0), since the assembler places #Imm >> 2 in the Word8 field.
	offset <<= 2

	if load {
		entry.Operator = "LDR"
		entry.Operand = fmt.Sprintf("R%d, [SP, #$%02x] ", reg, offset)

		return entry
	}

	entry.Operator = "STR"
	entry.Operand = fmt.Sprintf("R%d, [SP, #$%02x] ", reg, offset)

	return entry
}

func disasmLoadAddress(opcode uint16) DisasmEntry {
	// format 12 - Load address

	var entry DisasmEntry
	entry.Opcode = opcode

	sp := opcode&0x0800 == 0x800
	destReg := (opcode & 0x700) >> 8
	offset := opcode & 0x00ff

	// offset is a word aligned 10 bit address
	offset <<= 2

	if sp {
		entry.Operator = "ADD"
		entry.Operand = fmt.Sprintf("R%d, [SP, #$%02x] ", destReg, offset)

		return entry
	}

	entry.Operator = "ADD"
	entry.Operand = fmt.Sprintf("R%d, [PC, #$%02x] ", destReg, offset)

	return entry
}

func disasmAddOffsetToSP(opcode uint16) DisasmEntry {
	// format 13 - Add offset to stack pointer

	var entry DisasmEntry
	entry.Opcode = opcode

	sign := opcode&0x80 == 0x80
	imm := uint32(opcode & 0x7f)

	// The offset specified by #Imm can be up to -/+ 508, but must be word-aligned (ie with
	// bits 1:0 set to 0) since the assembler converts #Imm to an 8-bit sign + magnitude
	// number before placing it in field SWord7.
	imm <<= 2

	if sign {
		entry.Operator = "ADD"
		entry.Operand = fmt.Sprintf("SP, -#%d ", imm)

		return entry
	}

	entry.Operator = "ADD"
	entry.Operand = fmt.Sprintf("SP, #$%02x ", imm)

	return entry
}

func disasmPushPopRegisters(opcode uint16) DisasmEntry {
	// format 14 - Push/pop registers

	var entry DisasmEntry
	entry.Opcode = opcode

	load := opcode&0x0800 == 0x0800
	pclr := opcode&0x0100 == 0x0100
	regList := uint8(opcode & 0x00ff)

	// converts reglist to a string of register names separated by commas
	mnemonicFromBits := func(regList uint8) string {
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
		return s.String()
	}

	if load {
		if pclr {
			entry.Operator = "POP"
			entry.Operand = fmt.Sprintf("{%s,PC}", mnemonicFromBits(regList))
		} else {
			entry.Operator = "POP"
			entry.Operand = fmt.Sprintf("{%s}", mnemonicFromBits(regList))
		}

		return entry
	}

	if pclr {
		entry.Operator = "PUSH"
		entry.Operand = fmt.Sprintf("{%s,LR}", mnemonicFromBits(regList))
	} else {
		entry.Operator = "PUSH"
		entry.Operand = fmt.Sprintf("{%s}", mnemonicFromBits(regList))
	}

	return entry
}

func disasmMultipleLoadStore(opcode uint16) DisasmEntry {
	// format 15 - Multiple load/store

	var entry DisasmEntry
	entry.Opcode = opcode

	load := opcode&0x0800 == 0x0800
	baseReg := uint32(opcode&0x07ff) >> 8
	regList := opcode & 0xff

	if load {
		entry.Operator = "LDMIA"
	} else {
		entry.Operator = "STMIA"
	}
	entry.Operand = fmt.Sprintf("R%d!, {%#016b}", baseReg, regList)

	return entry
}

func disasmConditionalBranch(opcode uint16) DisasmEntry {
	// format 16 - Conditional branch

	var entry DisasmEntry
	entry.Opcode = opcode

	cond := (opcode & 0x0f00) >> 8
	offset := uint32(opcode & 0x00ff)

	switch cond {
	case 0b0000:
		entry.Operator = "BEQ"
	case 0b0001:
		entry.Operator = "BNE"
	case 0b0010:
		entry.Operator = "BCS"
	case 0b0011:
		entry.Operator = "BCC"
	case 0b0100:
		entry.Operator = "BMI"
	case 0b0101:
		entry.Operator = "BPL"
	case 0b0110:
		entry.Operator = "BVS"
	case 0b0111:
		entry.Operator = "BVC"
	case 0b1000:
		entry.Operator = "BHI"
	case 0b1001:
		entry.Operator = "BLS"
	case 0b1010:
		entry.Operator = "BGE"
	case 0b1011:
		entry.Operator = "BLT"
	case 0b1100:
		entry.Operator = "BGT"
	case 0b1101:
		entry.Operator = "BLE"
	case 0b1110:
		entry.Operator = "undefined branch"
	case 0b1111:
	}

	entry.Operand = fmt.Sprintf("$%04x", offset)

	return entry
}

func disasmSoftwareInterrupt(opcode uint16) DisasmEntry {
	// format 17 - Software interrupt"
	return DisasmEntry{Opcode: opcode}
}

func disasmUnconditionalBranch(opcode uint16) DisasmEntry {
	// format 18 - Unconditional branch

	var entry DisasmEntry
	entry.Opcode = opcode

	offset := uint32(opcode&0x07ff) << 1

	entry.Operator = "BAL"
	entry.Operand = fmt.Sprintf("$%04x ", offset)

	return entry
}

func disasmLongBranchWithLink(opcode uint16) DisasmEntry {
	// format 19 - Long branch with link

	var entry DisasmEntry
	entry.Opcode = opcode

	low := opcode&0x800 == 0x0800
	offset := uint32(opcode & 0x07ff)

	if low {
		entry.Operator = "BL"
		entry.Operand = fmt.Sprintf("$%08x", offset)

		return entry
	}

	// first bl instruction
	entry.Operator = ""
	entry.Operand = ""

	return entry
}
