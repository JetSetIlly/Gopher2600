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

import (
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/logger"
)

// The functions in this file are all about getting information in/out of the
// debugger that would otherwise be awkward or impossible through terminal
// commands.
//
// All of these functions are candidates for being replaced by terminal
// commands, with the understanding that doing so might: (a) be impossible to
// do so; (b) have a significant performance impact
//
// When calling these functions from another goroutine the PushRawEvent()
// function should be used to avoid an awkward critical section.

// GetQuantum returns the current quantum value.
func (dbg *Debugger) GetQuantum() QuantumMode {
	return dbg.quantum
}

// GetLastResult returns the formatted disasembly entry of the last CPU
// execution.
func (dbg *Debugger) GetLastResult() disassembly.Entry {
	return *dbg.lastResult
}

// HasBreak returns true if there is a breakpoint at the address. the second
// return value indicates if there is a breakpoint at the address AND bank.
func (dbg *Debugger) HasBreak(e *disassembly.Entry) BreakGroup {
	g, _ := dbg.breakpoints.hasBreak(e)
	return g
}

// TogglePCBreak sets or unsets a PC break at the address rerpresented by th
// disassembly entry.
func (dbg *Debugger) TogglePCBreak(e *disassembly.Entry) {
	dbg.breakpoints.togglePCBreak(e)
}

// PushRawEvent onto the event queue. This can be used to get information out
// of the debygger into another goroutine. Useful for when there is no
// equivalent terminal command.
func (dbg *Debugger) PushRawEvent(f func()) {
	select {
	case dbg.events.RawEvents <- f:
	default:
		logger.Log("debugger", "dropped raw event push")
	}
}
