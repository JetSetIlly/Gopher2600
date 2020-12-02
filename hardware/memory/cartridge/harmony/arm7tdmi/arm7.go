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

	"github.com/jetsetilly/gopher2600/logger"
)

// CPSR (current program status register)
type cpsr struct {
	thumb    bool
	negative bool
	zero     bool
	overflow bool
	carry    bool
}

func (s *cpsr) isCondition(instruction uint32) (bool, string) {
	cond := (instruction & 0xf0000000) >> 28
	switch cond {
	case 0b0000:
		return s.zero, "equal"
	case 0b0001:
		return !s.zero, "not equal"
	case 0b0010:
		return s.carry, "carry set / unsigned higher or same"
	case 0b0011:
		return !s.carry, "carry clear / unsigned lower"
	case 0b0100:
		return s.negative, "minus / negative"
	case 0b0101:
		return !s.negative, "plus / positive or zero"
	case 0b0110:
		return s.overflow, "overflow"
	case 0b0111:
		return !s.overflow, "no overflow"
	case 0b1000:
		return s.carry && !s.zero, "unsigned higher"
	case 0b1001:
		return !s.carry && s.zero, "unsigned lower of same"
	case 0b1010:
		return s.negative == s.overflow, "signed greater than or equal"
	case 0b1011:
		return s.negative != s.overflow, "signed less than"
	case 0b1100:
		return !s.zero && s.negative == s.overflow, "signed greater than"
	case 0b1101:
		return s.zero || s.negative != s.overflow, "signed less than or equal"
	case 0b1110:
		return true, "always (unconditional)"
	case 0b1111:
		fallthrough
	default:
		return false, "never (obsolete, unpredictable in ARM7TDMI)"
	}
}

// Memory defines the memory that can be accessed from the ARM type. Field
// names mirror the names used in the DPC+ static CartStatic implementation.
type Memory struct {
	Driver *[]byte
	Custom *[]byte
	Data   *[]byte
	Freq   *[]byte
}

const (
	memoryOrigin = 0x40000C00
)

// register names
const (
	r1 = iota
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
)

// ARM implements the ARM7TDMI-S LPC2103 processor.
type ARM struct {
	mem Memory

	status    cpsr
	registers [16]uint32

	parameters   []uint8
	parameterIdx int
}

const maxParameters = 255

// NewARM is the preferred method of initialisation for the ARM type.
func NewARM(mem Memory, setPC uint16) *ARM {
	arm := &ARM{
		mem:        mem,
		parameters: make([]uint8, maxParameters),
	}
	arm.registers[rPC] = uint32(setPC)
	return arm
}

func (arm *ARM) SetParameter(data uint8) {
	logger.Log("ARM7", fmt.Sprintf("function parameter (%02x)", data))
}

func (arm *ARM) CallFunction(data uint8) error {
	switch data {
	case 0xfe:
	case 0xff:
		return arm.executeInstruction()
	default:
	}

	return nil
}

func (arm *ARM) read32bit(block []uint8) uint32 {
	b1 := uint32(block[arm.registers[rPC]])
	b2 := uint32(block[arm.registers[rPC]+1]) << 8
	b3 := uint32(block[arm.registers[rPC]+2]) << 16
	b4 := uint32(block[arm.registers[rPC]+3]) << 24
	arm.registers[rPC] += 4
	return b1 | b2 | b3 | b4
}

func (arm *ARM) executeInstruction() error {
	pc := arm.registers[rPC]
	opcode := arm.read32bit(*arm.mem.Custom)

	_, condition := arm.status.isCondition(opcode)

	fmt.Printf("%04x: %032b %08x %s\n", pc, opcode, opcode, condition)

	return nil
}
