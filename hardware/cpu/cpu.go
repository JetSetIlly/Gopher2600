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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package cpu

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/cpu/execution"
	"gopher2600/hardware/cpu/instructions"
	"gopher2600/hardware/cpu/registers"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/bus"
	"log"
)

// CPU implements the 6507 found as found in the Atari 2600. Register logic is
// implemented by the Register type in the registers sub-package.
type CPU struct {
	PC     *registers.ProgramCounter
	A      *registers.Register
	X      *registers.Register
	Y      *registers.Register
	SP     *registers.Register
	Status *registers.StatusRegister

	// some operations only need an accumulator
	acc8  *registers.Register
	acc16 *registers.ProgramCounter

	mem          bus.CPUBus
	instructions []*instructions.Definition

	// isExecuting is used for sanity checks - to make sure we're not calling CPU
	// functions when we shouldn't
	isExecuting bool

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
}

// NewCPU is the preferred method of initialisation for the CPU structure
func NewCPU(mem bus.CPUBus) (*CPU, error) {
	mc := &CPU{mem: mem}

	mc.PC = registers.NewProgramCounter(0)
	mc.A = registers.NewRegister(0, "A")
	mc.X = registers.NewRegister(0, "X")
	mc.Y = registers.NewRegister(0, "Y")
	mc.SP = registers.NewRegister(0, "SP")
	mc.Status = registers.NewStatusRegister()

	mc.acc8 = registers.NewRegister(0, "accumulator")
	mc.acc16 = registers.NewProgramCounter(0)

	var err error

	mc.instructions, err = instructions.GetDefinitions()
	if err != nil {
		return nil, err
	}

	return mc, mc.Reset()
}

func (mc *CPU) String() string {
	return fmt.Sprintf("%s=%s %s=%s %s=%s %s=%s %s=%s %s=%s",
		mc.PC.Label(), mc.PC, mc.A.Label(), mc.A,
		mc.X.Label(), mc.X, mc.Y.Label(), mc.Y,
		mc.SP.Label(), mc.SP, mc.Status.Label(), mc.Status)
}

// Reset reinitialises all registers
func (mc *CPU) Reset() error {
	// we don't want the CPU to reset if we're in the middle of executing an
	// instruction.
	if mc.isExecuting {
		return errors.New(errors.InvalidOperationMidInstruction, "reset")
	}

	mc.LastResult.Reset()

	mc.PC.Load(0)
	mc.A.Load(0)
	mc.X.Load(0)
	mc.Y.Load(0)
	mc.SP.Load(255)
	mc.Status.Reset()
	mc.Status.Zero = mc.A.IsZero()
	mc.Status.Sign = mc.A.IsNegative()
	mc.Status.InterruptDisable = false
	mc.Status.Break = false
	mc.isExecuting = false
	mc.cycleCallback = nil
	mc.RdyFlg = true

	// not touching NoFlowControl

	return nil
}

// HasReset checks whether the CPU has recently been reset
func (mc CPU) HasReset() bool {
	return mc.LastResult.Address == 0 && mc.LastResult.Defn == nil
}

// LoadPCIndirect loads the contents of indirectAddress into the PC
func (mc *CPU) LoadPCIndirect(indirectAddress uint16) error {
	// changing the program counter mid-instruction could have unwanted side
	// effects
	if mc.isExecuting {
		return errors.New(errors.InvalidOperationMidInstruction, "load PC")
	}

	val, err := mc.read16Bit(indirectAddress)
	if err != nil {
		return err
	}
	mc.PC.Load(val)

	return nil
}

// LoadPC loads the contents of directAddress into the PC
func (mc *CPU) LoadPC(directAddress uint16) error {
	// changing the program counter mid-instruction could have unwanted side
	// effects
	if mc.isExecuting {
		return errors.New(errors.InvalidOperationMidInstruction, "load PC")
	}

	mc.PC.Load(directAddress)

	return nil
}

// read8Bit reads 8 bits from the specified address
//
// * note that read8Bit calls endCycle as appropriate
func (mc *CPU) read8Bit(address uint16) (uint8, error) {
	val, err := mc.mem.Read(address)

	if err != nil {
		if !errors.Is(err, errors.BusError) {
			return 0, err
		}
		mc.LastResult.BusError = err.Error()
	}

	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	return val, nil
}

// read8BitZero reads 8 bits from the specified zero page address
//
// * note that read8BitZeroPage calls endCycle as appropriate
func (mc *CPU) read8BitZeroPage(address uint8) (uint8, error) {
	val, err := mc.mem.ReadZeroPage(address)

	if err != nil {
		if !errors.Is(err, errors.BusError) {
			return 0, err
		}
		mc.LastResult.BusError = err.Error()
	}

	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	return val, nil
}

// write8Bit writes 8 bits to the specified address
//
// * note that write8Bit, unlike read8Bit(), does not call endCycle() this is
// because we need to differentiate between different addressing modes at
// different times.
func (mc *CPU) write8Bit(address uint16, value uint8) error {
	err := mc.mem.Write(address, value)

	if err != nil {
		// don't worry about unwritable addresses (unless strict addressing
		// is on)
		if !errors.Is(err, errors.BusError) {
			return err
		}
		mc.LastResult.BusError = err.Error()
	}

	return nil
}

// read16BitPC reads 16 bits from the address pointer to the program counter
//
// * note that read16Bit calls endCycle as appropriate
func (mc *CPU) read16Bit(address uint16) (uint16, error) {
	lo, err := mc.mem.Read(address)
	if err != nil {
		if !errors.Is(err, errors.BusError) {
			return 0, err
		}
		mc.LastResult.BusError = err.Error()
	}
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	hi, err := mc.mem.Read(address + 1)
	if err != nil {
		if !errors.Is(err, errors.BusError) {
			return 0, err
		}
		mc.LastResult.BusError = err.Error()
	}
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	val := uint16(hi) << 8
	val |= uint16(lo)

	return val, nil
}

// read8BitPC reads 8 bits from the address pointer to the program counter
func (mc *CPU) read8BitPC() (uint8, error) {
	op, err := mc.read8Bit(mc.PC.Address())
	if err != nil {
		return 0, err
	}

	// * note that this add operation does not require a call to endCycle. the
	// addition is implied as part of the call to read8Bit()
	carry, _ := mc.PC.Add(1)
	if carry {
		return 0, errors.New(errors.ProgramCounterCycled)
	}

	// bump the number of bytes read during instruction decode
	mc.LastResult.ByteCount++

	return op, nil
}

// read16BitPC reads 16 bits from the address pointer to the program counter
func (mc *CPU) read16BitPC() (uint16, error) {
	val, err := mc.read16Bit(mc.PC.Address())
	if err != nil {
		return 0, err
	}

	// strictly, PC should be incremented by one after reading the lo byte of
	// the next instruction but I don't believe this has any side-effects
	//
	// * note that this add operation does not require a call to endCycle. the
	// addition is implied as part of the call to read16Bit()
	carry, _ := mc.PC.Add(2)
	if carry {
		return 0, errors.New(errors.ProgramCounterCycled)
	}

	// bump the number of bytes read during instruction decode
	mc.LastResult.ByteCount += 2

	return val, nil
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
		// this is a bit wierd but without implementing the PC differently (with
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
// functionality
func (mc *CPU) endCycle() error {
	mc.LastResult.ActualCycles++
	if mc.cycleCallback == nil {
		return nil
	}
	return mc.cycleCallback()
}

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
	if mc.isExecuting {
		return errors.New(errors.InvalidOperationMidInstruction, "a previous call to ExecuteInstruction() has not yet completed")
	}

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
		mc.isExecuting = false
		mc.cycleCallback = nil
	}()

	var err error

	// read next instruction (end cycle part of read8BitPC)
	// +1 cycle
	opcode, err := mc.read8BitPC()
	if err != nil {
		return err
	}
	defn := mc.instructions[opcode]
	if defn == nil {
		return errors.New(errors.UnimplementedInstruction, opcode, mc.PC.Address()-1)
	}
	mc.LastResult.Defn = defn

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
	// StepResult and whether a page fault has occured. note that we don't do
	// this in the case of JSR
	switch defn.AddressingMode {
	case instructions.Implied:
		// implied mode does not use any additional bytes. however, the next
		// instruction is read but the PC is not incremented

		if defn.Mnemonic == "BRK" {
			// BRK is unusual in that it increases the PC by two bytes despite
			// being an implied addressing mode.
			// +1 cycle
			_, err = mc.read8BitPC()
			if err != nil {
				return err
			}

			// but we don't LastResult to show this
			mc.LastResult.ByteCount--
		} else {
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(mc.PC.Address())
			if err != nil {
				return err
			}
		}

	case instructions.Immediate:
		// for immediate mode, the value is the next byte in the program
		// therefore, we don't set the address and we read the value through the PC

		// +1 cycle
		value, err = mc.read8BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = value

	case instructions.Absolute:
		if defn.Effect != instructions.Subroutine {
			// +2 cycles
			address, err = mc.read16BitPC()
			if err != nil {
				return err
			}
			mc.LastResult.InstructionData = address
		}

		// else... for JSR, addresses are read slightly differently so we defer
		// this part of the operation to the mnemonic switch below

	case instructions.Relative:
		// relative addressing is only used for branch instructions, the address
		// is an offset value from the current PC position

		// most of the addressing cycles for this addressing mode are consumed
		// in the branch() function

		// +1 cycle
		value, err := mc.read8BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = value
		address = uint16(value)

	case instructions.ZeroPage:
		zeroPage = true

		// +1 cycle
		value, err := mc.read8BitPC()
		if err != nil {
			return err
		}
		address = uint16(value)
		mc.LastResult.InstructionData = address

	case instructions.IndexedZeroPageX:
		zeroPage = true

		// +1 cycles
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return err
		}

		mc.LastResult.InstructionData = indirectAddress
		mc.acc8.Load(indirectAddress)
		mc.acc8.Add(mc.X.Value(), false)
		address = mc.acc8.Address()

		// make a note of zero page index bug
		if uint16(indirectAddress+mc.X.Value())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.CPUBug = fmt.Sprintf("zero page index bug")
		}

		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case instructions.IndexedZeroPageY:
		zeroPage = true

		// used exclusively for LDX ZeroPage,y

		// +1 cycles
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return err
		}

		mc.LastResult.InstructionData = indirectAddress
		mc.acc8.Load(indirectAddress)
		mc.acc8.Add(mc.Y.Value(), false)
		address = mc.acc8.Address()

		// make a note of zero page index bug
		if uint16(indirectAddress+mc.Y.Value())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.CPUBug = fmt.Sprintf("zero page index bug")
		}

		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case instructions.Indirect:
		// indirect addressing (without indexing) is only used for the JMP command

		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

		// handle indirect addressing JMP bug
		if indirectAddress&0x00ff == 0x00ff {
			mc.LastResult.CPUBug = fmt.Sprintf("indirect addressing bug (JMP bug)")

			lo, err := mc.mem.Read(indirectAddress)
			if err != nil {
				if !errors.Is(err, errors.BusError) {
					return err
				}
				mc.LastResult.BusError = err.Error()
			}

			// +1 cycle
			err = mc.endCycle()
			if err != nil {
				if !errors.Is(err, errors.BusError) {
					return err
				}
				mc.LastResult.BusError = err.Error()
				return err
			}

			// in this bug path, the lower byte of the indirect address is on a
			// page boundary. because of the bug we must read high byte of JMP
			// address from the zero byte of the same page (rather than the
			// zero byte of the next page)
			hi, err := mc.mem.Read(indirectAddress & 0xff00)
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

	case instructions.PreIndexedIndirect: // x indexing
		// +1 cycle
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

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
			mc.LastResult.CPUBug = fmt.Sprintf("indirect addressing bug")
		}

		// +2 cycles
		address, err = mc.read16Bit(mc.acc8.Address())
		if err != nil {
			return err
		}

		// never a page fault wth pre-index indirect addressing

	case instructions.PostIndexedIndirect: // y indexing
		// +1 cycle
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

		// +2 cycles
		indexedAddress, err := mc.read16Bit(uint16(indirectAddress))
		if err != nil {
			return err
		}

		mc.acc16.Load(mc.Y.Address())
		mc.acc16.Add(indexedAddress & 0x00ff)
		address = mc.acc16.Address()

		// check for page fault
		if defn.PageSensitive && (address&0xff00 == 0x0100) {
			mc.LastResult.CPUBug = fmt.Sprintf("indirect addressing bug")
			mc.LastResult.PageFault = true
		}

		if mc.LastResult.PageFault || defn.Effect == instructions.Write || defn.Effect == instructions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indexedAddress & 0xff00)
		address = mc.acc16.Address()

	case instructions.AbsoluteIndexedX:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

		// add index to LSB of address
		mc.acc16.Load(mc.X.Address())
		mc.acc16.Add(indirectAddress & 0x00ff)
		address = mc.acc16.Address()

		// check for page fault
		mc.LastResult.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if mc.LastResult.PageFault || defn.Effect == instructions.Write || defn.Effect == instructions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indirectAddress & 0xff00)
		address = mc.acc16.Address()

	case instructions.AbsoluteIndexedY:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

		// add index to LSB of address
		mc.acc16.Load(mc.Y.Address())
		mc.acc16.Add(indirectAddress & 0x00ff)
		address = mc.acc16.Address()

		// check for page fault
		mc.LastResult.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if mc.LastResult.PageFault || defn.Effect == instructions.Write || defn.Effect == instructions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indirectAddress & 0xff00)
		address = mc.acc16.Address()

	default:
		log.Fatalf("unknown addressing mode for %s", defn.Mnemonic)
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
			r = mc.acc8
			r.Load(value)
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.ASL()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case "LSR":
		var r *registers.Register
		if defn.Effect == instructions.RMW {
			r = mc.acc8
			r.Load(value)
		} else {
			r = mc.A
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
			r = mc.acc8
			r.Load(value)
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.ROR(mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.Value()

	case "ROL":
		var r *registers.Register
		if defn.Effect == instructions.RMW {
			r = mc.acc8
			r.Load(value)
		} else {
			r = mc.A
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
		cmp := mc.acc8
		cmp.Load(mc.A.Value())

		// maybe surprisingly, CMP can be implemented with binary subtract even
		// if decimal mode is active (the meaning is the same)
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPX":
		cmp := mc.acc8
		cmp.Load(mc.X.Value())
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPY":
		cmp := mc.acc8
		cmp.Load(mc.Y.Value())
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "BIT":
		cmp := mc.acc8
		cmp.Load(value)
		mc.Status.Sign = cmp.IsNegative()
		mc.Status.Overflow = cmp.IsBitV()
		cmp.AND(mc.A.Value())
		mc.Status.Zero = cmp.IsZero()

	case "JMP":
		if !mc.NoFlowControl {
			mc.PC.Load(address)
		}

	case "BCC":
		err := mc.branch(!mc.Status.Carry, address)
		if err != nil {
			return err
		}

	case "BCS":
		err := mc.branch(mc.Status.Carry, address)
		if err != nil {
			return err
		}

	case "BEQ":
		err := mc.branch(mc.Status.Zero, address)
		if err != nil {
			return err
		}

	case "BMI":
		err := mc.branch(mc.Status.Sign, address)
		if err != nil {
			return err
		}

	case "BNE":
		err := mc.branch(!mc.Status.Zero, address)
		if err != nil {
			return err
		}

	case "BPL":
		err := mc.branch(!mc.Status.Sign, address)
		if err != nil {
			return err
		}

	case "BVC":
		err := mc.branch(!mc.Status.Overflow, address)
		if err != nil {
			return err
		}

	case "BVS":
		err := mc.branch(mc.Status.Overflow, address)
		if err != nil {
			return err
		}

	case "JSR":
		// +1 cycle
		lsb, err := mc.read8BitPC()
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
		msb, err := mc.read8BitPC()
		if err != nil {
			return err
		}

		address = (uint16(msb) << 8) | uint16(lsb)
		if !mc.NoFlowControl {
			mc.PC.Load(address)
		}

		// store address in theInstructionData field of result
		//
		// we would normally do this in the addressing mode switch above. however,
		// JSR uses absolute addressing and we deliberately do nothing in that
		// switch for 'sub-routine' commands
		mc.LastResult.InstructionData = address

	case "RTS":
		if !mc.NoFlowControl {
			// +1 cycle
			mc.SP.Add(1, false)
		}
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// +2 cycles
		rtsAddress, err := mc.read16Bit(mc.SP.Address())
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
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		err = mc.write8Bit(mc.SP.Address(), uint8(mc.PC.Address()))
		if err != nil {
			return err
		}
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
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// set the break flag
		mc.Status.Break = true

		// perform jump
		brkAddress, err := mc.read16Bit(addresses.IRQ)
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
		err = mc.endCycle()
		if err != nil {
			return err
		}

		value, err = mc.read8Bit(mc.SP.Address())
		if err != nil {
			return err
		}
		mc.Status.FromValue(value)

		// pull program counter (same effect as RTS)
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
		}

		rtiAddress, err := mc.read16Bit(mc.SP.Address())
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
		mc.X.Subtract(value, true)
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

	default:
		// this should never, ever happen
		log.Fatalf("WTF! unknown mnemonic! (%s)", defn.Mnemonic)
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
