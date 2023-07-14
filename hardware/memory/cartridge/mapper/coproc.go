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
	"fmt"
)

// CoProcExecutionState details the current condition of the coprocessor's execution
type CoProcExecutionState struct {
	Sync  CoProcSynchronisation
	Yield CoProcYield
}

// CoProcSynchronisation is used to describe the VCS synchronisation state of
// the coprocessor
type CoProcSynchronisation int

func (s CoProcSynchronisation) String() string {
	switch s {
	case CoProcIdle:
		return "idle"
	case CoProcNOPFeed:
		return "nop feed"
	case CoProcStrongARMFeed:
		return "strongarm feed"
	case CoProcParallel:
		return "parallel"
	}
	panic("unknown CoProcSynchronisation")
}

// List of valid CoProcSynchronisation values.
//
// A mapper will probably not employ all of these states depending on the
// synchronisation strategy. In reality the state will alternate between
// Idle-and-NOPFeed and StronARMFeed-and-Parallel.
//
// Other synchronisation strategies may need the addition of additional states
// or a different mechanism altogether.
const (
	// the idle state means that the coprocessor is not interacting with the
	// VCS at that moment. the coprocessor might be running but it is waiting
	// to be instructed by the VCS program
	CoProcIdle CoProcSynchronisation = iota

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

// CartYieldHook allows a cartridge to halt execution if the cartridge
// coprocessor has reached a breakpoint or some other yield point (eg.
// undefined behaviour)
type CartYieldHook interface {
	// CartYield returns true if the yield type cannot be handled without
	// breaking into a debugging loop
	//
	// CartYield will also return true if the cartridge mapper should cancel
	// coprocessing immediately
	//
	// all other yield reasons will return false
	CartYield(CoProcYieldType) bool
}

// StubCartYieldHook is a stub implementation for the CartYieldHook interface.
type StubCartYieldHook struct{}

// CartYield is a stub implementation for the CartYieldHook interface.
func (_ StubCartYieldHook) CartYield(_ CoProcYieldType) bool {
	return true
}

// CartCoProc is implemented by cartridge mappers that have a coprocessor that
// functions independently from the VCS.
type CartCoProc interface {
	CoProcID() string

	// set disassembler and developer hooks
	SetDisassembler(CartCoProcDisassembler)
	SetDeveloper(CartCoProcDeveloper)

	// the state of the coprocessor
	CoProcExecutionState() CoProcExecutionState

	// breakpoint control of coprocessor
	BreakpointsEnable(bool)

	// set interface for cartridge yields
	SetYieldHook(CartYieldHook)

	// the contents of register n. returns false if specified register is out
	// of range
	CoProcRegister(n int) (uint32, bool)
	CoProcRegisterSet(n int, value uint32) bool

	// returns the current stack frame
	CoProcStackFrame() uint32

	// read coprocessor memory address for 8/16/32 bit values. return false if
	// address is out of range
	CoProcRead8bit(addr uint32) (uint8, bool)
	CoProcRead16bit(addr uint32) (uint16, bool)
	CoProcRead32bit(addr uint32) (uint32, bool)
}

// CartCoProcRelocatable is implemented by cartridge mappers that are
// relocatable in coprocessor memory.
type CartCoProcRelocatable interface {
	// returns the offset of the named ELF section and whether the named
	// section exists. not all cartridges that implement this interface will be
	// able to meaningfully answer this function call
	ELFSection(string) ([]uint8, uint32, bool)
}

// CartCoProcNonRelocatable is implemented by cartridge mappers that are loaded
// into a specific coprocessor memory address.
type CartCoProcNonRelocatable interface {
	ExecutableOrigin() uint32
}

type CartCoProcProfileEntry struct {
	Addr   uint32
	Cycles float32
}

type CartCoProcProfiler struct {
	Entries []CartCoProcProfileEntry
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

	// returns the highest address used by the program. the coprocessor uses
	// this value to detect stack collisions
	HighAddress() uint32

	// checks if address has a breakpoint assigned to it
	CheckBreakpoint(addr uint32) bool

	// returns a map that can be used to count cycles for each PC address
	Profiling() *CartCoProcProfiler

	// notifies developer that the start of a new profiling session is about to begin
	StartProfiling()

	// instructs developer implementation to accumulate profiling data. there
	// can be many calls to profiling profiling for every call to start
	// profiling
	ProcessProfiling()

	// OnYield is called whenever the ARM yields to the VCS. It communicates the PC of the most
	// recent instruction, the current PC (as it is now), and the reason for the yield
	OnYield(instructionPC uint32, currentPC uint32, reason CoProcYield)
}

// CoProcYield describes a coprocessor yield state
type CoProcYield struct {
	Type   CoProcYieldType
	Error  error
	Detail []error
}

// CoProcYieldType specifies the type of yield. This is a broad categorisation
type CoProcYieldType int

func (t CoProcYieldType) String() string {
	switch t {
	case YieldProgramEnded:
		return "ended"
	case YieldSyncWithVCS:
		return "sync"
	case YieldBreakpoint:
		return "break"
	case YieldUndefinedBehaviour:
		return "undefined behaviour"
	case YieldUnimplementedFeature:
		return "unimplement feature"
	case YieldMemoryAccessError:
		return "memory error"
	case YieldExecutionError:
		return "execution error"
	case YieldRunning:
		return "running"
	}
	panic("unknown CoProcYieldType")
}

// Normal returns true if yield type is expected during normal operation of the
// coprocessor
func (t CoProcYieldType) Normal() bool {
	return t == YieldRunning || t == YieldProgramEnded || t == YieldSyncWithVCS
}

// List of CoProcYieldType values
const (
	// the coprocessor has yielded because the program has ended. in this instance the
	// CoProcessor is not considered to be in a "yielded" state and can be modified
	//
	// Expected YieldReason for CDF and DPC+ type ROMs
	YieldProgramEnded CoProcYieldType = iota

	// the coprocessor has reached a synchronisation point in the program. it
	// must wait for the VCS before continuing
	//
	// Expected YieldReason for ACE and ELF type ROMs
	YieldSyncWithVCS

	// a user supplied breakpoint has been encountered
	YieldBreakpoint

	// the program has triggered undefined behaviour in the coprocessor
	YieldUndefinedBehaviour

	// the program has triggered an unimplemented feature in the coprocessor
	YieldUnimplementedFeature

	// the program has tried to access memory illegally. details will have been
	// communicated by the IllegalAccess() function of the CartCoProcDeveloper
	// interface
	YieldMemoryAccessError

	// execution error indicates that something has gone very wrong
	YieldExecutionError

	// the coprocessor has not yet yielded and is still running
	YieldRunning
)

// CartCoProcDisasmSummary represents a summary of a coprocessor execution.
type CartCoProcDisasmSummary interface {
	String() string
}

// CartCoProcDisasmEntry represents a single decoded instruction by the coprocessor.
type CartCoProcDisasmEntry interface {
	String() string
	Key() string
	CSV() string
	Size() int
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
