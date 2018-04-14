package cpu

// TODO List
// ---------
// . NMOS indexed addressing extra read when crossing page boundaries
// . Binary Decimal Mode

import (
	"fmt"
	"headless/hardware/memory"
	"log"
	"runtime"
)

// CPU is the main container structure for the package
type CPU struct {
	PC     Register
	A      Register
	X      Register
	Y      Register
	SP     Register
	Status StatusRegister

	memory  memory.Bus
	opCodes definitionsTable

	// channels communicating success and error of each cycle
	stepResult chan InstructionResult
	stepError  chan error

	// stepNext stops the processor continuing with the instruction execution
	// until we signal with a true (false will cause the execution to halt)
	// we want this so that the execution doesn't race away and change the state
	// of the processor (and possible memory) before we have chance to service
	// the rest of the machine
	stepNext chan bool

	// endCycle is a closure that contains details of the current instruction
	// if it is undefined then no execution is currently being executed
	// (see IsRunning method)
	endCycle func()
}

// NewCPU is the constructor for the CPU type
func NewCPU(memory memory.Bus) *CPU {
	cpu := new(CPU)
	cpu.memory = memory

	cpu.stepResult = make(chan InstructionResult)
	cpu.stepError = make(chan error)
	cpu.stepNext = make(chan bool)

	var err error

	cpu.opCodes, err = getInstructionDefinitions()
	if err != nil {
		log.Fatalln(err)
	}

	cpu.PC = make(Register, 16)
	cpu.A = make(Register, 8)
	cpu.X = make(Register, 8)
	cpu.Y = make(Register, 8)
	cpu.SP = make(Register, 8)
	cpu.Status = *new(StatusRegister)

	cpu.Reset()

	return cpu
}

func (mc *CPU) endStep(result *InstructionResult) {
	result.Final = true
	mc.endCycle = nil
	mc.stepResult <- *result
}

func (mc *CPU) endStepInError(err error) {
	mc.endCycle = nil
	mc.stepError <- err
	runtime.Goexit()
}

// IsRunning is true if CPU is in the middle of executing an instruction
func (mc *CPU) IsRunning() bool {
	return mc.endCycle != nil
}

// Reset reinitialises all registers
func (mc *CPU) Reset() {
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
}

func (mc *CPU) read8Bit(address uint16) uint8 {
	val, err := mc.memory.Read(address)
	if err != nil {
		mc.endStepInError(err)
	}
	mc.endCycle()

	return val
}

func (mc *CPU) read16Bit(address uint16) uint16 {
	lo, err := mc.memory.Read(address)
	if err != nil {
		mc.endStepInError(err)
	}
	mc.endCycle()

	hi, err := mc.memory.Read(address + 1)
	if err != nil {
		mc.endStepInError(err)
	}
	mc.endCycle()

	var val uint16
	val = uint16(hi) << 8
	val |= uint16(lo)

	return val
}

func (mc *CPU) read8BitPC() uint8 {
	op := mc.read8Bit(mc.PC.ToUint16())
	mc.PC.Add(1, false)
	return op
}

func (mc *CPU) read16BitPC() uint16 {
	val := mc.read16Bit(mc.PC.ToUint16())
	mc.PC.Add(2, false)
	return val
}

func (mc *CPU) branch(flag bool, address uint16, result *InstructionResult) {
	if flag == true {
		// note that we've taken the branching path
		result.ActualCycles++

		// phantom read
		// +1 cycle
		mc.read8Bit(mc.PC.ToUint16())

		// note current PC for reference
		oldPC := mc.PC.ToUint16()

		// add LSB to PC
		// +1 cycle
		mc.PC.Add(address&0x00ff, false)

		// check to see whether branching has crossed a page
		if oldPC&0xff00 != mc.PC.ToUint16()&0xff00 {
			// phantom read
			// +1 cycle
			mc.read8Bit(mc.PC.ToUint16())

			// correct program counter
			mc.PC.Add(address&0xff00, false)

			// note that we've triggered a page fault
			result.PageFault = true
			result.ActualCycles++
		}
	}
}

func (mc *CPU) executeInstruction() {
	cont := <-mc.stepNext
	if cont == false {
		mc.endCycle = nil
		runtime.Goexit()
	}

	// prepare StepResult structure
	result := new(InstructionResult)
	result.ProgramCounter = mc.PC.ToUint16()

	// create endCycle function
	mc.endCycle = func() {
		result.ActualCycles++
		mc.stepResult <- *result
		cont := <-mc.stepNext
		if cont == false {
			mc.endCycle = nil
			runtime.Goexit()
		}
	}

	// read next instruction (end cycle part of read8BitPC)
	// +1 cycle
	operator := mc.read8BitPC()
	defn, found := mc.opCodes[operator]
	if !found {
		mc.endStepInError(fmt.Errorf("unimplemented instruction (0x%x)", operator))
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

	var err error

	// get address to use when reading/writing from/to memory (note that in the
	// case of immediate addressing, we are actually getting the value to use in
	// the instruction, not the address). we also take the opportuinity to set
	// the InstructionData value for the StepResult and whether a page fault has
	// occured
	switch defn.AddressingMode {
	case Implied:
		// phantom read
		// +1 cycle
		mc.read8Bit(mc.PC.ToUint16())

	case Immediate:
		// for immediate mode, the value is the next byte in the program
		// therefore, we don't set the address and we read the value through the PC

		// +1 cycle
		value = mc.read8BitPC()
		result.InstructionData = value

	case Relative:
		// relative addressing is only used for branch instructions, the address
		// is an offset value from the current PC position

		// +1 cycle
		address = uint16(mc.read8BitPC())
		result.InstructionData = address

	case Absolute:
		if defn.Effect != Subroutine {
			// +2 cycles
			address = mc.read16BitPC()
			result.InstructionData = address
		}

	case ZeroPage:
		// +1 cycle
		address = uint16(mc.read8BitPC())
		result.InstructionData = address

	case Indirect:
		// indirect addressing (without indexing) is only used for the JMP command

		// +2 cycles
		indirectAddress := mc.read16BitPC()

		// implement NMOS 6502 Indirect JMP bug
		if indirectAddress&0x00ff == 0x00ff {
			lo, err := mc.memory.Read(indirectAddress)
			if err != nil {
				mc.endStepInError(err)
			}

			// +1 cycle
			mc.endCycle()

			hi, err := mc.memory.Read(indirectAddress & 0xff00)
			if err != nil {
				mc.endStepInError(err)
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
			address = mc.read16Bit(indirectAddress)
			result.InstructionData = indirectAddress
		}

	case PreIndexedIndirect:
		// +1 cycle
		indirectAddress := mc.read8BitPC()

		// using 8bit addition because we don't want a page-fault
		adder, err := generateRegister(mc.X, 8)
		if err != nil {
			mc.endStepInError(err)
		}
		adder.Add(indirectAddress, false)

		// +1 cycle
		mc.endCycle()

		// +2 cycles
		address = mc.read16Bit(adder.ToUint16())

		// never a page fault wth pre-index indirect addressing
		result.InstructionData = indirectAddress

	case PostIndexedIndirect:
		// +1 cycle
		indirectAddress := mc.read8BitPC()

		// +2 cycles
		indexedAddress := mc.read16Bit(uint16(indirectAddress))

		adder, err := generateRegister(mc.Y, 16)
		if err != nil {
			mc.endStepInError(err)
		}
		adder.Add(indexedAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xff00 != indexedAddress&0xff00)
		if result.PageFault {
			// phantom read
			// +1 cycle
			mc.read8Bit(address)
			result.ActualCycles++

			adder.Add(indexedAddress&0xff00, false)
			address = adder.ToUint16()
		}

		result.InstructionData = indirectAddress

	case AbsoluteIndexedX:
		// +2 cycles
		indirectAddress := mc.read16BitPC()

		adder, err := generateRegister(mc.X, 16)
		if err != nil {
			mc.endStepInError(err)
		}

		// add index to LSB of address
		adder.Add(indirectAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xff00 != indirectAddress&0xff00)
		if result.PageFault {
			// phantom read
			// +1 cycle
			mc.read8Bit(address)
			result.ActualCycles++

			adder.Add(indirectAddress&0xff00, false)
			address = adder.ToUint16()
		}

		result.InstructionData = indirectAddress

	case AbsoluteIndexedY:
		// +2 cycles
		indirectAddress := mc.read16BitPC()

		adder, err := generateRegister(mc.Y, 16)
		if err != nil {
			mc.endStepInError(err)
		}

		// add index to LSB of address
		adder.Add(indirectAddress&0x00ff, false)
		address = adder.ToUint16()

		// check for page fault
		result.PageFault = defn.PageSensitive && (address&0xFF00 != indirectAddress&0xFF00)
		if result.PageFault {
			// phantom read
			// +1 cycle
			mc.read8Bit(address)
			result.ActualCycles++

			adder.Add(indirectAddress&0xff00, false)
			address = adder.ToUint16()
		}

		result.InstructionData = indirectAddress

	case IndexedZeroPageX:
		// +1 cycles
		indirectAddress := mc.read8BitPC()
		adder, err := generateRegister(indirectAddress, 8)
		if err != nil {
			mc.endStepInError(err)
		}
		adder.Add(mc.X, false)
		address = adder.ToUint16()
		result.InstructionData = indirectAddress

		// +1 cycle
		mc.endCycle()

	case IndexedZeroPageY:
		// used exclusively for LDX ZeroPage,y

		// +1 cycles
		indirectAddress := mc.read8BitPC()
		adder, err := generateRegister(indirectAddress, 8)
		if err != nil {
			mc.endStepInError(err)
		}
		adder.Add(mc.Y, false)
		address = adder.ToUint16()
		result.InstructionData = indirectAddress

		// +1 cycle
		mc.endCycle()

	default:
		log.Printf("unknown addressing mode for %s", defn.Mnemonic)
	}

	// read value from memory using address found in AddressingMode switch above only when:
	// a) addressing mode is not 'implied' or 'immediate'
	//	- for immediate modes, we already have the value in lieu of an address
	//  - for implied modes, we don't need a value
	// b) instruction is 'Read' OR 'ReadWrite'
	//  - for write modes, we only use the address to write a value we already have
	//  - for flow modes, the use of the address is very specific
	if !(defn.AddressingMode == Implied || defn.AddressingMode == Immediate) {
		if defn.Effect == Read {
			// +1 cycle
			value = mc.read8Bit(address)
		} else if defn.Effect == RMW {
			// +1 cycle
			value = mc.read8Bit(address)

			// phantom write
			// +1 cycle
			err = mc.memory.Write(address, value)
			if err != nil {
				mc.endStepInError(err)
			}
			mc.endCycle()
		}
	}

	// actually perform instruction based on mnemonic
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
		err = mc.memory.Write(mc.SP.ToUint16(), mc.A.ToUint8())
		if err != nil {
			mc.endStepInError(err)
		}
		mc.SP.Add(255, false)

	case "PLA":
		// +1 cycle
		mc.SP.Add(1, false)
		mc.endCycle()
		// +1 cycle
		value = mc.read8Bit(mc.SP.ToUint16())
		mc.A.Load(value)

	case "PHP":
		err = mc.memory.Write(mc.SP.ToUint16(), mc.Status.ToUint8())
		if err != nil {
			mc.endStepInError(err)
		}
		mc.SP.Add(255, false)

	case "PLP":
		// +1 cycle
		mc.SP.Add(1, false)
		mc.endCycle()
		// +1 cycle
		value = mc.read8Bit(mc.SP.ToUint16())
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
		err = mc.memory.Write(address, mc.A.ToUint8())
		if err != nil {
			mc.endStepInError(err)
		}

	case "STX":
		err = mc.memory.Write(address, mc.X.ToUint8())
		if err != nil {
			mc.endStepInError(err)
		}

	case "STY":
		err = mc.memory.Write(address, mc.Y.ToUint8())
		if err != nil {
			mc.endStepInError(err)
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
		mc.Status.Carry = mc.A.ASL()
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		if defn.Effect == RMW {
			value = mc.A.ToUint8()
		}

	case "LSR":
		mc.Status.Carry = mc.A.LSR()
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		if defn.Effect == RMW {
			value = mc.A.ToUint8()
		}

	case "ADC":
		mc.Status.Carry, mc.Status.Overflow = mc.A.Add(value, mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "SBC":
		mc.Status.Carry, mc.Status.Overflow = mc.A.Subtract(value, mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "ROR":
		mc.Status.Carry = mc.A.ROR(mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		if defn.Effect == RMW {
			value = mc.A.ToUint8()
		}

	case "ROL":
		mc.Status.Carry = mc.A.ROL(mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		if defn.Effect == RMW {
			value = mc.A.ToUint8()
		}

	case "INC":
		r, err := generateRegister(value, 8)
		if err != nil {
			mc.endStepInError(err)
		}
		r.Add(1, false)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		value = r.ToUint8()

	case "DEC":
		r, err := generateRegister(value, 8)
		if err != nil {
			mc.endStepInError(err)
		}
		r.Add(255, false)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		value = r.ToUint8()

	case "CMP":
		cmp, err := generateRegister(&mc.A, len(mc.A))
		if err != nil {
			mc.endStepInError(err)
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPX":
		cmp, err := generateRegister(&mc.X, len(mc.X))
		if err != nil {
			mc.endStepInError(err)
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPY":
		cmp, err := generateRegister(&mc.Y, len(mc.Y))
		if err != nil {
			mc.endStepInError(err)
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "BIT":
		cmp, err := generateRegister(&mc.A, len(mc.A))
		if err != nil {
			mc.endStepInError(err)
		}
		cmp.AND(value)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()
		mc.Status.Overflow = bool(cmp[1])

	case "JMP":
		mc.PC.Load(address)

	case "BCC":
		mc.branch(!mc.Status.Carry, address, result)

	case "BCS":
		mc.branch(mc.Status.Carry, address, result)

	case "BEQ":
		mc.branch(mc.Status.Zero, address, result)

	case "BMI":
		mc.branch(mc.Status.Sign, address, result)

	case "BNE":
		mc.branch(!mc.Status.Zero, address, result)

	case "BPL":
		mc.branch(!mc.Status.Sign, address, result)

	case "BVC":
		mc.branch(!mc.Status.Overflow, address, result)

	case "BVS":
		mc.branch(mc.Status.Overflow, address, result)

	case "JSR":
		// +1 cycle
		lsb := mc.read8BitPC()

		// the current value of the PC is now correct, even though we've only read
		// one byte of the address so far. remember, RTS increments the PC when
		// read from the stack, meaning that the PC will be correct at that point

		// with that in mind, we're not sure what this extra cycle is for
		// +1 cycle
		mc.endCycle()

		// push MSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.memory.Write(mc.SP.ToUint16(), uint8((mc.PC.ToUint16()&0xFF00)>>8))
		if err != nil {
			mc.endStepInError(err)
		}
		mc.SP.Add(255, false)
		mc.endCycle()

		// push LSB of PC onto stack, and decrement SP
		// +1 cycle
		err = mc.memory.Write(mc.SP.ToUint16(), uint8(mc.PC.ToUint16()&0x00FF))
		if err != nil {
			mc.endStepInError(err)
		}
		mc.SP.Add(255, false)
		mc.endCycle()

		// perform jump
		msb := mc.read8BitPC()
		address = (uint16(msb) << 8) | uint16(lsb)
		mc.PC.Load(address)

		// we would normally store the InstructionData in the addressing mode
		// switch but JSR bypasses all that so we'll do it here
		result.InstructionData = address

	case "RTS":
		// +1 cycle
		mc.SP.Add(1, false)
		mc.endCycle()

		// +2 cycles
		rtsAddress := mc.read16Bit(mc.SP.ToUint16())
		mc.SP.Add(1, false)

		// load and correct PC
		mc.PC.Load(rtsAddress)
		mc.PC.Add(1, false)

		// +1 cycle
		mc.endCycle()

	case "BRK":
		// TODO: implement BRK

	case "RTI":
		// TODO: implement RTI

	default:
		// this should never, ever happen
		log.Fatalf("WTF! unknown mnemonic! (%s)", defn.Mnemonic)
	}

	// write altered value back to memory for RMW instructions
	if defn.Effect == RMW {
		err = mc.memory.Write(address, value)
		if err != nil {
			mc.endStepInError(err)

		}
		// +1 cycle
		mc.endCycle()
	}

	// consume an extra cycle for the extra memory access for Write instructions
	if defn.Effect == Write {
		// +1 cycle
		mc.endCycle()
	}

	mc.endStep(result)
}
