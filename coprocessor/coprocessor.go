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

package coprocessor

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/coprocessor/developer/faults"
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

// YieldHookResponse is returned by the CartYieldHook implementation to instruct
// the mapper in how to proceed
type YieldHookResponse int

// List of valid YieldHookCommands
const (
	YieldHookContinue YieldHookResponse = iota
	YieldHookEnd
)

// CartYieldHook allows a cartridge to halt execution if the cartridge
// coprocessor has reached a breakpoint or some other yield point (eg.
// undefined behaviour)
type CartYieldHook interface {
	CartYield(CoProcYieldType) YieldHookResponse
}

// StubCartYieldHook is a stub implementation for the CartYieldHook interface.
type StubCartYieldHook struct{}

// CartYield is a stub implementation for the CartYieldHook interface.
func (_ StubCartYieldHook) CartYield(yld CoProcYieldType) YieldHookResponse {
	if yld.Normal() {
		return YieldHookContinue
	}
	return YieldHookEnd
}

// ExtendedRegisterGroup specifies the numeric range for a coprocessor register group
type ExtendedRegisterGroup struct {
	// name of the group
	Name string

	// the numeric range of the registers in this group
	Start int
	End   int

	// whether the register range is private to the implementation. a private
	// range means that is not meaningul in relation to DWARF
	Private bool

	// whether the registers in the group will return meaningful data from the
	// RegisterFormatted() function
	Formatted bool

	// the label to use for the register
	Label func(register int) string
}

// ExtendedRegisterSpec is the specification returned by CartCoProc.RegisterSpec() function
type ExtendedRegisterSpec []ExtendedRegisterGroup

// Group returns the ExtendedRegisterGroup from the specifciation if it exists.
// For the purposes of this function, group names are not case-sensitive
func (spec ExtendedRegisterSpec) Group(name string) (ExtendedRegisterGroup, bool) {
	name = strings.ToUpper(name)
	for _, grp := range spec {
		if strings.ToUpper(grp.Name) == name {
			return grp, true
		}
	}
	return ExtendedRegisterGroup{}, false
}

// The basic set of registers present in a coprocessor. Every implementation
// should specify this group at a minimum
const ExtendedRegisterCoreGroup = "Core"

// CartCoProc is implemented by processors that are used in VCS cartridges.
// Principally this means ARM type processors but other processor types may be
// possible.
type CartCoProc interface {
	ProcessorID() string

	// set disassembler and developer hooks
	SetDisassembler(CartCoProcDisassembler)
	SetDeveloper(CartCoProcDeveloper)

	// breakpoint control of coprocessor
	BreakpointsEnable(bool)

	// RegisterSpec returns the specification for the registers visible in the
	// coprocessor. Implementations should ensure that these conform to the
	// DWARF extended register specification for the processor type (if
	// avaiable)
	//
	// Additional registers not required by the DWARF specification may be
	// supported as required
	//
	// Implementations should include the ExtendedRegisterCoreGroup at a minimum
	RegisterSpec() ExtendedRegisterSpec

	// the contents of a register. the implementation should support extended
	// register values defined by DWARF for the coprocessor
	//
	// if the register is unrecognised or unsupported the function will return
	// false
	Register(register int) (uint32, bool)

	// the contents of the register and a formatted string appropriate for the
	// register type
	RegisterFormatted(register int) (uint32, string, bool)

	// as above but setting the value of the register
	RegisterSet(register int, value uint32) bool

	// returns the current stack frame
	StackFrame() uint32

	// read coprocessor memory address for 32bit value. return false if address is out of range
	Peek(addr uint32) (uint32, bool)
}

// CartCoProcBus is implemented by cartridge mappers that have a coprocessor
type CartCoProcBus interface {
	// return the actual coprocessor interface. if the cartridge implements the
	// CartCoProcBus then it should always return a non-nil CartCoProc instance
	GetCoProc() CartCoProc

	// set interface for cartridge yields
	SetYieldHook(CartYieldHook)

	// the state of the coprocessor
	CoProcExecutionState() CoProcExecutionState
}

// CartCoProcRelocatable is implemented by cartridge mappers where coprocessor
// programs can be located anywhere in the coprcessor's memory
type CartCoProcRelocatable interface {
	// returns the offset of the named ELF section and whether the named
	// section exists. not all cartridges that implement this interface will be
	// able to meaningfully answer this function call
	ELFSection(string) ([]uint8, uint32, bool)
}

// CartCoProcRelocate is implemented by cartridge mappers where coprocessor
// programs are located at a specific address
type CartCoProcRelocate interface {
	ExecutableOrigin() uint32
}

// CartCoProcProfileEntry indicates the number of coprocessor cycles used by the
// instruction at the specified adress
type CartCoProcProfileEntry struct {
	Addr   uint32
	Cycles float32
}

// CartCoProcProfiler is shared by CartCoProcDeveloper and used by a coprocessor
// to record profiling information
type CartCoProcProfiler struct {
	Entries []CartCoProcProfileEntry
}

// CartCoProcDeveloper is implemented by a coprocessor to provide functions
// available to developers when the source code is available.
type CartCoProcDeveloper interface {
	// a memory fault has occured
	MemoryFault(event string, explanation faults.Category, instructionAddr uint32, accessAddr uint32)

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

	// called whenever the ARM yields to the VCS. it communicates the PC of the
	// most recent instruction, the current PC (as it is now), and the reason
	// for the yield
	OnYield(currentPC uint32, reason CoProcYield)
}

// CoProcYield describes a coprocessor yield state
type CoProcYield struct {
	Type  CoProcYieldType
	Error error
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
	case YieldStackError:
		return "stack error"
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

// Bug returns true if the yield type indicates a likely bug
func (t CoProcYieldType) Bug() bool {
	return t == YieldUndefinedBehaviour || t == YieldUnimplementedFeature ||
		t == YieldExecutionError || t == YieldMemoryAccessError || t == YieldStackError
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

	// something has gone wrong with the stack
	YieldStackError

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
