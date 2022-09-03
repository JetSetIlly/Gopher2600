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

package mapper

import (
	"debug/dwarf"
	"fmt"
)

// CoProcState is used to describe the state of the coprocessor. We can think
// of these values as descriptions of the synchronisation state between the
// coprocessor and the VCS.
type CoProcState int

// List of valid CoProcState values. A mapper will probably not employ all of
// these states depending on the synchronisation strategy.
//
// In reality the state will alternate between Idle-and-NOPFeed and
// StronARMFeed-and-Parallel.
//
// Other synchronisation strategies may need the addition of additional states
// or a different mechanism altogether.
const (
	// the idle state means that the coprocessor is not interacting with the
	// VCS at that moment. the coprocessor might be running but it is waiting
	// to be instructed by the VCS program
	CoProcIdle CoProcState = iota

	// a NOP feed describes the state where a cartridge mapper is waiting for
	// the coprocessor to finish processing. in the meantime, the cartridge is
	// feeding NOP instructions to the VCS
	CoProcNOPFeed

	// a StrongARM feed describes the state where the coprocessor has yielded
	// to the VCS in order for the next instruction to be read by the 6507
	CoProcStrongARMFeed

	// parallel execution describes the state where the coprocessor is running
	// without immediate concern with VCS synchronisation
	CoProcParallel
)

// CartCoProc is implemented by cartridge mappers that have a coprocessor that
// functions independently from the VCS.
type CartCoProc interface {
	CoProcID() string

	// set disassembler and developer hooks
	SetDisassembler(CartCoProcDisassembler)
	SetDeveloper(CartCoProcDeveloper)

	// the state of the coprocessor
	CoProcState() CoProcState

	// breakpoint control of coprocessor
	BreakpointHasTriggered() bool
	ResumeAfterBreakpoint() error
	BreakpointsDisable(bool)

	// returns any DWARF data for the cartridge. not all cartridges that
	// implement the CartCoProc interface will be able to meaningfully
	// return any data but none-the-less would benefit from DWARF debugging
	// information. in those instances, the DWARF data must be retreived
	// elsewhere
	DWARF() *dwarf.Data

	// returns the offset of the named ELF section and whether the named
	// section exists. not all cartridges that implement this interface will be
	// able to meaningfully answer this function call
	ELFSection(string) (uint32, bool)
}

// the following interfaces are implemented by the coprocessor itself, rather
// than any cartridge that uses the coprocessor.

// CartCoProcDeveloper is implemented by a coprocessor to provide functions
// available to developers when the source code is available.
type CartCoProcDeveloper interface {
	// addr accessed illegally by instruction at pc address. should return the
	// empty string if no meaningful information could be found
	IllegalAccess(event string, pc uint32, addr uint32) string

	// address is the same as the null pointer, indicating the address access
	// is likely to be a null pointer dereference
	NullAccess(event string, pc uint32, addr uint32) string

	// stack has collided with variable memtop
	StackCollision(pc uint32, sp uint32) string

	// returns the highest address utilised by program memory. the coprocessor
	// uses this value to detect stack collisions. should return zero if no
	// variables information is available
	VariableMemtop() uint32

	// checks if address has a breakpoint assigned to it
	CheckBreakpoint(addr uint32) bool

	// returns a map that can be used to count cycles for each PC address
	Profiling() map[uint32]float32

	// instructs developer implementation to accumulate profiling data
	EndProfiling()
}

// CartCoProcDisasmSummary represents a summary of a coprocessor execution.
type CartCoProcDisasmSummary interface {
	String() string
}

// CartCoProcDisasmEntry represents a single decoded instruction by the coprocessor.
type CartCoProcDisasmEntry interface {
	Key() string
	CSV() string
}

// CartCoProcDisassembler defines the functions that must be defined for a
// disassembler to be attached to a coprocessor.
type CartCoProcDisassembler interface {
	// Start is called at the beginning of coprocessor program execution.
	Start()

	// Step called after every instruction in the coprocessor program.
	Step(CartCoProcDisasmEntry)

	// End is called when coprocessor program has finished.
	End(CartCoProcDisasmSummary)
}

// CartCoProcDisassemblerStdout is a minimial implementation of the CartCoProcDisassembler
// interface. It outputs entries to stdout immediately upon request.
type CartCoProcDisassemblerStdout struct {
}

// Start implements the CartCoProcDisassembler interface.
func (c *CartCoProcDisassemblerStdout) Start() {
}

// Instruction implements the CartCoProcDisassembler interface.
func (c *CartCoProcDisassemblerStdout) Step(e CartCoProcDisasmEntry) {
	fmt.Println(e)
}

// End implements the CartCoProcDisassembler interface.
func (c *CartCoProcDisassemblerStdout) End(s CartCoProcDisasmSummary) {
	fmt.Println(s)
}
