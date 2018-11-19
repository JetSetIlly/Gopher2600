package cpu

// TODO List
// ---------
// o NMOS indexed addressing extra read when crossing page boundaries
// o check that NoSideEffects is consistent in its intention
// o check that all calls to endCycle() occur when they're supposed to

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/register"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"log"
)

const irqInterruptVector = 0xfffe

// CPU is the main container structure for the package
type CPU struct {
	PC     *register.Register
	A      *register.Register
	X      *register.Register
	Y      *register.Register
	SP     *register.Register
	Status StatusRegister

	mem     memory.CPUBus
	opCodes []*definitions.InstructionDefinition

	// endCycle is called at the end of the imaginary CPU cycle. for example,
	// reading a byte from memory takes one cycle and so the emulation will
	// call endCycle() at that point. ExecuteInstruction() accepts an argument
	// cycleCallback which is called by endCycle for additional functionality
	//
	// by definition: if it is undefined then no execution is currently being
	// executed (see IsExecutingInstruction method)
	endCycle func()

	// controls whether cpu executes a cycle when it receives a clock tick (pin
	// 3 of the 6507)
	RdyFlg bool

	// it is somtimes useful to ignore branching instructions and other
	// side-effects. we use this in the disassembly package to make sure
	// we reach every part of the program
	NoSideEffects bool

	// silently ignore addressing errors unless StrictAddressing is true
	StrictAddressing bool
}

// NewCPU is the preferred method of initialisation for the CPU structure
func NewCPU(mem memory.CPUBus) (*CPU, error) {
	var err error

	mc := new(CPU)
	mc.mem = mem

	mc.PC = register.NewRegister(0, 16, "PC", "PC")
	mc.A = register.NewRegister(0, 8, "A", "A")
	mc.X = register.NewRegister(0, 8, "X", "X")
	mc.Y = register.NewRegister(0, 8, "Y", "Y")
	mc.SP = register.NewRegister(0, 8, "SP", "SP")
	mc.Status = NewStatusRegister("Status", "SR")

	mc.opCodes, err = definitions.GetInstructionDefinitions()
	if err != nil {
		return nil, err
	}

	mc.Reset()

	return mc, nil
}

// MachineInfoTerse returns the cpu information in terse format
func (mc *CPU) MachineInfoTerse() string {
	return fmt.Sprintf("%s %s %s %s %s %s", mc.PC.MachineInfoTerse(), mc.A.MachineInfoTerse(), mc.X.MachineInfoTerse(), mc.Y.MachineInfoTerse(), mc.SP.MachineInfoTerse(), mc.Status.MachineInfoTerse())
}

// MachineInfo returns the cpu information in verbose format
func (mc *CPU) MachineInfo() string {
	return fmt.Sprintf("%v\n%v\n%v\n%v\n%v\n%v", mc.PC, mc.A, mc.X, mc.Y, mc.SP, mc.Status)
}

// map String to MachineInfo
func (mc *CPU) String() string {
	return mc.MachineInfo()
}

// IsExecuting returns true if it is called during an ExecuteInstruction() callback
func (mc *CPU) IsExecuting() bool {
	return mc.endCycle != nil
}

// Reset reinitialises all registers
func (mc *CPU) Reset() error {
	// sanity check
	if mc.IsExecuting() {
		return errors.NewGopherError(errors.InvalidOperationMidInstruction, "reset")
	}

	mc.PC.Load(0)
	mc.A.Load(0)
	mc.X.Load(0)
	mc.Y.Load(0)
	mc.SP.Load(255)
	mc.Status.reset()
	mc.Status.Zero = mc.A.IsZero()
	mc.Status.Sign = mc.A.IsNegative()
	mc.Status.InterruptDisable = true
	mc.Status.Break = true
	mc.endCycle = nil
	mc.RdyFlg = true

	return nil
}

// LoadPC loads the contents of indirectAddress into the PC
func (mc *CPU) LoadPC(indirectAddress uint16) error {
	// sanity check
	if mc.IsExecuting() {
		return errors.NewGopherError(errors.InvalidOperationMidInstruction, "load PC")
	}

	// because we call this LoadPC() outside of the CPU's ExecuteInstruction()
	// cycle we need to make sure endCycle() is in a valid state for the duration
	// of the function
	mc.endCycle = func() {}
	defer func() {
		mc.endCycle = nil
	}()

	val, err := mc.read16Bit(indirectAddress)
	if err != nil {
		return err
	}
	mc.PC.Load(val)

	return nil
}

// note that write8Bit, unline read8Bit(), does not call endCycle() this is
// because we need to differentiate between different addressing modes at
// different times.
func (mc *CPU) write8Bit(address uint16, value uint8) error {
	if mc.NoSideEffects {
		return nil
	}

	err := mc.mem.Write(address, value)

	if err != nil {
		switch err := err.(type) {
		case errors.GopherError:
			// don't worry about unwritable addresses (unless strict addressing
			// is on)
			if mc.StrictAddressing || err.Errno != errors.UnwritableAddress {
				return err
			}
		default:
			return err
		}
	}

	return nil
}

// note that read8Bit calls endCycle as appropriate
func (mc *CPU) read8Bit(address uint16) (uint8, error) {
	val, err := mc.mem.Read(address)

	if err != nil {
		switch err := err.(type) {
		case errors.GopherError:
			// don't worry about unreadable addresses (unless strict addressing
			// is on)
			if mc.StrictAddressing || err.Errno != errors.UnreadableAddress {
				return 0, err
			}
		default:
			return 0, err
		}
	}

	mc.endCycle()

	return val, nil
}

func (mc *CPU) read16Bit(address uint16) (uint16, error) {
	lo, err := mc.mem.Read(address)
	if err != nil {
		return 0, err
	}
	mc.endCycle()

	hi, err := mc.mem.Read(address + 1)
	if err != nil {
		return 0, err
	}
	mc.endCycle()

	val := uint16(hi) << 8
	val |= uint16(lo)

	return val, nil
}

func (mc *CPU) read8BitPC() (uint8, error) {
	op, err := mc.read8Bit(mc.PC.ToUint16())
	if err != nil {
		return 0, err
	}
	carry, _ := mc.PC.Add(1, false)
	if carry {
		return 0, errors.NewGopherError(errors.ProgramCounterCycled, nil)
	}
	return op, nil
}

func (mc *CPU) read16BitPC() (uint16, error) {
	val, err := mc.read16Bit(mc.PC.ToUint16())
	if err != nil {
		return 0, err
	}

	// strictly, PC should be incremented by one after reading the lo byte of
	// the next instruction but I don't believe this has any side-effects
	carry, _ := mc.PC.Add(2, false)
	if carry {
		return 0, errors.NewGopherError(errors.ProgramCounterCycled, nil)
	}

	return val, nil
}

func (mc *CPU) branch(flag bool, address uint16, result *result.Instruction) error {
	// return early if NoSideEffects flag is turned on
	if mc.NoSideEffects {
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
		// phantom read
		// +1 cycle
		_, err := mc.read8Bit(mc.PC.ToUint16())
		if err != nil {
			return err
		}

		// note current PC for reference
		oldPC := mc.PC.ToUint16()

		// add LSB to PC
		// this is a bit wierd but without implementing the PC differently (with
		// two 8bit bytes perhaps) this is the only way I can see how to do it with
		// the desired cycle accuracy:
		//  o Add full (sign extended) 16bit address to PC
		//  o note whether a page fault has occurred
		//  o restore the MSB of the PC using the MSB of the old PC value
		mc.PC.Add(address, false)
		result.PageFault = oldPC&0xff00 != mc.PC.ToUint16()&0xff00
		mc.PC.Load(oldPC&0xff00 | mc.PC.ToUint16()&0x00ff)

		// check to see whether branching has crossed a page
		if result.PageFault {
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(mc.PC.ToUint16())
			if err != nil {
				return err
			}

			// correct program counter
			if address&0xff00 == 0xff00 {
				mc.PC.Add(0xff00, false)
			} else {
				mc.PC.Add(0x0100, false)
			}

			// note that we've triggered a page fault
			result.PageFault = true
		}
	}

	return nil
}

// ExecuteInstruction steps CPU forward one instruction, calling
// cycleCallback() after every cycle
func (mc *CPU) ExecuteInstruction(cycleCallback func(*result.Instruction)) (*result.Instruction, error) {
	// sanity check
	if mc.IsExecuting() {
		panic(fmt.Errorf("can't call cpu.ExecuteInstruction() in the middle of another cpu.ExecuteInstruction()"))
	}

	// do nothing and return nothing if ready flag is false
	if !mc.RdyFlg {
		cycleCallback(nil)
		return nil, nil
	}

	// prepare StepResult structure
	result := new(result.Instruction)
	result.Address = mc.PC.ToUint16()

	// register end cycle callback
	mc.endCycle = func() {
		result.ActualCycles++
		cycleCallback(result)
	}
	defer func() {
		mc.endCycle = nil
	}()

	var err error

	// read next instruction (end cycle part of read8BitPC)
	// +1 cycle
	operator, err := mc.read8BitPC()
	if err != nil {
		return nil, err
	}
	defn := mc.opCodes[operator]
	if defn == nil {
		if operator == 0xff {
			return nil, errors.NewGopherError(errors.NullInstruction, nil)
		}
		return nil, errors.NewGopherError(errors.UnimplementedInstruction, operator, mc.PC.ToUint16()-1)
	}
	result.Defn = defn

	// address is the actual address to use to access memory (after any indexing
	// has taken place)
	var address uint16

	// value is nil if addressing mode is implied and is read from the program for
	// immediate/relative mode, and from non-program memory for all other modes
	// note that for instructions which are read-modify-write, the value will
	// change during execution and be used to write back to memory
	var value uint8

	// get address to use when reading/writing from/to memory (note that in the
	// case of immediate addressing, we are actually getting the value to use in
	// the instruction, not the address). we also take the opportunity to set
	// the InstructionData value for the StepResult and whether a page fault has
	// occured
	switch defn.AddressingMode {
	case definitions.Implied:
		// implied mode does not use any additional bytes. however, the next
		// instruction is read but the PC is not incremented

		if defn.Mnemonic == "BRK" {
			// BRK is unusual in that it increases the PC by two bytes despite
			// being an implied addressing mode.
			// +1 cycle
			_, err = mc.read8BitPC()
			if err != nil {
				return nil, err
			}
		} else {
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(mc.PC.ToUint16())
			if err != nil {
				return nil, err
			}
		}

	case definitions.Immediate:
		// for immediate mode, the value is the next byte in the program
		// therefore, we don't set the address and we read the value through the PC

		// +1 cycle
		value, err = mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		result.InstructionData = value

	case definitions.Absolute:
		if defn.Effect != definitions.Subroutine {
			// +2 cycles
			address, err = mc.read16BitPC()
			if err != nil {
				return nil, err
			}
			result.InstructionData = address
		}

		// else... for JSR, addresses are read slightly differently so we defer
		// this part of the operation to the mnemonic switch below

	case definitions.Relative:
		// relative addressing is only used for branch instructions, the address
		// is an offset value from the current PC position

		// most of the addressing cycles for this addressing mode are consumed
		// in the branch() function

		// +1 cycle
		value, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		result.InstructionData = value
		address = uint16(value)

	case definitions.ZeroPage:
		// +1 cycle
		value, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		address = uint16(value)
		result.InstructionData = address

	case definitions.IndexedZeroPageX:
		// +1 cycles
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		adder := register.NewAnonRegister(indirectAddress, 8)
		adder.Add(mc.X, false)
		address = adder.ToUint16()
		result.InstructionData = indirectAddress

		// +1 cycle
		mc.endCycle()

	case definitions.IndexedZeroPageY:
		// used exclusively for LDX ZeroPage,y

		// +1 cycles
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		adder := register.NewAnonRegister(indirectAddress, 8)
		adder.Add(mc.Y, false)
		address = adder.ToUint16()
		result.InstructionData = indirectAddress

		// +1 cycle
		mc.endCycle()

	case definitions.Indirect:
		// indirect addressing (without indexing) is only used for the JMP command

		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return nil, err
		}

		// implement NMOS 6502 Indirect JMP bug
		if indirectAddress&0x00ff == 0x00ff {
			lo, err := mc.mem.Read(indirectAddress)
			if err != nil {
				return nil, err
			}

			// +1 cycle
			mc.endCycle()

			hi, err := mc.mem.Read(indirectAddress & 0xff00)
			if err != nil {
				return nil, err
			}
			address = uint16(hi) << 8
			address |= uint16(lo)

			result.InstructionData = indirectAddress
			result.Bug = fmt.Sprintf("Indirect JMP Bug")

			// +1 cycle
			mc.endCycle()

		} else {
			// normal, non-buggy behaviour

			// +2 cycles
			address, err = mc.read16Bit(indirectAddress)
			if err != nil {
				return nil, err
			}
			result.InstructionData = indirectAddress
		}

	case definitions.PreIndexedIndirect:
		// +1 cycle
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}

		// phantom read before adjusting the index
		// +1 cycle
		_, err = mc.read8Bit(uint16(indirectAddress))
		if err != nil {
			return nil, err
		}

		// using 8bit addition because we don't want a page-fault
		adder := register.NewAnonRegister(mc.X, 8)
		adder.Add(indirectAddress, false)

		// +2 cycles
		address, err = mc.read16Bit(adder.ToUint16())
		if err != nil {
			return nil, err
		}

		// never a page fault wth pre-index indirect addressing
		result.InstructionData = indirectAddress

	case definitions.PostIndexedIndirect:
		// +1 cycle
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}

		// +2 cycles
		indexedAddress, err := mc.read16Bit(uint16(indirectAddress))
		if err != nil {
			return nil, err
		}

		adder := register.NewAnonRegister(mc.Y, 16)
		adder.Add(indexedAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)

		if result.PageFault || defn.Effect == definitions.Write || defn.Effect == definitions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return nil, err
			}
		}

		// fix MSB of address
		adder.Add(indexedAddress&0xff00, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

	case definitions.AbsoluteIndexedX:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return nil, err
		}

		// add index to LSB of address
		adder := register.NewAnonRegister(mc.X, 16)
		adder.Add(indirectAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if result.PageFault || defn.Effect == definitions.Write || defn.Effect == definitions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return nil, err
			}
		}

		// fix MSB of address
		adder.Add(indirectAddress&0xff00, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

	case definitions.AbsoluteIndexedY:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return nil, err
		}

		// add index to LSB of address
		adder := register.NewAnonRegister(mc.Y, 16)
		adder.Add(indirectAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if result.PageFault || defn.Effect == definitions.Write || defn.Effect == definitions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return nil, err
			}
		}

		// fix MSB of address
		adder.Add(indirectAddress&0xff00, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

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
	if !(defn.AddressingMode == definitions.Implied || defn.AddressingMode == definitions.Immediate) {
		if defn.Effect == definitions.Read {
			// +1 cycle
			value, err = mc.read8Bit(address)
			if err != nil {
				return nil, err
			}
		} else if defn.Effect == definitions.RMW {
			// +1 cycle
			value, err = mc.read8Bit(address)
			if err != nil {
				return nil, err
			}

			// phantom write
			// +1 cycle
			err = mc.write8Bit(address, value)

			if err != nil {
				return nil, err
			}
			mc.endCycle()
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
		err = mc.write8Bit(mc.SP.ToUint16(), mc.A.ToUint8())
		if err != nil {
			return nil, err
		}
		mc.SP.Add(255, false)

	case "PLA":
		// +1 cycle
		mc.SP.Add(1, false)
		mc.endCycle()
		// +1 cycle
		value, err = mc.read8Bit(mc.SP.ToUint16())
		if err != nil {
			return nil, err
		}
		mc.A.Load(value)

	case "PHP":
		err = mc.write8Bit(mc.SP.ToUint16(), mc.Status.ToUint8())
		if err != nil {
			return nil, err
		}
		mc.SP.Add(255, false)

	case "PLP":
		// +1 cycle
		mc.SP.Add(1, false)
		mc.endCycle()
		// +1 cycle
		value, err = mc.read8Bit(mc.SP.ToUint16())
		if err != nil {
			return nil, err
		}
		mc.Status.FromUint8(value)

	case "TXA":
		mc.A.Load(mc.X)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "TAX":
		mc.X.Load(mc.A)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "TAY":
		mc.Y.Load(mc.A)
		mc.Status.Zero = mc.Y.IsZero()
		mc.Status.Sign = mc.Y.IsNegative()

	case "TYA":
		mc.A.Load(mc.Y)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "TSX":
		mc.X.Load(mc.SP)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "TXS":
		mc.SP.Load(mc.X)
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
		err = mc.write8Bit(address, mc.A.ToUint8())
		if err != nil {
			return nil, err
		}

	case "STX":
		err = mc.write8Bit(address, mc.X.ToUint8())
		if err != nil {
			return nil, err
		}

	case "STY":
		err = mc.write8Bit(address, mc.Y.ToUint8())
		if err != nil {
			return nil, err
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
		var r *register.Register
		if defn.Effect == definitions.RMW {
			r = register.NewAnonRegister(value, mc.A.Size())
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.ASL()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "LSR":
		var r *register.Register
		if defn.Effect == definitions.RMW {
			r = register.NewAnonRegister(value, mc.A.Size())
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.LSR()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "ADC":
		if mc.Status.DecimalMode {
			mc.Status.Carry = mc.A.AddDecimal(value, mc.Status.Carry)
			// decimal mode doesn't affect overflow flag (yet?)
		} else {
			mc.Status.Carry, mc.Status.Overflow = mc.A.Add(value, mc.Status.Carry)
		}
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "SBC":
		if mc.Status.DecimalMode {
			mc.Status.Carry = mc.A.SubtractDecimal(value, mc.Status.Carry)
			// decimal mode doesn't affect overflow flag (yet?)
		} else {
			mc.Status.Carry, mc.Status.Overflow = mc.A.Subtract(value, mc.Status.Carry)
		}
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "ROR":
		var r *register.Register
		if defn.Effect == definitions.RMW {
			r = register.NewAnonRegister(value, mc.A.Size())
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.ROR(mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "ROL":
		var r *register.Register
		if defn.Effect == definitions.RMW {
			r = register.NewAnonRegister(value, mc.A.Size())
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.ROL(mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "INC":
		r := register.NewAnonRegister(value, 8)
		r.Add(1, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "DEC":
		r := register.NewAnonRegister(value, 8)
		r.Add(255, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "CMP":
		cmp := register.NewAnonRegister(mc.A, mc.A.Size())

		// maybe surprisingly, CMP can be implemented with binary subtract even
		// if decimal mode is active (the meaning is the same)
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPX":
		cmp := register.NewAnonRegister(mc.X, mc.X.Size())
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPY":
		cmp := register.NewAnonRegister(mc.Y, mc.Y.Size())
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "BIT":
		cmp := register.NewAnonRegister(value, mc.A.Size())
		mc.Status.Sign = cmp.IsNegative()
		mc.Status.Overflow = cmp.IsBitV()
		cmp.AND(mc.A)
		mc.Status.Zero = cmp.IsZero()

	case "JMP":
		if !mc.NoSideEffects {
			mc.PC.Load(address)
		}

	case "BCC":
		err := mc.branch(!mc.Status.Carry, address, result)
		if err != nil {
			return nil, err
		}

	case "BCS":
		err := mc.branch(mc.Status.Carry, address, result)
		if err != nil {
			return nil, err
		}

	case "BEQ":
		err := mc.branch(mc.Status.Zero, address, result)
		if err != nil {
			return nil, err
		}

	case "BMI":
		err := mc.branch(mc.Status.Sign, address, result)
		if err != nil {
			return nil, err
		}

	case "BNE":
		err := mc.branch(!mc.Status.Zero, address, result)
		if err != nil {
			return nil, err
		}

	case "BPL":
		err := mc.branch(!mc.Status.Sign, address, result)
		if err != nil {
			return nil, err
		}

	case "BVC":
		err := mc.branch(!mc.Status.Overflow, address, result)
		if err != nil {
			return nil, err
		}

	case "BVS":
		err := mc.branch(mc.Status.Overflow, address, result)
		if err != nil {
			return nil, err
		}

	case "JSR":
		// +1 cycle
		lsb, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}

		// the current value of the PC is now correct, even though we've only read
		// one byte of the address so far. remember, RTS increments the PC when
		// read from the stack, meaning that the PC will be correct at that point

		// with that in mind, we're not sure what this extra cycle is for
		// +1 cycle
		mc.endCycle()

		// push MSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.ToUint16(), uint8((mc.PC.ToUint16()&0xFF00)>>8))
		if err != nil {
			return nil, err
		}
		mc.SP.Add(255, false)
		mc.endCycle()

		// push LSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.write8Bit(mc.SP.ToUint16(), uint8(mc.PC.ToUint16()&0x00FF))
		if err != nil {
			return nil, err
		}
		mc.SP.Add(255, false)
		mc.endCycle()

		// perform jump
		msb, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}

		address = (uint16(msb) << 8) | uint16(lsb)
		if !mc.NoSideEffects {
			mc.PC.Load(address)
		}

		// store address in theInstructionData field of result
		//
		// we would normally do this in the addressing mode switch above. however,
		// JSR uses absolute addressing and we deliberately do nothing in that
		// switch for 'sub-routine' commands
		result.InstructionData = address

	case "RTS":
		if !mc.NoSideEffects {
			// +1 cycle
			mc.SP.Add(1, false)
			mc.endCycle()

			// +2 cycles
			rtsAddress, err := mc.read16Bit(mc.SP.ToUint16())
			if err != nil {
				return nil, err
			}
			mc.SP.Add(1, false)

			// load and correct PC
			mc.PC.Load(rtsAddress)
			mc.PC.Add(1, false)

			// +1 cycle
			mc.endCycle()
		}

	case "BRK":
		// push PC onto register (same effect as JSR)
		err := mc.write8Bit(mc.SP.ToUint16(), uint8((mc.PC.ToUint16()&0xFF00)>>8))
		if err != nil {
			return nil, err
		}
		mc.SP.Add(255, false)
		mc.endCycle()

		err = mc.write8Bit(mc.SP.ToUint16(), uint8(mc.PC.ToUint16()&0x00FF))
		if err != nil {
			return nil, err
		}
		mc.SP.Add(255, false)
		mc.endCycle()

		// push status register (same effect as PHP)
		err = mc.write8Bit(mc.SP.ToUint16(), mc.Status.ToUint8())
		if err != nil {
			return nil, err
		}
		mc.SP.Add(255, false)
		mc.endCycle()

		// set the break flag
		mc.Status.Break = true

		// perform jump
		brkAddress, err := mc.read16Bit(irqInterruptVector)
		if err != nil {
			return nil, err
		}
		if !mc.NoSideEffects {
			mc.PC.Load(brkAddress)
		}

	case "RTI":
		// pull status register (same effect as PLP)
		mc.SP.Add(1, false)
		mc.endCycle()
		value, err = mc.read8Bit(mc.SP.ToUint16())
		if err != nil {
			return nil, err
		}
		mc.Status.FromUint8(value)

		// pull program counter (same effect as RTS)
		if !mc.NoSideEffects {
			mc.SP.Add(1, false)
			mc.endCycle()
			rtiAddress, err := mc.read16Bit(mc.SP.ToUint16())
			if err != nil {
				return nil, err
			}
			mc.SP.Add(1, false)
			mc.PC.Load(rtiAddress)
			mc.PC.Add(1, false)
			mc.endCycle()
		}

	// undocumented instructions

	case "dop":
		// does nothing (2 byte nop)

	case "lax":
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		mc.X.Load(value)

	default:
		// this should never, ever happen
		log.Fatalf("WTF! unknown mnemonic! (%s)", defn.Mnemonic)
	}

	// for Write instructions: consume an extra cycle for the extra memory
	// access we've already performed
	if defn.Effect == definitions.Write {
		// +1 cycle
		mc.endCycle()
	}

	// for RMW instructions: write altered value back to memory
	if defn.Effect == definitions.RMW {
		err = mc.write8Bit(address, value)
		if err != nil {
			return nil, err

		}

		// +1 cycle
		mc.endCycle()
	}

	// finalise result
	result.Final = true

	return result, nil
}
