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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

import (
	"github.com/jetsetilly/gopher2600/disassembly"
)

// GetQuantum returns the current quantum value
func (dbg *Debugger) GetQuantum() QuantumMode {
	return dbg.quantum
}

// GetLastBank returns the selected cartridge bank from before the last
// instruction was executed
func (dbg *Debugger) GetLastBank() int {
	return dbg.lastBank
}

// HasBreak returns true if there is a breakpoint at the address. the second
// return value indicates if there is a breakpoint at the address AND bank
func (dbg *Debugger) HasBreak(e *disassembly.Entry) BreakGroup {
	g, _ := dbg.breakpoints.hasBreak(e)
	return g
}

// TogglePCBreak sets or unsets a PC break at the address rerpresented by th
// disassembly entry
func (dbg *Debugger) TogglePCBreak(e *disassembly.Entry) {
	dbg.breakpoints.togglePCBreak(e)
}

// IsRunning returns true if emulation is being run
func (dbg *Debugger) IsRunning() bool {
	return dbg.running
}
