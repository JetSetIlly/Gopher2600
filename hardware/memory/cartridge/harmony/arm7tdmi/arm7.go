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

package arm7tdmi

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/logger"
)

type status struct {
	// CPSR (current program status register) bits
	negative bool
	zero     bool
	overflow bool
	carry    bool
}

func (sr *status) reset() {
	sr.negative = false
	sr.zero = false
	sr.overflow = false
	sr.carry = false
}

func (sr *status) setNegative(a uint32) {
	sr.negative = a&0x80000000 == 0x80000000
}

func (sr *status) setZero(a uint32) {
	sr.zero = a == 0x00
}

func (sr *status) setOverflow(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d >>= 31
	e := (c & 1) + ((a >> 31) & 1) + ((b >> 31) & 1)
	e >>= 31
	sr.overflow = (d^e)&0x01 == 0x01
}

func (sr *status) setCarry(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d = (d >> 31) + (a >> 31) + (b >> 31)
	sr.carry = d&0x02 == 0x02
}

// Memory defines the memory that can be accessed from the ARM type. Field
// names mirror the names used in the DPC+ static CartStatic implementation.
type Memory struct {
	Driver *[]byte
	Custom *[]byte
	Data   *[]byte
	Freq   *[]byte
}

func (mem *Memory) mapAddr(addr uint32) (*[]byte, uint32) {
	// driver ARM code (ROM copy)
	if addr < customOrigin {
		return mem.Driver, addr
	}

	// custom ARM code (ROM copy)
	if addr < driverOrigin {
		return mem.Custom, addr - customOrigin
	}

	// driver ARM code (RAM copy)
	if addr < dataOrigin {
		return mem.Driver, addr - driverOrigin
	}

	// data
	if addr < freqOrigin {
		return mem.Data, addr - dataOrigin
	}

	// frequency table
	if addr < mamOrigin {
		return mem.Freq, addr - freqOrigin
	}

	return nil, addr
}

// register names
const (
	r0 = iota
	r1
	r2
	r3
	r4
	r5
	r6
	r7
	r8
	r9
	r10
	r11
	r12
	rSP
	rLR
	rPC
	rCount
)

// ARM implements the ARM7TDMI-S LPC2103 processor.
type ARM struct {
	mem Memory

	// offset from memory origin for PC on reset
	resetOffset uint32

	thumb bool

	status    status
	registers [rCount]uint32
}

const (
	customOrigin = 0x00000c00
	driverOrigin = 0x40000000
	dataOrigin   = 0x40000c00
	freqOrigin   = 0x40001c00
	stackOrigin  = 0x40001fdc
	mamOrigin    = 0xe0000000
)

// NewARM is the preferred method of initialisation for the ARM type.
func NewARM(mem Memory, resetOffset uint32) *ARM {
	arm := &ARM{
		mem:         mem,
		resetOffset: resetOffset,
	}
	arm.reset()
	return arm
}

func (arm *ARM) reset() {
	arm.status.reset()
	for i := range arm.registers {
		arm.registers[i] = 0x00000000
	}
	arm.registers[rSP] = stackOrigin
	arm.registers[rLR] = customOrigin
	arm.registers[rPC] = (customOrigin + arm.resetOffset + 2)
}

func (arm *ARM) String() string {
	s := strings.Builder{}
	for i, r := range arm.registers {
		if i > 0 {
			if i%4 == 0 {
				s.WriteString("\n")
			} else {
				s.WriteString("\t\t")
			}
		}
		s.WriteString(fmt.Sprintf("R%-2d: %#08x", i, r))
	}
	return s.String()
}

func (arm *ARM) SetParameter(data uint8) {
	logger.Log("ARM7", fmt.Sprintf("function parameter (%02x)", data))
}

func (arm *ARM) CallFunction(data uint8) error {
	switch data {
	case 0xfe:
	case 0xff:
		arm.reset()
		for {
			if !arm.executeInstruction() {
				break
			}
		}
	default:
	}

	return nil
}

func (arm *ARM) read32bit(addr uint32) uint32 {
	var mem *[]uint8
	mem, addr = arm.mem.mapAddr(addr)
	if mem == nil {
		return 0
	}

	b1 := uint32((*mem)[addr])
	b2 := uint32((*mem)[addr+1]) << 8
	b3 := uint32((*mem)[addr+2]) << 16
	b4 := uint32((*mem)[addr+3]) << 24

	return b1 | b2 | b3 | b4
}

func (arm *ARM) write8bit(addr uint32, val uint8) {
	var mem *[]uint8
	mem, addr = arm.mem.mapAddr(addr)
	if mem == nil {
		return
	}

	(*mem)[addr] = val
}

func (arm *ARM) write32bit(addr uint32, val uint32) {
	var mem *[]uint8
	mem, addr = arm.mem.mapAddr(addr)
	if mem == nil {
		return
	}

	(*mem)[addr] = uint8(val)
	(*mem)[addr+1] = uint8(val >> 8)
	(*mem)[addr+2] = uint8(val >> 16)
	(*mem)[addr+3] = uint8(val >> 24)
}

func (arm *ARM) read8bit(addr uint32) uint8 {
	var mem *[]uint8
	mem, addr = arm.mem.mapAddr(addr)
	if mem == nil {
		return 0
	}

	return uint8((*mem)[addr])
}

func (arm *ARM) read16bitPC() uint16 {
	pc := arm.registers[rPC] - 2

	var mem *[]uint8
	mem, pc = arm.mem.mapAddr(pc)
	if mem == nil {
		return 0
	}

	b1 := uint16((*mem)[pc])
	b2 := uint16((*mem)[pc+1]) << 8

	arm.registers[rPC] += 2

	return b1 | b2
}

func (arm *ARM) executeInstruction() bool {
	cont := true

	pc := arm.registers[rPC] - 2
	opcode := arm.read16bitPC()

	fmt.Printf("\n%04x: %016b %04x ", pc, opcode, opcode)

	var operation string

	// working backwards up the table in Figure 5-1 of the THUMB instruction set reference
	if opcode&0xf000 == 0xf000 {
		// format 19
		operation = "Long branch with link"
		arm.executeLongBranchWithLink(opcode)
	} else if opcode&0xf000 == 0xe000 {
		// format 18
		operation = "Unconditional branch"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xff00 == 0xdf00 {
		// format 17
		operation = "Software Interrupt"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xf000 == 0xd000 {
		// format 16
		operation = "Conditional branch"
		arm.executeConditionalBranch(opcode)
	} else if opcode&0xf000 == 0xc000 {
		// format 15
		operation = "Multiple load/store"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xf600 == 0xb400 {
		// format 14
		operation = "Push/pop registers"
		arm.executePushPopRegisters(opcode)
	} else if opcode&0xff00 == 0xb000 {
		// format 13
		operation = "Add offset to stack pointer"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xf000 == 0xa000 {
		// format 12
		operation = "Load address"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xf000 == 0x9000 {
		// format 11
		operation = "SP-relative load/store"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xf000 == 0x8000 {
		// format 10
		operation = "Load/store halfword"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xe000 == 0x6000 {
		// format 9
		operation = "Load/store with immediate offset"
		arm.executeLoadStoreWithImmOffset(opcode)
	} else if opcode&0xf200 == 0x5200 {
		// format 8
		operation = "Load/store sign-extended byte/halfword"
		fmt.Printf("| %-38s ", operation)
	} else if opcode&0xf200 == 0x5000 {
		// format 7
		operation = "Load/store with register offset"
		arm.executeLoadStoreWithRegisterOffset(opcode)
	} else if opcode&0xf800 == 0x4800 {
		// format 6
		operation = "PC-relative load"
		arm.executePCrelativeLoad(opcode)
	} else if opcode&0xfc00 == 0x4400 {
		// format 5
		operation = "Hi register operations/branch exchange"
		cont = arm.executeHiRegisterOps(opcode)
	} else if opcode&0xfc00 == 0x4000 {
		// format 4
		operation = "ALU operations"
		arm.executeALUoperations(opcode)
	} else if opcode&0xe000 == 0x2000 {
		// format 3
		operation = "Move/compare/add/subtract immediate"
		arm.executeMovCmpAddSubImm(opcode)
	} else if opcode&0xf800 == 0x1800 {
		// format 2
		operation = "Add/subtract"
		arm.executeAddSubtract(opcode)
	} else if opcode&0xe000 == 0x0000 {
		// format 1
		operation = "Move shifted register"
		arm.executeMoveShiftedRegister(opcode)
	} else {
		panic("undecoded instruction")
	}

	return cont
}

func (arm *ARM) executeMoveShiftedRegister(opcode uint16) {
	// format 1
	op := (opcode & 0x1800) >> 11
	sourceReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07
	imm := (opcode & 0x7c0) >> 6

	v := arm.registers[destReg]

	switch op {
	case 0x00:
		fmt.Printf("| LSL R%d, R%d, #%d ", destReg, sourceReg, imm)
		arm.registers[destReg] = arm.registers[sourceReg] << imm
	case 0x01:
		fmt.Printf("| LSR R%d, R%d, #%d ", destReg, sourceReg, imm)
		arm.registers[destReg] = arm.registers[sourceReg] >> imm
	case 0x10:
		fmt.Printf("| ASR R%d, R%d, #%d ", destReg, sourceReg, imm)
		sr := arm.registers[sourceReg]
		if sr&0x80000000 == 0x8000000 {
			arm.registers[destReg] = arm.registers[sourceReg] >> imm
		} else {
			arm.registers[destReg] = arm.registers[sourceReg] >> imm
			// sign extend
			for i := uint16(0); i < imm; i++ {
				arm.registers[destReg] |= 0x80000000 >> i
			}
		}
	case 0x11:
		panic("illegal instruction")
	}

	arm.status.setZero(arm.registers[destReg])
	arm.status.setCarry(arm.registers[destReg], v, 0)
	arm.status.setOverflow(arm.registers[destReg], v, 0)
	arm.status.setNegative(arm.registers[destReg])
}

func (arm *ARM) executeAddSubtract(opcode uint16) {
	// format 2
	immediate := opcode&0x0400 == 0x0400
	subtract := opcode&0x0200 == 0x0200

	sourceReg := (opcode & 0x038) >> 3
	destReg := opcode & 0x07

	offset := uint32((opcode & 0x01c0) >> 6)
	val := offset
	if !immediate {
		val = arm.registers[offset]
	}

	v := arm.registers[destReg]

	if subtract {
		arm.registers[destReg] = arm.registers[sourceReg] - val
		arm.status.setCarry(arm.registers[destReg], v, 0)
		arm.status.setOverflow(arm.registers[destReg], v, 0)

		if immediate {
			fmt.Printf("| SUB R%d, R%d, #%d ", destReg, sourceReg, val)
		} else {
			fmt.Printf("| SUB R%d, R%d, R%d ", destReg, sourceReg, offset)
		}
	} else {
		arm.registers[destReg] = arm.registers[sourceReg] + val
		arm.status.setCarry(arm.registers[destReg], v, 0)
		arm.status.setOverflow(arm.registers[destReg], v, 0)

		if immediate {
			fmt.Printf("| ADD R%d, R%d, #%d ", destReg, sourceReg, val)
		} else {
			fmt.Printf("| ADD R%d, R%d, R%d ", destReg, sourceReg, offset)
		}
	}

	arm.status.setZero(arm.registers[destReg])
	arm.status.setNegative(arm.registers[destReg])
}

func (arm *ARM) executeMovCmpAddSubImm(opcode uint16) {
	// format 3
	op := (opcode & 0x1800) >> 11
	destReg := (opcode & 0x0700) >> 8
	imm := opcode & 0x00ff

	v := arm.registers[destReg]

	switch op {
	case 0b00:
		// mov
		arm.registers[destReg] = uint32(imm)
		arm.status.setZero(arm.registers[destReg])
		arm.status.setNegative(arm.registers[destReg])
		fmt.Printf("| MOV R%d, #%d ", destReg, imm)
	case 0b01:
		// cmp
		cmp := arm.registers[destReg] - uint32(imm)
		arm.status.setNegative(cmp)
		arm.status.setZero(cmp)
		arm.status.setCarry(arm.registers[destReg], uint32(imm)^0xffffffff, 1)
		arm.status.setOverflow(arm.registers[destReg], uint32(imm)^0xffffffff, 1)
		fmt.Printf("| CMP R%d, #%d ", destReg, imm)
	case 0b10:
		// add
		arm.registers[destReg] += uint32(imm)
		arm.status.setZero(arm.registers[destReg])
		arm.status.setNegative(arm.registers[destReg])
		arm.status.setCarry(arm.registers[destReg], v, 0)
		arm.status.setOverflow(arm.registers[destReg], v, 0)
		fmt.Printf("| ADD R%d, #%d ", destReg, imm)
	case 0b11:
		// sub
		arm.registers[destReg] -= uint32(imm)
		arm.status.setZero(arm.registers[destReg])
		arm.status.setNegative(arm.registers[destReg])
		arm.status.setCarry(arm.registers[destReg], uint32(imm)^0xffffffff, 1)
		arm.status.setOverflow(arm.registers[destReg], uint32(imm)^0xffffffff, 1)
		fmt.Printf("| SUB R%d, #%d ", destReg, imm)
	}

}

func (arm *ARM) executeALUoperations(opcode uint16) {
	// format 4
	op := (opcode & 0x03c0) >> 6
	sourceReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	v := arm.registers[destReg]

	switch op {
	case 0b0000:
		fmt.Printf("| AND R%d, R%d ", destReg, sourceReg)
		arm.registers[destReg] &= arm.registers[sourceReg]
	case 0b1100:
		fmt.Printf("| ORR R%d, R%d ", destReg, sourceReg)
		arm.registers[destReg] |= arm.registers[sourceReg]
	case 0b1110:
		fmt.Printf("| BIC R%d, R%d ", destReg, sourceReg)
		arm.registers[destReg] &= (arm.registers[sourceReg] ^ 0xffffffff)
	default:
		panic("unimplemented ALU operation")
	}

	arm.status.setZero(arm.registers[destReg])
	arm.status.setNegative(arm.registers[destReg])
	arm.status.setCarry(arm.registers[destReg], v, 0)
	arm.status.setOverflow(arm.registers[destReg], v, 0)
}

func (arm *ARM) executeHiRegisterOps(opcode uint16) bool {
	// format 5
	op := (opcode & 0x300) >> 8
	hi1 := opcode&0x80 == 0x80
	hi2 := opcode&0x40 == 0x40
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

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
		fmt.Printf("| ADD %s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)

		v := arm.registers[destReg]

		// two's complement?? doesn't say so in the spec
		arm.registers[destReg] += arm.registers[srcReg]
		arm.status.setZero(arm.registers[destReg])
		arm.status.setNegative(arm.registers[destReg])
		arm.status.setCarry(arm.registers[destReg], v, 0)
		arm.status.setOverflow(arm.registers[destReg], v, 0)
	case 0b01:
		fmt.Printf("| CMP %s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
		arm.status.zero = arm.registers[destReg] == arm.registers[srcReg]
	case 0b10:
		// status registers not set in this case
		fmt.Printf("| MOV %s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
		arm.registers[destReg] = arm.registers[srcReg]
	case 0b11:
		thumbMode := arm.registers[srcReg]&0x01 == 0x01

		var newPC uint32

		// If R15 is used as an operand, the value will be the address of the instruction + 4 with
		// bit 0 cleared. Executing a BX PC in THUMB state from a non-word aligned address
		// will result in unpredictable execution.
		if srcReg == 15 {
			// PC is already +2 from the instruction address
			newPC = arm.registers[rPC] + 2
		} else {
			newPC = (arm.registers[srcReg] & 0x7ffffffe) + 2
		}

		fmt.Printf("| BX %s%d ", srcLabel, srcReg)
		fmt.Printf("| PC=%#08x ", arm.registers[rPC])

		if thumbMode {
			fmt.Printf("[thumb]")
			arm.registers[rPC] = newPC
		} else {
			fmt.Printf("[arm]")

			// end of custom code
			return false
		}
	}

	return true
}

func (arm *ARM) executePCrelativeLoad(opcode uint16) {
	// // format 6
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode&0x00ff) << 2

	// notes says that "PC will be 4 bytes great than the instruction". we've
	// already advanced by 2; the additional 2 is to account for the prefetch
	pc := arm.registers[rPC]

	// pc must be word aligned
	pc &= 0xfffffffc

	fmt.Printf("| LDR R%d, [PC, #%d] ", destReg, imm)

	if imm&0x100 == 0x100 {
		// two's complement
		imm ^= 0x1ff
		arm.registers[destReg] = arm.read32bit(pc - imm)
	} else {
		arm.registers[destReg] = arm.read32bit(pc + imm)
	}
}

func (arm *ARM) executeLoadStoreWithRegisterOffset(opcode uint16) {
	// format 7
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x0400 == 0x0400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	addr := arm.registers[baseReg] + arm.registers[offsetReg]

	if load {
		if byteTransfer {
			fmt.Printf("| LDRB R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
			arm.registers[reg] = uint32(arm.read8bit(uint32(addr)))
		}
		fmt.Printf("| LDR R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		arm.registers[reg] = arm.read32bit(uint32(addr))
		return
	}

	if byteTransfer {
		fmt.Printf("| STRB R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		arm.write8bit(addr, uint8(arm.registers[reg]))
		return
	}

	fmt.Printf("| STR R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
	arm.write32bit(addr, arm.registers[reg])
}

func (arm *ARM) executeLoadStoreWithImmOffset(opcode uint16) {
	// format 9
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x1000 == 0x1000

	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// offset is 7-bit address with shifted in zero bits (meaning the offset is
	// word aligned)
	offset <<= 2

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[baseReg] + uint32(offset)

	if load {
		if byteTransfer {
			fmt.Printf("| LDRB R%d, [R%d, #%d] ", reg, baseReg, offset)
			arm.registers[reg] = uint32(arm.read8bit(addr))
			return
		}
		fmt.Printf("| LDR R%d, [R%d, #%d] ", reg, baseReg, offset)
		arm.registers[reg] = arm.read32bit(addr)
		return
	}

	// store
	if byteTransfer {
		fmt.Printf("| STRB R%d, [R%d, #%d] ", reg, baseReg, offset)
		arm.write8bit(addr, uint8(arm.registers[reg]))
		return
	}

	fmt.Printf("| STR R%d, [R%d, #%d] ", reg, baseReg, offset)
	arm.write32bit(addr, arm.registers[reg])
}

func (arm *ARM) executePushPopRegisters(opcode uint16) {
	// format 14

	// the ARM pushes registers in descending order and pops in ascending
	// order. in other words the LR is pushed first and PC is popped last

	load := opcode&0x0800 == 0x0800
	pclr := opcode&0x0100 == 0x0100
	regList := opcode & 0x00ff

	if load {
		if pclr {
			fmt.Printf("| POP {%#0b, PC} ", regList)
		} else {
			fmt.Printf("| POP {%#0b} ", regList)
		}

		// pop in ascending order
		for i := 0; i <= 7; i++ {
			r := regList >> i
			if r&0x01 == 0x01 {
				mem, addr := arm.mem.mapAddr(arm.registers[rSP])
				addr++
				// not checking for nil
				v0 := uint32((*mem)[addr]) << 24
				v1 := uint32((*mem)[addr+1]) << 16
				v2 := uint32((*mem)[addr+2]) << 8
				v3 := uint32((*mem)[addr+3])
				arm.registers[i] = v0 | v1 | v2 | v3
				arm.registers[rSP] += 4
			}
		}

		if pclr {
			mem, addr := arm.mem.mapAddr(arm.registers[rSP])
			// not checking for nil
			v0 := uint32((*mem)[addr]) << 24
			v1 := uint32((*mem)[addr+1]) << 16
			v2 := uint32((*mem)[addr+2]) << 8
			v3 := uint32((*mem)[addr+3])
			arm.registers[rPC] = v0 | v1 | v2 | v3
			arm.registers[rSP] += 4
		}

		return
	}

	// store
	if pclr {
		mem, addr := arm.mem.mapAddr(arm.registers[rSP])
		// not checking for nil
		val := arm.registers[rLR]
		(*mem)[addr] = uint8(val)
		(*mem)[addr-1] = uint8(val >> 8)
		(*mem)[addr-2] = uint8(val >> 16)
		(*mem)[addr-3] = uint8(val >> 24)
		arm.registers[rSP] -= 4
		fmt.Printf("| PUSH {%#0b, LR}", regList)
	} else {
		fmt.Printf("| PUSH {%#0b}", regList)
	}

	// push in descending order
	for i := 7; i >= 0; i-- {
		r := regList >> i
		if r&0x01 == 0x01 {
			mem, addr := arm.mem.mapAddr(arm.registers[rSP])
			// not checking for nil
			val := arm.registers[i]
			(*mem)[addr] = uint8(val)
			(*mem)[addr-1] = uint8(val >> 8)
			(*mem)[addr-2] = uint8(val >> 16)
			(*mem)[addr-3] = uint8(val >> 24)
			arm.registers[rSP] -= 4
		}
	}
}

func (arm *ARM) executeConditionalBranch(opcode uint16) {
	// format 16
	cond := (opcode & 0x0f00) >> 8
	offset := uint32(opcode & 0x00ff)

	operand := ""
	branch := false

	switch cond {
	case 0b0000:
		operand = "BEQ"
		branch = arm.status.zero
	case 0b0001:
		operand = "BNE"
		branch = !arm.status.zero
	case 0b0010:
		operand = "BCS"
		branch = arm.status.carry
	case 0b0011:
		operand = "BCC"
		branch = !arm.status.carry
	case 0b0100:
		operand = "BMI"
		branch = arm.status.negative
	case 0b0101:
		operand = "BPL"
		branch = !arm.status.negative
	case 0b0110:
		operand = "BVS"
		branch = arm.status.overflow
	case 0b0111:
		operand = "BVC"
		branch = !arm.status.overflow
	case 0b1000:
		operand = "BHI"
		branch = arm.status.carry && !arm.status.zero
	case 0b1001:
		operand = "BLS"
		branch = !arm.status.carry && arm.status.zero
	case 0b1010:
		operand = "BGE"
		branch = arm.status.negative == arm.status.overflow
	case 0b1011:
		operand = "BLT"
		branch = arm.status.negative != arm.status.overflow
	case 0b1100:
		operand = "BGT"
		branch = !arm.status.zero && arm.status.negative == arm.status.overflow
	case 0b1101:
		operand = "BLE"
		branch = arm.status.zero || arm.status.negative != arm.status.overflow
	case 0b1110:
		operand = "undefined branch"
		branch = true
	case 0b1111:
		branch = false
	}

	// offset is a nine-bit two's complement value
	offset <<= 1
	offset++

	var newPC uint32

	// get new PC value
	if offset&0x80 == 0x80 {
		// two's complement before subtraction
		offset ^= 0xff
		offset++
		newPC = arm.registers[rPC] - offset
	} else {
		newPC = arm.registers[rPC] + offset
	}

	// disassembly
	fmt.Printf("| %s %04x ", operand, newPC)

	// do branch
	if branch {
		arm.registers[rPC] = newPC + 1
	} else {
		fmt.Printf("| no branch ")
	}
}

func (arm *ARM) executeLongBranchWithLink(opcode uint16) {
	// format 19
	low := opcode&0x800 == 0x0800
	offset := uint32(opcode & 0x07ff)

	if low {
		// second instruction
		offset <<= 1
		arm.registers[rLR] += offset
		pc := arm.registers[rPC]
		arm.registers[rPC] = arm.registers[rLR]
		arm.registers[rLR] = pc - 1
		fmt.Printf("| BL %#08x", arm.registers[rPC])
		return
	}

	// first instruction
	offset <<= 12

	if offset&0x400000 == 0x400000 {
		// two's complement before subtraction
		offset ^= 0x7fffff
		offset++
		arm.registers[rLR] = arm.registers[rPC] - offset + 2
	} else {
		arm.registers[rLR] = arm.registers[rPC] + offset + 2
	}
}
