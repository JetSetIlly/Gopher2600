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
	"math/rand"
)

// these functions should be executed with runStrongArmFunction() and not
// setStrongArmFunction()

func randint(mem *elfMemory) {
	_ = mem.arm.RegisterSet(0, rand.Uint32())
}

func memset(mem *elfMemory) {
	addr := mem.strongarm.running.registers[0]
	set, origin := mem.MapAddress(addr, true)
	idx := addr - origin

	if set != nil {
		v := mem.strongarm.running.registers[1]
		l := mem.strongarm.running.registers[2]
		for i := uint32(0); i < l; i++ {
			(*set)[idx+i] = byte(v)
		}
	}
}

func memcpy(mem *elfMemory) {
	toAddr := mem.strongarm.running.registers[0]
	to, toOrigin := mem.MapAddress(toAddr, true)
	toIdx := toAddr - toOrigin

	fromAddr := mem.strongarm.running.registers[1]
	from, fromOrigin := mem.MapAddress(fromAddr, false)
	fromIdx := fromAddr - fromOrigin

	if to != nil && from != nil {
		l := mem.strongarm.running.registers[2]
		for i := uint32(0); i < l; i++ {
			(*to)[toIdx+i] = (*from)[fromIdx+i]
		}
	}
}

// incomplete implementation. it should perform divide by zero checking but
// for now just return immediately
func __aeabi_idiv(mem *elfMemory) {
	if mem.strongarm.running.registers[1] == 0 {
		_ = mem.arm.RegisterSet(0, 0)
	} else {
		_ = mem.arm.RegisterSet(0, uint32(int32(mem.strongarm.running.registers[0])/int32(mem.strongarm.running.registers[1])))
	}
}
