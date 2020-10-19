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
)

func (dsm *Disassembly) disassemble(mc *cpu.CPU, mem *disasmMemory) error {
	copiedBanks, err := dsm.cart.CopyBanks()
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
	startAddress := mc.PC.Value()

	for b := range dsm.entries {
		mc.PC.Load(startAddress)
		if dsm.blessSequence(b, mc.PC.Value(), false) {
			mc.PC.Load(startAddress)
			dsm.blessSequence(b, mc.PC.Value(), true)
		}
	}

	// list of start points for every bank
	blessings := make([][]uint16, len(dsm.entries))
	for b := range dsm.entries {
		blessings[b] = make([]uint16, 0)
	}

	// loop through every bank in the cartridge and collate a list of blessings
	// for the bank
	for b := range dsm.entries {
		for i := 0; i < len(dsm.entries[b]); i++ {
			e := dsm.entries[b][i]

			// if instruction is a JMP or JSR then check the target address and
			// which banks it can feasibly be in. add every matching bank to
			// the blessings list
			//
			// an example of interbank JMP/JSR jumping is the beginning of the
			// HeMan ROM. From bank 7 the following jumps to bank 5.
			//
			//	$fa03 JMP $f7e8
			//
			if e.Result.Defn.Mnemonic == "JMP" || e.Result.Defn.Mnemonic == "JSR" {
				jmpAddress := e.Result.InstructionData
				l := jmpTargets(copiedBanks, jmpAddress)
				for _, i := range l {
					blessings[i] = append(blessings[i], jmpAddress)
				}
			}

			// if instruction is a branch then add the address of the successful branch
			// to the blessing list.
			//
			// assumption here is that a branch will never branch to another bank
			//
			// TODO: find practical examples of interbank branch jumping
			if e.Result.Defn.IsBranch() {
				mc.PC.Load(e.Result.Address)
				mc.PC.Add(uint16(e.Result.Defn.Bytes))
				operand := e.Result.InstructionData
				if operand&0x0080 == 0x0080 {
					operand |= 0xff00
				}
				mc.PC.Add(operand)

				blessings[b] = append(blessings[b], mc.PC.Value())
			}
		}
	}

	// now that we have a list of blessings, we can bless each instruction in
	// sequence, starting at each blessing point accumulated above.
	for b := range blessings {
		for _, a := range blessings[b] {
			if dsm.blessSequence(b, a, false) {
				dsm.blessSequence(b, a, true)
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

func (dsm *Disassembly) blessSequence(b int, a uint16, commit bool) bool {
	// mask address into indexable range
	a &= memorymap.CartridgeBits

	// blessing can happen at the same time as iteration which is probably
	// being run from a different goroutine. acknowledge the critical section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// examine every entry from the starting point a. next entry determined in
	// program counter style (ie. address plus instruction byte count)
	//
	// sequence will stop if:
	//  . instruction is not defined/implemented
	//		(condition can be removed once all opcodes have been defined)
	//
	//  . there is already blessed instruction between this and the next entry
	//
	//  . a flow control instruction is encountered (this is normal and expected)
	//
	//  . if the end of cartridge data has been reached
	//
	for a < uint16(len(dsm.entries[b])) {
		// end run if entry has already been blessed
		if dsm.entries[b][a].Level == EntryLevelBlessed {
			return true
		}

		// if mnemonic is unknown than end the sequence.
		// !!TODO: remove this check once every opcode is defined/implemented
		mnemonic := dsm.entries[b][a].Result.Defn.Mnemonic
		if mnemonic == "??" {
			return false
		}

		next := a + uint16(dsm.entries[b][a].Result.ByteCount)

		// break if address has looped around. while this is possible
		// I'm not allowing it unless I can find an example of it being
		// used in actuality.
		if next > next&memorymap.CartridgeBits {
			return false
		}

		// if an entry between this entry and the next has already been
		// blessed then this track is probably not correct.
		for i := a + 1; i < next; i++ {
			if dsm.entries[b][i].Level == EntryLevelBlessed {
				return false
			}
		}

		// promote the entry
		if commit {
			dsm.entries[b][a].Level = EntryLevelBlessed
		}

		// return on interrupt or flow control instruction that is not a branch
		// instruction (ie. does not have relative addressing mode)
		effect := dsm.entries[b][a].Result.Defn.Effect
		if effect == instructions.Flow || effect == instructions.Interrupt {
			if dsm.entries[b][a].Result.Defn.AddressingMode != instructions.Relative {
				return true
			}
		}

		a = next
	}

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
