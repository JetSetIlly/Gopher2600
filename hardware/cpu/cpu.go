package cpu

// TODO List
// ---------
// . NMOS indexed addressing extra read when crossing page boundaries
// . Binary Decimal Mode

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/register"
	"gopher2600/hardware/memory"
	"log"
)

// CPU is the main container structure for the package
type CPU struct {
	PC     *register.Register
	A      *register.Register
	X      *register.Register
	Y      *register.Register
	SP     *register.Register
	Status StatusRegister

	mem     memory.CPUBus
	opCodes map[uint8]definitions.InstructionDefinition

	// endCycle is called at the end of the imaginary CPU cycle. for example,
	// reading a byte from memory takes one cycle and so the emulation will
	// call endCycle at that point. ExecuteInstruction accepts an argument
	// cycleCallback which in turn is called by endCycle
	// if it is undefined then no execution is currently being executed
	// (see IsExecutingInstruction method)
	endCycle func()

	// controls whether cpu is execute a cycle when it receives a clock tick (pin
	// 3 of the 6507)
	RdyFlg bool

	// it is somtimes useful to ignore branching instructions and other
	// side-effects. we use this in the disassembly package to make sure
	// we reach every part of the program
	NoSideEffects bool
}

// New is the preferred method of initialisation for the CPU structure
func New(mem memory.CPUBus) (*CPU, error) {
	var err error

	mc := new(CPU)
	mc.mem = mem

	mc.PC, err = register.New(0, 16, "PC", "PC")
	if err != nil {
		return nil, err
	}

	mc.A, err = register.New(0, 8, "A", "A")
	if err != nil {
		return nil, err
	}

	mc.X, err = register.New(0, 8, "X", "X")
	if err != nil {
		return nil, err
	}

	mc.Y, err = register.New(0, 8, "Y", "Y")
	if err != nil {
		return nil, err
	}

	mc.SP, err = register.New(0, 8, "SP", "SP")
	if err != nil {
		return nil, err
	}

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
		return fmt.Errorf("can't reset CPU in the middle of an instruction")
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
		return fmt.Errorf("can't alter program counter in the middle of an instruction")
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

func (mc *CPU) read8Bit(address uint16) (uint8, error) {
	val, err := mc.mem.Read(address)
	if err != nil {
		return 0, err
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
	mc.PC.Add(1, false)
	return op, nil
}

func (mc *CPU) read16BitPC() (uint16, error) {
	val, err := mc.read16Bit(mc.PC.ToUint16())
	if err != nil {
		return 0, err
	}
	mc.PC.Add(2, false)
	return val, nil
}

func (mc *CPU) branch(flag bool, address uint16, result *InstructionResult) error {
	// return early if IgnoreBranching flag is turned on
	if mc.NoSideEffects {
		return nil
	}

	// in the case of branchng (relative addressing) we've read an 8bit value
	// rather than a 16bit value to use as the "address". we do this kind of
	// thing all over the place and it normally doesn't matter but because we'll
	// sometimes be doing subtractions with this value we need to make sure the
	// sign bit of the 8bit value has been propogated into the most-significant
	// bits of the 16bit value.
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
// cycleCallback() after every cycle. note that the CPU will panic if a CPU
// method is called during a callback.
func (mc *CPU) ExecuteInstruction(cycleCallback func(*InstructionResult)) (*InstructionResult, error) {
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
	result := new(InstructionResult)
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
	defn, found := mc.opCodes[operator]
	if !found {
		if operator == 0xff {
			return nil, errors.GopherError{errors.NullInstruction, nil}
		}
		return nil, errors.GopherError{errors.UnimplementedInstruction, errors.Values{operator}}
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
	// the instruction, not the address). we also take the opportuinity to set
	// the InstructionData value for the StepResult and whether a page fault has
	// occured
	switch defn.AddressingMode {
	case definitions.Implied:
		// phantom read
		// +1 cycle
		_, err := mc.read8Bit(mc.PC.ToUint16())
		if err != nil {
			return nil, err
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

	case definitions.Relative:
		// relative addressing is only used for branch instructions, the address
		// is an offset value from the current PC position

		// +1 cycle
		value, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		result.InstructionData = value
		address = uint16(value)

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

	case definitions.ZeroPage:
		// +1 cycle
		value, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		address = uint16(value)
		result.InstructionData = address

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

		// using 8bit addition because we don't want a page-fault
		adder, err := register.NewAnonymous(mc.X, 8)
		if err != nil {
			return nil, err
		}
		adder.Add(indirectAddress, false)

		// +1 cycle
		mc.endCycle()

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

		adder, err := register.NewAnonymous(mc.Y, 16)
		if err != nil {
			return nil, err
		}
		adder.Add(indexedAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if result.PageFault {
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return nil, err
			}
			result.ActualCycles++
		}

		adder.Add(indexedAddress&0xff00, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

	case definitions.AbsoluteIndexedX:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return nil, err
		}

		adder, err := register.NewAnonymous(mc.X, 16)
		if err != nil {
			return nil, err
		}

		// add index to LSB of address
		adder.Add(indirectAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if result.PageFault {
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return nil, err
			}
		}

		adder.Add(indirectAddress&0xff00, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

	case definitions.AbsoluteIndexedY:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return nil, err
		}

		adder, err := register.NewAnonymous(mc.Y, 16)
		if err != nil {
			return nil, err
		}

		// add index to LSB of address
		adder.Add(indirectAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xFF00 == 0x0100)
		if result.PageFault {
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return nil, err
			}
		}

		adder.Add(indirectAddress&0xff00, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

	case definitions.IndexedZeroPageX:
		// +1 cycles
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return nil, err
		}
		adder, err := register.NewAnonymous(indirectAddress, 8)
		if err != nil {
			return nil, err
		}
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
		adder, err := register.NewAnonymous(indirectAddress, 8)
		if err != nil {
			return nil, err
		}
		adder.Add(mc.Y, false)
		address = adder.ToUint16()
		result.InstructionData = indirectAddress

		// +1 cycle
		mc.endCycle()

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
			if !mc.NoSideEffects {
				err = mc.mem.Write(address, value)

				if err != nil {
					return nil, err
				}
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
		if !mc.NoSideEffects {
			err = mc.mem.Write(mc.SP.ToUint16(), mc.A.ToUint8())
			if err != nil {
				return nil, err
			}
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
		if !mc.NoSideEffects {
			err = mc.mem.Write(mc.SP.ToUint16(), mc.Status.ToUint8())
			if err != nil {
				return nil, err
			}
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
		if !mc.NoSideEffects {
			err = mc.mem.Write(address, mc.A.ToUint8())
			if err != nil {
				return nil, err
			}
		}

	case "STX":
		if !mc.NoSideEffects {
			err = mc.mem.Write(address, mc.X.ToUint8())
			if err != nil {
				return nil, err
			}
		}

	case "STY":
		if !mc.NoSideEffects {
			err = mc.mem.Write(address, mc.Y.ToUint8())
			if err != nil {
				return nil, err
			}
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
			r, err = register.NewAnonymous(value, mc.A.Size())
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
			r, err = register.NewAnonymous(value, mc.A.Size())
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.LSR()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "ADC":
		mc.Status.Carry, mc.Status.Overflow = mc.A.Add(value, mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "SBC":
		mc.Status.Carry, mc.Status.Overflow = mc.A.Subtract(value, mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "ROR":
		var r *register.Register
		if defn.Effect == definitions.RMW {
			r, err = register.NewAnonymous(value, mc.A.Size())
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
			r, err = register.NewAnonymous(value, mc.A.Size())
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.ROL(mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "INC":
		r, err := register.NewAnonymous(value, 8)
		if err != nil {
			return nil, err
		}
		r.Add(1, false)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		value = r.ToUint8()

	case "DEC":
		r, err := register.NewAnonymous(value, 8)
		if err != nil {
			return nil, err
		}
		r.Add(255, false)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		value = r.ToUint8()

	case "CMP":
		cmp, err := register.NewAnonymous(mc.A, mc.A.Size())
		if err != nil {
			return nil, err
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPX":
		cmp, err := register.NewAnonymous(mc.X, mc.X.Size())
		if err != nil {
			return nil, err
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPY":
		cmp, err := register.NewAnonymous(mc.Y, mc.Y.Size())
		if err != nil {
			return nil, err
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "BIT":
		cmp, err := register.NewAnonymous(mc.A, mc.A.Size())
		if err != nil {
			return nil, err
		}
		cmp.AND(value)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()
		mc.Status.Overflow = cmp.IsBitV()

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

		if !mc.NoSideEffects {

			// push MSB of PC onto stack, and decrement SP
			// +1 cycle
			err = mc.mem.Write(mc.SP.ToUint16(), uint8((mc.PC.ToUint16()&0xFF00)>>8))
			if err != nil {
				return nil, err
			}
			mc.SP.Add(255, false)
			mc.endCycle()

			// push LSB of PC onto stack, and decrement SP
			// +1 cycle
			err = mc.mem.Write(mc.SP.ToUint16(), uint8(mc.PC.ToUint16()&0x00FF))
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
			mc.PC.Load(address)

			// store address in theInstructionData field of result
			//
			// we would normally do this in the addressing mode switch above. however,
			// JSR uses absolute addressing and we deliberately do nothing in that
			// switch for 'sub-routine' commands
			result.InstructionData = address
		}

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
		// TODO: implement BRK

	case "RTI":
		// TODO: implement RTI

	default:
		// this should never, ever happen
		log.Fatalf("WTF! unknown mnemonic! (%s)", defn.Mnemonic)
	}

	// write altered value back to memory for RMW instructions
	if defn.Effect == definitions.RMW {
		if !mc.NoSideEffects {
			err = mc.mem.Write(address, value)
			if err != nil {
				return nil, err

			}
		}
		// +1 cycle
		mc.endCycle()
	}

	// consume an extra cycle for the extra memory access for Write instructions
	if defn.Effect == definitions.Write {
		// +1 cycle
		mc.endCycle()
	}

	// finalise result
	result.Final = true

	return result, nil
}
