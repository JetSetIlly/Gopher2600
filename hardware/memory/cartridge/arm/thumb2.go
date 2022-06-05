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
// And the "ARMv7-M Architecture Reference Manual" (or "ARMv7-M" for brevity)
// can be found at:
//
// https://documentation-service.arm.com/static/606dc36485368c4c2b1bf62f

package arm

import "fmt"

func (arm *ARM) decodeThumb2(opcode uint16) func(uint16) {
	// condition tree built from the table in Figure 3-1 of the "Thumb-2 Supplement"
	//
	// for reference the branches are labelled with the equivalent descriptor
	// in the decodeThumb() function (prepended with ** for clairty). if the
	// name of that group is different in the table in the "Thumb-2 Supplement"
	// then the name is given on the next line
	//
	// note that format 2 and format 5 have been divided into two entries in
	// the Thumb-2 table
	//
	// format 13 and 14 have been combined into one "miscellaneous" entry in
	// the Thumb-2 table
	//
	// where possible the thumb*() instruction is used. this is because the
	// function is well tested already

	if opcode&0xf800 == 0xe800 || opcode&0xf000 == 0xf000 {
		// 32 bit instructions
		f := arm.decodeThumb2Upper32bit(opcode)
		return func(o uint16) {
			arm.function32bit = true
			arm.function32bitFunction = f
			arm.function32bitOpcode = o
		}
	} else {
		if opcode&0xf000 == 0xe000 {
			// ** format 18 Unconditional branch
			return arm.thumbUnconditionalBranch
		} else if opcode&0xff00 == 0xdf00 {
			// ** format 17 Software interrupt"
			// service (system) call
			return arm.thumbSoftwareInterrupt
		} else if opcode&0xff00 == 0xde00 {
			// undefined instruction
			panic(fmt.Sprintf("undefined 16-bit thumb-2 instruction (%04x)", opcode))
		} else if opcode&0xf000 == 0xd000 {
			// ** format 16 Conditional branch
			return arm.thumbConditionalBranch
		} else if opcode&0xf000 == 0xc000 {
			// ** format 15 Multiple load/store
			// load/store multiple
			return arm.thumbMultipleLoadStore
		} else if opcode&0xf000 == 0xb000 {
			// ** format 13/14 Add offset to stack pointer AND Push/pop registers
			// miscellaneous
			return arm.decodeThumb2Miscellaneous(opcode)
		} else if opcode&0xf000 == 0xa000 {
			// ** format 12 Load address
			// add to SP or PC
			return arm.thumbLoadAddress
		} else if opcode&0xf000 == 0x9000 {
			// ** format 11 SP-relative load/store
			// load from or store to stack
			return arm.thumbSPRelativeLoadStore
		} else if opcode&0xf000 == 0x8000 {
			// ** format 10 Load/store halfword
			// load/store halfword with immediate offset
			return arm.thumbLoadStoreHalfword
		} else if opcode&0xe000 == 0x6000 {
			// ** format 9 Load/store with immediate offset
			return arm.thumbLoadStoreWithImmOffset
		} else if opcode&0xf200 == 0x5200 {
			// ** format 8 Load/store sign-extended byte/halfword
			// load/store with register offset
			return arm.thumbLoadStoreSignExtendedByteHalford
		} else if opcode&0xf200 == 0x5000 {
			// ** format 7 Load/store with register offset
			return arm.thumbLoadStoreWithRegisterOffset
		} else if opcode&0xf800 == 0x4800 {
			// ** format 6 PC-relative load
			// load from literal pool
			return arm.thumbPCrelativeLoad
		} else if opcode&0xfc00 == 0x4400 {
			// ** format 5 Hi register operations/branch exchange
			// special data processing AND branch/exchange instruction set
			return arm.thumbHiRegisterOps
		} else if opcode&0xfc00 == 0x4000 {
			// ** format 4 ALU operations
			// data processing register
			return arm.thumbALUoperations
		} else if opcode&0xe000 == 0x2000 {
			// ** format 3 Move/compare/add/subtract immediate
			return arm.thumbMovCmpAddSubImm
		} else if opcode&0xf800 == 0x1800 {
			// ** format 2 Add/subtract
			// add/subtract register AND add/substract immediate
			return arm.thumbAddSubtract
		} else if opcode&0xe000 == 0x0000 {
			// ** format 1 Move shifted register
			// shift by immediate, move register
			return arm.thumbMoveShiftedRegister
		}
	}

	panic(fmt.Sprintf("undecoded 16-bit thumb-2 instruction (%04x)", opcode))
}

func (arm *ARM) decodeThumb2Upper32bit(opcode uint16) func(uint16) {
	if opcode&0xef00 == 0xef00 {
		// coprocessor
		panic("coprocessor")
	} else if opcode&0xf800 == 0xf000 {
		// branches, miscellaneous control
		return arm.thumb2LongBranchWithLink
	} else if opcode&0xfe40 == 0xe800 {
		// load and store multiple, RFE and SRS
		panic("load and store multiple, RFE and SRS")
	} else if opcode&0xfe40 == 0xe840 {
		// load and store double and exclusive and table branch
		panic("load and store double and exclusive and table branch")
	} else if opcode&0xfe00 == 0xf800 {
		// load and store single data item, memory hints
		panic("load and store single data item, memory hints")
	} else if opcode&0xee00 == 0xea00 {
		// data processing, no immediate operand
		panic("data processing, no immediate operand")
	} else if opcode&0xf800 == 0xf000 {
		// data processing: immediate, including bitfield and saturate
		panic("data processing: immediate, including bitfield and saturate")
	}

	panic(fmt.Sprintf("undecoded 32-bit thumb-2 instruction (upper half-word) (%04x)", opcode))
}

func (arm *ARM) thumb2LongBranchWithLink(opcode uint16) {
	// details in "A7.7.18 BL" of "ARMv7-M"

	arm.registers[rLR] = (arm.registers[rPC]-2)&0xfffffffe | 0x00000001

	s := uint32((arm.function32bitOpcode & 0x400) >> 10)
	j1 := uint32((opcode & 0x2000) >> 13)
	j2 := uint32((opcode & 0x800) >> 11)
	i1 := (^(j1 ^ s)) & 0x01
	i2 := (^(j2 ^ s)) & 0x01
	imm10 := uint32(arm.function32bitOpcode & 0x3ff)
	imm11 := uint32(opcode & 0x7ff)
	imm32 := (i1 << 23) | (i2 << 22) | (imm10 << 12) | (imm11 << 1)
	imm32 = imm32 | (s << 24) | (s << 25) | (s << 26) | (s << 27) | (s << 28) | (s << 29) | (s << 30) | (s << 31)
	arm.registers[rPC] += imm32
}

func (arm *ARM) decodeThumb2Miscellaneous(opcode uint16) func(uint16) {
	// condition tree built from the table in Figure 3-2 of the "Thumb-2 Supplement"
	//
	// thumb instruction format 13 and format 14 can be found in this tree

	if opcode&0xff00 == 0xbf00 {
		if opcode&0xff0f == 0xff00 {
			// nop-compatible hints
		} else {
			return arm.thumb2IfThen
		}
	} else {
		if opcode&0xff00 == 0xbe00 {
			// software breakpoint
		} else if opcode&0xff00 == 0xba00 {
			// reverse bytes
		} else if opcode&0xffe8 == 0xb668 {
			panic(fmt.Sprintf("unpredictable 16-bit (miscellaneous) thumb-2 instruction (%04x)", opcode))
		} else if opcode&0xffe8 == 0xb660 {
			// change processor state
		} else if opcode&0xfff0 == 0xb650 {
			// set endianness
		} else if opcode&0xfff0 == 0xb640 {
			panic(fmt.Sprintf("unpredictable 16-bit (miscellaneous) thumb-2 instruction (%04x)", opcode))
		} else if opcode&0xf600 == 0xb400 {
			// ** format 14 Push/pop registers
			// push/pop register list
			return arm.thumbPushPopRegisters
		} else if opcode&0xf500 == 0xb100 {
			// compare and branch on (non-)zero
		} else if opcode&0xff00 == 0xb200 {
			// sign/zero extend
		} else if opcode&0xff00 == 0xb000 {
			// ** format 13 Add offset to stack pointer
			// adjust stack pointer
			return arm.thumbAddOffsetToSP
		}
	}

	panic(fmt.Sprintf("undecoded 16-bit (miscellaneous) thumb-2 instruction (%04x)", opcode))
}

func (arm *ARM) thumb2IfThen(opcode uint16) {
	if arm.Status.itMask != 0b0000 {
		panic("unpredictable IT instruction - already in an IT block")
	}

	arm.Status.itMask = uint8(opcode & 0x000f)
	arm.Status.itCond = uint8((opcode & 0x00f0) >> 4)

	// switch table similar to the one in thumbConditionalBranch()
	switch arm.Status.itCond {
	case 0b1110:
		// any (al)
		if !(arm.Status.itMask == 0x1 || arm.Status.itMask == 0x2 || arm.Status.itMask == 0x4 || arm.Status.itMask == 0x8) {
			// it is not valid to specify an "else" for the "al" condition
			// because it is not possible to negate
			panic("unpredictable IT instruction - else for 'al' condition ")
		}
	case 0b1111:
		panic("unpredictable IT instruction - first condition data is 1111")
	}
}
