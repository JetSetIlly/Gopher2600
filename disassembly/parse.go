package disassembly

import (
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
)

func (dsm *Disassembly) parseLoop(mc *cpu.CPU) error {
	for {
		currentBank := dsm.Cart.Bank
		ir, err := mc.ExecuteInstruction(func(ir *result.Instruction) {})

		// filter out some errors
		if err != nil {
			switch err := err.(type) {
			case errors.FormattedError:
				switch err.Errno {
				case errors.ProgramCounterCycled:
					return nil
				case errors.UnimplementedInstruction:
					continue // for loop
				case errors.InvalidOpcode:
					continue // for loop
				default:
					return err
				}
			default:
				return err
			}
		}

		// check validity of instruction result
		err = ir.IsValid()
		if err != nil {
			return err
		}

		// if we've seen this before then finish the disassembly
		if dsm.put(currentBank, ir) == false {
			return nil
		}

		// we've disabled flow-control in the cpu but we still need to pay
		// attention to what's going on or we won't get to see all the areas of
		// the ROM.
		switch ir.Defn.Effect {
		case definitions.Flow:
			if ir.Defn.Mnemonic == "JMP" {
				if ir.Defn.AddressingMode == definitions.Indirect {
					if ir.InstructionData.(uint16) > dsm.Cart.Origin() {
						// note current location
						retBank := dsm.Cart.Bank
						retPC := mc.PC.ToUint16()

						// adjust program counter
						mc.LoadPCIndirect(ir.InstructionData.(uint16))

						// recurse
						err = dsm.parseLoop(mc)
						if err != nil {
							return err
						}

						// resume from where we left off
						dsm.Cart.BankSwitch(retBank)
						mc.LoadPC(retPC)
					} else {
						// it's entirely possible for the program to jump
						// outside of cartridge space and run inside RIOT RAM,
						// for instance (test-ane.bin does this for instance).
						//
						// it's difficult to see what we can do in these cases
						// without actually running the program for real (with
						// actual side-effects, that is)
					}
				} else {
					// absolute addressing

					// note current location
					retBank := dsm.Cart.Bank
					retPC := mc.PC.ToUint16()

					// adjust program counter
					mc.LoadPC(ir.InstructionData.(uint16))

					// recurse
					err = dsm.parseLoop(mc)
					if err != nil {
						return err
					}

					// resume from where we left off
					dsm.Cart.BankSwitch(retBank)
					mc.LoadPC(retPC)
				}
			} else {
				// branch instructions

				// note current location
				retBank := dsm.Cart.Bank
				retPC := mc.PC.ToUint16()

				// sign extend address and add to program counter
				address := uint16(ir.InstructionData.(uint8))
				if address&0x0080 == 0x0080 {
					address |= 0xff00
				}
				mc.PC.Add(address, false)

				// recurse
				err = dsm.parseLoop(mc)
				if err != nil {
					return err
				}

				// resume from where we left off
				dsm.Cart.BankSwitch(retBank)
				mc.LoadPC(retPC)
			}
		case definitions.Subroutine:
			if ir.Defn.Mnemonic == "RTS" {
				return nil
			}

			// note current location
			retBank := dsm.Cart.Bank
			retPC := mc.PC.ToUint16()

			// adjust program counter
			mc.LoadPC(ir.InstructionData.(uint16))

			// recurse
			err = dsm.parseLoop(mc)
			if err != nil {
				return err
			}

			// resume from where we left off
			dsm.Cart.BankSwitch(retBank)
			mc.LoadPC(retPC)
		case definitions.Interrupt:
			// do nothing with interrupts
		}
	}
}

// ParseMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control
func (dsm *Disassembly) ParseMemory(cart *memory.Cartridge, symtable *symbols.Table) error {
	dsm.Cart = cart
	dsm.Symtable = symtable
	dsm.program = make([]map[uint16]*result.Instruction, dsm.Cart.NumBanks)

	// create new memory
	mem, err := newMinimalMemory(dsm.Cart)
	if err != nil {
		return err
	}

	// create a new non-branching CPU to disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return err
	}

	mc.NoFlowControl = true

	// allocate memory for disassembly
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		dsm.Cart.BankSwitch(bank)
		dsm.program[bank] = make(map[uint16]*result.Instruction)
	}

	// make sure we're in the starting bank - at the beginning of the
	// disassembly and at the end
	dsm.Cart.BankSwitch(0)
	defer dsm.Cart.BankSwitch(0)

	mc.LoadPCIndirect(memory.AddressReset)

	return dsm.parseLoop(mc)
}
