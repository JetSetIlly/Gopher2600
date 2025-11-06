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
)

// CPU implements the 6507 found as found in the Atari 2600. Register logic is
// implemented by the Register type in the registers sub-package.
type CPU struct {
	PC     registers.ProgramCounter
	A      registers.Data
	X      registers.Data
	Y      registers.Data
	SP     registers.StackPointer
	Status registers.Status

	// some operations only need an accumulator
	acc8  registers.Data
	acc16 registers.ProgramCounter

	mem Memory

	// cycleCallback is called for additional emulator functionality
	cycleCallback func() error

	// controls whether cpu executes a cycle when it receives a clock tick (pin 3 of the 6507)
	RdyFlg bool

	// most recent execution result. the state of LastResult is used to detect if the CPU has just
	// been reset. it is also used to ensure that ExecuteInstruction() or LoadPC() is not called
	// when the CPU is not in a suitable state
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

	// whether the CPU is in an interrupt block of code. this field is >0 when
	// the PC has been loaded with the address pointed to by the NMI address. it
	// is possible for the the code to be interrupted while inside an interrupt,
	// which is why this field is an integer rather than a boolean
	//
	// we can think of this as an extended I status flag that is increased on a
	// NMI or IRQ signal (but not a BRK instruction) and reduced on RTI (when
	// the break flag in the status register is cleared)
	interruptDepth int

	// an interrupt has occured and so the next instruction will indicate that
	// it was executed as a result of the interrupt
	interrupt bool

	// Whether the last memory access by the CPU was a phantom access
	PhantomMemAccess bool

	// the cpu has encounted a KIL instruction. requires a Reset()
	Killed bool
}

const (
	// NMI is the address where the non-maskable interrupt address is stored.
	NMI = uint16(0xfffa)

	// Reset is the address where the reset address is stored.
	Reset = uint16(0xfffc)

	// IRQ is the address where the interrupt address is stored.
	IRQ = uint16(0xfffe)

	// for clarity, BRK is another name for IRQ. we use this when triggering
	// software interrupts
	BRK = IRQ
)

// it is believed that internalParameter is a value that is inherent to the brand of chip. whatever it is
// exactly, if is used in the XAA or LAX (immediate) instructions
//
// we call it internalParameter because that's how it's referred to in 64doc.txt
const (
	internalParameterXAA = 0xee
	internalParameterLAX = 0x00
)

// Memory interface to underlying implmentation. See MemoryAddressError
// interface for optional functions
type Memory interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

// NewCPU is the preferred method of initialisation for the CPU structure. Note
// that the CPU will be initialised in a random state.
func NewCPU(mem Memory) *CPU {
	return &CPU{
		mem:    mem,
		PC:     registers.NewProgramCounter(0),
		A:      registers.NewData(0, "A"),
		X:      registers.NewData(0, "X"),
		Y:      registers.NewData(0, "Y"),
		SP:     registers.NewStackPointer(0),
		Status: registers.NewStatus(),
		acc8:   registers.NewData(0, "accumulator"),
		acc16:  registers.NewProgramCounter(0),
	}
}

// Snapshot creates a copy of the CPU in its current state.
func (mc *CPU) Snapshot() *CPU {
	n := *mc
	return &n
}

// Plumb CPU into emulation
func (mc *CPU) Plumb(mem Memory) {
	mc.mem = mem
}

func (mc *CPU) String() string {
	return fmt.Sprintf("%s=%s %s=%s %s=%s %s=%s %s=%s %s=%s",
		mc.PC.Label(), mc.PC, mc.A.Label(), mc.A,
		mc.X.Label(), mc.X, mc.Y.Label(), mc.Y,
		mc.SP.Label(), mc.SP, mc.Status.Label(), mc.Status,
	)
}

// SetRDY sets the CPU RDY flag. equivalent to pin 3 of the 6507
func (mc *CPU) SetRDY(rdy bool) {
	mc.RdyFlg = rdy
}

// InInterrupt returns true if executed instructions will be between an NMI/IRQ
// and an RTI. Remember that it's possible for an NMI to interrupt on an ongoing
// interrupt handler, so an RTI will not necessarily cause InInterrupt to return
// false on the next call
func (mc *CPU) InInterrupt() bool {
	return mc.interruptDepth > 0
}

// Interrupt loads the PC with the 16bit value at the NMI address (when
// NMI is true) or at the IRQ address
func (mc *CPU) Interrupt(nonMaskable bool) error {
	// the interruptDepth field is now >0 and indicates that the CPU is in the
	// interrupted state. if the CPU has been interrupted previously without an
	// intervening RTI then the field will be >1
	mc.interruptDepth++

	// an interrupt has occurred and will be indicated in the restul for the next instruction
	mc.interrupt = true

	// IRQ interrupts only take effect if the InterruptDisable flag is unset
	if !nonMaskable {
		if mc.Status.InterruptDisable {
			return nil
		}
	}

	// push MSB of PC onto stack, and decrement SP
	err := mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()>>8), false)
	if err != nil {
		return err
	}
	mc.SP.Add(0xff, false)

	// push LSB of PC onto stack, and decrement SP
	err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()), false)
	if err != nil {
		return err
	}
	mc.SP.Add(0xff, false)

	// push status register
	err = mc.write8Bit(mc.SP.Address(), mc.Status.Value(), false)
	if err != nil {
		return err
	}
	mc.SP.Add(0xff, false)

	// set the interrupt disable flag after pushing the status register to the
	// stack. this is so the flag is cleared when the status register is restored
	//
	// NMI interrupts do not set the InterruptDisable flag
	if !nonMaskable {
		mc.Status.InterruptDisable = true
	}

	if nonMaskable {
		return mc.LoadPCIndirect(NMI)
	}
	return mc.LoadPCIndirect(IRQ)
}

type Random interface {
	Intn(int) int
}

// Reset CPU in either a random or non-random state. The random state is more accurate
func (mc *CPU) Reset(rnd Random) error {
	mc.LastResult.Reset()
	mc.Killed = false
	mc.cycleCallback = nil
	mc.interruptDepth = 0
	mc.interrupt = false

	if rnd == nil {
		mc.PC.Load(0x0000)
		mc.A.Load(0x00)
		mc.X.Load(0x00)
		mc.Y.Load(0x00)
		mc.Status.Load(0x00)
		mc.SP.Load(0x00)
	} else {
		mc.PC.Load(uint16(rnd.Intn(0xffff)))
		mc.A.Load(uint8(rnd.Intn(0xff)))
		mc.X.Load(uint8(rnd.Intn(0xff)))
		mc.Y.Load(uint8(rnd.Intn(0xff)))
		mc.Status.Load(uint8(rnd.Intn(0xff)))
		mc.SP.Load(uint8(rnd.Intn(0xff)))
	}

	// the interrupt disable flag is always set on reset
	// note that the zero and negative flags remain undefined and is unaffected by the value in the
	// A or any other register
	mc.Status.InterruptDisable = true

	// the reset procedure is special in that it leaves the information about the last executed
	// instruction in the "final" state
	mc.LastResult.Final = true

	// we simplify the reset procedure to reducing the stack value three times and then loading the
	// reset address into the PC. there's no need to emulate the full eight cycles of the
	// initilisation process
	mc.SP.Add(0xff, false)
	mc.SP.Add(0xff, false)
	mc.SP.Add(0xff, false)
	err := mc.LoadPCIndirect(Reset)
	if err != nil {
		return err
	}

	// cpu is ready immediately after reset
	mc.RdyFlg = true

	return nil
}

// HasReset checks whether the CPU has recently been reset.
func (mc *CPU) HasReset() bool {
	// instead of an explicit "has reset" flag, we can use the information about the last executed
	// to detect wehether the CPU has been recently reset
	//
	// the Final flag in LastResult is not normally set when the defintion field is nil. however, we
	// explicitely set the Final flag to true when resetting the CPU
	return mc.LastResult.Defn == nil && mc.LastResult.Final == true
}

// LoadPC loads the contents of directAddress into the PC.
func (mc *CPU) LoadPCIndirect(address uint16) error {
	if !mc.LastResult.Final {
		return fmt.Errorf("cpu: load PC invalid mid-instruction")
	}

	mc.PhantomMemAccess = false

	lo, err := mc.mem.Read(address)
	if err != nil {
		return err
	}
	hi, err := mc.mem.Read(address + 1)
	if err != nil {
		return err
	}
	mc.PC.Load((uint16(hi) << 8) | uint16(lo))

	return nil
}

// LoadPC loads the contents of directAddress into the PC.
func (mc *CPU) LoadPC(directAddress uint16) error {
	if !mc.LastResult.Final {
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
		return 0, err
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
		return err
	}

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return err
	}

	return nil
}

// read16Bit returns 16bit value from the specified address
//
// side-effects:
//   - calls cycleCallback after each 8bit read
func (mc *CPU) read16Bit(address uint16) (uint16, error) {
	lo, err := mc.mem.Read(address)
	if err != nil {
		return 0, err
	}

	// +1 cycle
	mc.LastResult.Cycles++
	err = mc.cycleCallback()
	if err != nil {
		return 0, err
	}

	// advance address and being careful to preserve page
	hi, err := mc.mem.Read((address & 0xff00) | ((address + 1) & 0x00ff))
	if err != nil {
		return 0, err
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
	loByte
	hiByte
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
		return err
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
		mc.LastResult.Defn = instructions.Definitions[v]

		// even though all opcodes are defined we'll leave this error check in
		// just in case something goes wrong with the instruction generator
		if mc.LastResult.Defn == nil {
			return fmt.Errorf("cpu: unimplemented instruction (%#02x) at (%#04x)", v, mc.PC.Address()-1)
		}

	case loByte:
		mc.LastResult.InstructionData = uint16(v)

	case hiByte:
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
		return err
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
		return err
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
	// the CPU does nothing if it is in the KIL state. however, the other
	// parts of the VCS continue
	if mc.Killed {
		return cycleCallback()
	}

	// a previous call to ExecuteInstruction() has not yet completed. it is
	// impossible to begin a new instruction
	if !mc.LastResult.Final {
		return fmt.Errorf("cpu: starting a new instruction is invalid mid-instruction")
	}

	// do nothing and return nothing if ready flag is false
	if !mc.RdyFlg {
		return cycleCallback()
	}

	// update cycle callback
	mc.cycleCallback = cycleCallback

	// prepare new round of results
	mc.LastResult.Reset()
	mc.LastResult.Address = mc.PC.Address()
	mc.LastResult.FromInterrupt = mc.interrupt
	mc.interrupt = false
	mc.LastResult.InInterrupt = mc.InInterrupt()

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
		err = mc.read8BitPC(loByte)
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
		err = mc.read8BitPC(loByte)
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
		// +1 cycle
		//
		// while we must trest the value as an address (ie. as uint16) we
		// actually only read an 8 bit value so we store the value as uint8
		err = mc.read8BitPC(loByte)
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
				return err
			}

			// +1 cycle
			mc.LastResult.Cycles++
			err = mc.cycleCallback()
			if err != nil {
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
		err = mc.read8BitPC(loByte)
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
		err = mc.read8BitPC(loByte)
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
		// +1 cycles
		err = mc.read8BitPC(loByte)
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
		// used exclusively for LDX ZeroPage,y

		// +1 cycles
		err = mc.read8BitPC(loByte)
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
		switch defn.Effect {
		case instructions.Read:
			// +1 cycle
			value, err = mc.read8Bit(address, false)
			if err != nil {
				return err
			}
		case instructions.RMW:
			// +1 cycle
			value, err = mc.read8Bit(address, false)
			if err != nil {
				return err
			}

			// phantom write
			// +1 cycle
			err = mc.write8Bit(address, value, true)
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

	case instructions.Pla:
		// +1 cycle
		value, err = mc.read8Bit(mc.SP.Address(), true)
		if err != nil {
			return err
		}
		mc.SP.Add(1, false)

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

	case instructions.Plp:
		// +1 cycle
		value, err = mc.read8Bit(mc.SP.Address(), true)
		if err != nil {
			return err
		}
		mc.SP.Add(1, false)

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

	case instructions.Stx:
		// +1 cycle
		err = mc.write8Bit(address, mc.X.Value(), false)
		if err != nil {
			return err
		}

	case instructions.Sty:
		// +1 cycle
		err = mc.write8Bit(address, mc.Y.Value(), false)
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
		var r *registers.Data
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
		var r *registers.Data
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
			mc.Status.Carry, mc.Status.Zero, mc.Status.Overflow, mc.Status.Sign = mc.A.AddDecimal(value, mc.Status.Carry)
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
		if defn.Effect == instructions.RMW {
			mc.acc8.Load(value)
			mc.Status.Carry = mc.acc8.ROR(mc.Status.Carry)
			mc.Status.Zero = mc.acc8.IsZero()
			mc.Status.Sign = mc.acc8.IsNegative()
			value = mc.acc8.Value()
		} else {
			mc.Status.Carry = mc.A.ROR(mc.Status.Carry)
			mc.Status.Zero = mc.A.IsZero()
			mc.Status.Sign = mc.A.IsNegative()
		}

	case instructions.Rol:
		if defn.Effect == instructions.RMW {
			mc.acc8.Load(value)
			mc.Status.Carry = mc.acc8.ROL(mc.Status.Carry)
			mc.Status.Zero = mc.acc8.IsZero()
			mc.Status.Sign = mc.acc8.IsNegative()
			value = mc.acc8.Value()
		} else {
			mc.Status.Carry = mc.A.ROL(mc.Status.Carry)
			mc.Status.Zero = mc.A.IsZero()
			mc.Status.Sign = mc.A.IsNegative()
		}

	case instructions.Inc:
		mc.acc8.Load(value)
		mc.acc8.Add(1, false)
		mc.Status.Zero = mc.acc8.IsZero()
		mc.Status.Sign = mc.acc8.IsNegative()
		value = mc.acc8.Value()

	case instructions.Dec:
		mc.acc8.Load(value)
		mc.acc8.Add(0xff, false)
		mc.Status.Zero = mc.acc8.IsZero()
		mc.Status.Sign = mc.acc8.IsNegative()
		value = mc.acc8.Value()

	case instructions.Cmp:
		mc.acc8.Load(mc.A.Value())

		// maybe surprisingly, CMP can be implemented with binary subtract even
		// if decimal mode is active (the meaning is the same)
		mc.Status.Carry, _ = mc.acc8.Subtract(value, true)
		mc.Status.Zero = mc.acc8.IsZero()
		mc.Status.Sign = mc.acc8.IsNegative()

	case instructions.Cpx:
		mc.acc8.Load(mc.X.Value())
		mc.Status.Carry, _ = mc.acc8.Subtract(value, true)
		mc.Status.Zero = mc.acc8.IsZero()
		mc.Status.Sign = mc.acc8.IsNegative()

	case instructions.Cpy:
		mc.acc8.Load(mc.Y.Value())
		mc.Status.Carry, _ = mc.acc8.Subtract(value, true)
		mc.Status.Zero = mc.acc8.IsZero()
		mc.Status.Sign = mc.acc8.IsNegative()

	case instructions.Bit:
		mc.acc8.Load(value)
		mc.Status.Sign = mc.acc8.IsNegative()
		mc.Status.Overflow = mc.acc8.IsBitV()
		mc.acc8.AND(mc.A.Value())
		mc.Status.Zero = mc.acc8.IsZero()

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
		err = mc.read8BitPC(loByte)
		if err != nil {
			return err
		}

		// dummy fetch from stack
		mc.read8Bit(mc.SP.Address(), true)

		// the current value of the PC is now correct, even though we've only read
		// one byte of the address so far. remember, RTS increments the PC when
		// read from the stack, meaning that the PC will be correct at that point

		// push MSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()>>8), false)
		if err != nil {
			return err
		}
		mc.SP.Add(0xff, false)

		// push LSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()), false)
		if err != nil {
			return err
		}
		mc.SP.Add(0xff, false)

		// +1 cycle
		err = mc.read8BitPC(hiByte)
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
		// dummy read of address at current SP before the pointer
		// is advanced for the real 16bit read
		//
		// +1 cycle
		_, err = mc.read8Bit(mc.SP.Address(), true)

		// adjust stack pointer
		mc.SP.Add(1, false)

		// +2 cycles
		var rtsAddress uint16
		rtsAddress, err = mc.read16Bit(mc.SP.Address())
		if err != nil {
			return err
		}

		mc.SP.Add(1, false)
		if !mc.NoFlowControl {
			mc.PC.Load(rtsAddress)
		}

		// +1 cycle
		_, err = mc.read8Bit(mc.PC.Address(), false)
		mc.PC.Add(1)

	case instructions.Brk:
		// push PC onto register (same effect as JSR)
		err := mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()>>8), false)
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(0xff, false)

		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()), false)
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(0xff, false)

		// push status register (same effect as PHP)
		err = mc.write8Bit(mc.SP.Address(), mc.Status.Value(), false)
		if err != nil {
			return err
		}

		mc.SP.Add(0xff, false)

		// set the break and interrupt disable flags after pushing the status
		// register to the stack. this is so the flags are cleared when the
		// status register is restored
		mc.Status.Break = true
		mc.Status.InterruptDisable = true

		// perform jump
		var brkAddress uint16
		brkAddress, err = mc.read16Bit(BRK)
		if err != nil {
			return err
		}
		if !mc.NoFlowControl {
			mc.PC.Load(brkAddress)
		}

	case instructions.Rti:
		// software breaks (the BRK instruction) can be distinguished from
		// hardware interrupts by the break flag
		if mc.Status.Break {
			if mc.interruptDepth > 0 {
				// reduce depth count by one. if the count is now zero then the CPU is
				// no longer in the interrupt state. if however, the an interrupt
				// happened whilst inside an interrupt, the count will still be >0
				mc.interruptDepth--
			} else {
				// if interruptDepth is zero then that means that RTI has been called outside of
				// interupt block
			}
		}

		// +1 cycles
		_, err = mc.read8Bit(mc.SP.Address(), true)
		if err != nil {
			return err
		}

		// pull status register (same effect as PLP)
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
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
		if defn.AddressingMode == instructions.Immediate {
			mc.A.Load((mc.A.Value() | internalParameterLAX) & value)
		} else {
			mc.A.Load(value)
		}
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		mc.X.Load(mc.A.Value())

	case instructions.DCP:
		// AND the contents of the A register with value...
		// decrease value...
		mc.acc8.Load(value)
		mc.acc8.Add(0xff, false)
		value = mc.acc8.Value()

		// ... and compare with the A register
		mc.acc8.Load(mc.A.Value())
		mc.Status.Carry, _ = mc.acc8.Subtract(value, true)
		mc.Status.Zero = mc.acc8.IsZero()
		mc.Status.Sign = mc.acc8.IsNegative()

	case instructions.ASR:
		mc.A.AND(value)

		// ... then LSR the result
		mc.Status.Carry = mc.A.LSR()
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.XAA:
		// description for "64doc.txt"
		//
		// "A = (A | #$EE) & X & #byte
		// same as
		// A = ((A & #$11 & X) | ( #$EE & X)) & #byte
		//
		// In real 6510/8502 the internal parameter #$11 may occasionally be #$10, #$01 or even
		// #$00. This occurs when the video chip starts DMA between the opcode fetch and the
		// parameter fetch of the instruction.  The value probably depends on the data that was left
		// on the bus by the VIC-II"
		//
		// note that XAA is referred to as ANE in 64doc.txt

		mc.A.Load((mc.A.Value() | internalParameterXAA) & mc.X.Value() & value)
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
		mc.acc8.Load(mc.A.Value())
		mc.acc8.AND(mc.X.Value())

		// +1 cycle
		err = mc.write8Bit(address, mc.acc8.Value(), false)
		if err != nil {
			return err
		}

	case instructions.ARR:
		// description for "64doc.txt"
		//
		// "This instruction seems to be a harmless combination of AND and ROR at
		// first sight, but it turns out that it affects the V flag and also has
		// a special kind of decimal mode. This is because the instruction has
		// inherited some properties of the ADC instruction ($69) in addition to
		// the ROR ($6A)"
		if !mc.Status.DecimalMode {
			// "In Binary mode (D flag clear), the instruction effectively does an AND
			// between the accumulator and the immediate parameter, and then shifts
			// the accumulator to the right, copying the C flag to the 8th bit. It
			// sets the Negative and Zero flags just like the ROR would. The ADC code
			// shows up in the Carry and oVerflow flags. The C flag will be copied
			// from the bit 6 of the result (which doesn't seem too logical), and the
			// V flag is the result of an Exclusive OR operation between the bit 6
			// and the bit 5 of the result.  This makes sense, since the V flag will
			// be normally set by an Exclusive OR, too"
			mc.A.AND(value)
			_ = mc.A.ROR(mc.Status.Carry)
			mc.Status.Zero = mc.A.IsZero()
			mc.Status.Sign = mc.A.IsNegative()
			mc.Status.Carry = (mc.A.Value() >> 6 & 0x01) == 0x01
			mc.Status.Overflow = (((mc.A.Value() >> 6) & 0x01) ^ ((mc.A.Value() >> 5) & 0x01)) == 0x01
		} else {
			// "In Decimal mode (D flag set), the ARR instruction first performs the
			// AND and ROR, just like in Binary mode. The N flag will be copied from
			// the initial C flag, and the Z flag will be set according to the ROR
			// result, as expected. The V flag will be set if the bit 6 of the
			// accumulator changed its state between the AND and the ROR, cleared
			// otherwise.
			//
			// Now comes the funny part. If the low nybble of the AND result,
			// incremented by its lowmost bit, is greater than 5, the low nybble in
			// the ROR result will be incremented by 6. The low nybble may overflow
			// as a consequence of this BCD fixup, but the high nybble won't be
			// adjusted. The high nybble will be BCD fixed in a similar way. If the
			// high nybble of the AND result, incremented by its lowmost bit, is
			// greater than 5, the high nybble in the ROR result will be incremented
			// by 6, and the Carry flag will be set. Otherwise the C flag will be
			// cleared"

			// "perform the AND"
			t := mc.A.Value() & value
			ah := t >> 4
			al := t & 0x0f

			// "separate the high and low nybbles"
			if mc.Status.Carry {
				mc.A.Load(0x80 | (t >> 1))
			} else {
				mc.A.Load(t >> 1)
			}

			// "set the N and Z flags traditionally"
			mc.Status.Sign = mc.Status.Carry
			mc.Status.Zero = mc.A.IsZero()

			// "and the V flag in a weird way"
			mc.Status.Overflow = (t^mc.A.Value())&0x40 == 0x040

			// "BCD 'fixup' for low nybble"
			if al+(al&0x01) > 5 {
				v := mc.A.Value()&0xf0 | (mc.A.Value()+6)&0x0f
				mc.A.Load(v)
			}

			// "set the carry flag"
			mc.Status.Carry = (ah + (ah & 1)) > 5

			// "BCD 'fixup' for high nybble"
			if mc.Status.Carry {
				v := mc.A.Value() + 0x60
				mc.A.Load(v)
			}
		}

	case instructions.SLO:
		mc.acc8.Load(value)
		mc.Status.Carry = mc.acc8.ASL()
		mc.Status.Zero = mc.acc8.IsZero()
		mc.Status.Sign = mc.acc8.IsNegative()
		value = mc.acc8.Value()
		mc.A.ORA(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.RLA:
		mc.acc8.Load(value)
		mc.Status.Carry = mc.acc8.ROL(mc.Status.Carry)
		value = mc.acc8.Value()
		mc.A.AND(mc.acc8.Value())
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.ISC:
		mc.acc8.Load(value)
		mc.acc8.Add(1, false)
		value = mc.acc8.Value()
		if mc.Status.DecimalMode {
			mc.Status.Carry, mc.Status.Zero,
				mc.Status.Overflow, mc.Status.Sign = mc.A.SubtractDecimal(value, mc.Status.Carry)
		} else {
			mc.Status.Carry, mc.Status.Overflow = mc.A.Subtract(value, mc.Status.Carry)
			mc.Status.Zero = mc.A.IsZero()
			mc.Status.Sign = mc.A.IsNegative()
		}

	case instructions.ANC:
		// immediate AND. puts bit 7 into the carry flag (in microcode terms
		// this is as though ASL had been enacted)
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		mc.Status.Carry = mc.A.Value()&0x80 == 0x80

	case instructions.SRE:
		mc.acc8.Load(value)
		mc.Status.Carry = mc.acc8.LSR()
		value = mc.acc8.Value()
		mc.A.EOR(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case instructions.RRA:
		mc.acc8.Load(value)
		mc.Status.Carry = mc.acc8.ROR(mc.Status.Carry)
		value = mc.acc8.Value()
		if mc.Status.DecimalMode {
			mc.Status.Carry, mc.Status.Zero,
				mc.Status.Overflow, mc.Status.Sign = mc.A.AddDecimal(mc.acc8.Value(), mc.Status.Carry)
		} else {
			mc.Status.Carry, mc.Status.Overflow = mc.A.Add(mc.acc8.Value(), mc.Status.Carry)
			mc.Status.Zero = mc.A.IsZero()
			mc.Status.Sign = mc.A.IsNegative()
		}

	case instructions.AHX:
		mc.acc8.Load(mc.A.Value())
		mc.acc8.AND(mc.X.Value())
		mc.acc8.AND(uint8(address>>8) + 1)

		// +1 cycle
		err = mc.write8Bit(address, mc.acc8.Value(), false)
		if err != nil {
			return err
		}

	case instructions.TAS:
		mc.acc8.Load(mc.A.Value())
		mc.acc8.AND(mc.X.Value())
		mc.SP.Load(mc.acc8.Value())
		mc.acc8.AND(uint8(address>>8) + 1)

		// +1 cycle
		err = mc.write8Bit(address, mc.acc8.Value(), false)
		if err != nil {
			return err
		}

	case instructions.SHY:
		mc.acc8.Load(mc.Y.Value())
		mc.acc8.AND(uint8(address>>8) + 1)

		// +1 cycle
		err = mc.write8Bit(address, mc.acc8.Value(), false)
		if err != nil {
			return err
		}

	case instructions.SHX:
		mc.acc8.Load(mc.X.Value())
		mc.acc8.AND(uint8(address>>8) + 1)

		// +1 cycle
		err = mc.write8Bit(address, mc.acc8.Value(), false)
		if err != nil {
			return err
		}

	case instructions.LAS:
		mc.SP.AND(value)
		mc.A.Load(mc.SP.Value())
		mc.X.Load(mc.SP.Value())
		mc.Status.Zero = mc.SP.IsZero()
		mc.Status.Sign = mc.SP.IsNegative()

	case instructions.KIL:
		if !mc.NoFlowControl {
			mc.Killed = true
		}

	default:
		return fmt.Errorf("cpu: unknown operator (%s)", defn.Operator)
	}

	// for RMW instructions: write altered value back to memory
	if defn.Effect == instructions.RMW {
		// +1 cycle
		err = mc.write8Bit(address, value, false)
		if err != nil {
			return err
		}
	}

	// record the CPU Rdy flag at the end of the instruction
	mc.LastResult.Rdy = mc.RdyFlg

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

	var SP registers.Data

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
