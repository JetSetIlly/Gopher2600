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

package cpu

import (
	"errors"
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/instance"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/logger"
)

// CPU implements the 6507 found as found in the Atari 2600. Register logic is
// implemented by the Register type in the registers sub-package.
type CPU struct {
	instance *instance.Instance

	PC     registers.ProgramCounter
	A      registers.Register
	X      registers.Register
	Y      registers.Register
	SP     registers.StackPointer
	Status registers.StatusRegister

	// some operations only need an accumulator
	acc8  registers.Register
	acc16 registers.ProgramCounter

	mem          cpubus.Memory
	instructions []*instructions.Definition

	// cycleCallback is called for additional emulator functionality
	cycleCallback func() error

	// controls whether cpu executes a cycle when it receives a clock tick (pin
	// 3 of the 6507)
	RdyFlg bool

	// last result. the address field is guaranteed to be always valid except
	// when the CPU has just been reset. we use this fact to help us decide
	// whether the CPU has just been reset (see HasReset() function)
	//
	// note a peculiarity in the current emulation means that LastResult is not
	// reset unless the RdyFlg is true at the start of the execution.
	LastResult execution.Result

	// NoFlowControl sets whether the cpu responds accurately to instructions
	// that affect the flow of the program (branches, JPS, subroutines and
	// interrupts).  we use this in the disassembly package to make sure we
	// reach every part of the program.
	//
	// note that the alteration of flow as a result of bank switching is still
	// possible even if NoFlowControl is true. this is because bank switching
	// is outside of the direct control of the CPU.
	NoFlowControl bool

	// Interrupted indicated that the CPU has been put into a state outside of
	// its normal operation. When true work may be done on the CPU that would
	// otherwise be considered an error. Resets to false on every call to
	// ExecuteInstruction()
	Interrupted bool

	// Whether the last memory access by the CPU was a phantom access
	PhantomMemAccess bool

	// the cpu has encounted a KIL instruction. requires a Reset()
	Killed bool
}

// NewCPU is the preferred method of initialisation for the CPU structure. Note
// that the CPU will be initialised in a random state.
func NewCPU(instance *instance.Instance, mem cpubus.Memory) *CPU {
	return &CPU{
		instance:     instance,
		mem:          mem,
		PC:           registers.NewProgramCounter(0),
		A:            registers.NewRegister(0, "A"),
		X:            registers.NewRegister(0, "X"),
		Y:            registers.NewRegister(0, "Y"),
		SP:           registers.NewStackPointer(0),
		Status:       registers.NewStatusRegister(),
		acc8:         registers.NewRegister(0, "accumulator"),
		acc16:        registers.NewProgramCounter(0),
		instructions: instructions.GetDefinitions(),
	}
}

// Snapshot creates a copy of the CPU in its current state.
func (mc *CPU) Snapshot() *CPU {
	n := *mc
	return &n
}

// Plumb a new CPUBus into the CPU.
func (mc *CPU) Plumb(mem cpubus.Memory) {
	mc.mem = mem
}

func (mc *CPU) String() string {
	return fmt.Sprintf("%s=%s %s=%s %s=%s %s=%s %s=%s %s=%s",
		mc.PC.Label(), mc.PC, mc.A.Label(), mc.A,
		mc.X.Label(), mc.X, mc.Y.Label(), mc.Y,
		mc.SP.Label(), mc.SP, mc.Status.Label(), mc.Status)
}

// Reset reinitialises all registers. Does not load PC with RESET vector. Use
// cpu.LoadPCIndirect(cpubus.Reset) when appropriate.
func (mc *CPU) Reset() {
	mc.LastResult.Reset()
	mc.Interrupted = true
	mc.Killed = false

	// checking for instance == nil because it's possible for NewCPU to be
	// called with a nil instance (test package)
	if mc.instance != nil && mc.instance.Prefs.RandomState.Get().(bool) {
		mc.PC.Load(uint16(mc.instance.Random.NoRewind(0xffff)))
		mc.A.Load(uint8(mc.instance.Random.NoRewind(0xff)))
		mc.X.Load(uint8(mc.instance.Random.NoRewind(0xff)))
		mc.Y.Load(uint8(mc.instance.Random.NoRewind(0xff)))
		mc.SP.Load(uint8(mc.instance.Random.NoRewind(0xff)))
		mc.Status.Load(uint8(mc.instance.Random.NoRewind(0xff)))
	} else {
		mc.PC.Load(0)
		mc.A.Load(0)
		mc.X.Load(0)
		mc.Y.Load(0)
		mc.SP.Load(0xff)
		mc.Status.Reset()
	}

	mc.Status.Zero = mc.A.IsZero()
	mc.Status.Sign = mc.A.IsNegative()
	mc.RdyFlg = true
	mc.cycleCallback = nil

	// not touching NoFlowControl
}

// HasReset checks whether the CPU has recently been reset.
func (mc *CPU) HasReset() bool {
	return mc.LastResult.Address == 0 && mc.LastResult.Defn == nil
}

// LoadPCIndirect loads the contents of indirectAddress into the PC.
func (mc *CPU) LoadPCIndirect(indirectAddress uint16) error {
	mc.PhantomMemAccess = false

	if !mc.LastResult.Final && !mc.Interrupted {
		return fmt.Errorf("cpu: load PC indirect invalid mid-instruction")
	}

	// read 16 bit address from specified indirect address

	lo, err := mc.mem.Read(indirectAddress)
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	hi, err := mc.mem.Read(indirectAddress + 1)
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	mc.PC.Load((uint16(hi) << 8) | uint16(lo))

	return nil
}

// LoadPC loads the contents of directAddress into the PC.
func (mc *CPU) LoadPC(directAddress uint16) error {
	if !mc.LastResult.Final && !mc.Interrupted {
		return fmt.Errorf("cpu: load PC invalid mid-instruction")
	}

	mc.PC.Load(directAddress)

	return nil
}

// read8Bit returns 8bit value from the specified address
//
// side-effects:
//   - calls cycleCallback after memory read
func (mc *CPU) read8Bit(address uint16, phantom bool) (uint8, error) {
	mc.PhantomMemAccess = phantom

	val, err := mc.mem.Read(address)
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return 0, err
	}

	return val, nil
}

// read8BitZero returns 8bit value from the specified zero-page address
//
// side-effects:
//   - calls cycleCallback after memory read
func (mc *CPU) read8BitZeroPage(address uint8) (uint8, error) {
	mc.PhantomMemAccess = false

	val, err := mc.mem.Read(uint16(address))
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return 0, err
	}

	return val, nil
}

// write8Bit writes 8 bits to the specified address. there are no side effects
// on the state of the CPU which means that *cycleCallback must be called by the
// calling function as appropriate*.
func (mc *CPU) write8Bit(address uint16, value uint8, phantom bool) error {
	mc.PhantomMemAccess = phantom

	err := mc.mem.Write(address, value)
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	return nil
}

// read16Bit returns 16bit value from the specified address
//
// side-effects:
//   - calls cycleCallback after each 8bit read
func (mc *CPU) read16Bit(address uint16) (uint16, error) {
	mc.PhantomMemAccess = false

	lo, err := mc.mem.Read(address)
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return 0, err
	}

	hi, err := mc.mem.Read(address + 1)
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return 0, err
	}

	return (uint16(hi) << 8) | uint16(lo), nil
}

// read 8bits from the PC location has a variety of additional side-effects
// depending on context.
type read8BitPCeffect int

const (
	brk read8BitPCeffect = iota
	newOpcode
	loNibble
	hiNibble
)

// read8BitPC reads 8 bits from the memory location pointed to by PC
//
// side-effects:
//   - updates program counter
//   - calls cycleCallback at end of function
//   - updates LastResult.ByteCount
//   - additional side effect updates LastResult as appropriate
func (mc *CPU) read8BitPC(effect read8BitPCeffect) error {
	v, err := mc.mem.Read(mc.PC.Address())

	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	// ignoring if program counter cycling
	mc.PC.Add(1)

	// bump the number of bytes read during instruction decode
	mc.LastResult.ByteCount++

	switch effect {
	case brk:
		// the BRK command causes the PC to advance by two but that case we
		// don't want to record that the additional byte has been read
		//
		// an alternative stategry would be to define the BRK command to have a
		// different addressing mode - rather than IMMEDIATE, a new mode called
		// IMMEDIATE_BRK could be defined. routines that check for execution
		// correctness would need to be made aware of the new addressing mode
		mc.LastResult.ByteCount--

	case newOpcode:
		// look up definition
		mc.LastResult.Defn = mc.instructions[v]

		// even though all opcodes are defined we'll leave this error check in
		// just in case something goes wrong with the instruction generator
		if mc.LastResult.Defn == nil {
			return fmt.Errorf("cpu: unimplemented instruction (%#02x) at (%#04x)", v, mc.PC.Address()-1)
		}

	case loNibble:
		mc.LastResult.InstructionData = uint16(v)

	case hiNibble:
		mc.LastResult.InstructionData = (uint16(v) << 8) | mc.LastResult.InstructionData
	}

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return err
	}

	return nil
}

// read16BitPC reads 16 bits from the memory location pointed to by PC
//
// side-effects:
//   - updates program counter
//   - calls cycleCallback after each 8 bit read
//   - updates LastResult.ByteCount
//   - updates InstructionData field, once before each call to cycleCallback
//   - no callback function because this function is only ever used
//     to read operands
func (mc *CPU) read16BitPC() error {
	lo, err := mc.mem.Read(mc.PC.Address())
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	// ignoring if program counter cycling
	mc.PC.Add(1)

	// bump the number of bytes read during instruction decode
	mc.LastResult.ByteCount++

	// update instruction data with partial operand
	mc.LastResult.InstructionData = uint16(lo)

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return err
	}

	hi, err := mc.mem.Read(mc.PC.Address())
	if err != nil {
		if !errors.Is(err, cpubus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	// ignoring if program counter cycling
	mc.PC.Add(1)

	// bump the number of bytes read during instruction decode
	mc.LastResult.ByteCount++

	// update instruction data with complete operand
	mc.LastResult.InstructionData = (uint16(hi) << 8) | uint16(lo)

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return err
	}

	return nil
}

func (mc *CPU) branch(flag bool, address uint16) error {
	// return early if NoFlowControl flag is turned on
	if mc.NoFlowControl {
		return nil
	}

	// in the case of branchng (relative addressing) we've read an 8bit value
	// rather than a 16bit value to use as the "address". we do this kind of
	// thing all over the place and it normally doesn't matter; but because
	// we'll sometimes be doing subtractions with this value we need to make
	// sure the sign bit of the 8bit value has been propogated into the
	// most-significant bits of the 16bit value.
	if address&0x0080 == 0x0080 {
		address |= 0xff00
	}

	// note branching result
	mc.LastResult.BranchSuccess = flag

	if flag {
		// note current PC for reference
		oldPC := mc.PC.Address()

		// phantom read
		// +1 cycle
		_, err := mc.read8Bit(mc.PC.Address(), true)
		if err != nil {
			return err
		}

		// add LSB to PC
		// this is a bit weird but without implementing the PC differently (with
		// two 8bit bytes perhaps) this is the only way I can see how to do it with
		// the desired cycle accuracy:
		//  o Add full (sign extended) 16bit address to PC
		//  o note whether a page fault has occurred
		//  o restore the MSB of the PC using the MSB of the old PC value
		mc.PC.Add(address)
		mc.LastResult.PageFault = oldPC&0xff00 != mc.PC.Address()&0xff00
		mc.PC.Load(oldPC&0xff00 | mc.PC.Address()&0x00ff)

		// check to see whether branching has crossed a page
		if mc.LastResult.PageFault {
			// phantom reed
			// +1 cycle
			_, err := mc.read8Bit(mc.PC.Address(), true)
			if err != nil {
				return err
			}

			// correct program counter
			if address&0xff00 == 0xff00 {
				mc.PC.Add(0xff00)
			} else {
				mc.PC.Add(0x0100)
			}

			// note that we've triggered a page fault
			mc.LastResult.PageFault = true
		}
	}

	return nil
}

// NilCycleCallback can be provided as an argument to ExecuteInstruction().
// It's a convenienct do-nothing function.
func NilCycleCallback() error {
	return nil
}

// sentinal errors returned by ExecuteInstruction.
var ResetMidInstruction = errors.New("cpu: appears to have been reset mid-instruction")

// ExecuteInstruction steps CPU forward one instruction. The basic process when
// executing an instruction is this:
//
//  1. read opcode and look up instruction definition
//  2. read operands (if any) according to the addressing mode of the instruction
//  3. using the operator as a guide, perform the instruction on the data
//
// All instructions take at least 2 cycle. After each cycle, the
// cycleCallback() function is run, thereby allowing the rest of the VCS
// hardware to operate.
//
// The cycleCallback arugment should *never* be nil. Use the NilCycleCallback()
// function in this package if you want a nil effect.
func (mc *CPU) ExecuteInstruction(cycleCallback func() error) error {
	// do nothing is CPU is in KIL state
	if mc.Killed {
		return nil
	}

	// a previous call to ExecuteInstruction() has not yet completed. it is
	// impossible to begin a new instruction
	if !mc.LastResult.Final && !mc.Interrupted {
		return fmt.Errorf("cpu: starting a new instruction is invalid mid-instruction")
	}

	// reset Interrupted flag
	mc.Interrupted = false

	// do nothing and return nothing if ready flag is false
	if !mc.RdyFlg {
		return cycleCallback()
	}

	// update cycle callback
	mc.cycleCallback = cycleCallback

	// prepare new round of results
	mc.LastResult.Reset()
	mc.LastResult.Address = mc.PC.Address()

	var err error

	// read next instruction (end cycle part of read8BitPC_opcode)
	// +1 cycle
	err = mc.read8BitPC(newOpcode)
	if err != nil {
		// even when there is an error we need to update some LastResult field
		// values before returning the error. the calling function might still
		// want to make use of LastResult even when an error has occurred and
		// there's no reason to disagree (see disassembly package for an exmple
		// of this)
		//
		// I don't believe similar treatment is necessary for other error
		// conditions in the rest of the ExecuteInstruction() function

		// firstly, the number of bytes read is by definition one
		mc.LastResult.ByteCount = 1

		// secondly, the definition field. this is only required while we have
		// undefined opcodes in the CPU definition.

		// finally, this is the final byte of the instruction
		mc.LastResult.Final = true

		return err
	}

	// address is the actual address to use to access memory (after any indexing
	// has taken place)
	var address uint16

	// value is nil if addressing mode is implied and is read from the program for
	// immediate/relative mode, and from non-program memory for all other modes
	// note that for instructions which are read-modify-write, the value will
	// change during execution and be used to write back to memory
	var value uint8

	// whether the data-read should be a zero page read or not
	var zeroPage bool

	// sometimes the CPU may be reset mid-instruction. if this happens
	// LastResult.Defn will be nil. there's nothing we can do except return
	// immediately
	defn := mc.LastResult.Defn
	if defn == nil {
		return ResetMidInstruction
	}

	// get address to use when reading/writing from/to memory (note that in the
	// case of immediate addressing, we are actually getting the value to use
	// in the instruction, not the address).
	//
	// we also take the opportunity to set the InstructionData value for the
	// StepResult and whether a page fault has occurred. note that we don't do
	// this in the case of JSR
	switch defn.AddressingMode {
	case instructions.Implied:
		// implied mode does not use any additional bytes. however, the next
		// instruction is read but the PC is not incremented

		if defn.Operator == instructions.Brk {
			// BRK is unusual in that it increases the PC by two bytes despite
			// being an implied addressing instruction
			// +1 cycle
			err = mc.read8BitPC(brk)
			if err != nil {
				return err
			}
		} else {
			// phantom read
			// +1 cycle
			_, err = mc.read8Bit(mc.PC.Address(), true)
			if err != nil {
				return err
			}
		}

	case instructions.Immediate:
		// for immediate mode, the value is the next byte in the program
		// therefore, we don't set the address and we read the value through the PC

		// +1 cycle
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}
		value = uint8(mc.LastResult.InstructionData)

	case instructions.Relative:
		// relative addressing is only used for branch instructions, the address
		// is an offset value from the current PC position

		// most of the addressing cycles for this addressing mode are consumed
		// in the branch() function

		// +1 cycle
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}
		address = mc.LastResult.InstructionData

	case instructions.Absolute:
		if defn.Effect != instructions.Subroutine {
			// +2 cycles
			err := mc.read16BitPC()
			if err != nil {
				return err
			}
			address = mc.LastResult.InstructionData
		}

		// else... for JSR, addresses are read slightly differently so we defer
		// this part of the operation to the operator switch below

	case instructions.ZeroPage:
		zeroPage = true

		// +1 cycle
		//
		// while we must trest the value as an address (ie. as uint16) we
		// actually only read an 8 bit value so we store the value as uint8
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}
		address = mc.LastResult.InstructionData

	case instructions.Indirect:
		// indirect addressing (without indexing) is only used for the JMP command

		// +2 cycles
		err := mc.read16BitPC()
		if err != nil {
			return err
		}
		indirectAddress := mc.LastResult.InstructionData

		// handle indirect addressing JMP bug
		if indirectAddress&0x00ff == 0x00ff {
			mc.LastResult.CPUBug = "indirect addressing bug (JMP bug)"

			var lo, hi uint8

			lo, err = mc.mem.Read(indirectAddress)
			if err != nil {
				if !errors.Is(err, cpubus.AddressError) {
					return err
				}
				mc.LastResult.Error = err.Error()
			}

			// +1 cycle
			mc.LastResult.Cycles++
			err = mc.cycleCallback()
			if err != nil {
				if !errors.Is(err, cpubus.AddressError) {
					return err
				}
				mc.LastResult.Error = err.Error()
				return err
			}

			// in this bug path, the lower byte of the indirect address is on a
			// page boundary. because of the bug we must read high byte of JMP
			// address from the zero byte of the same page (rather than the
			// zero byte of the next page)
			hi, err = mc.mem.Read(indirectAddress & 0xff00)
			if err != nil {
				return err
			}
			address = uint16(hi) << 8
			address |= uint16(lo)

			// +1 cycle
			mc.LastResult.Cycles++
			err = mc.cycleCallback()
			if err != nil {
				return err
			}
		} else {
			// normal, non-buggy behaviour

			// +2 cycles
			address, err = mc.read16Bit(indirectAddress)
			if err != nil {
				return err
			}
		}

	case instructions.IndexedIndirect: // x indexing
		// +1 cycle
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}
		indirectAddress := uint8(mc.LastResult.InstructionData)

		// phantom read before adjusting the index
		// +1 cycle
		_, err = mc.read8Bit(uint16(indirectAddress), true)
		if err != nil {
			return err
		}

		// using 8bit addition because of the 6507's indirect addressing bug -
		// we don't want indexed address t8 extend past the first page
		mc.acc8.Load(mc.X.Value())
		mc.acc8.Add(indirectAddress, false)

		// make a note of indirect addressig bug
		if uint16(indirectAddress+mc.X.Value())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.CPUBug = "indirect addressing bug"
		}

		// +2 cycles
		address, err = mc.read16Bit(mc.acc8.Address())
		if err != nil {
			return err
		}

		// never a page fault wth pre-index indirect addressing

	case instructions.IndirectIndexed: // y indexing
		// +1 cycle
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}
		indirectAddress := mc.LastResult.InstructionData

		// +2 cycles
		var indexedAddress uint16
		indexedAddress, err = mc.read16Bit(indirectAddress)
		if err != nil {
			return err
		}

		mc.acc16.Load(mc.Y.Address())
		mc.acc16.Add(indexedAddress & 0x00ff)
		address = mc.acc16.Address()

		// check for page fault
		if defn.PageSensitive && (address&0xff00 == 0x0100) {
			mc.LastResult.CPUBug = "indirect addressing bug"
			mc.LastResult.PageFault = true
		}

		if mc.LastResult.PageFault || defn.Effect == instructions.Write || defn.Effect == instructions.RMW {
			// phantom read (always happens for Write and RMW)
			// +1 cycle
			_, err = mc.read8Bit((indexedAddress&0xff00)|(address&0x00ff), true)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indexedAddress & 0xff00)
		address = mc.acc16.Address()

	case instructions.AbsoluteIndexedX:
		// +2 cycles
		err = mc.read16BitPC()
		if err != nil {
			return err
		}
		indirectAddress := mc.LastResult.InstructionData

		// add index to LSB of address
		mc.acc16.Load(mc.X.Address())
		mc.acc16.Add(indirectAddress & 0x00ff)
		address = mc.acc16.Address()

		// check for page fault
		mc.LastResult.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if mc.LastResult.PageFault || defn.Effect == instructions.Write || defn.Effect == instructions.RMW {
			// phantom read (always happens for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit((indirectAddress&0xff00)|(address&0x00ff), true)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indirectAddress & 0xff00)
		address = mc.acc16.Address()

	case instructions.AbsoluteIndexedY:
		// +2 cycles
		err = mc.read16BitPC()
		if err != nil {
			return err
		}
		indirectAddress := mc.LastResult.InstructionData

		// add index to LSB of address
		mc.acc16.Load(mc.Y.Address())
		mc.acc16.Add(indirectAddress & 0x00ff)
		address = mc.acc16.Address()

		// check for page fault
		mc.LastResult.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if mc.LastResult.PageFault || defn.Effect == instructions.Write || defn.Effect == instructions.RMW {
			// phantom read (always happens for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit((indirectAddress&0xff00)|(address&0x00ff), true)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indirectAddress & 0xff00)
		address = mc.acc16.Address()

	case instructions.ZeroPageIndexedX:
		zeroPage = true

		// +1 cycles
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}

		// phantom read from base address before index adjustment
		// +1 cycles
		_, err := mc.read8Bit(mc.LastResult.InstructionData, true)
		if err != nil {
			return err
		}

		indirectAddress := uint8(mc.LastResult.InstructionData)
		mc.acc8.Load(indirectAddress)
		mc.acc8.Add(mc.X.Value(), false)
		address = mc.acc8.Address()

		// make a note of zero page index bug
		if uint16(indirectAddress+mc.X.Value())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.CPUBug = "zero page index bug"
		}

	case instructions.ZeroPageIndexedY:
		zeroPage = true

		// used exclusively for LDX ZeroPage,y

		// +1 cycles
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}

		// phantom read from base address before index adjustment
		// +1 cycles
		_, err := mc.read8Bit(mc.LastResult.InstructionData, true)
		if err != nil {
			return err
		}

		indirectAddress := uint8(mc.LastResult.InstructionData)
		mc.acc8.Load(indirectAddress)
		mc.acc8.Add(mc.Y.Value(), false)
		address = mc.acc8.Address()

		// make a note of zero page index bug
		if uint16(indirectAddress+mc.Y.Value())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.CPUBug = "zero page index bug"
		}

	default:
		return fmt.Errorf("cpu: unknown addressing mode for %s", defn.Operator)
	}

	// read value from memory using address found in AddressingMode switch above only when:
	// a) addressing mode is not 'implied' or 'immediate'
	//	- for immediate modes, we already have the value in lieu of an address
	//  - for implied modes, we don't need a value
	// b) instruction is 'Read' OR 'ReadWrite'
	//  - for write modes, we only use the address to write a value we already have
	//  - for flow modes, the use of the address is very specific
	if !(defn.AddressingMode == instructions.Implied || defn.AddressingMode == instructions.Immediate) {
		if defn.Effect == instructions.Read {
			// +1 cycle
			if zeroPage {
				value, err = mc.read8BitZeroPage(uint8(address))
			} else {
				value, err = mc.read8Bit(address, false)
			}
			if err != nil {
				return err
			}
		} else if defn.Effect == instructions.RMW {
			// +1 cycle
			if zeroPage {
				value, err = mc.read8BitZeroPage(uint8(address))
			} else {
				value, err = mc.read8Bit(address, false)
			}
			if err != nil {
				return err
			}

			// phantom write
			// +1 cycle
			err = mc.write8Bit(address, value, true)
			if err != nil {
				return err
			}

			mc.LastResult.Cycles++
			err = mc.cycleCallback()
			if err != nil {
				return err
			}
		}
	}

	// actually perform instruction based on operator group
	switch defn.Operator {
	case instructions.Nop:
		// does nothing

	case instructions.Cli:
		mc.Status.InterruptDisable = false

	case instructions.Sei:
		mc.Status.InterruptDisable = true

	case instructions.Clc:
		mc.Status.Carry = false

	case instructions.Sec:
		mc.Status.Carry = true

	case instructions.Cld:
		mc.Status.DecimalMode = false

	case instructions.Sed:
		mc.Status.DecimalMode = true

	case instructions.Clv:
		mc.Status.Overflow = false

	case instructions.Pha:
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), mc.A.Value(), false)
		if err != nil {
			return err
		}
		mc.SP.Add(0xff, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.Pla:
		// +1 cycle
		mc.SP.Add(1, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// +1 cycle
		value, err = mc.read8Bit(mc.SP.Address(), false)
		if err != nil {
			return err
		}
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.Php:
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), mc.Status.Value(), false)
		if err != nil {
			return err
		}
		mc.SP.Add(0xff, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.Plp:
		// +1 cycle
		mc.SP.Add(1, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}
		// +1 cycle
		value, err = mc.read8Bit(mc.SP.Address(), false)
		if err != nil {
			return err
		}
		mc.Status.Load(value)

	case instructions.Txa:
		mc.A.Load(mc.X.Value())
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.Tax:
		mc.X.Load(mc.A.Value())
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case instructions.Tay:
		mc.Y.Load(mc.A.Value())
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case instructions.Tya:
		mc.A.Load(mc.Y.Value())
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.Tsx:
		mc.X.Load(mc.SP.Value())
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case instructions.Txs:
		mc.SP.Load(mc.X.Value())
		// does not affect status register

	case instructions.Eor:
		mc.A.EOR(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.Ora:
		mc.A.ORA(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.And:
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.Lda:
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.Ldx:
		mc.X.Load(value)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case instructions.Ldy:
		mc.Y.Load(value)
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case instructions.Sta:
		// +1 cycle
		err = mc.write8Bit(address, mc.A.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.Stx:
		// +1 cycle
		err = mc.write8Bit(address, mc.X.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.Sty:
		// +1 cycle
		err = mc.write8Bit(address, mc.Y.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.Inx:
		mc.X.Add(1, false)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case instructions.Iny:
		mc.Y.Add(1, false)
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case instructions.Dex:
		mc.X.Add(0xff, false)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case instructions.Dey:
		mc.Y.Add(0xff, false)
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case instructions.Asl:
		var r *registers.Register
		if defn.Effect == instructions.RMW {
			r = &mc.acc8
			r.Load(value)
		} else {
			r = &mc.A
		}
		mc.Status.Carry = r.ASL()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case instructions.Lsr:
		var r *registers.Register
		if defn.Effect == instructions.RMW {
			r = &mc.acc8
			r.Load(value)
		} else {
			r = &mc.A
		}
		mc.Status.Carry = r.LSR()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case instructions.Adc:
		if mc.Status.DecimalMode {
			mc.Status.Carry,
				mc.Status.Zero,
				mc.Status.Overflow,
				mc.Status.Sign = mc.A.AddDecimal(value, mc.Status.Carry)
		} else {
			mc.Status.Carry, mc.Status.Overflow = mc.A.Add(value, mc.Status.Carry)
			mc.Status.Zero = mc.A.IsZero()
			mc.Status.Sign = mc.A.IsNegative()
		}

	case instructions.SBC:
		// SBC is an undocumented sbc. not sure why it's undocumented because
		// it's the same as the regular sbc instruction
		fallthrough

	case instructions.Sbc:
		if mc.Status.DecimalMode {
			mc.Status.Carry,
				mc.Status.Zero,
				mc.Status.Overflow,
				mc.Status.Sign = mc.A.SubtractDecimal(value, mc.Status.Carry)
		} else {
			mc.Status.Carry, mc.Status.Overflow = mc.A.Subtract(value, mc.Status.Carry)
			mc.Status.Zero = mc.A.IsZero()
			mc.Status.Sign = mc.A.IsNegative()
		}

	case instructions.Ror:
		var r *registers.Register
		if defn.Effect == instructions.RMW {
			r = &mc.acc8
			r.Load(value)
		} else {
			r = &mc.A
		}
		mc.Status.Carry = r.ROR(mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case instructions.Rol:
		var r *registers.Register
		if defn.Effect == instructions.RMW {
			r = &mc.acc8
			r.Load(value)
		} else {
			r = &mc.A
		}
		mc.Status.Carry = r.ROL(mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case instructions.Inc:
		r := mc.acc8
		r.Load(value)
		r.Add(1, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case instructions.Dec:
		r := mc.acc8
		r.Load(value)
		r.Add(0xff, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case instructions.Cmp:
		r := mc.acc8
		r.Load(mc.A.Value())

		// maybe surprisingly, CMP can be implemented with binary subtract even
		// if decimal mode is active (the meaning is the same)
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case instructions.Cpx:
		r := mc.acc8
		r.Load(mc.X.Value())
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case instructions.Cpy:
		r := mc.acc8
		r.Load(mc.Y.Value())
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case instructions.Bit:
		r := mc.acc8
		r.Load(value)
		mc.Status.Sign = r.IsNegative()
		mc.Status.Overflow = r.IsBitV()
		r.AND(mc.A.Value())
		mc.Status.Zero = r.IsZero()

	case instructions.Jmp:
		if !mc.NoFlowControl {
			mc.PC.Load(address)
		}

	case instructions.Bcc:
		err = mc.branch(!mc.Status.Carry, address)
		if err != nil {
			return err
		}

	case instructions.Bcs:
		err = mc.branch(mc.Status.Carry, address)
		if err != nil {
			return err
		}

	case instructions.Beq:
		err = mc.branch(mc.Status.Zero, address)
		if err != nil {
			return err
		}

	case instructions.Bmi:
		err = mc.branch(mc.Status.Sign, address)
		if err != nil {
			return err
		}

	case instructions.Bne:
		err = mc.branch(!mc.Status.Zero, address)
		if err != nil {
			return err
		}

	case instructions.Bpl:
		err = mc.branch(!mc.Status.Sign, address)
		if err != nil {
			return err
		}

	case instructions.Bvc:
		err = mc.branch(!mc.Status.Overflow, address)
		if err != nil {
			return err
		}

	case instructions.Bvs:
		err = mc.branch(mc.Status.Overflow, address)
		if err != nil {
			return err
		}

	case instructions.Jsr:
		// +1 cycle
		err = mc.read8BitPC(loNibble)
		if err != nil {
			return err
		}

		// the current value of the PC is now correct, even though we've only read
		// one byte of the address so far. remember, RTS increments the PC when
		// read from the stack, meaning that the PC will be correct at that point

		// with that in mind, we're not sure what this extra cycle is for
		// +1 cycle
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// push MSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()>>8), false)
		if err != nil {
			return err
		}
		mc.SP.Add(0xff, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// push LSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()), false)
		if err != nil {
			return err
		}
		mc.SP.Add(0xff, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// perform jump
		err = mc.read8BitPC(hiNibble)
		if err != nil {
			return err
		}

		// address has been built in the read8BitPC callback functions.
		//
		// we would normally do this in the addressing mode switch above. however,
		// JSR uses absolute addressing and we deliberately do nothing in that
		// switch for 'sub-routine' commands
		address = mc.LastResult.InstructionData
		if !mc.NoFlowControl {
			mc.PC.Load(address)
		}

	case instructions.Rts:
		// +1 cycle
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// +2 cycles
		var rtsAddress uint16
		rtsAddress, err = mc.read16Bit(mc.SP.Address())
		if err != nil {
			return err
		}

		if !mc.NoFlowControl {
			mc.SP.Add(1, false)

			// load and correct PC
			mc.PC.Load(rtsAddress)
			mc.PC.Add(1)
		}

		// +1 cycle
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.Brk:
		// push PC onto register (same effect as JSR)
		err := mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()>>8), false)
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(0xff, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()), false)
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(0xff, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// push status register (same effect as PHP)
		err = mc.write8Bit(mc.SP.Address(), mc.Status.Value(), false)
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(0xff, false)
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// set the break flag
		mc.Status.Break = true

		// perform jump
		var brkAddress uint16
		brkAddress, err = mc.read16Bit(cpubus.BRK)
		if err != nil {
			return err
		}
		if !mc.NoFlowControl {
			mc.PC.Load(brkAddress)
		}

	case instructions.Rti:
		// pull status register (same effect as PLP)
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
		}

		// not sure when this cycle should occur
		// +1 cycle
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

		// +1 cycles
		value, err = mc.read8Bit(mc.SP.Address(), false)
		if err != nil {
			return err
		}
		mc.Status.Load(value)

		// pull program counter (same effect as RTS)
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
		}

		// +2 cycles
		var rtiAddress uint16
		rtiAddress, err = mc.read16Bit(mc.SP.Address())
		if err != nil {
			return err
		}

		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
			mc.PC.Load(rtiAddress)
			// unlike RTS there is no need to add one to return address
		}

	// undocumented instructions

	case instructions.NOP:
		// does nothing (2 byte nop)

	case instructions.LAX:
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		mc.X.Load(value)

	case instructions.DCP:
		// AND the contents of the A register with value...
		// decrease value...
		r := mc.acc8
		r.Load(value)
		r.Add(0xff, false)
		value = r.Value()

		// ... and compare with the A register
		r.Load(mc.A.Value())
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case instructions.ASR:
		mc.A.AND(value)

		// ... then LSR the result
		mc.Status.Carry = mc.A.LSR()
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.XAA:
		mc.A.Load(mc.X.Value())
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.AXS:
		mc.X.AND(mc.A.Value())

		// axs subtract behaves like CMP as far as carry and overflow flags are
		// concerned
		mc.Status.Carry, _ = mc.X.Subtract(value, true)

		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case instructions.SAX:
		r := mc.acc8
		r.Load(mc.A.Value())
		r.AND(mc.X.Value())

		// +1 cycle
		err = mc.write8Bit(address, r.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.ARR:
		mc.A.AND(value)
		mc.Status.Carry = mc.A.ROR(mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.SLO:
		r := mc.acc8
		r.Load(value)
		mc.Status.Carry = r.ASL()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()
		mc.A.ORA(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.RLA:
		r := mc.acc8
		r.Load(value)
		mc.Status.Carry = r.ROL(mc.Status.Carry)
		value = r.Value()
		mc.A.AND(r.Value())
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case instructions.ISC:
		r := mc.acc8
		r.Load(value)
		r.Add(1, false)
		value = r.Value()
		mc.Status.Carry, mc.Status.Overflow = mc.A.Subtract(value, mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.ANC:
		// immediate AND. puts bit 7 into the carry flag (in microcode terms
		// this is as though ASL had been enacted)
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		mc.Status.Carry = value&0x80 == 0x80

	case instructions.SRE:
		// untested
		r := mc.acc8
		r.Load(value)
		mc.Status.Carry = r.LSR()
		value = r.Value()
		mc.A.EOR(value)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case instructions.RRA:
		// untested
		r := mc.acc8
		r.Load(value)
		mc.Status.Carry = r.ROR(mc.Status.Carry)
		value = r.Value()
		mc.Status.Carry, mc.Status.Overflow = mc.A.Add(value, mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case instructions.AHX:
		// untested
		r := mc.acc8
		r.Load(mc.A.Value())
		r.AND(mc.X.Value())
		r.AND(uint8(mc.PC.Address() >> 8))

		// +1 cycle
		err = mc.write8Bit(address, r.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.TAS:
		// untested
		r := mc.acc8
		r.Load(mc.A.Value())
		r.AND(mc.X.Value())
		mc.SP.Load(r.Value())

		// continue working with r and store into address
		r.AND(uint8(mc.PC.Address() >> 8))

		// +1 cycle
		err = mc.write8Bit(address, r.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.SHY:
		// untested
		r := mc.acc8
		r.Load(mc.Y.Value())
		r.AND(uint8(mc.PC.Address() >> 8))

		// +1 cycle
		err = mc.write8Bit(address, r.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.SHX:
		// untested
		r := mc.acc8
		r.Load(mc.X.Value())
		r.AND(uint8(mc.PC.Address() >> 8))

		// +1 cycle
		err = mc.write8Bit(address, r.Value(), false)
		if err != nil {
			return err
		}
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}

	case instructions.LAS:
		// untested
		mc.SP.AND(value)
		mc.A.Load(mc.SP.Value())
		mc.X.Load(mc.SP.Value())
		mc.Status.Zero = mc.SP.IsZero()
		mc.Status.Sign = mc.SP.IsNegative()

	case instructions.KIL:
		if !mc.NoFlowControl {
			mc.Killed = true
			logger.Logf("CPU", "KIL instruction (%#04x)", mc.PC.Address())
		}

	default:
		return fmt.Errorf("cpu: unknown operator (%s)", defn.Operator)
	}

	// for RMW instructions: write altered value back to memory
	if defn.Effect == instructions.RMW {
		err = mc.write8Bit(address, value, false)
		if err != nil {
			return err
		}

		// +1 cycle
		mc.LastResult.Cycles++
		err = mc.cycleCallback()
		if err != nil {
			return err
		}
	}

	// finalise result
	if mc.LastResult.Defn != nil {
		mc.LastResult.Final = true
	}

	// validity check. there's no need to enable unless you've just added a new
	// opcode and wanting to check the validity of the definition.
	// err = mc.LastResult.IsValid()
	// if err != nil {
	// 	return err
	// }

	return nil
}

// adhoc interface exposing the Peek() function to the CPU
type predictRTS interface {
	Peek(address uint16) (uint8, error)
}

// PredictRTS returns the PC address that would result if RTS was run at the
// current moment.
func (mc *CPU) PredictRTS() (uint16, bool) {
	predict, ok := mc.mem.(predictRTS)
	if !ok {
		return 0, false
	}

	var SP registers.Register

	SP.Load(mc.SP.Value())
	SP.Add(1, false)

	lo, err := predict.Peek(SP.Address())
	if err != nil {
		return 0, false
	}

	hi, err := predict.Peek(SP.Address() + 1)
	if err != nil {
		return 0, false
	}

	return ((uint16(hi) << 8) | uint16(lo)) + 1, true
}
