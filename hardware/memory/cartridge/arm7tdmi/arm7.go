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

package arm7tdmi

import (
	"fmt"
	"math/bits"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm7tdmi/mapfile"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm7tdmi/memorymodel"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm7tdmi/objdump"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/logger"
)

// register names.
const (
	rSP = 13 + iota
	rLR
	rPC
	rCount
)

// the speed at which the arm is running at and the required stretching for
// access to flash memory. speed is in MHz. Access latency of Flash memory is
// 50ns which is 20MHz. Rounding up, this means that the clklen (clk stretching
// amount) is 4.
//
// "The pipelined nature of the ARM7TDMI-S processor bus interface means that
// there is a distinction between clock cycles and bus cycles. CLKEN can be
// used to stretch a bus cycle, so that it lasts for many clock cycles. The
// CLKEN input extends the timing of bus cycles in increments of of complete
// CLK cycles"
//
// Access speed of SRAM is 10ns which is fast enough not to require stretching.
// MAM also requires no stretching.
const Clk = float32(70)
const clklenFlash = float32(4)

// the maximum number of cycles allowed in a single ARM program execution.
// no idea if this value is accurate.
const CycleLimit = 400000

// ARM implements the ARM7TDMI-S LPC2103 processor.
type ARM struct {
	prefs *preferences.ARMPreferences
	mmap  memorymodel.Map
	mem   SharedMemory
	hook  CartridgeHook

	// execution flags. set to false and/or error when Run() function should end
	continueExecution bool
	executionError    error

	// ARM registers
	registers [rCount]uint32
	status    status

	// the PC of the instruction being executed
	executingPC uint32

	// "peripherals" connected to the variety of ARM7TDMI-S used in the Harmony
	// cartridge.
	timer timer
	mam   mam

	// the area the PC covers. once assigned we'll assume that the program
	// never reads outside this area. the value is assigned on reset()
	programMemory *[]uint8

	// length of program memory. in truth this is probably a constant but we
	// don't really know that for sure
	programMemoryLen int

	// the amount to adjust the memory address by so that it can be used to
	// index the programMemory array
	programMemoryOffset uint32

	// is set to true when an access to memory using a read/write function used
	// an unrecognised address. when this happens, the address is logged and
	// the Thumb program aborted (ie returns early)
	//
	// note: it is only set to true if abortOnIllegalMem is true
	memoryError bool

	// whether to foce an error on illegal memory access. set from ARM.prefs at
	// the start of every arm.Run()
	abortOnIllegalMem bool

	// collection of functionMap instances. indexed by programMemoryOffset to
	// retrieve a functionMap
	//
	// allocated in NewArm() and added to in findProgramMemory() if an entry
	// does not exist
	executionMap map[uint32][]func(_ uint16)

	// functionMap records the function that implements the instruction group for each
	// opcode in program memory. must be reset every time programMemory is reassigned
	//
	// note that when executing from RAM (which isn't normal) it's possible for
	// code to be modified (ie. self-modifying code). in that case functionMap
	// may be unreliable.
	functionMap []func(_ uint16)

	// interface to an optional disassembler
	disasm mapper.CartCoProcDisassembler

	// cache of disassembled entries
	disasmCache map[uint32]DisasmEntry

	// the level of disassemble to perform next instruction
	disasmLevel disasmLevel

	// the next disasmEntry to send to attached disassembler
	disasmEntry DisasmEntry

	// \/\/\/ the following fields relate to cycle counting. there's a possible
	// optimisation whereby we don't do any cycle counting at all (or minimise
	// it at least) if the emulation is running in immediate mode
	//
	// !TODO: optimisation for ARM immediate mode

	// the last cycle to be triggered, used to decide whether to merge I-S cycles
	lastCycle cycleType

	// the type of cycle next prefetch (the main PC increment in the Run()
	// loop) should be. either N or S type. never I type.
	prefetchCycle cycleType

	// total number of cycles for the entire program
	cyclesTotal float32

	// \/\/\/ the following are reset at the end of each Run() iteration \/\/\/

	// whether cycle count or not. set from ARM.prefs at the start of every arm.Run()
	//
	// used to cut out code that is required only for cycle counting. See
	// Icycle, Scycle and Ncycle fields which are called so frequently we
	// forego checking the immediateMode flag each time and have preset a stub
	// function if required
	immediateMode bool

	// number of cycles with CLKLEN modulation applied
	stretchedCycles float32

	// record the order in which cycles happen for a single instruction
	// - required for disasm only
	cycleOrder cycleOrder

	// whether a branch has used the branch trail latches or not
	// - required for disasm only
	branchTrail BranchTrail

	// whether an I cycle that is followed by an S cycle has been merged
	// - required for disasm only
	mergedIS bool

	// rather than call the cycle counting functions directly, we assign the
	// functions to these fields. in this way, we can use stubs when executing
	// in immediate mode (when cycle counting isn't necessary)
	//
	// other aspects of cycle counting are not expensive and can remain
	Icycle func()
	Scycle func(bus busAccess, addr uint32)
	Ncycle func(bus busAccess, addr uint32)

	// mapfile for binary (if available)
	mapfile *mapfile.Mapfile

	// obj dump for binary (if available)
	objdump *objdump.ObjDump

	// illegal accesses already encountered. duplicate accesses will not be logged.
	illegalAccesses map[string]bool
}

type disasmLevel int

const (
	disasmNone disasmLevel = iota

	// update entry only if the UpdateExecution is true.
	disasmUpdateOnly

	// update all disassembly fields (operator, operands, etc.). this doesn't
	// need to happen unless the entry is not in the disasm cache.
	disasmFull
)

// NewARM is the preferred method of initialisation for the ARM type.
func NewARM(mmap memorymodel.Map, prefs *preferences.ARMPreferences, mem SharedMemory, hook CartridgeHook, pathToROM string) *ARM {
	arm := &ARM{
		prefs:           prefs,
		mmap:            mmap,
		mem:             mem,
		hook:            hook,
		executionMap:    make(map[uint32][]func(_ uint16)),
		disasmCache:     make(map[uint32]DisasmEntry),
		illegalAccesses: make(map[string]bool),
	}

	arm.mam.mmap = mmap
	arm.timer.mmap = mmap

	err := arm.reset()
	if err != nil {
		logger.Logf("ARM7", "reset: %s", err.Error())
	}

	arm.mapfile, err = mapfile.NewMapFile(pathToROM)
	if err != nil {
		logger.Logf("ARM7", err.Error())
	}

	arm.objdump, err = objdump.NewObjDump(pathToROM)
	if err != nil {
		logger.Logf("ARM7", err.Error())
	}

	return arm
}

// CoProcID is the ID returned by the ARM type. This const value can be used
// for comparison purposes to check if a mapper.CartCoProcBus instance is of
// the ARM type.
const CoProcID = "ARM7TDMI"

// CoProcID implements the mapper.CartCoProcBus interface.
func (arm *ARM) CoProcID() string {
	return CoProcID
}

// SetDisassembler implements the mapper.CartCoProcBus interface.
func (arm *ARM) SetDisassembler(disasm mapper.CartCoProcDisassembler) {
	arm.disasm = disasm
}

// Plumb should be used to update the shared memory reference.
// Useful when used in conjunction with the rewind system.
func (arm *ARM) Plumb(mem SharedMemory, hook CartridgeHook) {
	arm.mem = mem
	arm.hook = hook
}

func (arm *ARM) reset() error {
	arm.status.reset()
	for i := range arm.registers {
		arm.registers[i] = 0x00000000
	}
	arm.registers[rSP], arm.registers[rLR], arm.registers[rPC] = arm.mem.ResetVectors()

	// reset execution flags
	arm.continueExecution = true
	arm.executionError = nil

	// reset cycles count
	arm.cyclesTotal = 0
	arm.prefetchCycle = S

	arm.memoryError = false

	return arm.findProgramMemory()
}

// find program memory using current program counter value.
func (arm *ARM) findProgramMemory() error {
	arm.programMemory, arm.programMemoryOffset = arm.mem.MapAddress(arm.registers[rPC], false)
	if arm.programMemory == nil {
		return curated.Errorf("ARM7: cannot find program memory")
	}

	arm.programMemoryOffset = arm.registers[rPC] - arm.programMemoryOffset

	if m, ok := arm.executionMap[arm.programMemoryOffset]; ok {
		arm.functionMap = m
	} else {
		arm.executionMap[arm.programMemoryOffset] = make([]func(_ uint16), len(*arm.programMemory))
		arm.functionMap = arm.executionMap[arm.programMemoryOffset]
	}

	arm.programMemoryLen = len(*arm.programMemory)

	return nil
}

func (arm *ARM) String() string {
	s := strings.Builder{}
	for i, r := range arm.registers {
		if i > 0 {
			if i%4 == 0 {
				s.WriteString("\n")
			} else {
				s.WriteString("\t\t")
			}
		}
		s.WriteString(fmt.Sprintf("R%-2d: %08x", i, r))
	}
	return s.String()
}

// Step moves the ARM on one cycle. Currently, the timer will only step forward
// when Step() is called and not during the Run() process. This might cause
// problems in some instances with some ARM programs.
func (arm *ARM) Step(vcsClock float32) {
	arm.timer.stepFromVCS(Clk, vcsClock)
}

func (arm *ARM) lookupSource() {
	if arm.mapfile != nil {
		programLabel := arm.mapfile.FindProgramAccess(arm.executingPC)
		if programLabel != "" {
			logger.Logf("ARM7", "mapfile: access in %s()", programLabel)
		}
	}
	if arm.objdump != nil {
		src := arm.objdump.FindProgramAccess(arm.executingPC)
		if src != "" {
			logger.Logf("ARM7", "objdump:\n%s", src)
		}
	}
}

func (arm *ARM) read8bit(addr uint32) uint8 {
	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if v, ok, comment := arm.timer.read(addr); ok {
			arm.disasmEntry.ExecutionNotes = comment
			return uint8(v)
		}
		if v, ok := arm.mam.read(addr); ok {
			return uint8(v)
		}
		arm.memoryError = arm.abortOnIllegalMem

		accessKey := fmt.Sprintf("%08x%08x", addr, arm.executingPC)
		if _, ok := arm.illegalAccesses[accessKey]; !ok {
			arm.illegalAccesses[accessKey] = true
			logger.Logf("ARM7", "read8bit: unrecognised address %08x (PC: %08x)", addr, arm.executingPC)
			arm.lookupSource()
		}
		return 0
	}

	return (*mem)[addr]
}

func (arm *ARM) write8bit(addr uint32, val uint8) {
	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if ok, comment := arm.timer.write(addr, uint32(val)); ok {
			arm.disasmEntry.ExecutionNotes = comment
			return
		}
		if ok := arm.mam.write(addr, uint32(val)); ok {
			return
		}
		arm.memoryError = arm.abortOnIllegalMem

		accessKey := fmt.Sprintf("%08x%08x", addr, arm.executingPC)
		if _, ok := arm.illegalAccesses[accessKey]; !ok {
			arm.illegalAccesses[accessKey] = true
			logger.Logf("ARM7", "write8bit: unrecognised address %08x (PC: %08x)", addr, arm.executingPC)
			arm.lookupSource()
		}
		return
	}

	(*mem)[addr] = val
}

func (arm *ARM) read16bit(addr uint32) uint16 {
	// check 16 bit alignment
	if addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit read (%08x) (PC: %08x)", addr, arm.registers[rPC])
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if v, ok, comment := arm.timer.read(addr); ok {
			arm.disasmEntry.ExecutionNotes = comment
			return uint16(v)
		}
		if v, ok := arm.mam.read(addr); ok {
			return uint16(v)
		}
		arm.memoryError = arm.abortOnIllegalMem

		accessKey := fmt.Sprintf("%08x%08x", addr, arm.executingPC)
		if _, ok := arm.illegalAccesses[accessKey]; !ok {
			arm.illegalAccesses[accessKey] = true
			logger.Logf("ARM7", "read16bit: unrecognised address %08x (PC: %08x)", addr, arm.executingPC)
			arm.lookupSource()
		}
		return 0
	}

	return uint16((*mem)[addr]) | (uint16((*mem)[addr+1]) << 8)
}

func (arm *ARM) write16bit(addr uint32, val uint16) {
	// check 16 bit alignment
	if addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit write (%08x) (PC: %08x)", addr, arm.registers[rPC])
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if ok, comment := arm.timer.write(addr, uint32(val)); ok {
			arm.disasmEntry.ExecutionNotes = comment
			return
		}
		if ok := arm.mam.write(addr, uint32(val)); ok {
			return
		}
		arm.memoryError = arm.abortOnIllegalMem

		accessKey := fmt.Sprintf("%08x%08x", addr, arm.executingPC)
		if _, ok := arm.illegalAccesses[accessKey]; !ok {
			arm.illegalAccesses[accessKey] = true
			logger.Logf("ARM7", "write16bit: unrecognised address %08x (PC: %08x)", addr, arm.executingPC)
			arm.lookupSource()
		}
		return
	}

	(*mem)[addr] = uint8(val)
	(*mem)[addr+1] = uint8(val >> 8)
}

func (arm *ARM) read32bit(addr uint32) uint32 {
	// check 32 bit alignment
	if addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit read (%08x) (PC: %08x)", addr, arm.registers[rPC])
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if v, ok, comment := arm.timer.read(addr); ok {
			arm.disasmEntry.ExecutionNotes = comment
			return v
		}
		if v, ok := arm.mam.read(addr); ok {
			return v
		}
		arm.memoryError = arm.abortOnIllegalMem

		accessKey := fmt.Sprintf("%08x%08x", addr, arm.executingPC)
		if _, ok := arm.illegalAccesses[accessKey]; !ok {
			arm.illegalAccesses[accessKey] = true
			logger.Logf("ARM7", "read32bit: unrecognised address %08x (PC: %08x)", addr, arm.executingPC)
			arm.lookupSource()
		}
		return 0
	}

	return uint32((*mem)[addr]) | (uint32((*mem)[addr+1]) << 8) | (uint32((*mem)[addr+2]) << 16) | uint32((*mem)[addr+3])<<24
}

func (arm *ARM) write32bit(addr uint32, val uint32) {
	// check 32 bit alignment
	if addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit write (%08x) (PC: %08x)", addr, arm.registers[rPC])
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if ok, comment := arm.timer.write(addr, val); ok {
			arm.disasmEntry.ExecutionNotes = comment
			return
		}
		if ok := arm.mam.write(addr, val); ok {
			return
		}
		arm.memoryError = arm.abortOnIllegalMem

		accessKey := fmt.Sprintf("%08x%08x", addr, arm.executingPC)
		if _, ok := arm.illegalAccesses[accessKey]; !ok {
			arm.illegalAccesses[accessKey] = true
			logger.Logf("ARM7", "write32bit: unrecognised address %08x (PC: %08x)", addr, arm.executingPC)
			arm.lookupSource()
		}
		return
	}

	(*mem)[addr] = uint8(val)
	(*mem)[addr+1] = uint8(val >> 8)
	(*mem)[addr+2] = uint8(val >> 16)
	(*mem)[addr+3] = uint8(val >> 24)
}

// Run will continue until the ARM program encounters a switch from THUMB mode
// to ARM mode. Note that currently, this means the ARM program may run
// forever.
//
// Returns the MAMCR state, the number of ARM cycles consumed and any errors.
func (arm *ARM) Run(mamcr uint32) (uint32, float32, error) {
	err := arm.reset()
	if err != nil {
		return arm.mam.mamcr, 0, err
	}

	// what we send at the end of the execution. not used if not disassembler is set
	programSummary := DisasmSummary{}

	// set mamcr on startup
	arm.mam.pref = arm.prefs.MAM.Get().(int)
	if arm.mam.pref == preferences.MAMDriver {
		arm.mam.setMAMCR(mamcr)
		arm.mam.mamtim = 4.0
	} else {
		arm.mam.setMAMCR(uint32(arm.mam.pref))
		arm.mam.mamtim = 4.0
	}

	// set cycle counting functions
	arm.immediateMode = arm.prefs.Immediate.Get().(bool)
	if arm.immediateMode {
		arm.Icycle = arm.iCycleStub
		arm.Scycle = arm.sCycleStub
		arm.Ncycle = arm.nCycleStub
		programSummary.ImmediateMode = true
	} else {
		arm.Icycle = arm.iCycle
		arm.Scycle = arm.sCycle
		arm.Ncycle = arm.nCycle
	}

	// start of program execution
	if arm.disasm != nil {
		arm.disasm.Start()
	}

	// how to handle illegal memory access
	arm.abortOnIllegalMem = arm.prefs.AbortOnIllegalMem.Get().(bool)

	// fill pipeline
	arm.registers[rPC] += 2

	// use to detect branches and whether to fill the pipeline (unused if
	// arm.immediateMode is true)
	var expectedPC uint32

	// loop through instructions until we reach an exit condition
	for arm.continueExecution && !arm.memoryError {
		// program counter to execute:
		//
		// from "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p1", page 1-2
		//
		// "The program counter points to the instruction being fetched rather than to the instruction
		// being executed. This is important because it means that the Program Counter (PC)
		// value used in an executing instruction is always two instructions ahead of the address."
		arm.executingPC = arm.registers[rPC] - 2

		// set disasmLevel for the next instruction
		if arm.disasm == nil {
			arm.disasmLevel = disasmNone
		} else {
			// full disassembly unless we can find a usable entry in the disasm cache
			arm.disasmLevel = disasmFull

			// check cache for existing disasm entry
			if e, ok := arm.disasmCache[arm.executingPC]; ok {
				// use cached entry
				arm.disasmEntry = e

				if arm.disasmEntry.updateNotes {
					arm.disasmEntry.ExecutionNotes = ""
					arm.disasmLevel = disasmUpdateOnly
				} else {
					arm.disasmLevel = disasmNone
				}
			}

			// if the entry has not been retreived from the cache make sure it is
			// in an initial state
			if arm.disasmLevel == disasmFull {
				arm.disasmEntry.Location = ""
				arm.disasmEntry.Address = fmt.Sprintf("%08x", arm.executingPC)
				arm.disasmEntry.Operator = ""
				arm.disasmEntry.Operand = ""
				arm.disasmEntry.Cycles = 0.0
				arm.disasmEntry.ExecutionNotes = ""
				arm.disasmEntry.updateNotes = false
			}
		}

		// check program counter
		memIdx := arm.executingPC - arm.programMemoryOffset
		if memIdx+1 >= uint32(arm.programMemoryLen) {
			// program counter is out-of-range so find program memory again
			// (using the PC value)
			err = arm.findProgramMemory()
			if err != nil {
				// can't find memory so we say the ARM program has finished inadvertently
				logger.Logf("ARM7", "PC out of range (%#08x). aborting thumb program early", arm.executingPC)
				break // for loop
			}

			// if it's still out-of-range then give up with an error
			memIdx = arm.executingPC - arm.programMemoryOffset
			if memIdx+1 >= uint32(arm.programMemoryLen) {
				// can't find memory so we say the ARM program has finished inadvertently
				logger.Logf("ARM7", "PC out of range (%#08x). aborting thumb program early", arm.executingPC)
				break // for loop
			}
		}

		// opcode for executed instruction
		opcode := uint16((*arm.programMemory)[memIdx]) | (uint16((*arm.programMemory)[memIdx+1]) << 8)

		// bump PC counter for prefetch. actual prefetch is done after execution
		arm.registers[rPC] += 2

		// the expected PC at the end of the execution. if the PC register
		// does not match fillPipeline() is called
		if !arm.immediateMode {
			expectedPC = arm.registers[rPC]
		}

		// run from functionMap if possible
		formatFunc := arm.functionMap[memIdx]
		if formatFunc != nil {
			formatFunc(opcode)
		} else {
			// make a reference of the opcode function
			var f func(opcode uint16)

			// working backwards up the table in Figure 5-1 of the ARM7TDMI Data Sheet.
			if opcode&0xf000 == 0xf000 {
				// format 19 - Long branch with link
				f = arm.executeLongBranchWithLink
			} else if opcode&0xf000 == 0xe000 {
				// format 18 - Unconditional branch
				f = arm.executeUnconditionalBranch
			} else if opcode&0xff00 == 0xdf00 {
				// format 17 - Software interrupt"
				f = arm.executeSoftwareInterrupt
			} else if opcode&0xf000 == 0xd000 {
				// format 16 - Conditional branch
				f = arm.executeConditionalBranch
			} else if opcode&0xf000 == 0xc000 {
				// format 15 - Multiple load/store
				f = arm.executeMultipleLoadStore
			} else if opcode&0xf600 == 0xb400 {
				// format 14 - Push/pop registers
				f = arm.executePushPopRegisters
			} else if opcode&0xff00 == 0xb000 {
				// format 13 - Add offset to stack pointer
				f = arm.executeAddOffsetToSP
			} else if opcode&0xf000 == 0xa000 {
				// format 12 - Load address
				f = arm.executeLoadAddress
			} else if opcode&0xf000 == 0x9000 {
				// format 11 - SP-relative load/store
				f = arm.executeSPRelativeLoadStore
			} else if opcode&0xf000 == 0x8000 {
				// format 10 - Load/store halfword
				f = arm.executeLoadStoreHalfword
			} else if opcode&0xe000 == 0x6000 {
				// format 9 - Load/store with immediate offset
				f = arm.executeLoadStoreWithImmOffset
			} else if opcode&0xf200 == 0x5200 {
				// format 8 - Load/store sign-extended byte/halfword
				f = arm.executeLoadStoreSignExtendedByteHalford
			} else if opcode&0xf200 == 0x5000 {
				// format 7 - Load/store with register offset
				f = arm.executeLoadStoreWithRegisterOffset
			} else if opcode&0xf800 == 0x4800 {
				// format 6 - PC-relative load
				f = arm.executePCrelativeLoad
			} else if opcode&0xfc00 == 0x4400 {
				// format 5 - Hi register operations/branch exchange
				f = arm.executeHiRegisterOps
			} else if opcode&0xfc00 == 0x4000 {
				// format 4 - ALU operations
				f = arm.executeALUoperations
			} else if opcode&0xe000 == 0x2000 {
				// format 3 - Move/compare/add/subtract immediate
				f = arm.executeMovCmpAddSubImm
			} else if opcode&0xf800 == 0x1800 {
				// format 2 - Add/subtract
				f = arm.executeAddSubtract
			} else if opcode&0xe000 == 0x0000 {
				// format 1 - Move shifted register
				f = arm.executeMoveShiftedRegister
			} else {
				panic("undecoded instruction")
			}

			// store function reference in functionMap and run for the first time
			arm.functionMap[memIdx] = f
			f(opcode)
		}

		if !arm.immediateMode {
			// add additional cycles required to fill pipeline before next iteration
			if expectedPC != arm.registers[rPC] {
				arm.fillPipeline()
			}

			// prefetch cycle for next instruction is associated with and counts
			// towards the total of the current instruction. most prefetch cycles
			// are S cycles but store instructions require an N cycle
			if arm.prefetchCycle == N {
				arm.Ncycle(prefetch, arm.registers[rPC])
			} else {
				arm.Scycle(prefetch, arm.registers[rPC])
			}

			// default to an S cycle for prefetch unless an instruction explicitly
			// says otherwise
			arm.prefetchCycle = S

			// increases total number of program cycles by the stretched cycles for this instruction
			arm.cyclesTotal += arm.stretchedCycles

			// update timer. assuming an APB divider value of one.
			arm.timer.step(arm.stretchedCycles)
		}

		// send disasm information to disassembler
		if arm.disasm != nil {
			arm.disasmEntry.MAMCR = int(arm.mam.mamcr)
			arm.disasmEntry.BranchTrail = arm.branchTrail
			arm.disasmEntry.MergedIS = arm.mergedIS
			arm.disasmEntry.CyclesSequence = arm.cycleOrder.String()

			// update cycle information
			arm.disasmEntry.Cycles = arm.cycleOrder.len()

			// update program cycles
			programSummary.add(arm.cycleOrder)

			// if no operator mnemonic has been defined, replace with
			// opcode value
			//
			// this shouldn't happen but we're showing something so that
			// the disasm output can be debugged
			if arm.disasmEntry.Operator == "" {
				arm.disasmEntry.Operator = fmt.Sprintf("%04x", opcode)
			}

			switch arm.disasmLevel {
			case disasmFull:
				// if this is not a cached entry then format operator and
				// operand fields and insert into cache
				arm.disasmEntry.Operator = fmt.Sprintf("%-4s", arm.disasmEntry.Operator)
				arm.disasmEntry.Operand = fmt.Sprintf("%-16s", arm.disasmEntry.Operand)
				arm.disasmCache[arm.executingPC] = arm.disasmEntry
			case disasmUpdateOnly:
				// entry is cached but notes may have changed so we recache
				// the entry
				arm.disasmCache[arm.executingPC] = arm.disasmEntry
			case disasmNone:
			}

			// we always send the instruction to the disasm interface
			arm.disasm.Step(arm.disasmEntry)
		}

		// reset cycle  information
		if !arm.immediateMode {
			arm.branchTrail = BranchTrailNotUsed
			arm.mergedIS = false
			arm.stretchedCycles = 0
			arm.cycleOrder.reset()

			// limit the number of cycles used by the ARM program
			if arm.cyclesTotal >= CycleLimit {
				logger.Logf("ARM7", "reached cycle limit of %d. ending execution early", CycleLimit)
				break
			}
		}
	}

	// indicate that program abort was because of illegal memory access
	if arm.memoryError {
		logger.Logf("ARM7", "illegal memory access detected. aborting thumb program early")
	}

	// end of program execution
	if arm.disasm != nil {
		arm.disasm.End(programSummary)
	}

	if arm.executionError != nil {
		return arm.mam.mamcr, 0, curated.Errorf("ARM7: %v", arm.executionError)
	}

	return arm.mam.mamcr, arm.cyclesTotal, nil
}

func (arm *ARM) executeMoveShiftedRegister(opcode uint16) {
	// format 1 - Move shifted register
	op := (opcode & 0x1800) >> 11
	shift := (opcode & 0x7c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// in this class of operation the src register may also be the dest
	// register so we need to make a note of the value before it is
	// overwrittten
	src := arm.registers[srcReg]

	switch op {
	case 0b00:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LSL"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d, #%02x ", destReg, srcReg, shift)
		}

		// if immed_5 == 0
		//	C Flag = unaffected
		//	Rd = Rm
		// else /* immed_5 > 0 */
		//	C Flag = Rm[32 - immed_5]
		//	Rd = Rm Logical_Shift_Left immed_5

		if shift == 0 {
			arm.registers[destReg] = src
		} else {
			m := uint32(0x01) << (32 - shift)
			arm.status.carry = src&m == m
			arm.registers[destReg] = arm.registers[srcReg] << shift
		}
	case 0b01:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LSR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d, #%02x ", destReg, srcReg, shift)
		}

		// if immed_5 == 0
		//		C Flag = Rm[31]
		//		Rd = 0
		// else /* immed_5 > 0 */
		//		C Flag = Rm[immed_5 - 1]
		//		Rd = Rm Logical_Shift_Right immed_5

		if shift == 0 {
			arm.status.carry = src&0x80000000 == 0x80000000
			arm.registers[destReg] = 0x00
		} else {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			arm.registers[destReg] = src >> shift
		}
	case 0b10:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ASR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d, #%02x ", destReg, srcReg, shift)
		}

		// if immed_5 == 0
		//		C Flag = Rm[31]
		//		if Rm[31] == 0 then
		//				Rd = 0
		//		else /* Rm[31] == 1 */]
		//				Rd = 0xFFFFFFFF
		// else /* immed_5 > 0 */
		//		C Flag = Rm[immed_5 - 1]
		//		Rd = Rm Arithmetic_Shift_Right immed_5

		if shift == 0 {
			arm.status.carry = src&0x80000000 == 0x80000000
			if arm.status.carry {
				arm.registers[destReg] = 0xffffffff
			} else {
				arm.registers[destReg] = 0x00000000
			}
		} else { // shift > 0
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			a := src >> shift
			if src&0x80000000 == 0x80000000 {
				a |= (0xffffffff << (32 - shift))
			}
			arm.registers[destReg] = a
		}

	case 0x11:
		panic("illegal instruction")
	}

	arm.status.isZero(arm.registers[destReg])
	arm.status.isNegative(arm.registers[destReg])

	if destReg == rPC {
		logger.Log("ARM7", "shift and store in PC is not possible in thumb mode")
	}

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	if shift > 0 {
		arm.Icycle()
	}
}

func (arm *ARM) executeAddSubtract(opcode uint16) {
	// format 2 - Add/subtract
	immediate := opcode&0x0400 == 0x0400
	subtract := opcode&0x0200 == 0x0200
	imm := uint32((opcode & 0x01c0) >> 6)
	srcReg := (opcode & 0x038) >> 3
	destReg := opcode & 0x07

	// value to work with is either an immediate value or is in a register
	val := imm
	if !immediate {
		val = arm.registers[imm]
	}

	if subtract {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "SUB"
			if immediate {
				arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d, #%02x ", destReg, srcReg, val)
			} else {
				arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d, R%02x ", destReg, srcReg, imm)
			}
		}

		arm.status.setCarry(arm.registers[srcReg], ^val, 1)
		arm.status.setOverflow(arm.registers[srcReg], ^val, 1)
		arm.registers[destReg] = arm.registers[srcReg] - val
	} else {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ADD"
			if immediate {
				arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d, #%02x ", destReg, srcReg, val)
			} else {
				arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d, R%02x ", destReg, srcReg, imm)
			}
		}

		arm.status.setCarry(arm.registers[srcReg], val, 0)
		arm.status.setOverflow(arm.registers[srcReg], val, 0)
		arm.registers[destReg] = arm.registers[srcReg] + val
	}

	arm.status.isZero(arm.registers[destReg])
	arm.status.isNegative(arm.registers[destReg])

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

// "The instructions in this group perform operations between a Lo register and
// an 8-bit immediate value".
func (arm *ARM) executeMovCmpAddSubImm(opcode uint16) {
	// format 3 - Move/compare/add/subtract immediate
	op := (opcode & 0x1800) >> 11
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode & 0x00ff)

	switch op {
	case 0b00:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "MOV"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, #%02x ", destReg, imm)
		}
		arm.registers[destReg] = imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b01:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "CMP"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, #%02x ", destReg, imm)
		}
		arm.status.setCarry(arm.registers[destReg], ^imm, 1)
		arm.status.setOverflow(arm.registers[destReg], ^imm, 1)
		cmp := arm.registers[destReg] - imm
		arm.status.isNegative(cmp)
		arm.status.isZero(cmp)
	case 0b10:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ADD"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, #%02x ", destReg, imm)
		}
		arm.status.setCarry(arm.registers[destReg], imm, 0)
		arm.status.setOverflow(arm.registers[destReg], imm, 0)
		arm.registers[destReg] += imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b11:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "SUB"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, #%02x ", destReg, imm)
		}
		arm.status.setCarry(arm.registers[destReg], ^imm, 1)
		arm.status.setOverflow(arm.registers[destReg], ^imm, 1)
		arm.registers[destReg] -= imm
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	}

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

// "The following instructions perform ALU operations on a Lo register pair".
func (arm *ARM) executeALUoperations(opcode uint16) {
	// format 4 - ALU operations
	op := (opcode & 0x03c0) >> 6
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	var shift uint32
	var mul bool
	var mulOperand uint32

	switch op {
	case 0b0000:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "AND"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.registers[destReg] &= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0001:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "EOR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.registers[destReg] ^= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0010:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LSL"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}

		shift = arm.registers[srcReg]

		// if Rs[7:0] == 0
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[7:0] < 32 then
		//		C Flag = Rd[32 - Rs[7:0]]
		//		Rd = Rd Logical_Shift_Left Rs[7:0]
		// else if Rs[7:0] == 32 then
		//		C Flag = Rd[0]
		//		Rd = 0
		// else /* Rs[7:0] > 32 */
		//		C Flag = 0
		//		Rd = 0
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected

		if shift > 0 && shift < 32 {
			m := uint32(0x01) << (32 - shift)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] <<= shift
		} else if shift == 32 {
			arm.status.carry = arm.registers[destReg]&0x01 == 0x01
			arm.registers[destReg] = 0x00
		} else if shift > 32 {
			arm.status.carry = false
			arm.registers[destReg] = 0x00
		}

		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0011:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LSR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}

		shift = arm.registers[srcReg]

		// if Rs[7:0] == 0 then
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[7:0] < 32 then
		//		C Flag = Rd[Rs[7:0] - 1]
		//		Rd = Rd Logical_Shift_Right Rs[7:0]
		// else if Rs[7:0] == 32 then
		//		C Flag = Rd[31]
		//		Rd = 0
		// else /* Rs[7:0] > 32 */
		//		C Flag = 0
		//		Rd = 0
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected

		if shift > 0 && shift < 32 {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] >>= shift
		} else if shift == 32 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
			arm.registers[destReg] = 0x00
		} else if shift > 32 {
			arm.status.carry = false
			arm.registers[destReg] = 0x00
		}

		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0100:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ASR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}

		shift = arm.registers[srcReg]

		// if Rs[7:0] == 0 then
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[7:0] < 32 then
		//		C Flag = Rd[Rs[7:0] - 1]
		//		Rd = Rd Arithmetic_Shift_Right Rs[7:0]
		// else /* Rs[7:0] >= 32 */
		//		C Flag = Rd[31]
		//		if Rd[31] == 0 then
		//			Rd = 0
		//		else /* Rd[31] == 1 */
		//			Rd = 0xFFFFFFFF
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected
		if shift > 0 && shift < 32 {
			src := arm.registers[destReg]
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			a := src >> shift
			if src&0x80000000 == 0x80000000 {
				a |= (0xffffffff << (32 - shift))
			}
			arm.registers[destReg] = a
		} else if shift >= 32 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
			if !arm.status.carry {
				arm.registers[destReg] = 0x00
			} else {
				arm.registers[destReg] = 0xffffffff
			}
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0101:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ADC"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		if arm.status.carry {
			arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 1)
			arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 1)
			arm.registers[destReg] += arm.registers[srcReg]
			arm.registers[destReg]++
		} else {
			arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 0)
			arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 0)
			arm.registers[destReg] += arm.registers[srcReg]
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0110:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "SBC"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		if !arm.status.carry {
			arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 0)
			arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 0)
			arm.registers[destReg] -= arm.registers[srcReg]
			arm.registers[destReg]--
		} else {
			arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
			arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
			arm.registers[destReg] -= arm.registers[srcReg]
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b0111:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ROR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}

		shift = arm.registers[srcReg]

		// if Rs[7:0] == 0 then
		//		C Flag = unaffected
		//		Rd = unaffected
		// else if Rs[4:0] == 0 then
		//		C Flag = Rd[31]
		//		Rd = unaffected
		// else /* Rs[4:0] > 0 */
		//		C Flag = Rd[Rs[4:0] - 1]
		//		Rd = Rd Rotate_Right Rs[4:0]
		// N Flag = Rd[31]
		// Z Flag = if Rd == 0 then 1 else 0
		// V Flag = unaffected
		if shift&0xff == 0 {
			// unaffected
		} else if shift&0x1f == 0 {
			arm.status.carry = arm.registers[destReg]&0x80000000 == 0x80000000
		} else {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] = bits.RotateLeft32(arm.registers[destReg], -int(shift))
		}
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1000:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "TST"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		w := arm.registers[destReg] & arm.registers[srcReg]
		arm.status.isZero(w)
		arm.status.isNegative(w)
	case 0b1001:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "NEG"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.status.setCarry(0, ^arm.registers[srcReg], 1)
		arm.status.setOverflow(0, ^arm.registers[srcReg], 1)
		arm.registers[destReg] = -arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1010:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "CMP"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
		arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
		cmp := arm.registers[destReg] - arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b1011:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "CMN"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.status.setCarry(arm.registers[destReg], arm.registers[srcReg], 0)
		arm.status.setOverflow(arm.registers[destReg], arm.registers[srcReg], 0)
		cmp := arm.registers[destReg] + arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b1100:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ORR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.registers[destReg] |= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1101:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "MUL"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}

		mul = true
		mulOperand = arm.registers[srcReg]

		arm.registers[destReg] *= arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1110:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "BIC"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.registers[destReg] &= ^arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	case 0b1111:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "MVN"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, R%d ", destReg, srcReg)
		}
		arm.registers[destReg] = ^arm.registers[srcReg]
		arm.status.isZero(arm.registers[destReg])
		arm.status.isNegative(arm.registers[destReg])
	default:
		panic(fmt.Sprintf("unimplemented ALU operation (%04b)", op))
	}

	// page 7-11 in "ARM7TDMI-S Technical Reference Manual r4p3"
	if shift > 0 && destReg == rPC {
		logger.Log("ARM7", "shift and store in PC is not possible in thumb mode")
	}

	if mul {
		// "7.7 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		//  and
		// "7.2 Instruction Cycle Count Summary"  in "ARM7TDMI-S Technical
		// Reference Manual r4p3" ...
		p := bits.OnesCount32(mulOperand & 0xffffff00)
		if p == 0 || p == 24 {
			// ... Is 1 if bits [32:8] of the multiplier operand are all zero or one.
			arm.Icycle()
		} else {
			p := bits.OnesCount32(mulOperand & 0xffff0000)
			if p == 0 || p == 16 {
				// ... Is 2 if bits [32:16] of the multiplier operand are all zero or one.
				arm.Icycle()
				arm.Icycle()
			} else {
				p := bits.OnesCount32(mulOperand & 0xff000000)
				if p == 0 || p == 8 {
					// ... Is 3 if bits [31:24] of the multiplier operand are all zero or one.
					arm.Icycle()
					arm.Icycle()
					arm.Icycle()
				} else {
					// ... Is 4 otherwise.
					arm.Icycle()
					arm.Icycle()
					arm.Icycle()
					arm.Icycle()
				}
			}
		}
	} else {
		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		if shift > 0 {
			arm.Icycle()
		}
	}
}

func (arm *ARM) executeHiRegisterOps(opcode uint16) {
	// format 5 - Hi register operations/branch exchange
	op := (opcode & 0x300) >> 8
	hi1 := opcode&0x80 == 0x80
	hi2 := opcode&0x40 == 0x40
	srcReg := (opcode & 0x38) >> 3
	destReg := opcode & 0x07

	// labels used to decoraate operands indicating Hi/Lo register usage
	destLabel := "R"
	srcLabel := "R"
	if hi1 {
		destReg += 8
		destLabel = "H"
	}
	if hi2 {
		srcReg += 8
		srcLabel = "H"
	}

	switch op {
	case 0b00:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ADD"
			arm.disasmEntry.Operand = fmt.Sprintf("%s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
		}

		// not two's complement
		arm.registers[destReg] += arm.registers[srcReg]

		// status register not changed

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return
	case 0b01:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "CMP"
			arm.disasmEntry.Operand = fmt.Sprintf("%s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
		}

		// alu_out = Rn - Rm
		// N Flag = alu_out[31]
		// Z Flag = if alu_out == 0 then 1 else 0
		// C Flag = NOT BorrowFrom(Rn - Rm)
		// V Flag = OverflowFrom(Rn - Rm)

		arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 1)
		arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 1)
		cmp := arm.registers[destReg] - arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)

		return
	case 0b10:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "MOV"
			arm.disasmEntry.Operand = fmt.Sprintf("%s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
		}
		arm.registers[destReg] = arm.registers[srcReg]

		// status register not changed

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return
	case 0b11:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "BX"
			arm.disasmEntry.Operand = fmt.Sprintf("%s%d ", srcLabel, srcReg)
		}

		thumbMode := arm.registers[srcReg]&0x01 == 0x01

		var newPC uint32

		// "ARM7TDMI Data Sheet" page 5-15:
		//
		// "If R15 is used as an operand, the value will be the address of the instruction + 4 with
		// bit 0 cleared. Executing a BX PC in THUMB state from a non-word aligned address
		// will result in unpredictable execution."
		if srcReg == rPC {
			newPC = arm.registers[rPC] + 2
		} else {
			newPC = (arm.registers[srcReg] & 0x7ffffffe) + 2
		}

		if thumbMode {
			arm.registers[rPC] = newPC

			if arm.disasmLevel != disasmNone {
				arm.disasmEntry.ExecutionNotes = "branch exchange to thumb code"
				arm.disasmEntry.updateNotes = true
			}

			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			return
		}

		// switch to ARM mode. emulate function call.
		res, err := arm.hook.ARMinterrupt(arm.registers[rPC]-4, arm.registers[2], arm.registers[3])
		if err != nil {
			arm.continueExecution = false
			arm.executionError = err
			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			//  - interrupted
			return
		}

		// update execution notes unless disasm level is disasmNone
		if arm.disasmLevel != disasmNone {
			if res.InterruptEvent != "" {
				arm.disasmEntry.ExecutionNotes = fmt.Sprintf("ARM function (%08x) %s", arm.registers[rPC]-4, res.InterruptEvent)
			} else {
				arm.disasmEntry.ExecutionNotes = fmt.Sprintf("ARM function (%08x)", arm.registers[rPC]-4)
			}
			arm.disasmEntry.updateNotes = true
		}

		// if ARMinterrupt returns false this indicates that the
		// function at the quoted program counter is not recognised and
		// has nothing to do with the cartridge mapping. at this point
		// we can assume that the main() function call is done and we
		// can return to the VCS emulation.
		if !res.InterruptServiced {
			arm.continueExecution = false
			// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
			//  - interrupted
			return
		}

		// ARM function updates the ARM registers
		if res.SaveResult {
			arm.registers[res.SaveRegister] = res.SaveValue
		}

		// the end of the emulated function will have an operation that
		// switches back to thumb mode, and copies the link register to the
		// program counter. we need to emulate that too.
		arm.registers[rPC] = arm.registers[rLR] + 2

		// add cycles used by the ARM program
		arm.armInterruptCycles(res)

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
	}
}

func (arm *ARM) executePCrelativeLoad(opcode uint16) {
	// format 6 - PC-relative load
	destReg := (opcode & 0x0700) >> 8
	imm := uint32(opcode&0x00ff) << 2

	// "Bit 1 of the PC value is forced to zero for the purpose of this
	// calculation, so the address is always word-aligned."
	pc := arm.registers[rPC] & 0xfffffffc

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "LDR"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, [PC, #%02x] ", destReg, imm)
	}

	// immediate value is not two's complement (surprisingly)
	addr := pc + imm
	arm.registers[destReg] = arm.read32bit(addr)

	// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
	arm.Ncycle(dataRead, addr)
	arm.Icycle()
}

func (arm *ARM) executeLoadStoreWithRegisterOffset(opcode uint16) {
	// format 7 - Load/store with register offset
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x0400 == 0x0400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	addr := arm.registers[baseReg] + arm.registers[offsetReg]

	if load {
		if byteTransfer {
			if arm.disasmLevel == disasmFull {
				arm.disasmEntry.Operator = "LDRB"
				arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d]", reg, baseReg, offsetReg)
			}

			arm.registers[reg] = uint32(arm.read8bit(addr))

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return
		}

		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		}

		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	if byteTransfer {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "STRB"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		}

		arm.write8bit(addr, uint8(arm.registers[reg]))

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "STR"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
	}

	arm.write32bit(addr, arm.registers[reg])

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) executeLoadStoreSignExtendedByteHalford(opcode uint16) {
	// format 8 - Load/store sign-extended byte/halfword
	hi := opcode&0x0800 == 0x800
	sign := opcode&0x0400 == 0x400
	offsetReg := (opcode & 0x01c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	addr := arm.registers[baseReg] + arm.registers[offsetReg]

	if sign {
		if hi {
			// load sign-extended halfword
			if arm.disasmLevel == disasmFull {
				arm.disasmEntry.Operator = "LDSH"
				arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
			}

			arm.registers[reg] = uint32(arm.read16bit(addr))

			// masking after cycle accumulation
			if arm.registers[reg]&0x8000 == 0x8000 {
				arm.registers[reg] |= 0xffff0000
			}

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return
		}
		// load sign-extended byte
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDSB"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		}

		arm.registers[reg] = uint32(arm.read8bit(addr))
		if arm.registers[reg]&0x0080 == 0x0080 {
			arm.registers[reg] |= 0xffffff00
		}

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	if hi {
		// load halfword
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDRH"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		}
		arm.registers[reg] = uint32(arm.read16bit(addr))

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	// store halfword
	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "STRH"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
	}

	arm.write16bit(addr, uint16(arm.registers[reg]))

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) executeLoadStoreWithImmOffset(opcode uint16) {
	// format 9 - Load/store with immediate offset
	load := opcode&0x0800 == 0x0800
	byteTransfer := opcode&0x1000 == 0x1000

	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "For word accesses (B = 0), the value specified by #Imm is a full 7-bit address, but must
	// be word-aligned (ie with bits 1:0 set to 0), since the assembler places #Imm >> 2 in
	// the Offset5 field." -- ARM7TDMI Data Sheet
	if !byteTransfer {
		offset <<= 2
	}

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[baseReg] + uint32(offset)

	if load {
		if byteTransfer {
			arm.registers[reg] = uint32(arm.read8bit(addr))

			if arm.disasmLevel == disasmFull {
				arm.disasmEntry.Operator = "LDRB"
				arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, #%02x] ", reg, baseReg, offset)
			}

			// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - fillPipeline() will be called if necessary
			arm.Ncycle(dataRead, addr)
			arm.Icycle()

			return
		}

		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, #%02x] ", reg, baseReg, offset)
		}
		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	// store
	if byteTransfer {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "STRB"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, #%02x] ", reg, baseReg, offset)
		}
		arm.write8bit(addr, uint8(arm.registers[reg]))

		// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)

		return
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "STR"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, #%02x] ", reg, baseReg, offset)
	}

	arm.write32bit(addr, arm.registers[reg])

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) executeLoadStoreHalfword(opcode uint16) {
	// format 10 - Load/store halfword
	load := opcode&0x0800 == 0x0800
	offset := (opcode & 0x07c0) >> 6
	baseReg := (opcode & 0x0038) >> 3
	reg := opcode & 0x0007

	// "#Imm is a full 6-bit address but must be halfword-aligned (ie with bit 0 set to 0) since
	// the assembler places #Imm >> 1 in the Offset5 field." -- ARM7TDMI Data Sheet
	offset <<= 1

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[baseReg] + uint32(offset)

	if load {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDRH"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, #%02x] ", reg, baseReg, offset)
		}

		arm.registers[reg] = uint32(arm.read16bit(addr))

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "STRH"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, #%02x] ", reg, baseReg, offset)
	}

	arm.write16bit(addr, uint16(arm.registers[reg]))

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) executeSPRelativeLoadStore(opcode uint16) {
	// format 11 - SP-relative load/store
	load := opcode&0x0800 == 0x0800
	reg := (opcode & 0x07ff) >> 8
	offset := uint32(opcode & 0xff)

	// The offset supplied in #Imm is a full 10-bit address, but must always be word-aligned
	// (ie bits 1:0 set to 0), since the assembler places #Imm >> 2 in the Word8 field.
	offset <<= 2

	// the actual address we'll be loading from (or storing to)
	addr := arm.registers[rSP] + offset

	if load {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [SP, #%02x] ", reg, offset)
		}

		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.Ncycle(dataRead, addr)
		arm.Icycle()

		return
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "STR"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, [SP, #%02x] ", reg, offset)
	}

	arm.write32bit(addr, arm.registers[reg])

	// "7.9 Store Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.storeRegisterCycles(addr)
}

func (arm *ARM) executeLoadAddress(opcode uint16) {
	// format 12 - Load address
	sp := opcode&0x0800 == 0x800
	destReg := (opcode & 0x700) >> 8
	offset := opcode & 0x00ff

	// offset is a word aligned 10 bit address
	offset <<= 2

	if sp {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ADD"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, SP, #%02x] ", destReg, offset)
		}

		arm.registers[destReg] = arm.registers[rSP] + uint32(offset)

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary

		return
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "ADD"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, PC, #%02x] ", destReg, offset)
	}

	arm.registers[destReg] = arm.registers[rPC] + uint32(offset)

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

func (arm *ARM) executeAddOffsetToSP(opcode uint16) {
	// format 13 - Add offset to stack pointer
	sign := opcode&0x80 == 0x80
	imm := uint32(opcode & 0x7f)

	// The offset specified by #Imm can be up to -/+ 508, but must be word-aligned (ie with
	// bits 1:0 set to 0) since the assembler converts #Imm to an 8-bit sign + magnitude
	// number before placing it in field SWord7.
	imm <<= 2

	if sign {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "ADD"
			arm.disasmEntry.Operand = fmt.Sprintf("SP, #-%d ", imm)
		}
		arm.registers[rSP] -= imm

		// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - no additional cycles

		return
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "ADD"
		arm.disasmEntry.Operand = fmt.Sprintf("SP, #%02x ", imm)
	}

	arm.registers[rSP] += imm

	// status register not changed

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - no additional cycles
}

func (arm *ARM) executePushPopRegisters(opcode uint16) {
	// format 14 - Push/pop registers

	// the ARM pushes registers in descending order and pops in ascending
	// order. in other words the LR is pushed first and PC is popped last

	load := opcode&0x0800 == 0x0800
	pclr := opcode&0x0100 == 0x0100
	regList := uint8(opcode & 0x00ff)

	if load {
		// start_address = SP
		// end_address = SP + 4*(R + Number_Of_Set_Bits_In(register_list))
		// address = start_address
		// for i = 0 to 7
		//		if register_list[i] == 1 then
		//			Ri = Memory[address,4]
		//			address = address + 4
		// if R == 1 then
		//		value = Memory[address,4]
		//		PC = value AND 0xFFFFFFFE
		// if (architecture version 5 or above) then
		//		T Bit = value[0]
		// address = address + 4
		// assert end_address = address
		// SP = end_address

		// start at stack pointer at work upwards
		addr := arm.registers[rSP]

		// read each register in turn (from lower to highest)
		numMatches := 0
		for i := 0; i <= 7; i++ {
			// shift single-bit mask
			m := uint8(0x01 << i)

			// read register if indicated by regList
			if regList&m == m {
				numMatches++

				// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - N cycle on first match
				// - S cycles on subsequent matches
				if numMatches == 1 {
					arm.Ncycle(dataRead, addr)
				} else {
					arm.Scycle(dataRead, addr)
				}

				arm.registers[i] = arm.read32bit(addr)
				addr += 4
			}
		}

		// load PC register after all other registers
		if pclr {
			numMatches++

			// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - N cycle if first match and S cycle otherwise
			// - fillPipeline() will be called if necessary
			if numMatches == 1 {
				arm.Ncycle(dataRead, addr)
			} else {
				arm.Scycle(dataRead, addr)
			}

			// chop the odd bit off the new PC value
			v := arm.read32bit(addr) & 0xfffffffe

			// adjust popped LR value before assigning to the PC
			arm.registers[rPC] = v + 2
			addr += 4

			if arm.disasmLevel == disasmFull {
				arm.disasmEntry.Operator = "POP"
				arm.disasmEntry.Operand = fmt.Sprintf("{%#08b, LR}", regList)
			}
		} else if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "POP"
			arm.disasmEntry.Operand = fmt.Sprintf("{%#08b}", regList)
		}

		// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Icycle()

		// leave stackpointer at final address
		arm.registers[rSP] = addr

		return
	}

	// store

	// start_address = SP - 4*(R + Number_Of_Set_Bits_In(register_list))
	// end_address = SP - 4
	// address = start_address
	// for i = 0 to 7
	//		if register_list[i] == 1
	//			Memory[address,4] = Ri
	//			address = address + 4
	// if R == 1
	//		Memory[address,4] = LR
	//		address = address + 4
	// assert end_address == address - 4
	// SP = SP - 4*(R + Number_Of_Set_Bits_In(register_list))

	// number of pushes to perform. count number of bits in regList and adjust
	// for PC/LR flag. each push requires 4 bytes of space
	var c uint32
	if pclr {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "PUSH"
			arm.disasmEntry.Operand = fmt.Sprintf("{%#08b, LR}", regList)
		}
		c = (uint32(bits.OnesCount8(regList)) + 1) * 4
	} else {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "PUSH"
			arm.disasmEntry.Operand = fmt.Sprintf("{%#08b}", regList)
		}
		c = uint32(bits.OnesCount8(regList)) * 4
	}

	// push occurs from the new low stack address upwards to the current stack
	// address (before the pushes)
	addr := arm.registers[rSP] - c

	// write each register in turn (from lower to highest)
	numMatches := 0
	for i := 0; i <= 7; i++ {
		// shift single-bit mask
		m := uint8(0x01 << i)

		// write register if indicated by regList
		if regList&m == m {
			numMatches++

			// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - storeRegisterCycles() on first match
			// - S cycles on subsequent match
			// - next prefetch cycle will be N
			if numMatches == 1 {
				arm.storeRegisterCycles(addr)
			} else {
				arm.Scycle(dataWrite, addr)
			}

			arm.write32bit(addr, arm.registers[i])
			addr += 4
		}
	}

	// write LR register after all the other registers
	if pclr {
		numMatches++

		lr := arm.registers[rLR]
		arm.write32bit(addr, lr)

		// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		if numMatches == 1 {
			arm.storeRegisterCycles(addr)
		} else {
			arm.Scycle(dataWrite, addr)
		}
	}

	// update stack pointer. note that this is the address we started the push
	// sequence from above. this is correct.
	arm.registers[rSP] -= c
}

func (arm *ARM) executeMultipleLoadStore(opcode uint16) {
	// format 15 - Multiple load/store
	load := opcode&0x0800 == 0x0800
	baseReg := uint32(opcode&0x07ff) >> 8
	regList := opcode & 0xff

	// load/store the registers in the list starting at address
	// in the base register
	addr := arm.registers[baseReg]

	if arm.disasmLevel == disasmFull {
		if load {
			arm.disasmEntry.Operator = "LDMIA"
		} else {
			arm.disasmEntry.Operator = "STMIA"
		}
		arm.disasmEntry.Operand = fmt.Sprintf("R%d!, {%#016b}", baseReg, regList)
	}

	// all ARM references say that the base register is updated as a result of
	// the multi-load. what isn't clear is what happens if the base register is
	// part of the update. observation of a bug in a confidential Andrew Davie
	// project however, demonstrates that we should *not* update the base
	// registere in those situations.
	//
	// this rule is not required for multiple store or for push/pop, where the
	// potential conflict never arises.
	updateBaseReg := true

	if load {
		numMatches := 0
		for i := 0; i <= 15; i++ {
			r := regList >> i
			if r&0x01 == 0x01 {
				// check if baseReg is being updated
				if i == int(baseReg) {
					updateBaseReg = false
				}

				numMatches++

				// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
				// - N cycle on first match
				// - S cycles on subsequent matches
				// - fillPipeline() will be called if PC register is matched
				if numMatches == 1 {
					arm.Ncycle(dataWrite, addr)
				} else {
					arm.Scycle(dataWrite, addr)
				}

				arm.registers[i] = arm.read32bit(addr)
				addr += 4
			}
		}

		// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Icycle()

		// no updating of base register if base register was part of the regList
		if updateBaseReg {
			arm.registers[baseReg] = addr
		}

		return
	}

	// store

	numMatches := 0
	for i := 0; i <= 15; i++ {
		r := regList >> i
		if r&0x01 == 0x01 {
			numMatches++

			// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// - storeRegisterCycles() on first match
			// - S cycles on subsequent match
			// - next prefetch cycle will be N
			if numMatches == 1 {
				arm.storeRegisterCycles(addr)
			} else {
				arm.Scycle(dataWrite, addr)
			}

			arm.write32bit(addr, arm.registers[i])
			addr += 4
		}
	}

	// write back the new base address
	arm.registers[baseReg] = addr
}

func (arm *ARM) executeConditionalBranch(opcode uint16) {
	// format 16 - Conditional branch
	cond := (opcode & 0x0f00) >> 8
	offset := uint32(opcode & 0x00ff)

	operator := ""
	b := false

	switch cond {
	case 0b0000:
		operator = "BEQ"
		b = arm.status.zero
	case 0b0001:
		operator = "BNE"
		b = !arm.status.zero
	case 0b0010:
		operator = "BCS"
		b = arm.status.carry
	case 0b0011:
		operator = "BCC"
		b = !arm.status.carry
	case 0b0100:
		operator = "BMI"
		b = arm.status.negative
	case 0b0101:
		operator = "BPL"
		b = !arm.status.negative
	case 0b0110:
		operator = "BVS"
		b = arm.status.overflow
	case 0b0111:
		operator = "BVC"
		b = !arm.status.overflow
	case 0b1000:
		operator = "BHI"
		b = arm.status.carry && !arm.status.zero
	case 0b1001:
		operator = "BLS"
		b = !arm.status.carry || arm.status.zero
	case 0b1010:
		operator = "BGE"
		b = (arm.status.negative && arm.status.overflow) || (!arm.status.negative && !arm.status.overflow)
	case 0b1011:
		operator = "BLT"
		b = (arm.status.negative && !arm.status.overflow) || (!arm.status.negative && arm.status.overflow)
	case 0b1100:
		operator = "BGT"
		b = !arm.status.zero && ((arm.status.negative && arm.status.overflow) || (!arm.status.negative && !arm.status.overflow))
	case 0b1101:
		operator = "BLE"
		b = arm.status.zero || ((arm.status.negative && !arm.status.overflow) || (!arm.status.negative && arm.status.overflow))
	case 0b1110:
		operator = "undefined branch"
		b = true
	case 0b1111:
		b = false
	}

	// offset is a nine-bit two's complement value
	offset <<= 1
	offset++

	var newPC uint32

	// get new PC value
	if offset&0x100 == 0x100 {
		// two's complement before subtraction
		offset ^= 0x1ff
		offset++
		newPC = arm.registers[rPC] - offset + 1
	} else {
		newPC = arm.registers[rPC] + offset + 1
	}

	// do branch
	if b {
		// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
		// - fillPipeline() will be called if necessary
		arm.registers[rPC] = newPC
	}

	switch arm.disasmLevel {
	case disasmFull:
		arm.disasmEntry.Operator = operator
		arm.disasmEntry.Operand = fmt.Sprintf("%04x", newPC-2)
		arm.disasmEntry.updateNotes = true
		fallthrough
	case disasmUpdateOnly:
		if b {
			arm.disasmEntry.ExecutionNotes = "branched"
		} else {
			arm.disasmEntry.ExecutionNotes = "next"
		}
	case disasmNone:
	}
}

func (arm *ARM) executeSoftwareInterrupt(opcode uint16) {
	// format 17 - Software interrupt"
	panic("Software interrupt")
}

func (arm *ARM) executeUnconditionalBranch(opcode uint16) {
	// format 18 - Unconditional branch
	offset := uint32(opcode&0x07ff) << 1

	if offset&0x800 == 0x0800 {
		// two's complement before subtraction
		offset ^= 0xfff
		offset++
		arm.registers[rPC] -= offset - 2
	} else {
		arm.registers[rPC] += offset + 2
	}

	if arm.disasmLevel == disasmFull {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "BAL"
			arm.disasmEntry.Operand = fmt.Sprintf("%04x ", arm.registers[rPC]-2)
		}
	}

	// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
	// - fillPipeline() will be called if necessary
}

func (arm *ARM) executeLongBranchWithLink(opcode uint16) {
	// format 19 - Long branch with link
	low := opcode&0x800 == 0x0800
	offset := uint32(opcode & 0x07ff)

	// there is no direct ARM equivalent for this instruction.

	if low {
		// second instruction

		offset <<= 1
		pc := arm.registers[rPC]
		arm.registers[rPC] = arm.registers[rLR] + offset
		arm.registers[rLR] = pc - 1

		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "BL"
			arm.disasmEntry.Operand = fmt.Sprintf("%#08x", arm.registers[rPC]-2)
		}

		// "7.4 Thumb Branch With Link" in "ARM7TDMI-S Technical Reference Manual r4p3"
		// -- no additional cycles for second instruction in BL
		// -- change of PC is captured by expectedPC check in Run() function loop

		return
	}

	// first instruction

	offset <<= 12

	if offset&0x400000 == 0x400000 {
		// two's complement before subtraction
		offset ^= 0x7fffff
		offset++
		arm.registers[rLR] = arm.registers[rPC] - offset + 2
	} else {
		arm.registers[rLR] = arm.registers[rPC] + offset + 2
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "bl"
		arm.disasmEntry.Operand = "-"
		arm.disasmEntry.ExecutionNotes = "first BL instruction"
	}

	// "7.4 Thumb Branch With Link" in "ARM7TDMI-S Technical Reference Manual r4p3"
	// -- no additional cycles for first instruction in BL
}