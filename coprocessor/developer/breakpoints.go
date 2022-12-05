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

package developer

// addBreakpoint adds an address to the list of addresses that will be checked
// each PC iteration.
func (src *Source) addBreakpoint(addr uint32) {
	src.Breakpoints[addr] = true
}

// removeBreakpoint removes an address from the list of breakpoint addresses.
func (src *Source) removeBreakpoint(addr uint32) {
	delete(src.Breakpoints, addr)
}

// ToggleBreakpoint adds or removes a breakpoint depending on whether the
// breakpoint already exists.
func (src *Source) ToggleBreakpoint(ln *SourceLine) {
	if len(ln.Disassembly) == 0 {
		return
	}
	addr := ln.Disassembly[0].Addr
	if src.checkBreakpointByAddr(addr) {
		src.removeBreakpoint(addr)
	} else {
		src.addBreakpoint(addr)
	}
}

// CheckBreakpoint returns true if there is a breakpoint on the specified line.
func (src *Source) CheckBreakpoint(ln *SourceLine) bool {
	if len(ln.Disassembly) == 0 {
		return false
	}
	return src.checkBreakpointByAddr(ln.Disassembly[0].Addr)
}

// checkBreapointByAddr handles the situation where an address is on the same
// line as the previous breakpoint check. we need this because we don't want to
// repeatedly break on the same line when nothing has really changed.
func (src *Source) checkBreakpointByAddr(addr uint32) bool {
	return src.Breakpoints[addr]
}
