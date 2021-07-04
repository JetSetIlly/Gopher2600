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
const clk = float32(70)
const clklenFlash = float32(4)

// ARM implements the ARM7TDMI-S LPC2103 processor.
type ARM struct {
	prefs *preferences.ARMPreferences
	mmap  MemoryMap
	mem   SharedMemory
	hook  CartridgeHook

	// execution flags. set to false and/or error when Run() function should end
	continueExecution bool
	executionError    error

	// ARM registers
	registers [rCount]uint32
	status    status

	// "peripherals" connected to the variety of ARM7TDMI-S used in the Harmony
	// cartridge.
	timer timer
	mam   mam

	// cycles this instruction. reset every Run() iteration
	N      int
	I      int
	S      int
	cycles float32

	// what happened with any branch that might have occured this instruction
	branchTrail BranchTrail
	mergedIS    bool
	mergedN     bool

	// a record of cycle types. used to decide whether to merge I-S cycles
	prevCycles [2]cycleType

	// the type of cycle next prefetch (the main PC increment in the Run()
	// loop) should be. either N or S type. never I type.
	prefetchCycle cycleType

	// total number of cycles for the entire program
	cyclesTotal float32

	// the area the PC covers. once assigned we'll assume that the program
	// never reads outside this area. the value is assigned on reset()
	programMemory *[]uint8

	// the amount to adjust the memory address by so that it can be used to
	// index the programMemory array
	programMemoryOffset uint32

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
	disasmCache map[uint32]mapper.CartCoProcDisasmEntry

	// the level of disassemble to perform next instruction
	disasmLevel disasmLevel

	// the next disasmEntry to send to attached disassembler
	disasmEntry mapper.CartCoProcDisasmEntry
}

type disasmLevel int

const (
	disasmNone disasmLevel = iota

	// update entry only if the UpdateExecution is true
	disasmUpdateOnly

	// update all disassembly fields (operator, operands, etc.). this doesn't
	// need to happen unless the entry is not in the disasm cache
	disasmFull
)

// NewARM is the preferred method of initialisation for the ARM type.
func NewARM(mmap MemoryMap, prefs *preferences.ARMPreferences, mem SharedMemory, hook CartridgeHook) *ARM {
	arm := &ARM{
		prefs:        prefs,
		mmap:         mmap,
		mem:          mem,
		hook:         hook,
		executionMap: make(map[uint32][]func(_ uint16)),
	}

	arm.mam.mmap = mmap
	arm.timer.mmap = mmap
	arm.Plumb()

	return arm
}

func (arm *ARM) Plumb() error {
	// reseting on Plumb() seems odd but we reset() on every call to Run()
	// anyway. that is to say the ARM isn't stateful between executions - apart
	// from shared memory but that's handled outside of the arm7tdmi package.
	err := arm.reset()
	if err != nil {
		return err
	}
	return nil
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
	if arm.disasm == nil {
		arm.disasmCache = nil
	} else {
		arm.disasmCache = make(map[uint32]mapper.CartCoProcDisasmEntry)
	}
}

// PlumbSharedMemory should be used to update the shared memory reference.
// Useful when used in conjunction with the rewind system.
func (arm *ARM) PlumbSharedMemory(mem SharedMemory) {
	arm.mem = mem
}

func (arm *ARM) reset() error {
	arm.status.reset()
	for i := range arm.registers {
		arm.registers[i] = 0x00000000
	}
	arm.registers[rSP], arm.registers[rLR], arm.registers[rPC] = arm.mem.ResetVectors()

	// a peculiarity of the ARM is that the PC is 2 bytes ahead of where we'll
	// be reading from. adjust PC so that this is correct.
	arm.registers[rPC] += 2

	// reset execution flags
	arm.continueExecution = true
	arm.executionError = nil

	// reset cycles count
	arm.cyclesTotal = 0
	arm.prefetchCycle = S

	return arm.findProgramMemory()
}

// find program memory using current program counter value.
func (arm *ARM) findProgramMemory() error {
	arm.programMemory, arm.programMemoryOffset = arm.mem.MapAddress(arm.registers[rPC], false)
	if arm.programMemory == nil {
		return curated.Errorf("ARM: cannot find program memory")
	}

	arm.programMemoryOffset = arm.registers[rPC] - arm.programMemoryOffset

	if m, ok := arm.executionMap[arm.programMemoryOffset]; ok {
		arm.functionMap = m
	} else {
		arm.executionMap[arm.programMemoryOffset] = make([]func(_ uint16), len(*arm.programMemory))
		arm.functionMap = arm.executionMap[arm.programMemoryOffset]
	}

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
	arm.timer.stepFromVCS(clk, vcsClock)
}

func (arm *ARM) read8bit(addr uint32) uint8 {
	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if v, ok := arm.timer.read(addr); ok {
			return uint8(v)
		}
		if v, ok := arm.mam.read(addr); ok {
			return uint8(v)
		}
		logger.Logf("ARM7", "read8bit: unrecognised address %08x", addr)
		return 0
	}

	return (*mem)[addr]
}

func (arm *ARM) write8bit(addr uint32, val uint8) {
	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if ok := arm.timer.write(addr, uint32(val)); ok {
			return
		}
		if ok := arm.mam.write(addr, uint32(val)); ok {
			return
		}
		logger.Logf("ARM7", "write8bit: unrecognised address %08x", addr)
		return
	}

	(*mem)[addr] = val
}

func (arm *ARM) read16bit(addr uint32) uint16 {
	// check 16 bit alignment
	if addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit read (%08x)", addr)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if v, ok := arm.timer.read(addr); ok {
			return uint16(v)
		}
		if v, ok := arm.mam.read(addr); ok {
			return uint16(v)
		}
		logger.Logf("ARM7", "read16bit: unrecognised address %08x", addr)
		return 0
	}

	return uint16((*mem)[addr]) | (uint16((*mem)[addr+1]) << 8)
}

func (arm *ARM) write16bit(addr uint32, val uint16) {
	// check 16 bit alignment
	if addr&0x01 != 0x00 {
		logger.Logf("ARM7", "misaligned 16 bit write (%08x)", addr)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if ok := arm.timer.write(addr, uint32(val)); ok {
			return
		}
		if ok := arm.mam.write(addr, uint32(val)); ok {
			return
		}
		logger.Logf("ARM7", "write16bit: unrecognised address %08x", addr)
		return
	}

	(*mem)[addr] = uint8(val)
	(*mem)[addr+1] = uint8(val >> 8)
}

func (arm *ARM) read32bit(addr uint32) uint32 {
	// check 32 bit alignment
	if addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit read (%08x)", addr)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, false)
	if mem == nil {
		if v, ok := arm.timer.read(addr); ok {
			return v
		}
		if v, ok := arm.mam.read(addr); ok {
			return v
		}
		logger.Logf("ARM7", "read32bit: unrecognised address %08x", addr)
		return 0
	}

	return uint32((*mem)[addr]) | (uint32((*mem)[addr+1]) << 8) | (uint32((*mem)[addr+2]) << 16) | uint32((*mem)[addr+3])<<24
}

func (arm *ARM) write32bit(addr uint32, val uint32) {
	// check 32 bit alignment
	if addr&0x03 != 0x00 {
		logger.Logf("ARM7", "misaligned 32 bit write (%08x)", addr)
	}

	var mem *[]uint8

	mem, addr = arm.mem.MapAddress(addr, true)
	if mem == nil {
		if ok := arm.timer.write(addr, val); ok {
			return
		}
		if ok := arm.mam.write(addr, val); ok {
			return
		}
		logger.Logf("ARM7", "write32bit: unrecognised address %08x", addr)
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
// Returns the number of ARM cycles and any errors.
func (arm *ARM) Run(mamcr uint32) (uint32, float32, error) {
	err := arm.reset()
	if err != nil {
		return arm.mam.mamcr, 0, err
	}

	// set mamcr on startup
	arm.mam.pref = arm.prefs.MAM.Get().(int)
	if arm.mam.pref == preferences.MAMDriver {
		arm.mam.setMAMCR(mamcr)
		arm.mam.mamtim = 4.0
	} else {
		arm.mam.setMAMCR(uint32(arm.mam.pref))
		arm.mam.mamtim = 4.0
	}

	// start of program execution
	if arm.disasm != nil {
		arm.disasm.Start()
	}

	// what we send at the end of the execution. not used if not disassembler is set
	programExecution := ExecutionDetails{}

	// loop through instructions until we reach an exit condition
	for arm.continueExecution {
		// reset instruction specific information
		arm.N = 0
		arm.I = 0
		arm.S = 0
		arm.cycles = 0
		arm.branchTrail = BranchTrailNotUsed
		arm.mergedIS = false
		arm.mergedN = false

		// -2 adjustment to PC register to account for pipeline
		pc := arm.registers[rPC] - 2

		// set disasmLevel for the next instruction
		if arm.disasm == nil {
			arm.disasmLevel = disasmNone
		} else {
			// full disassembly unless we can find a usable entry in the disasm cache
			arm.disasmLevel = disasmFull

			// check cache for existing disasm entry
			if e, ok := arm.disasmCache[pc]; ok {
				// use cached entry
				arm.disasmEntry = e

				if arm.disasmEntry.UpdateNotes {
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
				arm.disasmEntry.Address = fmt.Sprintf("%08x", pc)
				arm.disasmEntry.Operator = ""
				arm.disasmEntry.Operand = ""
				arm.disasmEntry.Cycles = 0.0
				arm.disasmEntry.ExecutionNotes = ""
				arm.disasmEntry.UpdateNotes = false
			}
		}

		// check program counter
		idx := pc - arm.programMemoryOffset
		if idx+1 >= uint32(len(*arm.programMemory)) {
			// program counter is out-of-range so find program memory again
			// (using the PC value)
			err = arm.findProgramMemory()
			if err != nil {
				// can't find memory so we say the ARM program has finished inadvertently
				logger.Logf("ARM7", "PC out of range (%#08x). finishing arm program early", arm.registers[rPC])
				return arm.mam.mamcr, arm.cyclesTotal, nil
			}

			// if it's still out-of-range then give up with an error
			idx = pc - arm.programMemoryOffset
			if idx+1 >= uint32(len(*arm.programMemory)) {
				// can't find memory so we say the ARM program has finished inadvertently
				logger.Logf("ARM7", "PC out of range (%#08x). finishing arm program early", arm.registers[rPC])
				return arm.mam.mamcr, arm.cyclesTotal, nil
			}
		}

		// store what the prefetch cycle was. we'll use this to adjust the
		// disassembly cycle information after the execution.
		thisPrefetchCycle := arm.prefetchCycle

		// read next instruction
		opcode := uint16((*arm.programMemory)[idx]) | (uint16((*arm.programMemory)[idx+1]) << 8)
		arm.registers[rPC] += 2
		if arm.prefetchCycle == N {
			arm.Ncycle(prefetch, arm.registers[rPC])

			// prefetch cycle is always S unless told otherwise
			arm.prefetchCycle = S
		} else {
			arm.Scycle(prefetch, arm.registers[rPC])
		}

		// run from executionMap if possible
		formatFunc := arm.functionMap[idx]
		if formatFunc != nil {
			formatFunc(opcode)
		} else {
			// working backwards up the table in Figure 5-1 of the ARM7TDMI Data Sheet.
			if opcode&0xf000 == 0xf000 {
				// format 19 - Long branch with link
				f := arm.executeLongBranchWithLink
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf000 == 0xe000 {
				// format 18 - Unconditional branch
				f := arm.executeUnconditionalBranch
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xff00 == 0xdf00 {
				// format 17 - Software interrupt"
				f := arm.executeSoftwareInterrupt
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf000 == 0xd000 {
				// format 16 - Conditional branch
				f := arm.executeConditionalBranch
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf000 == 0xc000 {
				// format 15 - Multiple load/store
				f := arm.executeMultipleLoadStore
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf600 == 0xb400 {
				// format 14 - Push/pop registers
				f := arm.executePushPopRegisters
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xff00 == 0xb000 {
				// format 13 - Add offset to stack pointer
				f := arm.executeAddOffsetToSP
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf000 == 0xa000 {
				// format 12 - Load address
				f := arm.executeLoadAddress
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf000 == 0x9000 {
				// format 11 - SP-relative load/store
				f := arm.executeSPRelativeLoadStore
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf000 == 0x8000 {
				// format 10 - Load/store halfword
				f := arm.executeLoadStoreHalfword
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xe000 == 0x6000 {
				// format 9 - Load/store with immediate offset
				f := arm.executeLoadStoreWithImmOffset
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf200 == 0x5200 {
				// format 8 - Load/store sign-extended byte/halfword
				f := arm.executeLoadStoreSignExtendedByteHalford
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf200 == 0x5000 {
				// format 7 - Load/store with register offset
				f := arm.executeLoadStoreWithRegisterOffset
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf800 == 0x4800 {
				// format 6 - PC-relative load
				f := arm.executePCrelativeLoad
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xfc00 == 0x4400 {
				// format 5 - Hi register operations/branch exchange
				f := arm.executeHiRegisterOps
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xfc00 == 0x4000 {
				// format 4 - ALU operations
				f := arm.executeALUoperations
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xe000 == 0x2000 {
				// format 3 - Move/compare/add/subtract immediate
				f := arm.executeMovCmpAddSubImm
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xf800 == 0x1800 {
				// format 2 - Add/subtract
				f := arm.executeAddSubtract
				arm.functionMap[idx] = f
				f(opcode)
			} else if opcode&0xe000 == 0x0000 {
				// format 1 - Move shifted register
				f := arm.executeMoveShiftedRegister
				arm.functionMap[idx] = f
				f(opcode)
			} else {
				panic("undecoded instruction")
			}
		}

		// adjust disassembly cycle information based on the this and the next
		// prefetch cycle type
		if arm.prefetchCycle == N && thisPrefetchCycle != N {
			arm.S--
			arm.N++
		}

		// increases total number of program cycles by the stretched cycles for this instruction
		arm.cyclesTotal += arm.cycles

		// update timer. assuming an APB divider value of one.
		arm.timer.step(arm.cycles)

		// send disasm information to disassembler
		if arm.disasm != nil {
			ed := ExecutionDetails{
				N:           int(arm.N),
				I:           int(arm.I),
				S:           int(arm.S),
				MAMCR:       int(arm.mam.mamcr),
				BranchTrail: arm.branchTrail,
				MergedIS:    arm.mergedIS,
				MergedN:     arm.mergedN,
			}

			// update cycle information
			arm.disasmEntry.Cycles = float32(arm.N + arm.I + arm.S)
			arm.disasmEntry.ExecutionDetails = ed

			// update program cycles
			programExecution.N += ed.N
			programExecution.I += ed.I
			programExecution.S += ed.S

			// only send if operator field is not equal to 'blfirst'. the first
			// instruction in a BL sequence is not shown
			if arm.disasmEntry.Operator != blfirst {
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
					arm.disasmEntry.ExecutionDetails = ed
					arm.disasmCache[pc] = arm.disasmEntry
				case disasmUpdateOnly:
					// entry is cached but notes may have changed so we recache
					// the entry
					arm.disasmEntry.ExecutionDetails = ed
					arm.disasmCache[pc] = arm.disasmEntry
				case disasmNone:
				}

				// we always send the instruction to the disasm interface
				arm.disasm.Step(arm.disasmEntry)
			}
		}
	}

	// end of program execution
	if arm.disasm != nil {
		arm.disasm.End(programExecution)
	}

	if arm.executionError != nil {
		return arm.mam.mamcr, 0, curated.Errorf("ARM: %v", arm.executionError)
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
			arm.registers[destReg] = arm.registers[srcReg]
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
			arm.status.carry = arm.registers[srcReg]&0x80000000 == 0x80000000
			arm.registers[destReg] = 0x00
		} else {
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			arm.registers[destReg] = arm.registers[srcReg] >> shift
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
			arm.status.carry = arm.registers[srcReg]&0x80000000 == 0x80000000
			if arm.status.carry {
				arm.registers[destReg] = 0xffffffff
			} else {
				arm.registers[destReg] = 0x00000000
			}
		} else { // shift > 0
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = src&m == m
			arm.registers[destReg] = uint32(int32(arm.registers[srcReg]) >> shift)
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
	if destReg == rPC {
		arm.pcCycle()
	}
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
	if destReg == rPC {
		arm.pcCycle()
	}
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
			m := uint32(0x01) << (shift - 1)
			arm.status.carry = arm.registers[destReg]&m == m
			arm.registers[destReg] = uint32(int32(arm.registers[destReg]) >> shift)
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
		if destReg == rPC {
			arm.pcCycle()
		}
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

		arm.status.setCarry(arm.registers[destReg], ^arm.registers[srcReg], 0)
		arm.status.setOverflow(arm.registers[destReg], ^arm.registers[srcReg], 0)
		cmp := arm.registers[destReg] - arm.registers[srcReg]
		arm.status.isZero(cmp)
		arm.status.isNegative(cmp)
	case 0b10:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "MOV"
			arm.disasmEntry.Operand = fmt.Sprintf("%s%d, %s%d ", destLabel, destReg, srcLabel, srcReg)
		}
		arm.registers[destReg] = arm.registers[srcReg]
		// status register not changed
	case 0b11:
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "BX"
			arm.disasmEntry.Operand = fmt.Sprintf("%s%d ", srcLabel, srcReg)
		}

		thumbMode := arm.registers[srcReg]&0x01 == 0x01

		var newPC uint32

		// If R15 is used as an operand, the value will be the address of the instruction + 4 with
		// bit 0 cleared. Executing a BX PC in THUMB state from a non-word aligned address
		// will result in unpredictable execution.
		if srcReg == 15 {
			// PC is already +2 from the instruction address
			newPC = arm.registers[rPC] + 2
		} else {
			newPC = (arm.registers[srcReg] & 0x7ffffffe) + 2
		}

		if thumbMode {
			arm.registers[rPC] = newPC
		} else {
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
				arm.disasmEntry.ExecutionNotes = res.InterruptEvent
				arm.disasmEntry.UpdateNotes = true
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
		}
	}

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	if destReg == rPC {
		arm.pcCycle()
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
	arm.Ncycle(dataRead, addr)
	arm.Icycle()
	if destReg == rPC {
		arm.pcCycle()
	}
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
			arm.Ncycle(dataRead, addr)
			arm.Icycle()
			if reg == rPC {
				arm.pcCycle()
			}

			return
		}

		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, R%d] ", reg, baseReg, offsetReg)
		}

		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Ncycle(dataRead, addr)
		arm.Icycle()
		if reg == rPC {
			arm.pcCycle()
		}

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
			arm.Ncycle(dataRead, addr)
			arm.Icycle()
			if reg == rPC {
				arm.pcCycle()
			}

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
		arm.Ncycle(dataRead, addr)
		arm.Icycle()
		if reg == rPC {
			arm.pcCycle()
		}

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
		arm.Ncycle(dataRead, addr)
		arm.Icycle()
		if reg == rPC {
			arm.pcCycle()
		}

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
			arm.Ncycle(dataRead, addr)
			arm.Icycle()
			if reg == rPC {
				arm.pcCycle()
			}

			return
		}

		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "LDR"
			arm.disasmEntry.Operand = fmt.Sprintf("R%d, [R%d, #%02x] ", reg, baseReg, offset)
		}
		arm.registers[reg] = arm.read32bit(addr)

		// "7.8 Load Register" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Ncycle(dataRead, addr)
		arm.Icycle()
		if reg == rPC {
			arm.pcCycle()
		}

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
		arm.Ncycle(dataRead, addr)
		arm.Icycle()
		if reg == rPC {
			arm.pcCycle()
		}

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
		arm.Ncycle(dataRead, addr)
		arm.Icycle()
		if reg == rPC {
			arm.pcCycle()
		}

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
		if destReg == rPC {
			arm.pcCycle()
		}

		return
	}

	if arm.disasmLevel == disasmFull {
		arm.disasmEntry.Operator = "ADD"
		arm.disasmEntry.Operand = fmt.Sprintf("R%d, PC, #%02x] ", destReg, offset)
	}

	arm.registers[destReg] = arm.registers[rPC] + uint32(offset)

	// "7.6 Data Operations" in "ARM7TDMI-S Technical Reference Manual r4p3"
	if destReg == rPC {
		arm.pcCycle()
	}
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
		num := 0
		for i := 0; i <= 7; i++ {
			// shift single-bit mask
			m := uint8(0x01 << i)

			// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			if i == 0 {
				arm.Ncycle(dataRead, addr)
			} else {
				arm.Scycle(dataRead, addr)
			}

			// read register if indicated by regList
			if regList&m == m {
				arm.registers[i] = arm.read32bit(addr)
				addr += 4
				num++
			}
		}

		// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Icycle()

		// load PC register after all other registers
		if pclr {
			// chop the odd bit off the new PC value
			v := arm.read32bit(addr) & 0xfffffffe

			// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			arm.Ncycle(dataRead, addr)

			// add two to the new PC value. not sure why this is. it's not
			// described in the pseudo code above but I think it's to do with
			// how the ARM CPU does prefetching and when the adjustment is
			// applied. anwyay, this works but it might be worth figuring out
			// where else to apply the adjustment and whether that would be any
			// clearer.
			v += 2

			arm.registers[rPC] = v
			addr += 4

			if arm.disasmLevel == disasmFull {
				arm.disasmEntry.Operator = "POP"
				arm.disasmEntry.Operand = fmt.Sprintf("{%#0b, LR}", regList)
			}

			num++
		} else if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "POP"
			arm.disasmEntry.Operand = fmt.Sprintf("{%#0b}", regList)
		}

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
			arm.disasmEntry.Operand = fmt.Sprintf("{%#0b, LR}", regList)
		}
		c = (uint32(bits.OnesCount8(regList)) + 1) * 4
	} else {
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "PUSH"
			arm.disasmEntry.Operand = fmt.Sprintf("{%#0b}", regList)
		}
		c = uint32(bits.OnesCount8(regList)) * 4
	}

	// push occurs from the new low stack address upwards to the current stack
	// address (before the pushes)
	addr := arm.registers[rSP] - c

	// write each register in turn (from lower to highest)
	num := 0
	for i := 0; i <= 7; i++ {
		// shift single-bit mask
		m := uint8(0x01 << i)

		// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		if i == 0 {
			arm.Ncycle(dataWrite, addr)
		} else {
			arm.Scycle(dataWrite, addr)
		}

		// write register if indicated by regList
		if regList&m == m {
			arm.write32bit(addr, arm.registers[i])
			addr += 4
			num++
		}
	}

	// write LR register after all the other registers
	if pclr {
		lr := arm.registers[rLR]
		arm.write32bit(addr, lr)
		num++

		// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Ncycle(dataWrite, addr)
	}

	// update stack pointer. note that this is the address we started the push
	// sequence from above. this is correct.
	arm.registers[rSP] -= c

	// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.Icycle()
	arm.storeRegisterCycles(addr)
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
		arm.disasmEntry.Operand = fmt.Sprintf("R%d!, {%#0b}", baseReg, regList)
	}

	num := 0
	for i := 0; i <= 15; i++ {
		r := regList >> i
		if r&0x01 == 0x01 {
			// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
			if i == 0 {
				arm.Ncycle(dataWrite, addr)
			} else {
				arm.Scycle(dataWrite, addr)
			}

			if load {
				arm.registers[i] = arm.read32bit(addr)
				addr += 4

			} else {
				arm.write32bit(addr, arm.registers[i])
				addr += 4
			}
			num++
		}
	}

	// write back the new base address
	arm.registers[baseReg] = addr

	if load {
		// "7.10 Load Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Icycle()
	} else {
		// "7.11 Store Multiple Registers" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.storeRegisterCycles(addr)
	}
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
		newPC = arm.registers[rPC] - offset
	} else {
		newPC = arm.registers[rPC] + offset
	}

	// do branch
	if b {
		// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Ncycle(branch, arm.registers[rPC])
		arm.Scycle(prefetch, newPC+1)

		arm.registers[rPC] = newPC + 1
	}

	switch arm.disasmLevel {
	case disasmFull:
		arm.disasmEntry.Operator = operator
		arm.disasmEntry.Operand = fmt.Sprintf("%04x", newPC)
		arm.disasmEntry.UpdateNotes = true
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

	oldPC := arm.registers[rPC]

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
			arm.disasmEntry.Operand = fmt.Sprintf("%04x ", arm.registers[rPC])
		}
	}

	// "7.3 Branch ..." in "ARM7TDMI-S Technical Reference Manual r4p3"
	arm.Ncycle(branch, oldPC)
	arm.Scycle(prefetch, arm.registers[rPC])
}

// special operator mnemonic to mark the first instruction in the BL sequence
const blfirst = "BLFIRST"

func (arm *ARM) executeLongBranchWithLink(opcode uint16) {
	// format 19 - Long branch with link
	low := opcode&0x800 == 0x0800
	offset := uint32(opcode & 0x07ff)

	// there is no direct ARM equivalent for this instruction.

	if low {
		offset <<= 1
		arm.registers[rLR] += offset
		pc := arm.registers[rPC]
		arm.registers[rPC] = arm.registers[rLR]
		arm.registers[rLR] = pc - 1
		if arm.disasmLevel == disasmFull {
			arm.disasmEntry.Operator = "BL"
			arm.disasmEntry.Operand = fmt.Sprintf("%#08x", arm.registers[rPC])
		}

		// "7.4 Thumb Branch With Link" in "ARM7TDMI-S Technical Reference Manual r4p3"
		arm.Scycle(prefetch, arm.registers[rPC]) // first instruction
		arm.Ncycle(prefetch, arm.registers[rPC]) // second instruction
		arm.Ncycle(prefetch, arm.registers[rPC])
		arm.Ncycle(prefetch, arm.registers[rPC])

		return
	}

	// first instruction. we'll defer cycle accumulation until the second
	// instruction branch above.

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
		arm.disasmEntry.Operator = blfirst
		arm.disasmEntry.Operand = fmt.Sprintf("%#08x", arm.registers[rPC])
	}

	// cycles accumulated on second half of the instruction
}
