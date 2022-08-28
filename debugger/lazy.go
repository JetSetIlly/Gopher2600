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
)

// The functions in this file are intended to be used with the lazy values
// system created for the sdlimgui GUI. the purpose of the lazy values system
// is to allow the GUI to access values from the emulator without having to
// worry about critical sectioning - all the criticial sectioning is done by
// the lazy value system.
//
// however, the system works by copying values which was fine at first but it
// is proving very cumbersome now that the project has moved on. moreover, in
// cases where the lazy system has proven wholly inadequate, a borrow mechanism
// has been implemented and without any adverse effects on performance.
//
// it is planned therefore for the lazy system to be superceded entirely by a
// borrow mechanism. as such, these functions will no longer be required.

// LazyGetQuantum returns the current quantum value.
func (dbg *Debugger) LazyGetQuantum() Quantum {
	return dbg.stepQuantum
}

// LazyGetLiveDisasmEntry returns the formatted disasembly entry of the last CPU
// execution and the bank information.
func (dbg *Debugger) LazyGetLiveDisasmEntry() disassembly.Entry {
	if dbg.liveDisasmEntry == nil {
		return disassembly.Entry{}
	}

	return *dbg.liveDisasmEntry
}

// LazyBreakpointsQuery allows others packages to query the currently set
// breakpoints.
type LazyBreakpointsQuery interface {
	HasPCBreak(addr uint16, bank int) (bool, int)
}

// LazyQueryBreakpoints returns an instance of LazyBreakpointsQuery.
func (dbg *Debugger) LazyQueryBreakpoints() LazyBreakpointsQuery {
	bq := *dbg.halting.breakpoints
	bq.breaks = make([]breaker, len(dbg.halting.breakpoints.breaks))
	copy(bq.breaks, dbg.halting.breakpoints.breaks)
	return bq
}

// LazyHasChanged returns true if emulation state has changed since last call to
// the function.
func (dbg *Debugger) LazyHasChanged() bool {
	v := dbg.hasChanged
	dbg.hasChanged = false
	return v
}
