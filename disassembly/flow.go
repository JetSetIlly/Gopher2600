package disassembly

import (
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/result"
)

// flowDisassembly decodes those cartridge addresses that follow the flow from
// the address pointed to by the reset address of the cartridge.
//
// every branch and subroutine is considered. however, it is possible for real
// execution of the ROM to reach places not considered by the flow disassembly.
// for example:
//
//		o addresses stuffed into the stack and RTS being called, without an
//			explicit JSR
//		o branching of jumping to non-cartridge memory. (ie. RAM) and executing
//			code there. self-modifying code.

func (dsm *Disassembly) flowDisassembly(mc *cpu.CPU) error {
	for {
		err := mc.ExecuteInstruction(nil)

		// filter out the predictable errors
		if err != nil {
			if !errors.IsAny(err) {
				return err
			}

			switch err.(errors.AtariError).Errno {
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
		}

		// check validity of instruction result
		err = mc.LastResult.IsValid()
		if err != nil {
			return err
		}

		bank := dsm.Cart.GetBank(mc.LastResult.Address)

		// if we've seen this before then finish the disassembly
		if dsm.flow[bank][mc.LastResult.Address&disasmMask].IsInstruction() {
			return nil
		}

		dsm.flow[bank][mc.LastResult.Address&disasmMask] = Entry{
			style:                 result.StyleDisasm,
			instruction:           mc.LastResult.GetString(dsm.Symtable, result.StyleDisasm),
			instructionDefinition: mc.LastResult.Defn}

		// we've disabled flow-control in the cpu but we still need to pay
		// attention to what's going on or we won't get to see all the areas of
		// the ROM.
		switch mc.LastResult.Defn.Effect {

		case definitions.Flow:
			if mc.LastResult.Defn.Mnemonic == "JMP" {
				if mc.LastResult.Defn.AddressingMode == definitions.Indirect {
					if mc.LastResult.InstructionData.(uint16) > dsm.Cart.Origin() {
						// note current location
						state := dsm.Cart.SaveState()
						retPC := mc.PC.Address()

						// adjust program counter
						mc.LoadPCIndirect(mc.LastResult.InstructionData.(uint16))

						// recurse
						err = dsm.flowDisassembly(mc)
						if err != nil {
							return err
						}

						// resume from where we left off
						dsm.Cart.RestoreState(state)
						mc.PC.Load(retPC)
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
					state := dsm.Cart.SaveState()
					retPC := mc.PC.Address()

					// adjust program counter
					mc.PC.Load(mc.LastResult.InstructionData.(uint16))

					// recurse
					err = dsm.flowDisassembly(mc)
					if err != nil {
						return err
					}

					// resume from where we left off
					dsm.Cart.RestoreState(state)
					mc.PC.Load(retPC)
				}
			} else {
				// branch instructions

				// note current location
				state := dsm.Cart.SaveState()
				retPC := mc.PC.Address()

				// sign extend address and add to program counter
				address := uint16(mc.LastResult.InstructionData.(uint8))
				if address&0x0080 == 0x0080 {
					address |= 0xff00
				}
				mc.PC.Add(address)

				// recurse
				err = dsm.flowDisassembly(mc)
				if err != nil {
					return err
				}

				// resume from where we left off
				dsm.Cart.RestoreState(state)
				mc.PC.Load(retPC)
			}

		case definitions.Subroutine:
			if mc.LastResult.Defn.Mnemonic == "RTS" {
				// sometimes, a ROM will call RTS despite never having called
				// JSR. in these instances, the ROM has probably stuffed the
				// stack manually with a return address. this disassembly
				// routine currently doesn't handle these instances.
				//
				// Krull does this. one of the very first things it does at
				// address 0xb038 (bank 0) is load the stack with a return
				// address. the first time the "extra" RTS occurs is at 0xb0ad
				dsm.forcedRTS = true
				return nil
			}

			// note current location
			retPC := mc.PC.Address()

			// adjust program counter
			mc.PC.Load(mc.LastResult.InstructionData.(uint16))

			// recurse
			err = dsm.flowDisassembly(mc)
			if err != nil {
				return err
			}

			// resume from where we left off
			mc.PC.Load(retPC)

			// subroutines don't care about cartridge banks
			// -- if we JSR in bank 0 and RTS in bank 1 then that execution
			// will continue in bank 1. that's expected VCS behaviour.

		case definitions.Interrupt:
			// do nothing with interrupts
			dsm.interrupts = true
		}
	}
}
