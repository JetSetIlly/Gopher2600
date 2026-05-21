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

package breakpoints

import (
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
)

type Breakpoints struct {
	breakpoints map[uint32]bool
}

// NewBreakpoints is the preferred method of initialiasation for the Breakpoints type
func NewBreakpoints() Breakpoints {
	return Breakpoints{
		breakpoints: make(map[uint32]bool),
	}
}

// Write writes out the current callstack
func (bp *Breakpoints) Write(w io.Writer) {
	for b := range bp.breakpoints {
		fmt.Fprintf(w, "%08x\n", b)
	}
}

// HasBreakpoint returns true if there is a breakpoint on the specified line
func (bp *Breakpoints) Check(addr uint32) bool {
	return bp.breakpoints[addr]
}

// addBreakpoint adds an address to the list of addresses that will be checked
// each PC iteration
func (bp *Breakpoints) addBreakpoint(addr uint32) {
	bp.breakpoints[addr] = true
}

// removeBreakpoint removes an address from the list of breakpoint addresses
func (bp *Breakpoints) removeBreakpoint(addr uint32) {
	delete(bp.breakpoints, addr)
}

// ToggleBreakpoint adds or removes a breakpoint depending on whether the
// breakpoint already exists
func (bp *Breakpoints) ToggleBreakpoint(ln *dwarf.SourceLine) {
	// rather than toggle individual break addresses, we're toggling the line
	//
	// this means that if there is any address in the line with a breakpoint,
	// all the addresses in the line are removed. otherwise all the break
	// addresses are added
	has := bp.HasBreakpoint(ln)

	for _, addr := range ln.BreakAddresses {
		if has {
			bp.removeBreakpoint(addr)
		} else {
			bp.addBreakpoint(addr)
		}
	}
}

// HasBreakpoint returns true if there is a breakpoint on the specified line
func (bp *Breakpoints) HasBreakpoint(ln *dwarf.SourceLine) bool {
	// look for any address in the line and not just the break addresses
	for _, i := range ln.Instruction {
		if bp.breakpoints[i.Addr] {
			return true
		}
	}
	return false
}

// CanBreakpoint returns true if the specified line can have a breakpoint applied to it
func (bp *Breakpoints) CanBreakpoint(ln *dwarf.SourceLine) bool {
	return len(ln.BreakAddresses) > 0
}
