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

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
)

type strongArmFunction func()

type strongarm struct {
	function  strongArmFunction
	state     int
	registers [arm.NumRegisters]uint32
}

// strongARM functions need to return to the main program with a branch exchange
var strongArmStub = []byte{
	0x70, 0x47, // BX LR
	0x00, 0x00,
}

func (mem *elfMemory) vcsJsr6() {
	switch mem.strongarm.state {
	case 0:
		mem.gpioB[fromArm_Opcode] = 0x20
		mem.strongarm.state++
	case 1:
		mem.gpioB[fromArm_Opcode] = uint8(mem.strongarm.registers[0])
		mem.strongarm.state++
	case 2:
		mem.gpioB[fromArm_Opcode] = uint8(mem.strongarm.registers[0] >> 8)

		mem.gpioA[toArm_address] = uint8(mem.strongarm.registers[0])
		mem.gpioA[toArm_address+1] = uint8(mem.strongarm.registers[0] >> 8)
		mem.gpioA[toArm_address+2] = uint8(mem.strongarm.registers[0] >> 16)
		mem.gpioA[toArm_address+3] = uint8(mem.strongarm.registers[0] >> 24)

		mem.strongarm.function = nil
	}
}
