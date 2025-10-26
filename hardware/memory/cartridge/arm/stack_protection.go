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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/faults"
)

func (arm *ARM) stackProtectCheckSP() {
	// do nothing if stack has already collided
	if arm.state.stackHasErrors {
		return
	}

	// get memory block that the stack point is pointing to
	stackMemory, stackOrigin := arm.mem.MapAddress(arm.state.registers[rSP], true, false)

	if stackMemory == nil {
		arm.state.yield.Type = coprocessor.YieldMemoryFault
		arm.state.yield.Error = fmt.Errorf("illegal stack address (%08x)", arm.state.registers[rSP])

	} else if stackMemory == arm.state.programMemory {
		arm.state.yield.Type = coprocessor.YieldMemoryFault
		arm.state.yield.Error = fmt.Errorf("stack is in program memory (%08x)", arm.state.registers[rSP])

	} else if arm.state.protectVariableMemTop {
		// return is stack and variable memory blocks are different
		_, variableOrigin := arm.mem.MapAddress(arm.state.variableMemtop, true, false)
		if stackOrigin != variableOrigin {
			return
		}

		// return is stack pointer is above the top of variable memory
		if arm.state.registers[rSP] > arm.state.variableMemtop {
			return
		}

		arm.state.yield.Type = coprocessor.YieldMemoryFault
		arm.state.yield.Error = fmt.Errorf("stack collides (SP %08x) with variables (memtop %08x)",
			arm.state.registers[rSP], arm.state.variableMemtop)
	} else {
		return
	}

	arm.state.stackHasErrors = true

	if arm.dev != nil {
		arm.dev.MemoryFault(arm.state.yield.Error.Error(), faults.StackError, arm.state.executingPC, arm.state.registers[rSP])
	}
}

// called whenever program memory changes
func (arm *ARM) stackProtectCheckProgramMemory() {
	if arm.state.stackHasErrors {
		return
	}

	stackMemory, _ := arm.mem.MapAddress(arm.state.registers[rSP], true, false)
	if stackMemory == arm.state.programMemory {
		arm.state.yield.Type = coprocessor.YieldMemoryFault
		arm.state.yield.Error = fmt.Errorf("stack is in program memory (%08x)", arm.state.registers[rSP])
		arm.state.stackHasErrors = true
	} else {
		return
	}

	arm.state.stackHasErrors = true

	if arm.dev != nil {
		arm.dev.MemoryFault(arm.state.yield.Error.Error(), faults.StackError, arm.state.executingPC, arm.state.registers[rSP])
	}
}
