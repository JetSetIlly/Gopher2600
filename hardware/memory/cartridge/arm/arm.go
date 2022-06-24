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

package arm

import (
	"fmt"
	"math"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/memorymodel"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/logger"
)

// register names.
const (
	rSB = 9 + iota // static base
	rSL            // stack limit
	rFP            // frame pointer
	rIP            // intra-procedure-call scratch register
	rSP
	rLR
	rPC
	rCount
)

// the maximum number of cycles allowed in a single ARM program execution.
// no idea if this value is sufficient.
//
// 03/02/2022 - raised to accomodate CDFJBoulderDash development
const cycleLimit = 1000000

// the maximum number of instructions to execute. like cycleLimit but for when
// running in immediate mode
const instructionsLimit = cycleLimit / 3

// Architecture defines the features of the ARM core.
type Architecture string

// List of defined Architecture values. Not all features of the listed
// architectures may be implemented.
const (
	ARM7TDMI Architecture = "ARM7TDMI"
	ARMv7_M  Architecture = "ARMv7-M"
)

const (
	// accesses below this address are deemed to be probably null accesses. value
	// is arbitrary and was suggested by John Champeau (09/04/2022)
	nullAccessBoundaryARM7TDMI = 0x751

	// writing to GPIO "addresses" is allowed
	nullAccessBoundaryARMv7_m = 0x00
)

// ARM implements the ARM7TDMI-S LPC2103 processor.
type ARM struct {
	prefs *preferences.ARMPreferences
	arch  Architecture
	mmap  memorymodel.Map
	mem   SharedMemory
	hook  CartridgeHook

	// ARM registers
	registers [rCount]uint32
	Status    status

	// "peripherals" connected to the variety of ARM7TDMI-S used in the Harmony
	// cartridge.
	timer timer
	mam   mam

	// whether to foce an error on illegal memory access. set from ARM.prefs at
	// the start of every arm.Run()
	abortOnIllegalMem bool

	// whether to foce an error on illegal memory access. set from ARM.prefs at
	// the start of every arm.Run()
	abortOnStackCollision bool

	// nullAccessBoundary differs depending on the architecture
	nullAccessBoundary uint32

	// execution flags. set to false and/or error when Run() function should end
	continueExecution bool
	executionError    error

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
	//
	// updated from prefs on every Run() invocation
	Clk         float32
	clklenFlash float32

	// the PC of the instruction being executed
	executingPC uint32

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

	// the next disasmEntry to send to attached disassembler
	disasmExecutionNotes string
	disasmUpdateNotes    bool

	// the summary of the most recent disassembly
	disasmSummary DisasmSummary

	// interface to an option development package
	dev mapper.CartCoProcDeveloper

	// top of variable memory for stack pointer collision testing
	// * only valid if dev is not nil
	variableMemtop uint32

	// once the stack has been found to have collided with memory then all
	// memory accesses are deemed suspect and illegal accesses are no longer
	// logged
	stackHasCollided bool

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

	// \/\/\/ the following are reset at the end of each execution iteration \/\/\/

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

	// addresses of instructions that have been executed
	executedAddresses map[uint32]float32

	// control of 32bit thumb-2 function decoding.
	function32bit       bool
	function32bitOpcode uint16

	// disassembly of 32bit thumb-2
	// * temporary construct until thumb2Disassemble() is written
	fudge_thumb2disassemble32bit string
	fudge_thumb2disassemble16bit string
	fudge_thumb2disassemblePrint bool

	// address watches - apply to 32bit reads only
	readWatches []uint32

	// whether the previous execution stopped because of a yield
	yield bool
}

// NewARM is the preferred method of initialisation for the ARM type.
func NewARM(arch Architecture, mamcr MAMCR, mmap memorymodel.Map, prefs *preferences.ARMPreferences, mem SharedMemory, hook CartridgeHook, pathToROM string) *ARM {
	arm := &ARM{
		arch:         arch,
		prefs:        prefs,
		mmap:         mmap,
		mem:          mem,
		hook:         hook,
		executionMap: make(map[uint32][]func(_ uint16)),
		disasmCache:  make(map[uint32]DisasmEntry),

		// updated on every UpdatePrefs(). these are reasonable defaults
		Clk:         70.0,
		clklenFlash: 4.0,
	}

	// the mamcr to use as a preference
	arm.mam.preferredMAMCR = mamcr

	switch arm.arch {
	case ARM7TDMI:
		arm.nullAccessBoundary = nullAccessBoundaryARM7TDMI
	case ARMv7_M:
		arm.nullAccessBoundary = nullAccessBoundaryARMv7_m
	default:
		panic("unhandled ARM architecture: cannot set nullAccessBoundary")
	}

	arm.mam.mmap = mmap
	arm.timer.mmap = mmap

	arm.reset()
	arm.UpdatePrefs()

	return arm
}

// CoProcID implements the mapper.CartCoProcBus interface.
//
// CoProcID is the ID returned by the ARM type. This const value can be used
// for comparison purposes to check if a mapper.CartCoProcBus instance is of
// the ARM type.
func (arm *ARM) CoProcID() string {
	return string(arm.arch)
}

// SetDisassembler implements the mapper.CartCoProcBus interface.
func (arm *ARM) SetDisassembler(disasm mapper.CartCoProcDisassembler) {
	arm.disasm = disasm
}

// SetDeveloper implements the mapper.CartCoProcBus interface.
func (arm *ARM) SetDeveloper(dev mapper.CartCoProcDeveloper) {
	arm.dev = dev
}

// Plumb should be used to update the shared memory reference.
// Useful when used in conjunction with the rewind system.
func (arm *ARM) Plumb(mem SharedMemory, hook CartridgeHook) {
	arm.mem = mem
	arm.hook = hook
}

// ClearCaches should be used very rarely. It empties the instruction and
// disassembly caches.
func (arm *ARM) ClearCaches() {
	arm.executionMap = make(map[uint32][]func(_ uint16))
	arm.disasmCache = make(map[uint32]DisasmEntry)
}

// reset ARM registers.
func (arm *ARM) reset() {
	arm.Status.reset()

	for i := 0; i < rSP; i++ {
		arm.registers[i] = 0x00000000
	}

	arm.registers[rSP], arm.registers[rLR], arm.registers[rPC] = arm.mem.ResetVectors()
}

// AddReadWatch adds an address to the list of addresses that will cause a
// yield on a 32bit data read (not opcode reads).
func (arm *ARM) AddReadWatch(addr uint32) {
	arm.readWatches = append(arm.readWatches, addr)
}

// UpdatePrefs should be called periodically to ensure that the current
// preference values are being used in the ARM emulation.
func (arm *ARM) UpdatePrefs() {
	// update clock value from preferences
	arm.Clk = float32(arm.prefs.Clock.Get().(float64))

	// latency in megahertz
	latencyInMhz := (1 / (arm.prefs.FlashLatency.Get().(float64) / 1000000000)) / 1000000
	arm.clklenFlash = float32(math.Ceil(float64(arm.Clk) / latencyInMhz))

	// set mamcr on startup
	arm.mam.pref = arm.prefs.MAM.Get().(int)
	if arm.mam.pref == preferences.MAMDriver {
		arm.mam.setPreferredMamcr()
		arm.mam.mamtim = 4.0
	} else {
		arm.mam.setMAMCR(MAMCR(arm.mam.pref))
		arm.mam.mamtim = 4.0
	}

	// set cycle counting functions
	arm.immediateMode = arm.prefs.Immediate.Get().(bool)
	if arm.immediateMode {
		arm.Icycle = arm.iCycleStub
		arm.Scycle = arm.sCycleStub
		arm.Ncycle = arm.nCycleStub
		arm.disasmSummary.ImmediateMode = true
	} else {
		arm.Icycle = arm.iCycle
		arm.Scycle = arm.sCycle
		arm.Ncycle = arm.nCycle
	}

	// how to handle illegal memory access
	arm.abortOnIllegalMem = arm.prefs.AbortOnIllegalMem.Get().(bool)
	arm.abortOnStackCollision = arm.prefs.AbortOnStackCollision.Get().(bool)
}

// resetExecution is differnt to ARM in that it does not reset the state of the
// ARM processor itself only the state related to the current execution (cycles
// consumed etc.)
func (arm *ARM) resetExecution() error {
	// reset cycles count
	arm.cyclesTotal = 0
	arm.prefetchCycle = S

	// reset execution flags
	arm.continueExecution = true
	arm.executionError = nil
	arm.yield = false

	// reset disasm notes/flags
	arm.disasmExecutionNotes = ""
	arm.disasmUpdateNotes = false

	arm.memoryError = false

	if arm.dev != nil {
		arm.executedAddresses = make(map[uint32]float32)
	}

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
	arm.timer.stepFromVCS(arm.Clk, vcsClock)
}

// Run will continue until the ARM program encounters a switch from THUMB mode
// to ARM mode. Execution will halt under the following conditions
//
// 1) The number of cycles for the entire program is too great
// 2) A yield condition has been met (eg. a watch address has been triggered)
// 3) Execution mode has changed from Thumb to ARM (ARM7TDMI architecture only)
//
// Returns the number of ARM cycles consumed and any errors.
func (arm *ARM) Run() (float32, error) {
	arm.reset()

	// must happen after reset()
	err := arm.resetExecution()
	if err != nil {
		return 0, err
	}

	// fill pipeline must happen after resetExecution()
	arm.registers[rPC] += 2

	return arm.run()
}

// Continue execution from a previously yielded execution. ARM program is
// unpredictable if it was not initially started with a call to Run().
//
// Returns the number of ARM cycles consumed and any errors.
func (arm *ARM) Continue() (float32, error) {
	err := arm.resetExecution()
	if err != nil {
		return 0, err
	}

	return arm.run()
}

func (arm *ARM) run() (float32, error) {
	if arm.dev != nil {
		// update variableMemtop - probably hasn't changed but you never know
		arm.variableMemtop = arm.dev.VariableMemtop()
		// profile executed addresses at end of function
		defer arm.dev.ExecutionProfile(arm.executedAddresses)
	}

	if arm.disasm != nil {
		// start of program execution
		arm.disasmSummary = DisasmSummary{}
		arm.disasm.Start()

		defer arm.disasm.End(arm.disasmSummary)
	}

	var err error

	// use to detect branches and whether to fill the pipeline (unused if
	// arm.immediateMode is true)
	var expectedPC uint32

	// number of iterations. only used when in immediate mode
	var iterations int

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

		// check program counter
		memIdx := arm.executingPC - arm.programMemoryOffset
		if memIdx < 0 || memIdx+1 >= uint32(arm.programMemoryLen) {
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
			if memIdx < 0 || memIdx+1 >= uint32(arm.programMemoryLen) {
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

		// note stack pointer. we'll use this to check if stack pointer has
		// collided with variables memory
		stackPointerBeforeExecution := arm.registers[rSP]

		// run from functionMap if possible
		switch arm.arch {
		case ARM7TDMI:
			f := arm.functionMap[memIdx]
			if f == nil {
				f = arm.decodeThumb(opcode)
				arm.functionMap[memIdx] = f
			}
			f(opcode)
		case ARMv7_M:
			var f func(uint16)

			// taking a note of whether this is a resolution of a 32bit
			// instruction. we use this later during the fudge_ disassembly
			// printing
			fudge_resolving32bitInstruction := arm.function32bit

			// process a 32 bit or 16 bit instruction as appropriate
			if arm.function32bit {
				arm.function32bit = false
				f = arm.functionMap[memIdx]
				if f == nil {
					f = arm.decode32bitThumb2(arm.function32bitOpcode)
					arm.functionMap[memIdx] = f
				}
			} else {
				if arm.is32BitThumb2(opcode) {
					arm.function32bit = true
					arm.function32bitOpcode = opcode

					// we need something for the emulation to run. this is a
					// clearer alternative to having a flag
					f = func(_ uint16) {}
				} else {
					f = arm.functionMap[memIdx]
					if f == nil {
						f = arm.decodeThumb2(opcode)
						arm.functionMap[memIdx] = f
					}
				}
			}

			// whether instruction was prevented from executing by IT block. we
			// use this later during the fudge_ disassembly printing
			fudge_notExecuted := false

			// new 32bit functions always execute
			// if the opcode indicates that this is a 32bit thumb instruction
			// then we need to resolve that regardless of any IT block
			// conditional execution of instructions
			if arm.Status.itMask != 0b0000 {
				r := arm.Status.condition(arm.Status.itCond)

				if r {
					f(opcode)
				} else {
					// "A7.3.2: Conditional execution of undefined instructions
					//
					// If an undefined instruction fails a condition check in Armv7-M, the instruction
					// behaves as a NOP and does not cause an exception"
					//
					// page A7-179 of the "ARMv7-M Architecture Reference Manual"
					fudge_notExecuted = true
				}

				// update IT conditions only if the opcode is not a 32bit opcode
				// update LSB of IT condition by copying the MSB of the IT mask
				arm.Status.itCond &= 0b1110
				arm.Status.itCond |= (arm.Status.itMask >> 3)

				// shift IT mask
				arm.Status.itMask = (arm.Status.itMask << 1) & 0b1111
			} else {
				f(opcode)
			}

			// if arm.function32bitOpcode == 0xf084 && opcode == 0x0280 {
			// 	arm.fudge_thumb2disassemblePrint = true
			// }

			if arm.fudge_thumb2disassemblePrint {
				if fudge_notExecuted {
					fmt.Print("*** ")
				}
				if fudge_resolving32bitInstruction {
					fmt.Printf("%04x %04x :: %s\n", arm.function32bitOpcode, opcode, arm.fudge_thumb2disassemble32bit)
					fmt.Println(arm.String())
					fmt.Println(arm.Status.String())
					fmt.Println("====================")
				} else if !arm.function32bit {
					if arm.fudge_thumb2disassemble16bit != "" {
						fmt.Printf("%04x :: %s\n", opcode, arm.fudge_thumb2disassemble16bit)
					} else {
						fmt.Printf("%04x :: %s\n", opcode, thumbDisassemble(opcode).String())
					}
					fmt.Println(arm.String())
					fmt.Println(arm.Status.String())
					fmt.Println("====================")
				}
			}
			arm.fudge_thumb2disassemble32bit = ""
			arm.fudge_thumb2disassemble16bit = ""

		default:
			panic("unsupported ARM architecture")
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
			var cached bool
			var d DisasmEntry

			d, cached = arm.disasmCache[arm.executingPC]
			if !cached {
				d = Disassemble(opcode)
				d.Address = fmt.Sprintf("%08x", arm.executingPC)
			}

			d.MAMCR = int(arm.mam.mamcr)
			d.BranchTrail = arm.branchTrail
			d.MergedIS = arm.mergedIS
			d.CyclesSequence = arm.cycleOrder.String()
			d.ExecutionNotes = arm.disasmExecutionNotes

			// update cycle information
			d.Cycles = arm.cycleOrder.len()

			// update cache if necessary
			if !cached || arm.disasmUpdateNotes {
				arm.disasmCache[arm.executingPC] = d
			}

			arm.disasmExecutionNotes = ""
			arm.disasmUpdateNotes = false

			// update program cycles
			arm.disasmSummary.add(arm.cycleOrder)

			// we always send the instruction to the disasm interface
			arm.disasm.Step(d)
		}

		// accumulate execution counts
		if arm.dev != nil {
			arm.executedAddresses[arm.executingPC] += arm.stretchedCycles
		}

		// reset cycle information
		if !arm.immediateMode {
			arm.branchTrail = BranchTrailNotUsed
			arm.mergedIS = false
			arm.stretchedCycles = 0
			arm.cycleOrder.reset()

			// limit the number of cycles used by the ARM program
			if arm.cyclesTotal >= cycleLimit {
				logger.Logf("ARM7", "reached cycle limit of %d. ending execution early", cycleLimit)
				panic(1)
				break
			}
		} else {
			iterations++
			if iterations > instructionsLimit {
				logger.Logf("ARM7", "reached instructions limit of %d. ending execution early", instructionsLimit)
				panic(2)
				break
			}
		}

		// check stack pointer before iterating loop again
		if arm.dev != nil && stackPointerBeforeExecution != arm.registers[rSP] {
			if !arm.stackHasCollided && arm.registers[rSP] <= arm.variableMemtop {
				event := "Stack"
				logger.Logf("ARM7", "%s: collision with program memory (%08x)", event, arm.registers[rSP])

				log := arm.dev.StackCollision(arm.executingPC, arm.registers[rSP])
				if log != "" {
					logger.Logf("ARM7", "%s: %s", event, log)
				}

				if arm.abortOnStackCollision {
					logger.Logf("ARM7", "aborting thumb program early")
					break
				}

				// set stackHasCollided flag. this means that memory accesses
				// will no longer be checked for legality
				arm.stackHasCollided = true
			}
		}

		// abort if a watch has been triggered
		if arm.yield {
			break
		}
	}

	// indicate that program abort was because of illegal memory access
	if arm.memoryError {
		logger.Logf("ARM7", "illegal memory access detected. aborting thumb program early")
	}

	// end of program execution

	if arm.executionError != nil {
		return 0, curated.Errorf("ARM7: %v", arm.executionError)
	}

	return arm.cyclesTotal, nil
}
