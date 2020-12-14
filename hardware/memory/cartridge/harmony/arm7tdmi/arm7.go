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
	"math/bits"
	"strings"
)

// SharedMemory represents the memory passed between the parent
// cartridge-mapper implementation and the ARM.
type SharedMemory interface {
	// Return memory block and array offset for the requested address. Memory
	// blocks mays be different for read and write operations.
	MapAddress(addr uint32, write bool) (*[]byte, uint32)

	// Return reset addreses for the Stack Pointer register; the Link Register;
	// and Program Counter
	ResetVectors() (uint32, uint32, uint32)
}

type status struct {
	// CPSR (current program status register) bits
	negative bool
	zero     bool
	overflow bool
	carry    bool
}

func (sr *status) String() string {
	s := strings.Builder{}
	if sr.negative {
		s.WriteRune('N')
	} else {
		s.WriteRune('n')
	}
	if sr.zero {
		s.WriteRune('Z')
	} else {
		s.WriteRune('z')
	}
	if sr.overflow {
		s.WriteRune('V')
	} else {
		s.WriteRune('v')
	}
	if sr.carry {
		s.WriteRune('C')
	} else {
		s.WriteRune('c')
	}
	return s.String()
}

func (sr *status) reset() {
	sr.negative = false
	sr.zero = false
	sr.overflow = false
	sr.carry = false
}

func (sr *status) isNegative(a uint32) {
	sr.negative = a&0x80000000 == 0x80000000
}

func (sr *status) isZero(a uint32) {
	sr.zero = a == 0x00
}

func (sr *status) setOverflow(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d >>= 31
	e := (d & 0x01) + ((a >> 31) & 0x01) + ((b >> 31) & 0x01)
	e >>= 1
	sr.overflow = (d^e)&0x01 == 0x01
}

func (sr *status) setCarry(a, b, c uint32) {
	d := (a & 0x7fffffff) + (b & 0x7fffffff) + c
	d = (d >> 31) + (a >> 31) + (b >> 31)
	sr.carry = d&0x02 == 0x02
}

// register names
const (
	rSP = 13 + iota
	rLR
	rPC
	rCount
)

// ARM implements the ARM7TDMI-S LPC2103 processor.
type ARM struct {
	mem SharedMemory

	// memory for values that fall outside of the areas defined above. map
	// access is slower than array/slice access but the potential range of
	// scratch addresses is very large and there's no way of knowning how much
	// space is required ahead of time. a map is good compromise.
	scratch map[uint32]byte

	status    status
	registers [rCount]uint32

	noDebugNext bool
}

// NewARM is the preferred method of initialisation for the ARM type.
func NewARM(mem SharedMemory) *ARM {
	arm := &ARM{
		mem: mem,
	}
	arm.reset()

	arm.scratch = make(map[uint32]byte)

	return arm
}

func (arm *ARM) reset() {
	arm.status.reset()
	for i := range arm.registers {
		arm.registers[i] = 0x00000000
	}
	arm.registers[rSP], arm.registers[rLR], arm.registers[rPC] = arm.mem.ResetVectors()

	// a perculiarity of the ARM is that the PC is 2 bytes ahead of where we'll
	// be reading from. adjust PC so that this is correct.
	arm.registers[rPC] += 2
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
		s.WriteString(fmt.Sprintf("R%-2d: %08x", i, r))
	}
	return s.String()
}

// func (arm *ARM) stack() string {
// 	o := arm.registers[rSP] - freqOriginRAM
// 	return fmt.Sprintf("%v", (*arm.mem.Freq)[o:stackOriginRAM-freqOriginRAM])
// }

func (arm *ARM) Run() error {
	arm.reset()
	for arm.executeInstruction() {
	}
	return nil
}

func (arm *ARM) read8bit(addr uint32) uint8 {
	var mem *[]uint8
	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		return arm.scratch[addr]
	}
	return (*mem)[addr]
}

func (arm *ARM) write8bit(addr uint32, val uint8) {
	var mem *[]uint8
	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		arm.scratch[addr] = val
		return
	}
	(*mem)[addr] = val
}

func (arm *ARM) read16bit(addr uint32) uint16 {
	var mem *[]uint8
	mem, addr = arm.mem.MapAddress(addr, false)

	if mem == nil {
		b1 := uint16(arm.scratch[addr])
		b2 := uint16(arm.scratch[addr+1]) << 8
		return b1 | b2
	}

	b1 := uint16((*mem)[addr])
	b2 := uint16((*mem)[addr+1]) << 8
	return b1 | b2
}

func (arm *ARM) write16bit(addr uint32, val uint16) {
	var mem *[]uint8
	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		arm.scratch[addr] = uint8(val)
		arm.scratch[addr+1] = uint8(val >> 8)
		return
	}
	(*mem)[addr] = uint8(val)
	(*mem)[addr+1] = uint8(val >> 8)
}

func (arm *ARM) read32bit(addr uint32) uint32 {
	var mem *[]uint8
	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		b1 := uint32(arm.scratch[addr])
		b2 := uint32(arm.scratch[addr+1]) << 8
		b3 := uint32(arm.scratch[addr+2]) << 16
		b4 := uint32(arm.scratch[addr+3]) << 24
		return b1 | b2 | b3 | b4
	}
	b1 := uint32((*mem)[addr])
	b2 := uint32((*mem)[addr+1]) << 8
	b3 := uint32((*mem)[addr+2]) << 16
	b4 := uint32((*mem)[addr+3]) << 24

	return b1 | b2 | b3 | b4
}

func (arm *ARM) write32bit(addr uint32, val uint32) {
	var mem *[]uint8
	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		arm.scratch[addr] = uint8(val)
		arm.scratch[addr+1] = uint8(val >> 8)
		arm.scratch[addr+2] = uint8(val >> 16)
		arm.scratch[addr+3] = uint8(val >> 24)
		return
	}
	(*mem)[addr] = uint8(val)
	(*mem)[addr+1] = uint8(val >> 8)
	(*mem)[addr+2] = uint8(val >> 16)
	(*mem)[addr+3] = uint8(val >> 24)
}

func (arm *ARM) read16bitPC() uint16 {
	pc := arm.registers[rPC] - 2

	var mem *[]uint8
	mem, pc = arm.mem.MapAddress(pc, false)
	if mem == nil {
		return 0
	}
	b1 := uint16((*mem)[pc])
	b2 := uint16((*mem)[pc+1]) << 8

	arm.registers[rPC] += 2

	return b1 | b2
}

func (arm *ARM) executeInstruction() bool {
	// fmt.Println()
	if arm.noDebugNext {
		arm.noDebugNext = false
	} else {
		// fmt.Println(arm.String())
		// fmt.Println(arm.status.String())
	}

	cont := true

	// pc := arm.registers[rPC] - 2
	opcode := arm.read16bitPC()

	// fmt.Printf("\n>>> %04x :: %08x :: ", pc, opcode)
	// // fmt.Printf(" [40000ce0=%04x]", arm.read16bit(0x40000ce0))
	// // fmt.Printf(" [40000ce1=%04x]", arm.read16bit(0x40000ce1))
	// fmt.Printf(" [40000ce2=%04x]", arm.read16bit(0x40000ce2))
	// // fmt.Printf(" [40000ce3=%04x]", arm.read16bit(0x40000ce3))
	// // fmt.Printf(" [40000ce4=%04x]", arm.read16bit(0x40000ce4))

	// working backwards up the table in Figure 5-1 of the THUMB instruction set reference
	if opcode&0xf000 == 0xf000 {
		// format 19 - Long branch with link
		arm.executeLongBranchWithLink(opcode)
	} else if opcode&0xf000 == 0xe000 {
		// format 18 - Unconditional branch
		arm.executeUnconditionalBranch(opcode)
	} else if opcode&0xff00 == 0xdf00 {
		// format 17 - Software interrupt"
		arm.executeSoftwareInterrupt(opcode)
	} else if opcode&0xf000 == 0xd000 {
		// format 16 - Conditional branch
		arm.executeConditionalBranch(opcode)
	} else if opcode&0xf000 == 0xc000 {
		// format 15 - Multiple load/store
		arm.executeMultipleLoadStore(opcode)
	} else if opcode&0xf600 == 0xb400 {
		// format 14 - Push/pop registers
		arm.executePushPopRegisters(opcode)
	} else if opcode&0xff00 == 0xb000 {
		// format 13 - Add offset to stack pointer
		arm.executeAddOffsetToSP(opcode)
	} else if opcode&0xf000 == 0xa000 {
		// format 12 - Load address
		arm.executeLoadAddress(opcode)
	} else if opcode&0xf000 == 0x9000 {
		// format 11 - SP-relative load/store
		arm.executeSPRelativeLoadStore(opcode)
	} else if opcode&0xf000 == 0x8000 {
		// format 10 - Load/store halfword
		arm.executeLoadStoreHalfword(opcode)
	} else if opcode&0xe000 == 0x6000 {
		// format 9 - Load/store with immediate offset
		arm.executeLoadStoreWithImmOffset(opcode)
	} else if opcode&0xf200 == 0x5200 {
		// format 8 - Load/store sign-extended byte/halfword
		arm.executeLoadStoreSignExtendedByteHalford(opcode)
	} else if opcode&0xf200 == 0x5000 {
		// format 7 - Load/store with register offset
		arm.executeLoadStoreWithRegisterOffset(opcode)
	} else if opcode&0xf800 == 0x4800 {
		// format 6 - PC-relative load
		arm.executePCrelativeLoad(opcode)
	} else if opcode&0xfc00 == 0x4400 {
		// format 5 - Hi register operations/branch exchange
		cont = arm.executeHiRegisterOps(opcode)
	} else if opcode&0xfc00 == 0x4000 {
		// format 4 - ALU operations
		arm.executeALUoperations(opcode)
	} else if opcode&0xe000 == 0x2000 {
		// format 3 - Move/compare/add/subtract immediate
		arm.executeMovCmpAddSubImm(opcode)
	} else if opcode&0xf800 == 0x1800 {
		// format 2 - Add/subtract
		arm.executeAddSubtract(opcode)
	} else if opcode&0xe000 == 0x0000 {
		// format 1 - Move shifted register
		arm.executeMoveShiftedRegister(opcode)
	} else {
		panic("undecoded instruction")
	}

	return cont
}

func (arm *ARM) executeMoveShiftedRegister(opcode uint16) {
	// format 1 - Move shifted register
	op := (opcode & 0x1800) >> 11
	shift := (opcode & 0x7c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// in this class of operation the src register may also be the dest
	// register so we need to make a note of the value before it is
	// overwrittten
	src := arm.registers[srcReg]

	switch op {
	case 0b00:
		// fmt.Printf("| LSL R%d, R%d, #%02x ", destReg, srcReg, shift)

		// if immed_5 == 0
		//	C Flag = unaffected
		//	Rd = Rm
		// else /* immed_5 > 0 */
		//	C Flag = Rm[32 - immed_5]
		//	Rd = Rm Logical_Shift_Left immed_5

		if shift == 0 {
			arm.registers[destReg] = arm.registers[srcReg]
		} else {
			m := uint32(0x01) << (32 - shift)
			arm.status.carry = src&m == m
			arm.registers[destReg] = arm.registers[srcReg] << shift
		}
	case 0b01:
		// fmt.Printf("| LSR R%d, R%d, #%02x ", destReg, srcReg, shift)

		// if immed_5 == 0
		//		C Flag = Rm[31]
		//		Rd = 0
		// else /* immed_5 > 0 */
		//		C Flag = Rm[immed_5 - 1]
		//		Rd = Rm Logical_Shift_Right immed_5

		if shift == 0 {
			arm.status.carry = arm.registers[srcReg]&0x80000000 == 0x80000000
			arm.registers[destReg] = 0x00
		} else {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			arm.registers[destReg] = arm.registers[srcReg] >> shift
		}
	case 0b10:
		// fmt.Printf("| ASR R%d, R%d, #%02x ", destReg, srcReg, shift)

		// if immed_5 == 0
		//		C Flag = Rm[31]
		//		if Rm[31] == 0 then
		//				Rd = 0
		//		else /* Rm[31] == 1 */]
		//				Rd = 0xFFFFFFFF
		// else /* immed_5 > 0 */
		//		C Flag = Rm[immed_5 - 1]
		//		Rd = Rm Arithmetic_Shift_Right immed_5

		if shift == 0 {
			arm.status.carry = arm.registers[srcReg]&0x80000000 == 0x80000000
			if arm.status.carry {
				arm.registers[destReg] = 0xffffffff
			} else {
				arm.registers[destReg] = 0x00000000
			}
		} else { // shift > 0
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			arm.registers[destReg] = uint32(int32(arm.registers[srcReg]) >> shift)
		}

	case 0x11:
		panic("illegal instruction")
	}

	arm.status.isZero(arm.registers[destReg])
	arm.status.isNegative(arm.registers[destReg])
}

func (arm *ARM) executeAddSubtract(opcode uint16) {
	// format 2 - Add/subtract
	immediate := opcode&0x0400 == 0x0400
	subtract := opcode&0x0200 == 0x0200
	imm := uint32((opcode & 0x01c0) >> 6)
	srcReg := (opcode & 0x038) >> 3
	destReg := opcode & 0x07

	val := imm
	if !immediate {
		val = arm.registers[imm]
	}

	if subtract {
		if immediate {
			// fmt.Printf("| SUB R%d, R%d, #%02x ", destReg, srcReg, val)
		} else {
			// fmt.Printf("| SUB R%d, R%d, R%d ", destReg, srcReg, imm)
		}
		arm.status.setCarry(arm.registers[srcReg], ^val, 1)
		arm.status.setOverflow(arm.registers[srcReg], ^val, 1)
		arm.registers[destReg] = arm.registers[srcReg] - val
	} else {
		if immediate {
			// fmt.Printf("| ADD R%d, R%d, #%02x ", destReg, srcReg, val)
		} else {
			// fmt.Printf("| ADD R%d, R%d, R%d ", destReg, srcReg, imm)
		}
		arm.status.setCarry(arm.registers[srcReg], val, 0)
		arm.status.setOverflow(arm.registers[srcReg], val, 0)
		arm.registers[destReg] = arm.registers[srcReg] + val
	}

	arm.status.isZero(arm.registers[destReg])
	arm.status.isNegative(arm.registers[destReg])
}

// "The instructions in this group perform operations between a Lo register and
// an 8-bit immediate value."
func (arm *ARM) executeMovCmpAddSubImm(opcode uint16) {
	// format 3 - Move/compare/add/subtract immediate
	op := (opcode & 0x1800) >> 11
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode & 0x00ff)

	switch op {
	case 0b00:
		// fmt.Printf("| MOV R%d, #%02x ", destReg, imm)
		arm.registers[destReg] = imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b01:
		// fmt.Printf("| CMP R%d, #%02x ", destReg, imm)
		arm.status.setCarry(arm.registers[destReg], ^imm, 1)
		arm.status.setOverflow(arm.registers[destReg], ^imm, 1)
		cmp := arm.registers[destReg] - imm
		arm.status.isNegative(cmp)
		arm.status.isZero(cmp)
	case 0b10:
		// fmt.Printf("| ADD R%d, #%02x ", destReg, imm)
		arm.status.setCarry(arm.registers[destReg], imm, 0)
		arm.status.setOverflow(arm.registers[destReg], imm, 0)
		arm.registers[destReg] += imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b11:
		// fmt.Printf("| SUB R%d, #%02x ", destReg, imm)
		arm.status.setCarry(arm.registers[destReg], ^imm, 1)
		arm.status.setOverflow(arm.registers[destReg], ^imm, 1)
		arm.registers[destReg] -= imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	}
}

// "The following instructions perform ALU operations on a Lo register pair."
func (arm *ARM) executeALUoperations(opcode uint16) {
	// format 4 - ALU operations
	op := (opcode & 0x03c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// fmt.Printf("%04b", op)

	switch op {
	case 0b0000:
		// fmt.Printf("| AND R%d, R%d ", destReg, srcReg)
		arm.registers[destReg] &= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0001:
		// fmt.Printf("| EOR R%d, R%d ", destReg, srcReg)
		arm.registers[destReg] ^= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0010:
		// fmt.Printf("| LSL R%d, R%d ", destReg, srcReg)

		// if Rs[7:0] == 0
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[7:0] < 32 then
		//		C Flag = Rd[32 - Rs[7:0]]
		//		Rd = Rd Logical_Shift_Left Rs[7:0]
		// else if Rs[7:0] == 32 then
		//		C Flag = Rd[0]
		//		Rd = 0
		// else /* Rs[7:0] > 32 */
		//		C Flag = 0
		//		Rd = 0
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected

		shift := arm.registers[srcReg]

		if shift > 0 && shift < 32 {
			m := uint32(0x01) << (32 - shift)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] <<= shift
		} else if shift == 32 {
			arm.status.carry = arm.registers[destReg]&0x01 == 0x01
			arm.registers[destReg] = 0x00
		} else if shift > 32 {
			arm.status.carry = false
			arm.registers[destReg] = 0x00
		}

		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0011:
		// fmt.Printf("| LSR R%d, R%d ", destReg, srcReg)

		// if Rs[7:0] == 0 then
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[7:0] < 32 then
		//		C Flag = Rd[Rs[7:0] - 1]
		//		Rd = Rd Logical_Shift_Right Rs[7:0]
		// else if Rs[7:0] == 32 then
		//		C Flag = Rd[31]
		//		Rd = 0
		// else /* Rs[7:0] > 32 */
		//		C Flag = 0
		//		Rd = 0
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected

		shift := arm.registers[srcReg]

		if shift > 0 && shift < 32 {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] >>= shift
		} else if shift == 32 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
			arm.registers[destReg] = 0x00
		} else if shift > 32 {
			arm.status.carry = false
			arm.registers[destReg] = 0x00
		}

		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0100:
		// fmt.Printf("| ASR R%d, R%d ", destReg, srcReg)

		// if Rs[7:0] == 0 then
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[7:0] < 32 then
		//		C Flag = Rd[Rs[7:0] - 1]
		//		Rd = Rd Arithmetic_Shift_Right Rs[7:0]
		// else /* Rs[7:0] >= 32 */
		//		C Flag = Rd[31]
		//		if Rd[31] == 0 then
		//			Rd = 0
		//		else /* Rd[31] == 1 */
		//			Rd = 0xFFFFFFFF
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected
		shift := arm.registers[srcReg]
		if shift > 0 && shift < 32 {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] = uint32(int32(arm.registers[destReg]) >> shift)
		} else if shift >= 32 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
			if !arm.status.carry {
				arm.registers[destReg] = 0x00
			} else {
				arm.registers[destReg] = 0xffffffff
			}
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0101:
		// fmt.Printf("| ADC R%d, R%d ", destReg, srcReg)
		if arm.status.carry {
			arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 1)
			arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 1)
			arm.registers[destReg] += arm.registers[srcReg]
			arm.registers[destReg]++
		} else {
			arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 0)
			arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 0)
			arm.registers[destReg] += arm.registers[srcReg]
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0110:
		// fmt.Printf("| SBC R%d, R%d ", destReg, srcReg)
		if !arm.status.carry {
			arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 0)
			arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 0)
			arm.registers[destReg] -= arm.registers[srcReg]
			arm.registers[destReg]--
		} else {
			arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
			arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
			arm.registers[destReg] -= arm.registers[srcReg]
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0111:
		// fmt.Printf("| ROR R%d, R%d ", destReg, srcReg)

		// if Rs[7:0] == 0 then
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[4:0] == 0 then
		//		C Flag = Rd[31]
		//		Rd = unaffected
		// else /* Rs[4:0] > 0 */
		//		C Flag = Rd[Rs[4:0] - 1]
		//		Rd = Rd Rotate_Right Rs[4:0]
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected
		shift := arm.registers[srcReg]
		if shift&0xff == 0 {
			// unaffected
		} else if shift&0x1f == 0 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
		} else {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] = bits.RotateLeft32(arm.registers[destReg], -int(shift))
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1000:
		// fmt.Printf("| TST R%d, R%d ", destReg, srcReg)
		w := arm.registers[destReg] & arm.registers[srcReg]
		arm.status.isZero(w)
		arm.status.isNegative(w)
	case 0b1001:
		// fmt.Printf("| NEG R%d, R%d ", destReg, srcReg)
		arm.status.setCarry(0, ^arm.registers[srcReg], 1)
		arm.status.setOverflow(0, ^arm.registers[srcReg], 1)
		arm.registers[destReg] = -arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1010:
		// fmt.Printf("| CMP R%d, R%d ", destReg, srcReg)
		arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
		arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
		cmp := arm.registers[destReg] - arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b1011:
		// fmt.Printf("| CMN R%d, R%d ", destReg, srcReg)
		arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 0)
		arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 0)
		cmp := arm.registers[destReg] + arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b1100:
		// fmt.Printf("| ORR R%d, R%d ", destReg, srcReg)
		arm.registers[destReg] |= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1101:
		// fmt.Printf("| MUL R%d, R%d ", destReg, srcReg)
		arm.registers[destReg] *= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1110:
		// fmt.Printf("| BIC R%d, R%d ", destReg, srcReg)
		arm.registers[destReg] &= ^arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1111:
		// fmt.Printf("| MVN R%d, R%d ", destReg, srcReg)
		arm.registers[destReg] = ^arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	default:
		panic(fmt.Sprintf("unimplemented ALU operation (%04b)", op))
	}

}

func (arm *ARM) executeHiRegisterOps(opcode uint16) bool {
	// format 5 - Hi register operations/branch exchange
	op := (opcode & 0x300) >> 8
	hi1 := opcode&0x80 == 0x80
	hi2 := opcode&0x40 == 0x40
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// destLabel := "R"
	// srcLabel := "R"
	if hi1 {
		destReg += 8
		// destLabel = "H"
	}
	if hi2 {
		srcReg += 8
		// srcLabel = "H"
	}

	switch op {
	case 0b00:
		// fmt.Printf("| ADD %s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)

		// not two's complement
		arm.registers[destReg] += arm.registers[srcReg]

		// status register not changed
	case 0b01:
		// fmt.Printf("| CMP %s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)

		// alu_out = Rn - Rm
		// N Flag = alu_out[31]
		// Z Flag = if alu_out == 0 then 1 else 0
		// C Flag = NOT BorrowFrom(Rn - Rm)
		// V Flag = OverflowFrom(Rn - Rm)

		arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 0)
		arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 0)
		cmp := arm.registers[destReg] - arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b10:
		// fmt.Printf("| MOV %s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
		arm.registers[destReg] = arm.registers[srcReg]
		// status register not changed
	case 0b11:
		// fmt.Printf("| BX %s%d ", srcLabel, srcReg)

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

		if thumbMode {
			// fmt.Printf("[thumb]")
			arm.registers[rPC] = newPC
			// fmt.Printf("| PC=%#08x ", arm.registers[rPC])
		} else {
			// fmt.Printf("[arm]")

			// end of custom code
			return false
		}
	}

	return true
}

func (arm *ARM) executePCrelativeLoad(opcode uint16) {
	// format 6 - PC-relative load
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode&0x00ff) << 2

	// "Bit 1 of the PC value is forced to zero for the purpose of this
	// calculation, so the address is always word-aligned."
	pc := arm.registers[rPC] & 0xfffffffc

	// fmt.Printf("| LDR R%d, [PC, #%02x] ", destReg, imm)
	// fmt.Printf("| %08x ", pc)

	// immediate value is not two's complement (surprisingly)
	arm.registers[destReg] = arm.read32bit(pc + imm)
}

func (arm *ARM) executeLoadStoreWithRegisterOffset(opcode uint16) {
	// format 7 - Load/store with register offset
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x0400 == 0x0400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	addr := arm.registers[baseReg] + arm.registers[offsetReg]

	if load {
		if byteTransfer {
			// fmt.Printf("| LDRB R%d, [R%d, R%d]", reg, baseReg, offsetReg)
			arm.registers[reg] = uint32(arm.read8bit(addr))
			return
		}
		// fmt.Printf("| LDR R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		arm.registers[reg] = arm.read32bit(addr)
		return
	}

	if byteTransfer {
		// fmt.Printf("| STRB R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		arm.write8bit(addr, uint8(arm.registers[reg]))
		return
	}

	// fmt.Printf("| STR R%d, [R%d, R%d] ", reg, baseReg, offsetReg)

	arm.write32bit(addr, arm.registers[reg])
}

func (arm *ARM) executeLoadStoreSignExtendedByteHalford(opcode uint16) {
	// format 8 - Load/store sign-extended byte/halfword
	hi := opcode&0x0800 == 0x800
	sign := opcode&0x0400 == 0x400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	addr := arm.registers[baseReg] + arm.registers[offsetReg]

	if sign {
		if hi {
			// load sign-extended halfword
			// fmt.Printf("| LDSH R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
			arm.registers[reg] = uint32(arm.read16bit(addr))
			if arm.registers[reg]&0x8000 == 0x8000 {
				arm.registers[reg] |= 0xffff0000
			}
			return
		}
		// load sign-extended byte
		// fmt.Printf("| LDSB R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		arm.registers[reg] = uint32(arm.read8bit(addr))
		if arm.registers[reg]&0x0080 == 0x0080 {
			arm.registers[reg] |= 0xffffff00
		}
		return
	}

	if hi {
		// load halfword
		// fmt.Printf("| LDRH R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		arm.registers[reg] = uint32(arm.read16bit(addr))
		return
	}

	// store halfword
	// fmt.Printf("| STRH R%d, [R%d, R%d] ", reg, baseReg, offsetReg)

	arm.write16bit(addr, uint16(arm.registers[reg]))
}

func (arm *ARM) executeLoadStoreWithImmOffset(opcode uint16) {
	// format 9 - Load/store with immediate offset
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

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[baseReg] + uint32(offset)

	if load {
		if byteTransfer {
			// fmt.Printf("| LDRB R%d, [R%d, #%02x] ", reg, baseReg, offset)
			arm.registers[reg] = uint32(arm.read8bit(addr))
			// fmt.Printf("addr=%08x ", addr)
			return
		}
		// fmt.Printf("| LDR R%d, [R%d, #%02x] ", reg, baseReg, offset)
		arm.registers[reg] = arm.read32bit(addr)
		return
	}

	// store
	if byteTransfer {
		// fmt.Printf("| STRB R%d, [R%d, #%02x] ", reg, baseReg, offset)
		arm.write8bit(addr, uint8(arm.registers[reg]))
		return
	}

	// fmt.Printf("| STR R%d, [R%d, #%02x] ", reg, baseReg, offset)

	arm.write32bit(addr, arm.registers[reg])
}

func (arm *ARM) executeLoadStoreHalfword(opcode uint16) {
	// format 10 - Load/store halfword
	load := opcode&0x0800 == 0x0800
	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "#Imm is a full 6-bit address but must be halfword-aligned (ie with bit 0 set to 0) since
	// the assembler places #Imm >> 1 in the Offset5 field." -- ARM7TDMI Data Sheet
	offset <<= 1

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[baseReg] + uint32(offset)

	if load {
		// fmt.Printf("| LDRH R%d, [R%d, #%02x] ", reg, baseReg, offset)
		arm.registers[reg] = uint32(arm.read16bit(addr))
		return
	}

	// fmt.Printf("| STRH R%d, [R%d, #%02x] ", reg, baseReg, offset)

	arm.write16bit(addr, uint16(arm.registers[reg]))
}

func (arm *ARM) executeSPRelativeLoadStore(opcode uint16) {
	// format 11 - SP-relative load/store
	load := opcode&0x0800 == 0x0800
	reg := (opcode & 0x07ff) >> 8
	offset := uint32(opcode & 0xff)

	// The offset supplied in #Imm is a full 10-bit address, but must always be word-aligned
	// (ie bits 1:0 set to 0), since the assembler places #Imm >> 2 in the Word8 field.
	offset <<= 2

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[rSP] + offset

	if load {
		// fmt.Printf("| LDR R%d, [SP, #%02x] ", reg, offset)
		arm.registers[reg] = arm.read32bit(addr)
		return
	}

	// fmt.Printf("| STR R%d, [SP, #%02x] ", reg, offset)

	arm.write32bit(addr, arm.registers[reg])
}

func (arm *ARM) executeLoadAddress(opcode uint16) {
	// format 12 - Load address
	sp := opcode&0x0800 == 0x800
	destReg := (opcode & 0x700) >> 8
	offset := opcode & 0x00ff

	// offset is a word aligned 10 bit address
	offset <<= 2

	if sp {
		arm.registers[destReg] = arm.registers[rSP] + uint32(offset)
		return
	}

	arm.registers[destReg] = arm.registers[rPC] + uint32(offset)
}

func (arm *ARM) executeAddOffsetToSP(opcode uint16) {
	// format 13 - Add offset to stack pointer
	sign := opcode&0x80 == 0x80
	imm := uint32(opcode & 0x7f)

	// The offset specified by #Imm can be up to -/+ 508, but must be word-aligned (ie with
	// bits 1:0 set to 0) since the assembler converts #Imm to an 8-bit sign + magnitude
	// number before placing it in field SWord7.
	imm <<= 2

	if sign {
		// fmt.Printf("| ADD SP, #-%d ", imm)
		arm.registers[rSP] -= imm
		return
	}

	// fmt.Printf("| ADD SP, #%02x ", imm)

	arm.registers[rSP] += imm

	// status register not changed
}

func (arm *ARM) executePushPopRegisters(opcode uint16) {
	// format 14 - Push/pop registers

	// the ARM pushes registers in descending order and pops in ascending
	// order. in other words the LR is pushed first and PC is popped last

	load := opcode&0x0800 == 0x0800
	pclr := opcode&0x0100 == 0x0100
	regList := uint8(opcode & 0x00ff)

	if load {
		// start_address = SP
		// end_address = SP + 4*(R + Number_Of_Set_Bits_In(register_list))
		// address = start_address
		// for i = 0 to 7
		//		if register_list[i] == 1 then
		//			Ri = Memory[address,4]
		//			address = address + 4
		// if R == 1 then
		//		value = Memory[address,4]
		//		PC = value AND 0xFFFFFFFE
		// if (architecture version 5 or above) then
		//		T Bit = value[0]
		// address = address + 4
		// assert end_address = address
		// SP = end_address

		if pclr {
			// fmt.Printf("| POP {%#0b, PC}", regList)
		} else {
			// fmt.Printf("| POP {%#0b}", regList)
		}

		// start at stack pointer at work upwards
		addr := arm.registers[rSP]

		// read each register in turn (from lower to highest)
		for i := 0; i <= 7; i++ {

			// shit single-big mask
			m := uint8(0x01 << i)

			// read register if indicated by regList
			if regList&m == m {
				arm.registers[i] = arm.read32bit(addr)
				addr += 4
			}
		}

		// load PC register after all other registers
		if pclr {
			// chop the odd bit off the new PC value
			v := arm.read32bit(addr) & 0xfffffffe

			// add two to the new PC value. not sure why this is. it's not
			// descibed in the pseudo code above but I think it's to do with
			// how the ARM CPU does prefetching and when the adjustment is
			// applied. anwyay, this works but it might be worth figuring out
			// where else to apply the adjustment and whether that would be any
			// clearer.
			v += 2

			arm.registers[rPC] = v
			addr += 4
		}

		// leave stackpointer at final address
		arm.registers[rSP] = addr

		return
	}

	// store

	// start_address = SP - 4*(R + Number_Of_Set_Bits_In(register_list))
	// end_address = SP - 4
	// address = start_address
	// for i = 0 to 7
	//		if register_list[i] == 1
	//			Memory[address,4] = Ri
	//			address = address + 4
	// if R == 1
	//		Memory[address,4] = LR
	//		address = address + 4
	// assert end_address == address - 4
	// SP = SP - 4*(R + Number_Of_Set_Bits_In(register_list))

	// number of pushes to perform. count number of bits in regList and adjust
	// for PC/LR flag. each push requires 4 bytes of space
	var c uint32
	if pclr {
		// fmt.Printf("| PUSH {%#0b, LR}", regList)
		c = (uint32(bits.OnesCount8(regList)) + 1) * 4
	} else {
		// fmt.Printf("| PUSH {%#0b}", regList)
		c = uint32(bits.OnesCount8(regList)) * 4
	}

	// push occurs from the new low stack address upwards to the current stack
	// address (before the pushes)
	addr := arm.registers[rSP] - c

	// write each register in turn (from lower to highest)
	for i := 0; i <= 7; i++ {

		// shit single-big mask
		m := uint8(0x01 << i)

		// write register if indicated by regList
		if regList&m == m {
			arm.write32bit(addr, arm.registers[i])
			addr += 4
		}
	}

	// write LR register after all the other registers
	if pclr {
		lr := arm.registers[rLR]
		arm.write32bit(addr, lr)
	}

	// update stack pointer. note that this is the address we started the push
	// sequence from above. this is correct.
	arm.registers[rSP] -= c
}

func (arm *ARM) executeMultipleLoadStore(opcode uint16) {
	// format 15 - Multiple load/store
	load := opcode&0x0800 == 0x0800
	baseReg := uint32(opcode&0x07ff) >> 8
	regList := opcode & 0xff

	// load/store the registers in the list starting at address
	// in the base register
	addr := arm.registers[baseReg]

	for i := 0; i <= 7; i++ {
		r := regList >> i
		if r&0x01 == 0x01 {
			if load {
				arm.registers[i] = arm.read32bit(addr)
				addr += 4
			} else {
				arm.write32bit(addr, arm.registers[i])
				addr += 4
			}
		}
	}

	// write back the new base address
	arm.registers[baseReg] = addr
}

func (arm *ARM) executeConditionalBranch(opcode uint16) {
	// format 16 - Conditional branch
	cond := (opcode & 0x0f00) >> 8
	offset := uint32(opcode & 0x00ff)

	// operand := ""
	branch := false

	switch cond {
	case 0b0000:
		// operand = "BEQ"
		branch = arm.status.zero
	case 0b0001:
		// operand = "BNE"
		branch = !arm.status.zero
	case 0b0010:
		// operand = "BCS"
		branch = arm.status.carry
	case 0b0011:
		// operand = "BCC"
		branch = !arm.status.carry
	case 0b0100:
		// operand = "BMI"
		branch = arm.status.negative
	case 0b0101:
		// operand = "BPL"
		branch = !arm.status.negative
	case 0b0110:
		// operand = "BVS"
		branch = arm.status.overflow
	case 0b0111:
		// operand = "BVC"
		branch = !arm.status.overflow
	case 0b1000:
		// operand = "BHI"
		branch = arm.status.carry && !arm.status.zero
	case 0b1001:
		// operand = "BLS"
		branch = !arm.status.carry || arm.status.zero
	case 0b1010:
		// operand = "BGE"
		branch = (arm.status.negative && arm.status.overflow) || (!arm.status.negative && !arm.status.overflow)
	case 0b1011:
		// operand = "BLT"
		branch = (arm.status.negative && !arm.status.overflow) || (!arm.status.negative && arm.status.overflow)
	case 0b1100:
		// operand = "BGT"
		branch = !arm.status.zero && ((arm.status.negative && arm.status.overflow) || (!arm.status.negative && !arm.status.overflow))
	case 0b1101:
		// operand = "BLE"
		branch = arm.status.zero || ((arm.status.negative && !arm.status.overflow) || (!arm.status.negative && arm.status.overflow))
	case 0b1110:
		// operand = "undefined branch"
		branch = true
	case 0b1111:
		branch = false
	}

	// offset is a nine-bit two's complement value
	offset <<= 1
	offset++

	var newPC uint32

	// get new PC value
	if offset&0x100 == 0x100 {
		// two's complement before subtraction
		offset ^= 0x1ff
		offset++
		newPC = arm.registers[rPC] - offset
	} else {
		newPC = arm.registers[rPC] + offset
	}

	// disassembly
	// fmt.Printf("| %s %04x ", operand, newPC)

	// do branch
	if branch {
		arm.registers[rPC] = newPC + 1
	} else {
		// fmt.Printf("| no branch ")
	}
}

func (arm *ARM) executeSoftwareInterrupt(opcode uint16) {
	// format 17 - Software interrupt"
	panic("Software interrupt")
}

func (arm *ARM) executeUnconditionalBranch(opcode uint16) {
	// format 18 - Unconditional branch
	offset := uint32(opcode&0x07ff) << 1

	if offset&0x800 == 0x0800 {
		// two's complement before subtraction
		offset ^= 0xfff
		offset++
		arm.registers[rPC] -= offset - 2
	} else {
		arm.registers[rPC] += offset + 2
	}

	// disassembly
	// fmt.Printf("| BAL %04x ", arm.registers[rPC])

}

func (arm *ARM) executeLongBranchWithLink(opcode uint16) {
	// format 19 - Long branch with link
	low := opcode&0x800 == 0x0800
	offset := uint32(opcode & 0x07ff)

	if low {
		// second instruction
		offset <<= 1
		arm.registers[rLR] += offset
		pc := arm.registers[rPC]
		arm.registers[rPC] = arm.registers[rLR]
		arm.registers[rLR] = pc - 1
		// fmt.Printf("| BL %#08x", arm.registers[rPC])
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

	arm.noDebugNext = true
}
