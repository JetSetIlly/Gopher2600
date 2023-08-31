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

// The "ARM Architecture Reference Manual Thumb-2 Supplement" referenced in
// this document ("Thumb-2 Supplement" for brevity) can be found at:
//
// https://documentation-service.arm.com/static/5f1066ca0daa596235e7e90a
//
// Where the Thumb-2 Supplement is not clear the "ARMv7-M Architecture Reference
// Manual" (or "ARMv7-M" for brevity) was referenced. For example, in the list
// of load and store functions in Table 3.3.3 of the Thumb-2 Supplement, it is
// not clear which form of the instructions is required.
//
// The "ARMv7-M Architecture Reference Manual" document can be found at:
//
// https://documentation-service.arm.com/static/606dc36485368c4c2b1bf62f

package arm

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/logger"
)

func (arm *ARM) decodeThumb2(opcode uint16) decodeFunction {
	// condition tree built from the table in Figure 3-1 of the "Thumb-2 Supplement"
	//
	// for reference the branches are labelled with the equivalent descriptor
	// in the decodeThumb() function (prepended with ** for clairty). if the
	// name of that group is different in the table in the "Thumb-2 Supplement"
	// then the name is given on the next line
	//
	// where possible the thumb*() instruction is used. this is because the
	// function is well tested already
	//
	// note that format 2 and format 5 have been divided into two entries in
	// the Thumb-2 table (in the reference material). we don't do that here.
	//
	// format 13 and 14 have been combined into one "miscellaneous" entry in
	// the Thumb-2 table. we do the same here and call the miscellanous
	// function. from there the original thumb*() function is called as
	// appropriate

	if opcode&0xf800 == 0xe800 || opcode&0xf000 == 0xf000 {
		// 32 bit instructions
		panic(fmt.Sprintf("32-bit thumb-2 hi opcode (%04x) should have been handled before this", opcode))
	} else {
		if opcode&0xf000 == 0xe000 {
			// ** format 18 Unconditional branch
			return arm.decodeThumbUnconditionalBranch(opcode)
		} else if opcode&0xff00 == 0xdf00 {
			// ** format 17 Software interrupt"
			// service (system) call
			return arm.decodeThumbSoftwareInterrupt(opcode)
		} else if opcode&0xff00 == 0xde00 {
			// undefined instruction
			panic(fmt.Sprintf("undefined 16-bit thumb-2 instruction (%04x)", opcode))
		} else if opcode&0xf000 == 0xd000 {
			// ** format 16 Conditional branch
			return arm.decodeThumbConditionalBranch(opcode)
		} else if opcode&0xf000 == 0xc000 {
			// ** format 15 Multiple load/store
			// load/store multiple
			return arm.decodeThumbMultipleLoadStore(opcode)
		} else if opcode&0xf000 == 0xb000 {
			// ** format 13/14 Add offset to stack pointer AND Push/pop registers
			// miscellaneous
			return arm.decodeThumb2Miscellaneous(opcode)
		} else if opcode&0xf000 == 0xa000 {
			// ** format 12 Load address
			// add to SP or PC
			return arm.decodeThumbLoadAddress(opcode)
		} else if opcode&0xf000 == 0x9000 {
			// ** format 11 SP-relative load/store
			// load from or store to stack
			return arm.decodeThumbSPRelativeLoadStore(opcode)
		} else if opcode&0xf000 == 0x8000 {
			// ** format 10 Load/store halfword
			// load/store halfword with immediate offset
			return arm.decodeThumbLoadStoreHalfword(opcode)
		} else if opcode&0xe000 == 0x6000 {
			// ** format 9 Load/store with immediate offset
			return arm.decodeThumbLoadStoreWithImmOffset(opcode)
		} else if opcode&0xf200 == 0x5200 {
			// ** format 8 Load/store sign-extended byte/halfword
			// load/store with register offset
			return arm.decodeThumbLoadStoreSignExtendedByteHalford(opcode)
		} else if opcode&0xf200 == 0x5000 {
			// ** format 7 Load/store with register offset
			return arm.decodeThumbLoadStoreWithRegisterOffset(opcode)
		} else if opcode&0xf800 == 0x4800 {
			// ** format 6 PC-relative load
			// load from literal pool
			return arm.decodeThumbPCrelativeLoad(opcode)
		} else if opcode&0xfc00 == 0x4400 {
			// ** format 5 Hi register operations/branch exchange
			// special data processing AND branch/exchange instruction set
			return arm.decodeThumbHiRegisterOps(opcode)
		} else if opcode&0xfc00 == 0x4000 {
			// ** format 4 ALU operations
			// data processing register
			return arm.decodeThumbALUoperations(opcode)
		} else if opcode&0xe000 == 0x2000 {
			// ** format 3 Move/compare/add/subtract immediate
			return arm.decodeThumbMovCmpAddSubImm(opcode)
		} else if opcode&0xf800 == 0x1800 {
			// ** format 2 Add/subtract
			// add/subtract register AND add/substract immediate
			return arm.decodeThumbAddSubtract(opcode)
		} else if opcode&0xe000 == 0x0000 {
			// ** format 1 Move shifted register
			// shift by immediate, move register
			return arm.decodeThumbMoveShiftedRegister(opcode)
		}
	}

	panic(fmt.Sprintf("undecoded 16-bit thumb-2 instruction (%04x)", opcode))
}

func (arm *ARM) decodeThumb2Miscellaneous(opcode uint16) decodeFunction {
	// "3.2.1 Miscellaneous Instructions" of "Thumb-2 Supplement"
	// condition tree built from the table in Figure 3-2
	//
	// thumb instruction format 13 and format 14 can be found in this tree

	if opcode&0xff00 == 0xbf00 {
		if opcode&0xff0f == 0xbf00 {
			return arm.decodeThumb2MemoryHints(opcode)
		} else {
			return arm.decodeThumb2IfThen(opcode)
		}
	} else {
		if opcode&0xff00 == 0xbe00 {
			// software breakpoint
			return func() *DisasmEntry {
				arm.state.yield.Type = coprocessor.YieldSyncWithVCS
				arm.state.yield.Error = nil
				return nil
			}
		} else if opcode&0xff00 == 0xba00 {
			return arm.decodeThumb2ReverseBytes(opcode)
		} else if opcode&0xffe8 == 0xb668 {
			panic(fmt.Sprintf("unpredictable 16-bit (miscellaneous) thumb-2 instruction (%04x)", opcode))
		} else if opcode&0xffe8 == 0xb660 {
			return arm.decodeThumb2ChangeProcessorState(opcode)
		} else if opcode&0xfff0 == 0xb650 {
			// set endianness
		} else if opcode&0xfff0 == 0xb640 {
			panic(fmt.Sprintf("unpredictable 16-bit (miscellaneous) thumb-2 instruction (%04x)", opcode))
		} else if opcode&0xf600 == 0xb400 {
			// ** format 14 Push/pop registers
			// push/pop register list
			return arm.decodeThumbPushPopRegisters(opcode)
		} else if opcode&0xf500 == 0xb100 {
			// compare and branch on (non-)zero
			return arm.decodeThumb2CompareAndBranchOnNonZero(opcode)
		} else if opcode&0xff00 == 0xb200 {
			// sign/zero extend
			return arm.decodeThumb2SignZeroExtend(opcode)
		} else if opcode&0xff00 == 0xb000 {
			// ** format 13 Add offset to stack pointer
			// adjust stack pointer
			return arm.decodeThumbAddOffsetToSP(opcode)
		}
	}

	panic(fmt.Sprintf("undecoded 16-bit (miscellaneous) thumb-2 instruction (%04x)", opcode))
}

func (arm *ARM) decodeThumb2ReverseBytes(opcode uint16) decodeFunction {
	opc := (opcode & 0x00c0) >> 6
	Rm := (opcode & 0x0038) >> 3
	Rd := opcode & 0x0007

	switch opc {
	case 0b00:
		return func() *DisasmEntry {
			// "4.6.111 REV" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "REV",
					Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
				}
			}

			v := arm.state.registers[Rm]
			r := ((v & 0x000000ff) << 24) | ((v & 0x0000ff00) << 8) | ((v & 0x00ff0000) >> 8) | ((v & 0xff000000) >> 24)
			arm.state.registers[Rd] = r
			return nil
		}
	case 0b01:
		return func() *DisasmEntry {
			// "4.6.112 REV16" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "REV16",
					Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
				}
			}

			v := arm.state.registers[Rm]
			r := ((v & 0x00ff0000) << 8) | ((v & 0xff000000) >> 8) | ((v & 0x000000ff) << 8) | ((v & 0x0000ff00) >> 8)
			arm.state.registers[Rd] = r
			return nil
		}
	default:
		panic(fmt.Sprintf("unimplemented thumb2 reverse byte instruction (%02b)", opc))
	}
}

func (arm *ARM) decodeThumb2ChangeProcessorState(opcode uint16) decodeFunction {
	return func() *DisasmEntry {
		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "CPSID",
			}
		}
		logger.Logf("ARM7", "CPSID instruction does nothing")
		return nil
	}
}

func (arm *ARM) decodeThumb2MemoryHints(opcode uint16) decodeFunction {
	hint := (opcode & 0x00f0) >> 4

	switch hint {
	case 0b000:
		return func() *DisasmEntry {
			// "4.6.88 NOP" of "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "NOP",
				}
			}
			return nil
		}
	case 0b001:
		panic("unimplemented YIELD instruction")
	case 0b010:
		panic("unimplemented WFE instruction")
	case 0b011:
		panic("unimplemented WFI instruction")
	case 0b100:
		panic("unimplemented SEV instruction")
	}

	panic(fmt.Sprintf("undecoded 16bit (memory hint) thumb-2 instruction (%03b)", hint))
}

func (arm *ARM) decodeThumb2IfThen(opcode uint16) decodeFunction {
	mask := uint8(opcode & 0x000f)
	cond := uint8((opcode & 0x00f0) >> 4)

	return func() *DisasmEntry {
		// "4.6.39 IT" of "Thumb-2 Supplement"

		if arm.decodeOnly {
			_, mnemonic := arm.state.status.condition(cond)
			if len(mnemonic) > 0 {
				mnemonic = mnemonic[1:]
			}
			if len(mnemonic) == 0 {
				mnemonic = "AL"
			}
			return &DisasmEntry{
				Operator: "IT",
				Operand:  strings.ToLower(mnemonic),
			}
		}

		if arm.state.status.itMask != 0b0000 {
			panic("unpredictable IT instruction - already in an IT block")
		}

		arm.state.status.itMask = mask
		arm.state.status.itCond = cond

		// switch table similar to the one in thumbConditionalBranch()
		switch arm.state.status.itCond {
		case 0b1110:
			// any (al)
			if !(arm.state.status.itMask == 0x1 || arm.state.status.itMask == 0x2 || arm.state.status.itMask == 0x4 || arm.state.status.itMask == 0x8) {
				// it is not valid to specify an "else" for the "AL" condition
				// because it is not possible to negate
				panic("unpredictable IT instruction - else for 'AL' condition ")
			}
		case 0b1111:
			panic("unpredictable IT instruction - first condition data is 1111")
		}

		return nil
	}
}

func (arm *ARM) decodeThumb2CompareAndBranchOnNonZero(opcode uint16) decodeFunction {
	// "4.6.22 CBNZ" of "Thumb-2 Supplement"
	//
	// and
	//
	// "4.6.23 CBZ" of "Thumb-2 Supplement"

	nonZero := opcode&0x0800 == 0x0800
	Rn := opcode & 0x0007
	i := (opcode & 0x0200) >> 9
	imm5 := (opcode & 0x00f8) >> 3
	imm32 := (imm5 << 1) | (i << 6)

	if nonZero {
		return func() *DisasmEntry {
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "CBNZ",
					Operand:  fmt.Sprintf("R%d, %d", Rn, imm32),
				}
			}

			if arm.state.registers[Rn] != 0 {
				arm.state.registers[rPC] += uint32(imm32) + 2
			}

			return nil
		}
	}

	return func() *DisasmEntry {
		if arm.decodeOnly {
			return &DisasmEntry{
				Operator: "CBZ",
				Operand:  fmt.Sprintf("R%d, %d", Rn, imm32),
			}
		}

		if arm.state.registers[Rn] == 0 {
			arm.state.registers[rPC] += uint32(imm32) + 2
		}

		return nil
	}
}

func (arm *ARM) decodeThumb2SignZeroExtend(opcode uint16) decodeFunction {
	op := (opcode & 0xc0) >> 6
	Rm := (opcode & 0x38) >> 3
	Rd := opcode & 0x07

	switch op {
	case 0b00:
		return func() *DisasmEntry {
			// "4.6.187 SXTH" in "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "SXTH",
					Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
				}
			}

			arm.state.registers[Rd] = arm.state.registers[Rm] & 0x0000ffff
			if arm.state.registers[Rd]&0x8000 == 0x8000 {
				arm.state.registers[Rd] |= 0xffff0000
			}
			return nil
		}
	case 0b01:
		return func() *DisasmEntry {
			// "4.6.185 SXTB" in "Thumb-2 Supplement"
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "SXTB",
					Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
				}
			}

			arm.state.registers[Rd] = arm.state.registers[Rm] & 0x000000ff
			if arm.state.registers[Rd]&0x80 == 0x80 {
				arm.state.registers[Rd] |= 0xffffff00
			}
			return nil
		}
	case 0b10:
		// unsigned extend halfword
		return func() *DisasmEntry {
			// "4.6.226 UXTH" in "Thumb-2 Supplement"
			// T1 Encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "UXTH",
					Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
				}
			}

			arm.state.registers[Rd] = arm.state.registers[Rm] & 0x0000ffff
			return nil
		}
	case 0b11:
		return func() *DisasmEntry {
			// unsigned extend byte UXTB
			// "4.6.224 UXTB" in "Thumb-2 Supplement"
			// T1 Encoding
			if arm.decodeOnly {
				return &DisasmEntry{
					Operator: "UXTB",
					Operand:  fmt.Sprintf("R%d, R%d", Rd, Rm),
				}
			}

			arm.state.registers[Rd] = arm.state.registers[Rm] & 0x000000ff
			return nil
		}
	}

	panic(fmt.Sprintf("unhandled sign/zero extend instruction (op %02b)", op))
}
