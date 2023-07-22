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

package yield

import (
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// State records the most recent yield.
type State struct {
	Addr           uint32
	Reason         mapper.CoProcYieldType
	LocalVariables []*dwarf.SourceVariableLocal
}

// Cmp returns true if two YieldStates are equal.
func (y State) Cmp(w State) bool {
	return y.Addr == w.Addr && y.Reason == w.Reason
}
