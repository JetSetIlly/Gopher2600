package disassembly

import (
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/result"
)

func (dsm *Disassembly) runLoop(mc *cpu.CPU) error {
	for {
		currentBank := dsm.Cart.Bank
		ir, err := mc.ExecuteInstruction(func(ir *result.Instruction) {})

		// filter out some errors
		if err != nil {
			switch err := err.(type) {
			case errors.FormattedError:
				switch err.Errno {
				case errors.ProgramCounterCycled:
					// originally, a cycled program counter caused the
					// disassembly to end but thinking about it a bit more,
					// we can see that simply continuing with the loop makes
					// more sense
					continue // for loop
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
						err = dsm.runLoop(mc)
						if err != nil {
							return err
						}

						// resume from where we left off
						dsm.Cart.BankSwitch(retBank)
						mc.LoadPC(retPC)
					} else {
						// it's entirely possible for the program to jump
						// outside of cartridge space and run inside RIOT RAM
						// (for instance, test-ane.bin does this).
						//
						// it's difficult to see what we can do in these cases
						// without actually running the program for real (with
						// actual side-effects)
						//
						// for now, we'll just tolerate it
						dsm.selfModifyingCode = true
					}
				} else {
					// absolute JMP addressing

					// note current location
					retBank := dsm.Cart.Bank
					retPC := mc.PC.ToUint16()

					// adjust program counter
					mc.LoadPC(ir.InstructionData.(uint16))

					// recurse
					err = dsm.runLoop(mc)
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
				err = dsm.runLoop(mc)
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
			retPC := mc.PC.ToUint16()

			// adjust program counter
			mc.LoadPC(ir.InstructionData.(uint16))

			// recurse
			err = dsm.runLoop(mc)
			if err != nil {
				return err
			}

			// resume from where we left off
			mc.LoadPC(retPC)

			// subroutines don't care about cartridge banks
			// -- if we JSR in bank 0 and RTS in bank 1 then that execution
			// will continue in bank 1. that's expected VCS behaviour.

		case definitions.Interrupt:
			// do nothing with interrupts
			dsm.interrupts = true
		}
	}
}
