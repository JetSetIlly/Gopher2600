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

package disassembly

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/instructions"
	"gopher2600/hardware/memory/memorymap"
	"strings"
)

// Analysis (best effort) of the cartridge
type Analysis struct {
	// discovered/inferred cartridge attributes
	ExecuteFromRAM bool
	Interrupts     bool
	ForcedRTS      bool
}

// Analysis returns a summary of anything interesting found during disassembly.
func (ana Analysis) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("Execute from RAM: %v\n", ana.ExecuteFromRAM))
	s.WriteString(fmt.Sprintf("Interrupts: %v\n", ana.Interrupts))
	s.WriteString(fmt.Sprintf("Forced RTS: %v\n", ana.ForcedRTS))
	return s.String()
}

func (dsm *Disassembly) flowAnalysis(mc *cpu.CPU, flowedFrom uint16, subroutineDepth int) error {
	for {
		// get bank now before executing instruction. the bank may change
		// as a result of the instruction execution. we cannot stop this withe
		// the CPU's NoFlow mechanism
		bank := dsm.cart.GetBank(mc.PC.Address())

		// exeute the instruction
		err := mc.ExecuteInstruction(nil)

		// filter out the predictable errors
		if err != nil {
			if !errors.IsAny(err) {
				return err
			}

			switch err.(errors.AtariError).Message {
			case errors.ProgramCounterCycled:
				// originally, a cycled program counter caused the disassembly
				// to end but thinking about it a bit more, we can see that
				// simply continuing with the loop makes more sense
				continue // for loop
			case errors.UnimplementedInstruction:
				continue // for loop
			default:
				return err
			}
		}

		// fail on invalid results
		if err := mc.LastResult.IsValid(); err != nil {
			return err
		}

		// check that the instruction was executed from cartridge space
		_, area := memorymap.MapAddress(mc.LastResult.Address, true)
		if area != memorymap.Cartridge {
			if area == memorymap.RAM {
				dsm.Analysis.ExecuteFromRAM = true
			} else {
				// executing from somewhere other than cartridge or RMA
				// seems very serious to me. I suppose it's possible so there's
				// no reason to panic() but maybe it should be noted in
				// someway.
			}
			return nil
		}

		// if we've seen this before but it was not from then flow pass then
		// finish the disassembly and flowedFrom is zero
		d := dsm.Entries[bank][mc.LastResult.Address&memorymap.AddressMaskCart]
		if d != nil && d.Type == EntryTypeAnalysis {
			return nil
		}

		// create new disassembly entry
		d, err = dsm.FormatResult(mc.LastResult)
		if err != nil {
			return err
		}

		// add bank information
		d.Bank = bank

		// indicate that it was generated from the flow pass
		d.Type = EntryTypeAnalysis

		// indicate the instruction address from which the new instruction was
		// Jumped/Branched.
		if flowedFrom != 0 {
			d.Prev = append(d.Prev, flowedFrom)
			flowedFrom = 0
		}

		// updated. we need to do it before any jumping/branching because the
		// information needs to be there for the iterated function (loop
		// detection)
		//
		// in the event that a jump/branch has been encountered we update the
		// entry again after appending the next address
		//
		// !!TODO: what happens if LastResult address is not in cart memory?
		dsm.Entries[bank][mc.LastResult.Address&memorymap.AddressMaskCart] = d
		e := &d

		// we've disabled flow-control in the cpu but we still need to pay
		// attention to what's going on or we won't get to see all the areas of
		// the ROM.
		switch mc.LastResult.Defn.Effect {

		case instructions.Flow:
			if mc.LastResult.Defn.Mnemonic == "JMP" {
				if mc.LastResult.Defn.AddressingMode == instructions.Indirect {
					// note current location
					state := dsm.cart.SaveState()
					retPC := mc.PC.Address()

					// adjust program counter
					mc.LoadPCIndirect(mc.LastResult.InstructionData.(uint16))

					// record next address
					(*e).Next = append((*e).Next, mc.PC.Address())

					// recurse
					err = dsm.flowAnalysis(mc, mc.LastResult.Address, subroutineDepth)
					if err != nil {
						return err
					}

					// resume from where we left off
					dsm.cart.RestoreState(state)
					mc.PC.Load(retPC)
				} else {
					// absolute JMP addressing

					// note current location
					state := dsm.cart.SaveState()
					retPC := mc.PC.Address()

					// adjust program counter
					mc.PC.Load(mc.LastResult.InstructionData.(uint16))
					dsm.Entries[bank][mc.LastResult.Address&memorymap.AddressMaskCart] = d

					// record next address
					(*e).Next = append((*e).Next, mc.PC.Address())

					// recurse
					err = dsm.flowAnalysis(mc, mc.LastResult.Address, subroutineDepth)
					if err != nil {
						return err
					}

					// resume from where we left off
					dsm.cart.RestoreState(state)
					mc.PC.Load(retPC)
				}
			} else {
				// branch instructions

				// note current location
				state := dsm.cart.SaveState()
				retPC := mc.PC.Address()

				// sign extend address and add to program counter
				address := uint16(mc.LastResult.InstructionData.(uint8))
				if address&0x0080 == 0x0080 {
					address |= 0xff00
				}
				mc.PC.Add(address)

				// record next address
				(*e).Next = append((*e).Next, mc.PC.Address())

				// recurse
				err = dsm.flowAnalysis(mc, mc.LastResult.Address, subroutineDepth)
				if err != nil {
					return err
				}

				// resume from where we left off
				dsm.cart.RestoreState(state)
				mc.PC.Load(retPC)
			}

		case instructions.Subroutine:
			if mc.LastResult.Defn.Mnemonic == "RTS" {
				// sometimes, a ROM will call RTS despite never having called
				// JSR. in these instances, the ROM has probably stuffed the
				// stack manually with a return address. this disassembly
				// routine currently doesn't handle these instances.
				//
				// Krull does this. one of the very first things it does at
				// address 0xb038 (bank 0) is load the stack with a return
				// address. the first time the "extra" RTS occurs is at 0xb0ad
				if subroutineDepth == 0 {
					dsm.Analysis.ForcedRTS = true
				}
				return nil
			}

			// note current location
			retPC := mc.PC.Address()

			// adjust program counter
			mc.PC.Load(mc.LastResult.InstructionData.(uint16))

			// record next address
			(*e).Next = append((*e).Next, mc.PC.Address())

			// recurse
			err = dsm.flowAnalysis(mc, mc.LastResult.Address, subroutineDepth+1)
			if err != nil {
				return err
			}

			// resume from where we left off
			mc.PC.Load(retPC)

			// subroutines don't care about cartridge banks
			// -- if we JSR in bank 0 and RTS in bank 1 then that execution
			// will continue in bank 1. that's expected VCS behaviour.

		case instructions.Interrupt:
			// do nothing with interrupts
			dsm.Analysis.Interrupts = true
		}
	}
}
