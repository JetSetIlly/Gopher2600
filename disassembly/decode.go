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
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

func (dsm *Disassembly) disassemble(mc *cpu.CPU) error {
	// basic decoding pass
	err := dsm.decode(mc)
	if err != nil {
		return err
	}

	// reinitialise cartridge
	dsm.cart.Initialise()

	// bless those entries which we're reasonably sure are real instructions
	err = dsm.bless(mc)
	if err != nil {
		return err
	}

	return nil
}

func (dsm *Disassembly) bless(mc *cpu.CPU) error {
	// list of start points for every bank
	blessings := make([][]uint16, len(dsm.disasm))
	for b := range dsm.disasm {
		blessings[b] = make([]uint16, 0)
	}

	// get start bank for cartridge before we do anything else
	mc.Reset(false)
	err := mc.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return err
	}
	bank := dsm.cart.GetBank(mc.PC.Value())
	if bank.IsRAM {
		return nil
	}

	// get the reset address contained in the reset vector of the active bank
	blessings[bank.Number] = append(blessings[bank.Number], mc.PC.Value()&memorymap.CartridgeBits)

	// loop through every bank in the cartridge and collate a list of blessings
	// for the bank. deliberately not using IterateCart for this.
	for b := range dsm.disasm {
		bitr, _, err := dsm.NewBankIteration(EntryLevelDecoded, b)
		if err != nil {
			return err
		}

		for _, e := bitr.Start(); e != nil; _, e = bitr.Next() {

			// if instruciton is a JMP or JSR then take the jump address to be a
			// blessing and add it to the list
			if e.Result.Defn.Mnemonic == "JMP" || e.Result.Defn.Mnemonic == "JSR" {
				blessings[b] = append(blessings[b], e.Result.InstructionData&memorymap.CartridgeBits)
			}

			// if instruction is a branch then add the address of the
			// successful branch to the blessing list
			if e.Result.Defn.IsBranch() {
				mc.PC.Load(e.Result.Address)
				mc.PC.Add(uint16(e.Result.Defn.Bytes))
				operand := e.Result.InstructionData
				if operand&0x0080 == 0x0080 {
					operand |= 0xff00
				}
				mc.PC.Add(operand)

				blessings[b] = append(blessings[b], mc.PC.Value()&memorymap.CartridgeBits)
			}
		}
	}

	// blessing can happen at the same time as iteration which is probably
	// being run from a different goroutine. acknowledge the critical section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// now that we have a list of blessings, we'll performa a linear traversal
	// and bless each instruction in sequence, starting at each blessing point.
	//
	// we only bless instructions that naturally follow on from the previous
	// instruction. we also stop when a significant flow control event or
	// interrupt has occurred
	for b := range blessings {
		for _, a := range blessings[b] {
			for a < uint16(len(dsm.disasm[b])) && dsm.disasm[b][a].Level != EntryLevelBlessed {
				if dsm.disasm[b][a].Level == EntryLevelNotMappable {
					a++
					continue
				}

				// if mnemonic is unknown than end the sequence.
				// !TODO: remove this check once every opcode is defined/implemented
				mnemonic := dsm.disasm[b][a].Result.Defn.Mnemonic
				if mnemonic == "??" {
					break
				}

				// promote the entry
				dsm.disasm[b][a].Level = EntryLevelBlessed

				// not breaking on JSR because the sequence will continue if
				// the jumped-to sequence has an RTS (which it probably will
				// have)
				if mnemonic == "JMP" || mnemonic == "RTS" || mnemonic == "BRK" {
					break
				}

				a += uint16(dsm.disasm[b][a].Result.ByteCount)

				// break if address has looped around. while this is possible
				// I'm not allowing it unless I can find an example of it being
				// used in actuality.
				if a > a&memorymap.CartridgeBits {
					break
				}
			}
		}
	}

	return nil
}

func (dsm *Disassembly) decode(mc *cpu.CPU) error {
	// make sure cpu is in initial state
	mc.Reset(false)

	// decoding can happen at the same time as iteration which is probably
	// being run from a different goroutine. acknowledge the critical section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	for b := range dsm.disasm {
		mc.Reset(false)

		for seg := 0; seg < dsm.numSegments; seg++ {
			// origin and memtop
			origin := dsm.mirrorOrigin + (uint16(seg) * dsm.segmentSize)
			memtop := origin + dsm.segmentBits

			// by default the EntryLevel we'll use in the decoding is
			// EntryLevelDecoded. we'll flip this if necessary
			entryLevel := EntryLevelDecoded

			// set bank every iteration because of segmented cartridge memory
			if err := dsm.cart.SetBank(origin, b); err != nil {
				if !errors.Is(err, errors.CartridgeNotMappable) {
					return err
				}

				// we tried setting the bank for the address but couldn't do
				// it, so we flip the entryLevel to NotMappable. the traversal
				// of the bank will continue in the normal way.
				//
				// we could just continue the seg loop and not bother with the
				// reset of the loop. however we would then be left with a
				// sparse array which would complicate iteration. it does mean
				// we'll have entries in the array that are not valid in the
				// strictest sense but with the EntryLevelNotMappable group the
				// method is easier/safer and reasonably clear
				entryLevel = EntryLevelNotMappable
			}

			// we're using uint16 for addresses so if memtop is defined to be
			// 0xffff the clause address <= memtop will always be true. the
			// additional clause detects the overflow condition
			for address := origin; address <= memtop && address >= origin; address++ {
				// execute instruction at address
				mc.PC.Load(address)
				err := mc.ExecuteInstruction(nil)

				unimplementedInstruction := errors.Is(err, errors.UnimplementedInstruction)
				programCounterCycled := errors.Is(err, errors.ProgramCounterCycled)

				// filter out the predictable errors
				if err != nil && !unimplementedInstruction && !programCounterCycled {
					return err
				}

				// create a new disassembly entry using last result
				ent, err := dsm.formatResult(memorymap.BankDetails{Number: b}, mc.LastResult, entryLevel)
				if err != nil {
					return err
				}

				if !unimplementedInstruction && !programCounterCycled {
					if err = mc.LastResult.IsValid(); err != nil {
						return err
					}

					// update field formatting information
					dsm.fields.updateWidths(ent)
				}

				// add entry to disassmebly. we do this even if we've encountered a
				// unimplemented instruction or some other error
				dsm.disasm[b][address&memorymap.CartridgeBits] = ent
			}
		}
	}

	return nil
}
