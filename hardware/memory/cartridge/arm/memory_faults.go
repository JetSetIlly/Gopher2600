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

func (arm *ARM) memoryFault(event string, fault faults.Category, addr uint32) {
	if arm.state.stackHasErrors {
		return
	}

	if arm.dev != nil {
		arm.dev.MemoryFault(event, fault, arm.state.instructionPC, addr)
	}

	arm.state.yield.Type = coprocessor.YieldMemoryFault
	arm.state.yield.Error = fmt.Errorf("%s: %s: %08x (PC: %08x)", fault, event, addr, arm.state.instructionPC)
}

func (arm *ARM) unimplemented(name string, addr uint32) {
	arm.memoryFault(name, faults.UnimplementedPeripheral, addr)
}

func (arm *ARM) illegalAccess(event string, addr uint32) {
	arm.memoryFault(event, faults.IllegalAddress, addr)
}

func (arm *ARM) nullAccess(event string, addr uint32) {
	arm.memoryFault(event, faults.NullDereference, addr)
}

func (arm *ARM) misalignedAccess(event string, addr uint32) {
	arm.memoryFault(event, faults.MisalignedAddressing, addr)
}
