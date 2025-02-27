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
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm"
)

// DisasmAnnotation is used as part of the implementation of the
// arm.CartridgeHookDisassembly interface
type DisasmAnnotation struct {
	StrongarmFunction string
}

func (a DisasmAnnotation) String() string {
	return a.StrongarmFunction
}

// AnnotateDisassembly implements the arm.CartridgeHookDisassembly interface
func (cart *Elf) AnnotateDisassembly(e *arm.DisasmEntry) fmt.Stringer {
	// subtracting strongArmOrigin may put the address outside of the sparse
	// array. we don't want to index the array in that case
	if e.Addr < cart.mem.strongArmOrigin || e.Addr > cart.mem.strongArmMemtop {
		return nil
	}

	f := cart.mem.strongArmFunctions[e.Addr-cart.mem.strongArmOrigin]
	if f == nil {
		return nil
	}

	return DisasmAnnotation{
		StrongarmFunction: f.name,
	}
}
