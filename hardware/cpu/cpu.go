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
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/cpu/registers"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
)

// CPU implements the 6507 found as found in the Atari 2600. Register logic is
// implemented by the Register type in the registers sub-package.
type CPU struct {
	prefs *preferences.Preferences

	PC     registers.ProgramCounter
	A      registers.Register
	X      registers.Register
	Y      registers.Register
	SP     registers.Register
	Status registers.StatusRegister

	// some operations only need an accumulator
	acc8  registers.Register
	acc16 registers.ProgramCounter

	mem          bus.CPUBus
	instructions []*instructions.Definition

	// cycleCallback is called by endCycle() for additional emulator
	// functionality
	cycleCallback func() error

	// controls whether cpu executes a cycle when it receives a clock tick (pin
	// 3 of the 6507)
	RdyFlg bool

	// last result. the address field is guaranteed to be always valid except
	// when the CPU has just been reset. we use this fact to help us decide
	// whether the CPU has just been reset (see HasReset() function)
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
}

// NewCPU is the preferred method of initialisation for the CPU structure. Note
// that the CPU will be initialised in a random state.
func NewCPU(prefs *preferences.Preferences, mem bus.CPUBus) *CPU {
	return &CPU{
		prefs:        prefs,
		mem:          mem,
		PC:           registers.NewProgramCounter(0),
		A:            registers.NewRegister(0, "A"),
		X:            registers.NewRegister(0, "X"),
		Y:            registers.NewRegister(0, "Y"),
		SP:           registers.NewRegister(0, "SP"),
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
func (mc *CPU) Plumb(mem bus.CPUBus) {
	mc.mem = mem
}

func (mc *CPU) String() string {
	return fmt.Sprintf("%s=%s %s=%s %s=%s %s=%s %s=%s %s=%s",
		mc.PC.Label(), mc.PC, mc.A.Label(), mc.A,
		mc.X.Label(), mc.X, mc.Y.Label(), mc.Y,
		mc.SP.Label(), mc.SP, mc.Status.Label(), mc.Status)
}

// Reset reinitialises all registers.
func (mc *CPU) Reset() {
	mc.LastResult.Reset()
	mc.LastResult.Final = true

	if mc.prefs != nil && mc.prefs.RandomState.Get().(bool) {
		mc.PC.Load(uint16(mc.prefs.RandSrc.Intn(0xffff)))
		mc.A.Load(uint8(mc.prefs.RandSrc.Intn(0xff)))
		mc.X.Load(uint8(mc.prefs.RandSrc.Intn(0xff)))
		mc.Y.Load(uint8(mc.prefs.RandSrc.Intn(0xff)))
		mc.SP.Load(uint8(mc.prefs.RandSrc.Intn(0xff)))
		mc.Status.FromValue(uint8(mc.prefs.RandSrc.Intn(0xff)))
	} else {
		mc.PC.Load(0)
		mc.A.Load(0)
		mc.X.Load(0)
		mc.Y.Load(0)
		mc.SP.Load(255)
		mc.Status.Reset()
	}

	mc.Status.Zero = mc.A.IsZero()
	mc.Status.Sign = mc.A.IsNegative()
	mc.RdyFlg = true
	mc.cycleCallback = nil

	// not touching NoFlowControl
}

// HasReset checks whether the CPU has recently been reset.
func (mc CPU) HasReset() bool {
	return mc.LastResult.Address == 0 && mc.LastResult.Defn == nil
}

// LoadPCIndirect loads the contents of indirectAddress into the PC.
func (mc *CPU) LoadPCIndirect(indirectAddress uint16) error {
	if !mc.LastResult.Final && !mc.Interrupted {
		return curated.Errorf("cpu: load PC indirect invalid mid-instruction")
	}

	// read 16 bit address from specified indirect address

	lo, err := mc.mem.Read(indirectAddress)
	if err != nil {
		if !curated.Has(err, bus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	hi, err := mc.mem.Read(indirectAddress + 1)
	if err != nil {
		if !curated.Has(err, bus.AddressError) {
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
		return curated.Errorf("cpu: load PC invalid mid-instruction")
	}

	mc.PC.Load(directAddress)

	return nil
}

// read8Bit returns 8bit value from the specified address
//
// side-effects
//	* calls endCycle after memory read
func (mc *CPU) read8Bit(address uint16) (uint8, error) {
	val, err := mc.mem.Read(address)

	if err != nil {
		if !curated.Has(err, bus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	return val, nil
}

// read8BitZero returns 8bit value from the specified zero-page address
//
// side-effects
//	* calls endCycle after memory read
func (mc *CPU) read8BitZeroPage(address uint8) (uint8, error) {
	val, err := mc.mem.(bus.CPUBusZeroPage).ReadZeroPage(address)

	if err != nil {
		if !curated.Has(err, bus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	return val, nil
}

// write8Bit writes 8 bits to the specified address. there are no side effects
// on the state of the CPU which means that *endCycle must be called by the
// calling function as appropriate*.
func (mc *CPU) write8Bit(address uint16, value uint8) error {
	err := mc.mem.Write(address, value)

	if err != nil {
		if !curated.Has(err, bus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	return nil
}

// read16Bit returns 16bit value from the specified address
//
// side-effects
//	* calls endCycle after each 8bit read
func (mc *CPU) read16Bit(address uint16) (uint16, error) {
	lo, err := mc.mem.Read(address)
	if err != nil {
		if !curated.Has(err, bus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	hi, err := mc.mem.Read(address + 1)
	if err != nil {
		if !curated.Has(err, bus.AddressError) {
			return 0, err
		}
		mc.LastResult.Error = err.Error()
	}

	// +1 cycle
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	return (uint16(hi) << 8) | uint16(lo), nil
}

// read8BitPC reads 8 bits from the memory location pointed to by PC
//
// side-effects
//	* updates program counter
//	* calls endCycle at end of function
//	* updates LastResult.ByteCount
//	* callback function in which LastResult should be updated as appropriate
//		- probably updating InstructionData field
//		- but can be used to read opcode too
func (mc *CPU) read8BitPC(f func(val uint8) error) error {
	v, err := mc.mem.Read(mc.PC.Address())

	if err != nil {
		if !curated.Has(err, bus.AddressError) {
			return err
		}
		mc.LastResult.Error = err.Error()
	}

	// ignoring if program counter cycling
	mc.PC.Add(1)

	// bump the number of bytes read during instruction decode
	mc.LastResult.ByteCount++

	// callback function
	if f != nil {
		err = f(v)
		if err != nil {
			return err
		}
	}

	// +1 cycle
	err = mc.endCycle()
	if err != nil {
		return err
	}

	return nil
}

// read16BitPC reads 16 bits from the memory location pointed to by PC
//
// side-effects
//	* updates program counter
//	* calls endCycle after each 8 bit read
//	* updates LastResult.ByteCount
//	* updates InstructionData field, once before each call to endCycle
//		- no callback function because this function is only ever used
//	 	to read operands
func (mc *CPU) read16BitPC() error {
	lo, err := mc.mem.Read(mc.PC.Address())
	if err != nil {
		if !curated.Has(err, bus.AddressError) {
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
	err = mc.endCycle()
	if err != nil {
		return err
	}

	hi, err := mc.mem.Read(mc.PC.Address())
	if err != nil {
		if !curated.Has(err, bus.AddressError) {
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
	err = mc.endCycle()
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
		_, err := mc.read8Bit(mc.PC.Address())
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
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(mc.PC.Address())
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

// endCycle is called at the end of the imaginary CPU cycle. for example,
// reading a byte from memory takes one cycle and so the emulation will call
// endCycle() at that point.
//
// CPU.cycleCallback() is called from this function for additional
// functionality.
func (mc *CPU) endCycle() error {
	mc.LastResult.Cycles++
	if mc.cycleCallback == nil {
		return nil
	}
	return mc.cycleCallback()
}

// Sentinal error returned by ExecuteInstruction if an unimplemented opcode is encountered.
const (
	UnimplementedInstruction = "cpu: unimplemented instruction (%#02x) at (%#04x)"
)

// ExecuteInstruction steps CPU forward one instruction. The basic process when
// executing an instruction is this:
//
//	1. read opcode and look up instruction definition
//	2. read operands (if any) according to the addressing mode of the instruction
//	3. using the mnemonic as a guide, perform the instruction on the data
//
// All instructions take at least 2 cycle. After each cycle, the
// cycleCallback() function is run, thereby allowing the rest of the VCS
// hardware to operate.
func (mc *CPU) ExecuteInstruction(cycleCallback func() error) error {
	// a previous call to ExecuteInstruction() has not yet completed. it is
	// impossible to begin a new instruction
	if !mc.LastResult.Final && !mc.Interrupted {
		return curated.Errorf("cpu: starting a new instruction is invalid mid-instruction")
	}

	// reset Interrupted flag
	mc.Interrupted = false

	// update cycle callback
	mc.cycleCallback = cycleCallback

	// do nothing and return nothing if ready flag is false
	if !mc.RdyFlg {
		err := cycleCallback()
		return err
	}

	// prepare new round of results
	mc.LastResult.Reset()
	mc.LastResult.Address = mc.PC.Address()

	// register end cycle callback
	defer func() {
		mc.cycleCallback = nil
	}()

	var err error
	var opcode uint8
	var defn *instructions.Definition

	// read next instruction (end cycle part of read8BitPC)
	// +1 cycle
	err = mc.read8BitPC(func(val uint8) error {
		opcode = val
		defn = mc.instructions[val]

		// !!TODO: remove this once all opcodes are defined/implemented
		if defn == nil {
			return curated.Errorf(UnimplementedInstruction, val, mc.PC.Address()-1)
		}

		mc.LastResult.Defn = defn

		return nil
	})
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

		// if there is no definition create a fake one
		// !!TODO: remove this once all opcodes are defined/implemented
		if mc.LastResult.Defn == nil {
			mc.LastResult.Defn = &instructions.Definition{
				OpCode:   opcode,
				Mnemonic: "??",
				Bytes:    1,
				Cycles:   0,
				// remaining fields are undefined
			}
		}

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

		if defn.Mnemonic == "BRK" {
			// BRK is unusual in that it increases the PC by two bytes despite
			// being an implied addressing mode.
			// +1 cycle
			err = mc.read8BitPC(nil)
			if err != nil {
				return err
			}

			// but we don't LastResult to show this
			mc.LastResult.ByteCount--
		} else {
			// phantom read
			// +1 cycle
			_, err = mc.read8Bit(mc.PC.Address())
			if err != nil {
				return err
			}
		}

	case instructions.Immediate:
		// for immediate mode, the value is the next byte in the program
		// therefore, we don't set the address and we read the value through the PC

		// +1 cycle
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
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
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
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
		// this part of the operation to the mnemonic switch below

	case instructions.ZeroPage:
		zeroPage = true

		// +1 cycle
		err = mc.read8BitPC(func(val uint8) error {
			// while we must trest the value as an address (ie. as uint16) we
			// actually only read an 8 bit value so we store the value as uint8
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
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
				if !curated.Has(err, bus.AddressError) {
					return err
				}
				mc.LastResult.Error = err.Error()
			}

			// +1 cycle
			err = mc.endCycle()
			if err != nil {
				if !curated.Has(err, bus.AddressError) {
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
			err = mc.endCycle()
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
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
		if err != nil {
			return err
		}
		indirectAddress := uint8(mc.LastResult.InstructionData)

		// phantom read before adjusting the index
		// +1 cycle
		_, err = mc.read8Bit(uint16(indirectAddress))
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
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
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
			_, err = mc.read8Bit((indexedAddress & 0xff00) | (address & 0x00ff))
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
			_, err := mc.read8Bit((indirectAddress & 0xff00) | (address & 0x00ff))
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
			_, err := mc.read8Bit((indirectAddress & 0xff00) | (address & 0x00ff))
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
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
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

		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case instructions.ZeroPageIndexedY:
		zeroPage = true

		// used exclusively for LDX ZeroPage,y

		// +1 cycles
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
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

		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

	default:
		return curated.Errorf("cpu: unknown addressing mode for %s", defn.Mnemonic)
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
				value, err = mc.read8Bit(address)
			}
			if err != nil {
				return err
			}
		} else if defn.Effect == instructions.RMW {
			// +1 cycle

			if zeroPage {
				value, err = mc.read8BitZeroPage(uint8(address))
			} else {
				value, err = mc.read8Bit(address)
			}
			if err != nil {
				return err
			}

			// phantom write
			// +1 cycle
			err = mc.write8Bit(address, value)
			if err != nil {
				return err
			}

			err = mc.endCycle()
			if err != nil {
				return err
			}
		}
	}

	// actually perform instruction based on mnemonic group
	switch defn.Mnemonic {
	case "NOP":
		// does nothing

	case "CLI":
		mc.Status.InterruptDisable = false

	case "SEI":
		mc.Status.InterruptDisable = true

	case "CLC":
		mc.Status.Carry = false

	case "SEC":
		mc.Status.Carry = true

	case "CLD":
		mc.Status.DecimalMode = false

	case "SED":
		mc.Status.DecimalMode = true

	case "CLV":
		mc.Status.Overflow = false

	case "PHA":
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), mc.A.Value())
		if err != nil {
			return err
		}
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "PLA":
		// +1 cycle
		mc.SP.Add(1, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// +1 cycle
		value, err = mc.read8Bit(mc.SP.Address())
		if err != nil {
			return err
		}
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "PHP":
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), mc.Status.Value())
		if err != nil {
			return err
		}
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "PLP":
		// +1 cycle
		mc.SP.Add(1, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}
		// +1 cycle
		value, err = mc.read8Bit(mc.SP.Address())
		if err != nil {
			return err
		}
		mc.Status.FromValue(value)

	case "TXA":
		mc.A.Load(mc.X.Value())
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "TAX":
		mc.X.Load(mc.A.Value())
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "TAY":
		mc.Y.Load(mc.A.Value())
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case "TYA":
		mc.A.Load(mc.Y.Value())
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "TSX":
		mc.X.Load(mc.SP.Value())
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "TXS":
		mc.SP.Load(mc.X.Value())
		// does not affect status register

	case "EOR":
		mc.A.EOR(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "ORA":
		mc.A.ORA(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "AND":
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "LDA":
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "LDX":
		mc.X.Load(value)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "LDY":
		mc.Y.Load(value)
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case "STA":
		// +1 cycle
		err = mc.write8Bit(address, mc.A.Value())
		if err != nil {
			return err
		}
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "STX":
		// +1 cycle
		err = mc.write8Bit(address, mc.X.Value())
		if err != nil {
			return err
		}
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "STY":
		// +1 cycle
		err = mc.write8Bit(address, mc.Y.Value())
		if err != nil {
			return err
		}
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "INX":
		mc.X.Add(1, false)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "INY":
		mc.Y.Add(1, false)
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case "DEX":
		mc.X.Add(255, false)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "DEY":
		mc.Y.Add(255, false)
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case "ASL":
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

	case "LSR":
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

	case "ADC":
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

	case "SBC":
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

	case "ROR":
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

	case "ROL":
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

	case "INC":
		r := mc.acc8
		r.Load(value)
		r.Add(1, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case "DEC":
		r := mc.acc8
		r.Load(value)
		r.Add(255, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case "CMP":
		r := mc.acc8
		r.Load(mc.A.Value())

		// maybe surprisingly, CMP can be implemented with binary subtract even
		// if decimal mode is active (the meaning is the same)
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case "CPX":
		r := mc.acc8
		r.Load(mc.X.Value())
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case "CPY":
		r := mc.acc8
		r.Load(mc.Y.Value())
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case "BIT":
		r := mc.acc8
		r.Load(value)
		mc.Status.Sign = r.IsNegative()
		mc.Status.Overflow = r.IsBitV()
		r.AND(mc.A.Value())
		mc.Status.Zero = r.IsZero()

	case "JMP":
		if !mc.NoFlowControl {
			mc.PC.Load(address)
		}

	case "BCC":
		err = mc.branch(!mc.Status.Carry, address)
		if err != nil {
			return err
		}

	case "BCS":
		err = mc.branch(mc.Status.Carry, address)
		if err != nil {
			return err
		}

	case "BEQ":
		err = mc.branch(mc.Status.Zero, address)
		if err != nil {
			return err
		}

	case "BMI":
		err = mc.branch(mc.Status.Sign, address)
		if err != nil {
			return err
		}

	case "BNE":
		err = mc.branch(!mc.Status.Zero, address)
		if err != nil {
			return err
		}

	case "BPL":
		err = mc.branch(!mc.Status.Sign, address)
		if err != nil {
			return err
		}

	case "BVC":
		err = mc.branch(!mc.Status.Overflow, address)
		if err != nil {
			return err
		}

	case "BVS":
		err = mc.branch(mc.Status.Overflow, address)
		if err != nil {
			return err
		}

	case "JSR":
		// +1 cycle
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = uint16(val)
			return nil
		})
		if err != nil {
			return err
		}

		// the current value of the PC is now correct, even though we've only read
		// one byte of the address so far. remember, RTS increments the PC when
		// read from the stack, meaning that the PC will be correct at that point

		// with that in mind, we're not sure what this extra cycle is for
		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// push MSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()>>8))
		if err != nil {
			return err
		}
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// push LSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()))
		if err != nil {
			return err
		}
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// perform jump
		err = mc.read8BitPC(func(val uint8) error {
			mc.LastResult.InstructionData = (uint16(val) << 8) | mc.LastResult.InstructionData
			return nil
		})
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

	case "RTS":
		// +1 cycle
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
		}
		err = mc.endCycle()
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
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "BRK":
		// push PC onto register (same effect as JSR)
		err := mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()>>8))
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()))
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// push status register (same effect as PHP)
		err = mc.write8Bit(mc.SP.Address(), mc.Status.Value())
		if err != nil {
			return err
		}

		// +1 cycle
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// set the break flag
		mc.Status.Break = true

		// perform jump
		var brkAddress uint16
		brkAddress, err = mc.read16Bit(addresses.IRQ)
		if err != nil {
			return err
		}
		if !mc.NoFlowControl {
			mc.PC.Load(brkAddress)
		}

	case "RTI":
		// pull status register (same effect as PLP)
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
		}

		// not sure when this cycle should occur
		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// +1 cycles
		value, err = mc.read8Bit(mc.SP.Address())
		if err != nil {
			return err
		}
		mc.Status.FromValue(value)

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

	case "nop":
		// does nothing (2 byte nop)

	case "lax":
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		mc.X.Load(value)

	case "skw":
		// does nothing (2 byte skip)
		// differs to dop because the second byte is actually read

	case "dcp":
		// AND the contents of the A register with value...
		// decrease value...
		r := mc.acc8
		r.Load(value)
		r.Add(255, false)
		value = r.Value()

		// ... and compare with the A register
		r.Load(mc.A.Value())
		mc.Status.Carry, _ = r.Subtract(value, true)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case "asr":
		mc.A.AND(value)

		// ... then LSR the result
		mc.Status.Carry = mc.A.LSR()
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "xaa":
		mc.A.Load(mc.X.Value())
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "axs":
		mc.X.AND(mc.A.Value())

		// axs subtract behaves like CMP as far as carry and overflow flags are
		// concerned
		mc.Status.Carry, _ = mc.X.Subtract(value, true)

		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "sax":
		r := mc.acc8
		r.Load(mc.A.Value())
		r.AND(mc.X.Value())

		// +1 cycle
		err = mc.write8Bit(address, r.Value())
		if err != nil {
			return err
		}
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "arr":
		mc.A.AND(value)
		mc.Status.Carry = mc.A.ROR(mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "slo":
		r := mc.acc8
		r.Load(value)
		mc.Status.Carry = r.ASL()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()
		mc.A.ORA(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "rla":
		r := mc.acc8
		r.Load(value)
		mc.Status.Carry = r.ROL(mc.Status.Carry)
		value = r.Value()
		mc.A.AND(r.Value())
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()

	case "isc":
		r := mc.acc8
		r.Load(value)
		r.Add(1, false)
		value = r.Value()
		mc.Status.Carry, mc.Status.Overflow = mc.A.Subtract(value, mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "anc":
		// immediate AND. puts bit 7 into the carry flag (in microcode terms
		// this is as though ASL had been enacted)
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		mc.Status.Carry = value&0x80 == 0x80

	default:
		return curated.Errorf("cpu: unknown mnemonic (%s)", defn.Mnemonic)
	}

	// for RMW instructions: write altered value back to memory
	if defn.Effect == instructions.RMW {
		err = mc.write8Bit(address, value)
		if err != nil {
			return err
		}

		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}
	}

	// finalise result
	mc.LastResult.Final = true

	// validity check. there's no need to enable unless you've just added a new
	// opcode and wanting to check the validity of the definition.
	// err = mc.LastResult.IsValid()
	// if err != nil {
	// 	return err
	// }

	return nil
}
