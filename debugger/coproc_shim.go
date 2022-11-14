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

package debugger

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"

// the coproc shim is a convenient way of retreiving corprocessor interfaces
// without exposing too much of the emulation
//
// more importantly the shim can be passed to another package and not have to
// worry about updating the reference to the cartridge after a rewind event
type coprocShim struct {
	dbg *Debugger
}

// GetCoProc implements the CartCoProcDeveloper interface in the coprocessor
// developer package
func (shim coprocShim) GetCoProc() mapper.CartCoProc {
	return shim.dbg.vcs.Mem.Cart.GetCoProc()
}

// PushFunction implements the CartCoProcDeveloper interface in the coprocessor
// developer package
func (shim coprocShim) PushFunction(f func()) {
	shim.dbg.PushFunction(f)
}
