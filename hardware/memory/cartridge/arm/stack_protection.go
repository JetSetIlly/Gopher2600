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
	"errors"
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

func (arm *ARM) stackProtectCheckSP() {
	// do nothing if stack has already collided
	if arm.state.stackHasCollided {
		return
	}

	// get memory block that the stack point is pointing to
	stackMemory, stackOrigin := arm.mem.MapAddress(arm.state.registers[rSP], true)

	if stackMemory == nil {
		arm.state.yield.Type = mapper.YieldStackError
		arm.state.yield.Error = fmt.Errorf("SP is not pointing to a valid address")

	} else if stackMemory == arm.state.programMemory {
		arm.state.yield.Type = mapper.YieldStackError
		arm.state.yield.Error = fmt.Errorf("SP is pointing to program memory")

	} else if arm.state.protectVariableMemTop {
		// return is stack and variable memory blocks are different
		_, variableOrigin := arm.mem.MapAddress(arm.state.variableMemtop, true)
		if stackOrigin != variableOrigin {
			return
		}

		// return is stack pointer is above the top of variable memory
		if arm.state.registers[rSP] > arm.state.variableMemtop {
			return
		}

		// set yield type
		arm.state.yield.Type = mapper.YieldStackError
		arm.state.yield.Error = fmt.Errorf("stack collision (SP %08x) with variables (top %08x) ",
			arm.state.registers[rSP], arm.state.variableMemtop)
	} else {
		return
	}

	arm.state.stackHasCollided = true

	// add developer details if possible
	if arm.dev != nil {
		detail := arm.dev.StackCollision(arm.state.executingPC, arm.state.registers[rSP])
		if detail != "" {
			arm.state.yield.Detail = append(arm.state.yield.Detail, errors.New(detail))
		}
	}
}

func (arm *ARM) stackProtectCheckProgramMemory() {
	if arm.state.stackHasCollided {
		return
	}

	stackMemory, _ := arm.mem.MapAddress(arm.state.registers[rSP], true)
	if stackMemory == arm.state.programMemory {
		arm.state.yield.Type = mapper.YieldStackError
		arm.state.yield.Error = fmt.Errorf("SP is pointing to program memory")
		arm.state.stackHasCollided = true
	} else {
		return
	}

	// add developer details if possible
	if arm.dev != nil {
		detail := arm.dev.StackCollision(arm.state.executingPC, arm.state.registers[rSP])
		if detail != "" {
			arm.state.yield.Detail = append(arm.state.yield.Detail, errors.New(detail))
		}
	}
}
