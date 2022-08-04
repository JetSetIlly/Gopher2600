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

package elf

// these functions work like you might expect the standard C implementations of
// these function to work except that they all end with a call to
// endStrongArmFunction()

func memset(mem *elfMemory) {
	addr := mem.strongarm.running.registers[0]
	m, o := mem.MapAddress(addr, true)

	v := mem.strongarm.running.registers[1]
	l := mem.strongarm.running.registers[2]
	for i := uint32(0); i < l; i++ {
		(*m)[o+i] = byte(v)
	}

	mem.endStrongArmFunction()
}

func memcpy(mem *elfMemory) {
	addr := mem.strongarm.running.registers[0]
	m, o := mem.MapAddress(addr, true)

	addrB := mem.strongarm.running.registers[1]
	mB, oB := mem.MapAddress(addrB, true)

	l := mem.strongarm.running.registers[2]
	for i := uint32(0); i < l; i++ {
		(*m)[o+i] = (*mB)[oB+i]
	}

	mem.endStrongArmFunction()
}

// incomplete implementation. it should perform divide by zero checking but
// for now just return immediately
func __aeabi_idiv(mem *elfMemory) {
	if mem.strongarm.running.registers[1] == 0 {
		mem.strongarm.running.registers[0] = 0
	} else {
		mem.strongarm.running.registers[0] = uint32(int32(mem.strongarm.running.registers[0]) / int32(mem.strongarm.running.registers[1]))
	}

	mem.arm.SetRegisters(mem.strongarm.running.registers)
	mem.endStrongArmFunction()
}
