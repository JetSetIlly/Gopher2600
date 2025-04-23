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
	"time"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	"github.com/jetsetilly/gopher2600/debugger/govern"
)

// State records the most recent yield
type State struct {
	Addr           uint32
	Reason         coprocessor.CoProcYieldType
	LocalVariables []*dwarf.SourceVariableLocal

	Strobe                bool
	StrobeAddr            uint32
	StrobedLocalVariables []*dwarf.SourceVariableLocal
	StrobeTicker          *time.Ticker
}

// Cmp returns true if two YieldStates are equal
func (yld State) Cmp(w State) bool {
	return yld.Addr == w.Addr && yld.Reason == w.Reason
}

// LocalVariableView returns either the LocalVariables or StrobedLocalVariables
// array depending on the running state and whether a strobe is active
func (yld State) LocalVariableView(state govern.State) (uint32, []*dwarf.SourceVariableLocal) {
	if state == govern.Running && yld.Strobe {
		return yld.StrobeAddr, yld.StrobedLocalVariables
	}
	return yld.Addr, yld.LocalVariables
}

// Address returns the current address field
func (yld State) YieldAddress() uint32 {
	return yld.Addr
}
