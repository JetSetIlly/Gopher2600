package cpu

// TODO List
// ---------
// . concurrency: yield control back to the clock manager after every cycle
// . NMOS indexed addressing extra read when crossing page boundaries
// . Binary Decimal Mode

import (
	"fmt"
	"headless/hardware/memory"
	"log"
)

// CPU is the main container structure for the package
type CPU struct {
	PC     Register
	A      Register
	X      Register
	Y      Register
	SP     Register
	Status StatusRegister

	memory memory.Memory

	opCodes definitionsTable
}

// NewCPU is the constructor for the CPU type
func NewCPU(memory memory.Memory) *CPU {
	mc := new(CPU)
	mc.memory = memory

	var err error

	mc.opCodes, err = getInstructionDefinitions()
	if err != nil {
		log.Fatalln(err)
	}

	mc.PC = make(Register, 16)
	mc.A = make(Register, 8)
	mc.X = make(Register, 8)
	mc.Y = make(Register, 8)
	mc.SP = make(Register, 8)
	mc.Status = *new(StatusRegister)

	mc.Reset()

	return mc
}

// Reset reinitialises all registers
func (mc *CPU) Reset() {
	mc.PC.Load(0)
	mc.A.Load(0)
	mc.X.Load(0)
	mc.Y.Load(0)
	mc.SP.Load(255)
	mc.Status.FromUint8(0)
	mc.Status.Zero = mc.A.IsZero()
	mc.Status.Sign = mc.A.IsNegative()
	mc.Status.InterruptDisable = true
	mc.Status.Break = true
}

func (mc *CPU) read8Bit(address uint16) uint8 {
	val, err := mc.memory.Read(address)
	if err != nil {
		log.Fatalln(err)
	}

	return val
}

func (mc *CPU) read16Bit(address uint16) uint16 {
	lo, err := mc.memory.Read(address)
	if err != nil {
		log.Fatalln(err)
	}

	hi, err := mc.memory.Read(address + 1)
	if err != nil {
		log.Fatalln(err)
	}

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

// Step executes the next instruction in the program
func (mc *CPU) Step() (*StepResult, error) {
	// read next instruction
	operator := mc.read8BitPC()
	defn, found := mc.opCodes[operator]
	if !found {
		return nil, fmt.Errorf("unimplemented instruction (0x%x)", operator)
	}

	var address uint16
	var value uint8
	var err error

	// prepare StepResult structure
	result := new(StepResult)
	result.ProgramCounter = mc.PC.ToUint16()
	result.Defn = defn
	result.ActualCycles = defn.Cycles

	// get address to use when reading/writing from/to memory (note that in the
	// case of immediate addressing, we are actually getting the value to use in
	// the instruction, not the address). we also take the opportuinity to set
	// the InstructionData value for the StepResult and whether a page fault has
	// occured
	switch defn.AddressingMode {
	case Implied:
		// do nothing for implied addressing

	case Immediate:
		// for immediate mode, the value is the next byte in the program
		// therefore, we don't set the address and we read the value through the PC
		value = mc.read8BitPC()
		result.InstructionData = value

	case Relative:
		// relative addressing is only used for branch instructions, the address
		// is an offset value from the current PC position
		address = uint16(mc.read8BitPC())
		result.InstructionData = address

	case Absolute:
		address = mc.read16BitPC()
		result.InstructionData = address

	case ZeroPage:
		address = uint16(mc.read8BitPC())
		result.InstructionData = address

	case Indirect:
		// indirect addressing (without indexing) is only used for the JMP command
		indirectAddress := mc.read16BitPC()

		// implement NMOS 6502 Indirect JMP bug
		if indirectAddress&0x00ff == 0x00ff {
			lo, err := mc.memory.Read(indirectAddress)
			if err != nil {
				log.Fatalln(err)
			}
			hi, err := mc.memory.Read(indirectAddress & 0xff00)
			if err != nil {
				log.Fatalln(err)
			}
			address = uint16(hi) << 8
			address |= uint16(lo)

			result.InstructionData = indirectAddress
			result.Bug = fmt.Sprintf("Indirect JMP Bug")

		} else {
			// normal, non-buggy behaviour
			address = mc.read16Bit(indirectAddress)
			result.InstructionData = indirectAddress
		}

	case PreIndexedIndirect:
		indirectAddress := mc.read8BitPC()
		adder, err := generateRegister(indirectAddress, 8)
		if err != nil {
			log.Fatalln(err)
		}
		adder.Add(mc.X, false)
		address = mc.read16Bit(adder.ToUint16())

		result.InstructionData = indirectAddress
		// never a page fault wth pre-index indirect addressing because the we only
		// ever read from the first page - we discard any carry from the addition
		// and allow the indexing to "wrap around"

	case PostIndexedIndirect:
		indirectAddress := mc.read8BitPC()
		indexedAddress := mc.read16Bit(uint16(indirectAddress))
		adder, err := generateRegister(indexedAddress, 16)
		if err != nil {
			log.Fatalln(err)
		}
		adder.Add(mc.Y, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress
		result.PageFault = defn.PageSensitive && (address&0xFF00 != indexedAddress&0xFF00)

	case AbsoluteIndexedX:
		indirectAddress := mc.read16BitPC()
		adder, err := generateRegister(indirectAddress, 16)
		if err != nil {
			log.Fatalln(err)
		}
		adder.Add(mc.X, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress
		result.PageFault = defn.PageSensitive && (address&0xFF00 != indirectAddress&0xFF00)

	case AbsoluteIndexedY:
		indirectAddress := mc.read16BitPC()
		adder, err := generateRegister(indirectAddress, 16)
		if err != nil {
			log.Fatalln(err)
		}
		adder.Add(mc.Y, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress
		result.PageFault = defn.PageSensitive && (address&0xFF00 != indirectAddress&0xFF00)

	case IndexedZeroPageX:
		indirectAddress := mc.read8BitPC()
		adder, err := generateRegister(indirectAddress, 8)
		if err != nil {
			log.Fatalln(err)
		}
		adder.Add(mc.X, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

	case IndexedZeroPageY:
		// used exclusively for LDX ZeroPage,y
		indirectAddress := mc.read8BitPC()
		adder, err := generateRegister(indirectAddress, 8)
		if err != nil {
			log.Fatalln(err)
		}
		adder.Add(mc.Y, false)
		address = adder.ToUint16()

		result.InstructionData = indirectAddress

	default:
		log.Printf("unknown addressing mode for %s", defn.Mnemonic)
	}

	// adjust number of cycles used if there has been a page fault
	if result.PageFault {
		result.ActualCycles++
	}

	// read value from memory using address found in AddressingMode switch above only when:
	// a) addressing mode is not 'implied' or 'immediate'
	//	- for immediate modes, we already have the value in lieu of an address
	//  - for implied modes, we don't need a value
	// b) instruction is 'Read' OR 'ReadWrite'
	//  - for write modes, we only use the address to write a value we already have
	//  - for flow modes, the use of the address is very specific
	if !(defn.AddressingMode == Implied || defn.AddressingMode == Immediate) {
		if defn.Effect == Read || defn.Effect == RMW {
			value = mc.read8Bit(address)
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
			log.Fatalln(err)
		}
		mc.SP.Add(255, false)

	case "PLA":
		mc.SP.Add(1, false)
		value = mc.read8Bit(mc.SP.ToUint16())
		mc.A.Load(value)

	case "PHP":
		err = mc.memory.Write(mc.SP.ToUint16(), mc.Status.ToUint8())
		if err != nil {
			log.Fatalln(err)
		}
		mc.SP.Add(255, false)

	case "PLP":
		mc.SP.Add(1, false)
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
			log.Fatalln(err)
		}

	case "STX":
		err = mc.memory.Write(address, mc.X.ToUint8())
		if err != nil {
			log.Fatalln(err)
		}

	case "STY":
		err = mc.memory.Write(address, mc.Y.ToUint8())
		if err != nil {
			log.Fatalln(err)
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

	case "LSR":
		mc.Status.Carry = mc.A.LSR()
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

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

	case "ROL":
		mc.Status.Carry = mc.A.ROL(mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "INC":
		r, err := generateRegister(value, 8)
		if err != nil {
			log.Fatalln(err)
		}
		r.Add(1, false)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		err = mc.memory.Write(address, r.ToUint8())
		if err != nil {
			log.Fatalln(err)
		}

	case "DEC":
		r, err := generateRegister(value, 8)
		if err != nil {
			log.Fatalln(err)
		}
		r.Add(255, false)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()
		err = mc.memory.Write(address, r.ToUint8())
		if err != nil {
			log.Fatalln(err)
		}

	case "CMP":
		cmp, err := generateRegister(&mc.A, len(mc.A))
		if err != nil {
			log.Fatalln(err)
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPX":
		cmp, err := generateRegister(&mc.X, len(mc.X))
		if err != nil {
			log.Fatalln(err)
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPY":
		cmp, err := generateRegister(&mc.Y, len(mc.Y))
		if err != nil {
			log.Fatalln(err)
		}
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "BIT":
		cmp, err := generateRegister(&mc.A, len(mc.A))
		if err != nil {
			log.Fatalln(err)
		}
		cmp.AND(value)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()
		mc.Status.Overflow = bool(cmp[1])

	case "JMP":
		mc.PC.Load(address)

	case "BCC":
		if mc.Status.Carry == false {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "BCS":
		if mc.Status.Carry == true {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "BEQ":
		if mc.Status.Zero == true {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "BMI":
		if mc.Status.Sign == true {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "BNE":
		if mc.Status.Zero == false {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "BPL":
		if mc.Status.Sign == false {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "BVC":
		if mc.Status.Overflow == false {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "BVS":
		if mc.Status.Overflow == true {
			mc.PC.Add(address, false)
			result.ActualCycles++
		}

	case "JSR":
		rtsAddress, err := generateRegister(&mc.PC, len(mc.PC))
		if err != nil {
			log.Fatalln(err)
		}
		rtsAddress.Add(65535, false)
		v := rtsAddress.ToUint16()
		err = mc.memory.Write(mc.SP.ToUint16(), uint8((v&0xFF00)>>8))
		if err != nil {
			log.Fatalln(err)
		}
		mc.SP.Add(255, false)
		err = mc.memory.Write(mc.SP.ToUint16(), uint8(v&0x00FF))
		if err != nil {
			log.Fatalln(err)
		}
		mc.SP.Add(255, false)
		mc.PC.Load(address)

	case "RTS":
		mc.SP.Add(1, false)
		rtsAddress := mc.read16Bit(mc.SP.ToUint16())
		mc.SP.Add(1, false)
		mc.PC.Load(rtsAddress)
		mc.PC.Add(1, false)

	case "BRK":
		// TODO: implement BRK

	case "RTI":
		// TODO: implement RTI

	default:
		// this should never, ever happen
		log.Fatalf("WTF! unknown mnemonic! (%s)", defn.Mnemonic)
	}

	return result, nil
}
