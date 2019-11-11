package cpu

// TODO: List
// ----------
// !!TODO NMOS indexed addressing extra read when crossing page boundaries
// !!TODO check that NoFlowControl is consistent in its intention
// !!TODO check that all calls to endCycle() occur when they're supposed to

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/register"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
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

	// some operations only need an accumulator
	acc8  *register.Register
	acc16 *register.Register

	mem     memory.CPUBus
	opCodes []*definitions.InstructionDefinition

	// executing is used for sanity checks - to make sure we're not calling CPU
	// functions when we shouldn't
	executing bool

	// cycleCallback is called by endCycle() for additional emulator
	// functionality
	cycleCallback func() error

	// controls whether cpu executes a cycle when it receives a clock tick (pin
	// 3 of the 6507)
	RdyFlg bool

	// last result
	LastResult result.Instruction

	// silently ignore addressing errors unless StrictAddressing is true
	StrictAddressing bool

	// NoFlowControl sets whehter the cpu responds accurately to instructions
	// that affect the flow of the program (branches, JPS, subroutines and
	// interrupts).  we use this in the disassembly package to make sure we
	// reach every part of the program.
	//
	// note that the alteration of flow as a result of bank switching is still
	// possible even if NoFlowControl is true
	NoFlowControl bool
}

// NewCPU is the preferred method of initialisation for the CPU structure
func NewCPU(mem memory.CPUBus) (*CPU, error) {
	var err error

	mc := new(CPU)
	mc.mem = mem

	mc.PC = register.NewRegister(0, 16, "PC")
	mc.A = register.NewRegister(0, 8, "A")
	mc.X = register.NewRegister(0, 8, "X")
	mc.Y = register.NewRegister(0, 8, "Y")
	mc.SP = register.NewRegister(0, 8, "SP")
	mc.Status = NewStatusRegister("SR")

	mc.acc8 = register.NewAnonRegister(0, 8)
	mc.acc16 = register.NewAnonRegister(0, 16)

	mc.opCodes, err = definitions.GetInstructionDefinitions()
	if err != nil {
		return nil, err
	}

	return mc, mc.Reset()
}

func (mc *CPU) String() string {
	return fmt.Sprintf("%s %s %s %s %s %s", mc.PC, mc.A, mc.X, mc.Y, mc.SP, mc.Status)
}

// Reset reinitialises all registers
func (mc *CPU) Reset() error {
	// sanity check
	if mc.executing {
		return errors.New(errors.InvalidOperationMidInstruction, "reset")
	}

	mc.PC.Load(0)
	mc.A.Load(0)
	mc.X.Load(0)
	mc.Y.Load(0)
	mc.SP.Load(255)
	mc.Status.reset()
	mc.Status.Zero = mc.A.IsZero()
	mc.Status.Sign = mc.A.IsNegative()
	mc.Status.InterruptDisable = false
	mc.Status.Break = false
	mc.executing = false
	mc.cycleCallback = nil
	mc.RdyFlg = true

	// not touching StrictAddressing and NoFlowControl

	return nil
}

// LoadPCIndirect loads the contents of indirectAddress into the PC
func (mc *CPU) LoadPCIndirect(indirectAddress uint16) error {
	// sanity check
	if mc.executing {
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
	// sanity check
	if mc.executing {
		return errors.New(errors.InvalidOperationMidInstruction, "load PC")
	}

	mc.PC.Load(directAddress)

	return nil
}

// note that write8Bit, unline read8Bit(), does not call endCycle() this is
// because we need to differentiate between different addressing modes at
// different times.
func (mc *CPU) write8Bit(address uint16, value uint8) error {
	err := mc.mem.Write(address, value)

	if err != nil {
		// don't worry about unwritable addresses (unless strict addressing
		// is on)
		if mc.StrictAddressing || !errors.Is(err, errors.UnwritableAddress) {
			return err
		}
	}

	return nil
}

// note that read8Bit calls endCycle as appropriate
func (mc *CPU) read8Bit(address uint16) (uint8, error) {
	val, err := mc.mem.Read(address)

	if err != nil {
		// don't worry about unreadable addresses (unless strict addressing
		// is on)
		if mc.StrictAddressing || !errors.Is(err, errors.UnreadableAddress) {
			return 0, err
		}
	}

	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	return val, nil
}

// note that read16Bit calls endCycle as appropriate
func (mc *CPU) read16Bit(address uint16) (uint16, error) {
	lo, err := mc.mem.Read(address)
	if err != nil {
		return 0, err
	}
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

	hi, err := mc.mem.Read(address + 1)
	if err != nil {
		return 0, err
	}
	err = mc.endCycle()
	if err != nil {
		return 0, err
	}

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
		return 0, errors.New(errors.ProgramCounterCycled)
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
		return 0, errors.New(errors.ProgramCounterCycled)
	}

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
		mc.LastResult.PageFault = oldPC&0xff00 != mc.PC.ToUint16()&0xff00
		mc.PC.Load(oldPC&0xff00 | mc.PC.ToUint16()&0x00ff)

		// check to see whether branching has crossed a page
		if mc.LastResult.PageFault {
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
			mc.LastResult.PageFault = true
		}
	}

	return nil
}

// endCycle is called at the end of the imaginary CPU cycle. for example,
// reading a byte from memory takes one cycle and so the emulation will
// call endCycle() at that point. ExecuteInstruction() accepts an argument
// cycleCallback which is called by endCycle for additional functionality
func (mc *CPU) endCycle() error {
	mc.LastResult.ActualCycles++
	if mc.cycleCallback == nil {
		return nil
	}
	return mc.cycleCallback()
}

// ExecuteInstruction steps CPU forward one instruction, calling
// cycleCallback() after every cycle
func (mc *CPU) ExecuteInstruction(cycleCallback func() error) error {
	// sanity check
	if mc.executing {
		panic(fmt.Sprintf("can't call cpu.ExecuteInstruction() in the middle of another cpu.ExecuteInstruction()"))
	}

	// update cycle callback
	mc.cycleCallback = cycleCallback

	// do nothing and return nothing if ready flag is false
	if !mc.RdyFlg {
		err := cycleCallback()
		return err
	}

	// prepare new round of results
	mc.LastResult.Address = mc.PC.ToUint16()
	mc.LastResult.Defn = nil
	mc.LastResult.Final = false
	mc.LastResult.ActualCycles = 0
	mc.LastResult.PageFault = false
	mc.LastResult.Bug = ""

	// register end cycle callback
	defer func() {
		mc.executing = false
		mc.cycleCallback = nil
	}()

	var err error

	// read next instruction (end cycle part of read8BitPC)
	// +1 cycle
	operator, err := mc.read8BitPC()
	if err != nil {
		return err
	}
	defn := mc.opCodes[operator]
	if defn == nil {
		// any byte in which the higher nibble has a value which is numerically
		// odd, is an invalid 6502 opcode. this probably means that execution
		// has wandered into data memory - most likely to occur during
		// disassembly.
		if (operator>>4)%2 == 1 {
			return errors.New(errors.InvalidOpcode, fmt.Sprintf("%02x", operator))
		}

		return errors.New(errors.UnimplementedInstruction, operator, mc.PC.ToUint16()-1)
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

	// get address to use when reading/writing from/to memory (note that in the
	// case of immediate addressing, we are actually getting the value to use
	// in the instruction, not the address).
	//
	// we also take the opportunity to set the InstructionData value for the
	// StepResult and whether a page fault has occured. note that we don't do
	// this in the case of JSR
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
				return err
			}
		} else {
			// phantom read
			// +1 cycle
			_, err := mc.read8Bit(mc.PC.ToUint16())
			if err != nil {
				return err
			}
		}

	case definitions.Immediate:
		// for immediate mode, the value is the next byte in the program
		// therefore, we don't set the address and we read the value through the PC

		// +1 cycle
		value, err = mc.read8BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = value

	case definitions.Absolute:
		if defn.Effect != definitions.Subroutine {
			// +2 cycles
			address, err = mc.read16BitPC()
			if err != nil {
				return err
			}
			mc.LastResult.InstructionData = address
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
			return err
		}
		mc.LastResult.InstructionData = value
		address = uint16(value)

	case definitions.ZeroPage:
		// +1 cycle
		value, err := mc.read8BitPC()
		if err != nil {
			return err
		}
		address = uint16(value)
		mc.LastResult.InstructionData = address

	case definitions.IndexedZeroPageX:
		// +1 cycles
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return err
		}

		mc.LastResult.InstructionData = indirectAddress
		mc.acc8.Load(indirectAddress)
		mc.acc8.Add(mc.X, false)
		address = mc.acc8.ToUint16()

		// handle zero page index bug
		if (uint16(indirectAddress)+mc.X.ToUint16())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.Bug = fmt.Sprintf("zero page index bug")
		}

		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case definitions.IndexedZeroPageY:
		// used exclusively for LDX ZeroPage,y

		// +1 cycles
		indirectAddress, err := mc.read8BitPC()
		if err != nil {
			return err
		}

		mc.LastResult.InstructionData = indirectAddress
		mc.acc8.Load(indirectAddress)
		mc.acc8.Add(mc.Y, false)
		address = mc.acc8.ToUint16()

		// handle zero page index bug
		if (uint16(indirectAddress)+mc.Y.ToUint16())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.Bug = fmt.Sprintf("zero page index bug")
		}

		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case definitions.Indirect:
		// indirect addressing (without indexing) is only used for the JMP command

		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

		// handle indirect addressing JMP bug
		if indirectAddress&0x00ff == 0x00ff {
			mc.LastResult.Bug = fmt.Sprintf("indirect addressing bug (JMP bug)")

			lo, err := mc.mem.Read(indirectAddress)
			if err != nil {
				return err
			}

			// +1 cycle
			err = mc.endCycle()
			if err != nil {
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

	case definitions.PreIndexedIndirect: // x indexing
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

		// using 8bit addition because of the 6502's indirect addressing bug -
		// we don't want indexed address t8 extend past the first page
		mc.acc8.Load(mc.X)
		mc.acc8.Add(indirectAddress, false)

		// note whether indirect addressing / page boundary bug has occurred
		if (uint16(indirectAddress)+mc.X.ToUint16())&0xff00 != uint16(indirectAddress)&0xff00 {
			mc.LastResult.Bug = fmt.Sprintf("indirect addressing bug")
		}

		// +2 cycles
		address, err = mc.read16Bit(mc.acc8.ToUint16())
		if err != nil {
			return err
		}

		// never a page fault wth pre-index indirect addressing

	case definitions.PostIndexedIndirect: // y indexing
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

		mc.acc16.Load(mc.Y)
		mc.acc16.Add(indexedAddress&0x00ff, false)
		address = mc.acc16.ToUint16()

		// check for page fault
		if defn.PageSensitive && (address&0xff00 == 0x0100) {
			mc.LastResult.Bug = fmt.Sprintf("indirect addressing bug")
			mc.LastResult.PageFault = true
		}

		if mc.LastResult.PageFault || defn.Effect == definitions.Write || defn.Effect == definitions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indexedAddress&0xff00, false)
		address = mc.acc16.ToUint16()

	case definitions.AbsoluteIndexedX:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

		// add index to LSB of address
		mc.acc16.Load(mc.X)
		mc.acc16.Add(indirectAddress&0x00ff, false)
		address = mc.acc16.ToUint16()

		// check for page fault
		mc.LastResult.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if mc.LastResult.PageFault || defn.Effect == definitions.Write || defn.Effect == definitions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indirectAddress&0xff00, false)
		address = mc.acc16.ToUint16()

	case definitions.AbsoluteIndexedY:
		// +2 cycles
		indirectAddress, err := mc.read16BitPC()
		if err != nil {
			return err
		}
		mc.LastResult.InstructionData = indirectAddress

		// add index to LSB of address
		mc.acc16.Load(mc.Y)
		mc.acc16.Add(indirectAddress&0x00ff, false)
		address = mc.acc16.ToUint16()

		// check for page fault
		mc.LastResult.PageFault = defn.PageSensitive && (address&0xff00 == 0x0100)
		if mc.LastResult.PageFault || defn.Effect == definitions.Write || defn.Effect == definitions.RMW {
			// phantom read (always happends for Write and RMW)
			// +1 cycle
			_, err := mc.read8Bit(address)
			if err != nil {
				return err
			}
		}

		// fix MSB of address
		mc.acc16.Add(indirectAddress&0xff00, false)
		address = mc.acc16.ToUint16()

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
				return err
			}
		} else if defn.Effect == definitions.RMW {
			// +1 cycle
			value, err = mc.read8Bit(address)
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
		err = mc.write8Bit(mc.SP.ToUint16(), mc.A.ToUint8())
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
		value, err = mc.read8Bit(mc.SP.ToUint16())
		if err != nil {
			return err
		}
		mc.A.Load(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "PHP":
		// +1 cycle
		err = mc.write8Bit(mc.SP.ToUint16(), mc.Status.ToUint8())
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
		value, err = mc.read8Bit(mc.SP.ToUint16())
		if err != nil {
			return err
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
		// +1 cycle
		err = mc.write8Bit(address, mc.A.ToUint8())
		if err != nil {
			return err
		}
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "STX":
		// +1 cycle
		err = mc.write8Bit(address, mc.X.ToUint8())
		if err != nil {
			return err
		}
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "STY":
		// +1 cycle
		err = mc.write8Bit(address, mc.Y.ToUint8())
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
		var r *register.Register
		if defn.Effect == definitions.RMW {
			r = mc.acc8
			r.Load(value)
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
			r = mc.acc8
			r.Load(value)
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
			r = mc.acc8
			r.Load(value)
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
			r = mc.acc8
			r.Load(value)
		} else {
			r = mc.A
		}
		mc.Status.Carry = r.ROL(mc.Status.Carry)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "INC":
		r := mc.acc8
		r.Load(value)
		r.Add(1, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "DEC":
		r := mc.acc8
		r.Load(value)
		r.Add(255, false)
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()

	case "CMP":
		cmp := mc.acc8
		cmp.Load(mc.A)

		// maybe surprisingly, CMP can be implemented with binary subtract even
		// if decimal mode is active (the meaning is the same)
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPX":
		cmp := mc.acc8
		cmp.Load(mc.X)
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "CPY":
		cmp := mc.acc8
		cmp.Load(mc.Y)
		mc.Status.Carry, _ = cmp.Subtract(value, true)
		mc.Status.Zero = cmp.IsZero()
		mc.Status.Sign = cmp.IsNegative()

	case "BIT":
		cmp := mc.acc8
		cmp.Load(value)
		mc.Status.Sign = cmp.IsNegative()
		mc.Status.Overflow = cmp.IsBitV()
		cmp.AND(mc.A)
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
		err = mc.write8Bit(mc.SP.ToUint16(), uint8((mc.PC.ToUint16()&0xFF00)>>8))
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
		err = mc.write8Bit(mc.SP.ToUint16(), uint8(mc.PC.ToUint16()&0x00FF))
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
		rtsAddress, err := mc.read16Bit(mc.SP.ToUint16())
		if err != nil {
			return err
		}

		if !mc.NoFlowControl {
			mc.SP.Add(1, false)

			// load and correct PC
			mc.PC.Load(rtsAddress)
			mc.PC.Add(1, false)
		}
		// +1 cycle
		err = mc.endCycle()
		if err != nil {
			return err
		}

	case "BRK":
		// push PC onto register (same effect as JSR)
		err := mc.write8Bit(mc.SP.ToUint16(), uint8((mc.PC.ToUint16()&0xFF00)>>8))
		if err != nil {
			return err
		}
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		err = mc.write8Bit(mc.SP.ToUint16(), uint8(mc.PC.ToUint16()&0x00FF))
		if err != nil {
			return err
		}
		mc.SP.Add(255, false)
		err = mc.endCycle()
		if err != nil {
			return err
		}

		// push status register (same effect as PHP)
		err = mc.write8Bit(mc.SP.ToUint16(), mc.Status.ToUint8())
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

		value, err = mc.read8Bit(mc.SP.ToUint16())
		if err != nil {
			return err
		}
		mc.Status.FromUint8(value)

		// pull program counter (same effect as RTS)
		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
		}

		rtiAddress, err := mc.read16Bit(mc.SP.ToUint16())
		if err != nil {
			return err
		}

		if !mc.NoFlowControl {
			mc.SP.Add(1, false)
			mc.PC.Load(rtiAddress)
			mc.PC.Add(1, false)
		}

	// undocumented instructions

	case "dop":
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
		value = r.ToUint8()

		// ... and compare with the A register
		r.Load(mc.A)
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
		mc.A.Load(mc.X)
		mc.A.AND(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "sax":
		mc.X.AND(mc.A)
		mc.X.Subtract(value, true)
		mc.Status.Zero = mc.X.IsZero()
		mc.Status.Sign = mc.X.IsNegative()

	case "arr":
		mc.A.AND(value)
		mc.Status.Carry = mc.A.ROR(mc.Status.Carry)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	case "slo":
		// the slo opcode starts off with an ASL operation
		// all versions of this opcode are RMW so we always work with
		// the anonymous register
		r := mc.acc8
		r.Load(value)
		mc.Status.Carry = r.ASL()
		mc.Status.Zero = r.IsZero()
		mc.Status.Sign = r.IsNegative()
		value = r.ToUint8()
		mc.A.ORA(value)
		mc.Status.Zero = mc.A.IsZero()
		mc.Status.Sign = mc.A.IsNegative()

	default:
		// this should never, ever happen
		log.Fatalf("WTF! unknown mnemonic! (%s)", defn.Mnemonic)
	}

	// for RMW instructions: write altered value back to memory
	if defn.Effect == definitions.RMW {
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

	return nil
}
