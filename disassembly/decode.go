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

package disassembly

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

func (dsm *Disassembly) disassemble(mc *cpu.CPU, mem *disasmMemory) error {
	copiedBanks, err := dsm.vcs.Mem.Cart.CopyBanks()
	if err != nil {
		return curated.Errorf("decode: %v", err)
	}

	// basic decoding pass
	err = dsm.decode(mc, mem, copiedBanks)
	if err != nil {
		return err
	}

	// bless those entries which we're reasonably sure are real instructions
	err = dsm.bless(mc, copiedBanks)
	if err != nil {
		return err
	}

	// convert addresses to preferred mirror
	dsm.setCartMirror()

	return nil
}

func (dsm *Disassembly) bless(mc *cpu.CPU, copiedBanks []mapper.BankContent) error {
	// we can be sure the machine reset address is a starting point for a blessing
	err := mc.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return err
	}

	resetAddress, resetArea := memorymap.MapAddress(mc.PC.Value(), true)
	if resetArea != memorymap.Cartridge {
		return curated.Errorf("bless: cartridge does not reset to an address in the cartridge area.")
	}

	// bless from reset address for every bank. note that we do not test to see
	// if the blessSequence is "correct" before committal because execution
	// from the reset address is likely
	for b := range dsm.entries {
		mc.PC.Load(resetAddress)
		_ = dsm.blessSequence(b, mc.PC.Value(), true)
	}

	moreBlessing := true
	attempts := 0

	for moreBlessing {
		moreBlessing = false

		// make sure we're not going bezerk. a limit of 100 is probably far too high
		attempts++
		if attempts > 100 {
			return curated.Errorf("bless: cannot discover a finite bless path through the program")
		}

		// loop through every bank in the cartridge and collate a list of blessings
		// for the bank
		for b := range dsm.entries {
			for i := range dsm.entries[b] {
				e := dsm.entries[b][i]

				// ignore any non-blessed entry
				if e.Level != EntryLevelBlessed {
					continue
				}

				// if instruction is a JMP or JSR then check the target address and
				// which banks it can feasibly be in. add every matching bank to
				// the blessings list
				//
				// an example of interbank JMP/JSR jumping is the beginning of the
				// HeMan ROM. From bank 7 the following jumps to bank 5.
				//
				//	$fa03 JMP $f7e8
				if e.Result.Defn.Operator == "JMP" || e.Result.Defn.Operator == "JSR" {
					jmpAddress, area := memorymap.MapAddress(e.Result.InstructionData, true)

					if area == memorymap.Cartridge {
						l := jmpTargets(copiedBanks, jmpAddress)

						for _, jb := range l {
							if dsm.blessSequence(jb, jmpAddress, false) {
								if dsm.blessSequence(jb, jmpAddress, true) {
									moreBlessing = true
								}
							}

							// add label to the symbols table
							dsm.sym.AddLabelAuto(jb, jmpAddress)
						}
					}

				} else {
					// if instruction is a branch then add the address of the successful branch
					// to the blessing list. assumption here is that a branch will never branch
					// to another bank
					// !!TODO: find practical examples of interbank branch jumping
					if e.Result.Defn.IsBranch() {
						mc.PC.Load(e.Result.Address)
						mc.PC.Add(uint16(e.Result.Defn.Bytes))
						operand := e.Result.InstructionData
						if operand&0x0080 == 0x0080 {
							operand |= 0xff00
						}
						mc.PC.Add(operand)

						pcAddress, area := memorymap.MapAddress(mc.PC.Value(), true)
						if area == memorymap.Cartridge {
							if dsm.blessSequence(b, pcAddress, false) {
								if dsm.blessSequence(b, pcAddress, true) {
									moreBlessing = true
								}
							}

							// add label to the symbols table
							dsm.sym.AddLabelAuto(b, pcAddress)
						}
					}
				}
			}
		}
	}

	// remove any labels that have been added to entries that have not been
	// blessed (these labels were added speculatively).
	//
	// this also removes labels that have been added from any symbols file that
	// pertain to entries that are unblessed. this is inentional because the
	// symbols.ReadSymbolsFile() function is far too permissive.
	for b := range dsm.entries {
		for i := range dsm.entries[b] {
			if dsm.entries[b][i].Level < EntryLevelBlessed {
				_ = dsm.sym.RemoveLabel(b, dsm.entries[b][i].Result.Address)
			}
		}
	}

	return nil
}

// jmpTargets returns a list of banks that a JMP or JSR target address may
// feasibly be in.
func jmpTargets(copiedBanks []mapper.BankContent, jmpAddress uint16) []int {
	l := make([]int, 0, len(copiedBanks))

	// only the significant bits of the jmp and origin addresses are compared.
	// this is the easiest way of handling different cartridge mirrors
	jmpAddress &= memorymap.CartridgeBits

	// find banks which can be mapped to cover the supplied jmpAddress
	for _, b := range copiedBanks {
		for _, o := range b.Origins {
			o &= memorymap.CartridgeBits

			if jmpAddress >= o && jmpAddress <= o+uint16(len(b.Data)) {
				l = append(l, b.Number)
				break
			}
		}
	}

	return l
}

// returns false if sequence is false. generally, the function should be called
// twice for a bank/addr. once with commit set to false and then if the return
// value is true, called again with commit set to true.
//
// if commit is true without the sequence being sure, a log entry will be made
// in the event of mistake
//
// it does not matter if addr is a normalised or unormalised address.
func (dsm *Disassembly) blessSequence(bank int, addr uint16, commit bool) bool {
	// mask address into indexable range
	a := addr & memorymap.CartridgeBits

	// blessing can happen at the same time as iteration which is probably
	// being run from a different goroutine. acknowledge the critical section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	hasCommitted := false

	// examine every entry from the starting point a. next entry determined in
	// program counter style (ie. address plus instruction byte count)
	//
	// sequence will stop if:
	//  . an unknown opcode has been encountered
	//  . if the end of cartridge data has been reached
	//  . there is already blessed instruction between this and the next entry
	//  . a flow control instruction is encountered (this is normal and expected)
	//  . an instruction looks like fill characters
	//
	// the following will stop the blessing but indicate that the blessing is
	// "correct" if the following have been encountered:
	//
	//  . an RTS instruction
	//  . a branch instruction
	//  . an interrupt
	for a < uint16(len(dsm.entries[bank])) {
		instruction := dsm.entries[bank][a]

		// end run if entry has already been blessed
		if instruction.Level == EntryLevelBlessed {
			if hasCommitted {
				logger.Logf("disassembly", "blessSequence has blessed an instruction in a false sequence. discoverd at bank %d: %s", bank, instruction.String())
			}
			return false
		}

		// if operator is unknown than end the sequence.
		operator := instruction.Result.Defn.Operator
		if operator == "??" {
			if hasCommitted {
				logger.Logf("disassembly", "blessSequence has blessed an instruction in a false sequence. discoverd at bank %d: %s", bank, instruction.String())
			}
			return false
		}

		next := a + uint16(instruction.Result.ByteCount)

		// break if address has looped around
		if next > next&memorymap.CartridgeBits {
			if hasCommitted {
				logger.Logf("disassembly", "blessSequence has blessed an instruction in a false sequence. discoverd at bank %d: %s", bank, instruction.String())
			}
			return false
		}

		// if an entry between this entry and the next has already been
		// blessed then this track is probably not correct.
		for i := a + 1; i < next; i++ {
			if instruction.Level == EntryLevelBlessed {
				if hasCommitted {
					logger.Logf("disassembly", "blessSequence has blessed an instruction in a false sequence. discoverd at bank %d: %s", bank, instruction.String())
				}
				return false
			}
		}

		// do not bless decoded instructions that look like fill characters
		if instruction.Result.Defn.OpCode == 0xff && instruction.Result.InstructionData == 0xffff {
			if hasCommitted {
				logger.Logf("disassembly", "blessSequence has blessed an instruction in a false sequence. discoverd at bank %d: %s", bank, instruction.String())
			}
			return false
		}

		// promote the entry
		if commit {
			hasCommitted = true
			instruction.Level = EntryLevelBlessed
		}

		// end the blessing sequence if we encounted an instruction that breaks
		// the flow with no possibilty of resumption. In practical terms this
		// means JMP, RTS and Interrupt instructions.
		if instruction.Operator == "JMP" || instruction.Operator == "RTS" ||
			instruction.Result.Defn.Effect == instructions.Interrupt {
			return true
		}

		// next adress
		a = next
	}

	// reached end of the bank without encountering any other halt condition
	return false
}

func (dsm *Disassembly) decode(mc *cpu.CPU, mem *disasmMemory, copiedBanks []mapper.BankContent) error {
	// decoding can happen at the same time as iteration which is probably
	// being run from a different goroutine. acknowledge the critical section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	for _, bank := range copiedBanks {
		// point memory implementation to iterated bank
		mem.bank = bank

		// try each bank in each possible segment - banks that are smaller than
		// the cartridge addressing range can often be mapped into different
		// "segments" of cartridge memory
		for _, origin := range bank.Origins {
			// make sure origin address is rooted correctly. we'll convert all
			// addresses to the preferred mirror at the end of the disassembly
			mem.origin = (origin & memorymap.CartridgeBits) | memorymap.OriginCart

			// the memtop for the bank
			memtop := origin + uint16(len(bank.Data)) - 1

			// reset CPU for each bank/origin
			mc.Reset()

			// loop over entire address space for cartridge. even then bank
			// sizes are smaller than the address space it makes things easier
			// later if we put a valid entry at every index. entries outside of
			// the bank space will have level == EntryLevelUnused
			for address := memorymap.OriginCart; address <= memorymap.MemtopCart; address++ {
				// continue if entry has already been decoded
				e := dsm.entries[bank.Number][address&memorymap.CartridgeBits]
				if e != nil && e.Level > EntryLevelUnmappable {
					continue
				}

				// decide whether address is mappable or not. even if it isn't, we'll still go
				// through the decoding process so that we have a usuable entry at all points
				// in the disassembly. this simplifies future iterations.
				entryLevel := EntryLevelUnmappable
				if address >= origin && address <= memtop {
					entryLevel = EntryLevelDecoded
				}

				// execute instruction at address
				mc.PC.Load(address)
				err := mc.ExecuteInstruction(nil)

				// filter out (allow) unimplemented instruction errors
				if err != nil && !curated.Is(err, cpu.UnimplementedInstruction) {
					return curated.Errorf("decode: %v", err)
				}

				// create a new disassembly entry using last result
				ent, err := dsm.FormatResult(mapper.BankInfo{Number: bank.Number}, mc.LastResult, entryLevel)
				if err != nil {
					return curated.Errorf("decode: %v", err)
				}

				// error on invalid instruction execution
				if err = mc.LastResult.IsValid(); err != nil {
					return curated.Errorf("decode: %v", err)
				}

				// add entry to disassembly
				dsm.entries[bank.Number][address&memorymap.CartridgeBits] = ent
			}
		}
	}

	// sanity checks
	for b := range dsm.entries {
		for _, a := range dsm.entries[b] {
			if a == nil {
				return curated.Errorf("decode: not every address has been decoded")
			}
			if a.Level == EntryLevelUnmappable {
				if a.Result.Defn.OpCode != 0x00 {
					return curated.Errorf("decode: an unmappable bank address [%#04x bank %d] has a non 0x00 opcode", a.Result.Address, b)
				}
			}
		}
	}

	return nil
}
