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
	if src.CheckBreakpoint(addr) {
		src.removeBreakpoint(addr)
	} else {
		src.addBreakpoint(addr)
	}
}

// CheckBreakpointBySourceLine returns true if there is a breakpoint on the
// specified SourceLine
func (src *Source) CheckBreakpointBySourceLine(ln *SourceLine) bool {
	if len(ln.Disassembly) == 0 {
		return false
	}
	return src.CheckBreakpoint(ln.Disassembly[0].Addr)
}

// CheckBreakpoint returns true if there is a breakpoint on the
// specified address
func (src *Source) CheckBreakpoint(addr uint32) bool {
	if _, ok := src.Breakpoints[addr]; ok {
		return true
	}
	return false
}
