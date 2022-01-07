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
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/disassembly"
)

// The functions in this file are all about getting information in/out of the
// debugger that would otherwise be awkward or too slow to serviced through
// terminal commands.
//
// All of these functions are candidates for being replaced by terminal
// commands, with the understanding that doing so might: (a) be impossible to
// do so; (b) have a significant performance impact
//
// When calling these functions from another goroutine the PushRawEvent()
// function should be used to avoid an awkward critical section.

// GetQuantum returns the current quantum value.
func (dbg *Debugger) GetQuantum() Quantum {
	return dbg.stepQuantum
}

// GetLastResult returns the formatted disasembly entry of the last CPU
// execution.
func (dbg *Debugger) GetLastResult() disassembly.Entry {
	return *dbg.lastResult
}

// BreakpointsQuery allows others packages to query the currently set
// breakpoints.
type BreakpointsQuery interface {
	HasPCBreak(addr uint16, bank int) (bool, int)
}

// QueryBreakpoints returns an instance of BreakpointsQuery.
func (dbg *Debugger) QueryBreakpoints() BreakpointsQuery {
	bq := *dbg.halting.breakpoints
	bq.breaks = make([]breaker, len(dbg.halting.breakpoints.breaks))
	copy(bq.breaks, dbg.halting.breakpoints.breaks)
	return bq
}

// TogglePCBreak sets or unsets a PC break at the address rerpresented by th
// disassembly entry.
func (dbg *Debugger) TogglePCBreak(e *disassembly.Entry) {
	dbg.halting.breakpoints.togglePCBreak(e)
}

// HasChanged returns true if emulation state has changed since last call to
// the function.
func (dbg *Debugger) HasChanged() bool {
	v := dbg.hasChanged
	dbg.hasChanged = false
	return v
}

// InsertCartridge into running emulation.
func (dbg *Debugger) InsertCartridge(filename string) error {
	cartload, err := cartridgeloader.NewLoader(filename, "AUTO")
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}
	err = dbg.attachCartridge(cartload)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}
	if dbg.forcedROMselection != nil {
		dbg.forcedROMselection <- true
	}
	return nil
}
