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
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

func (dsm *Disassembly) disassemble(mc *cpu.CPU, mem *disasmMemory, startAddress ...uint16) error {
	// new disassembly pass so we initialise field width
	dsm.fields.initialise()

	// basic decoding pass
	err := dsm.decode(mc, mem)
	if err != nil {
		return err
	}

	// after the decoding pass we won't be using the memory again except to to
	// read the reset address. to do this however it's best we access the
	// cartridge in the natural way. setting the bank field to nil will do
	// this.
	mem.bank = nil

	// reinitialise cartridge in case its internal state has been changed
	//
	// we shouldn't really need this and in some case it's downright dangerous
	// to do (eg. Supercharger) but it's harmless in most cases.
	//
	// !!TODO: satisfy ourselves that cartridge initialisation during disassembly is not needed
	if dsm.cart.ID() != supercharger.MappingID {
		dsm.cart.Initialise()
	}

	// bless those entries which we're reasonably sure are real instructions
	err = dsm.bless(mc, startAddress...)
	if err != nil {
		return err
	}

	// convert addresses to preferred mirror
	dsm.setCartMirror()

	return nil
}

func (dsm *Disassembly) bless(mc *cpu.CPU, startAddress ...uint16) error {
	if len(startAddress) == 0 {
		// if no startPoints have been supplied then we start off with the
		// machine reset address. this is the only sequence that we can be sure
		// of so we do this first (see blessSequence() function for interleave
		// detection).
		//
		// if we add it to the list of start points we accumulate below, we can't
		// be sure it will be first to run
		mc.Reset(false)
		err := mc.LoadPCIndirect(addresses.Reset)
		if err != nil {
			return err
		}
		bank := dsm.cart.GetBank(mc.PC.Value())
		if !bank.NonCart {
			dsm.blessSequence(bank.Number, mc.PC.Value()&memorymap.CartridgeBits)
		}

	} else {

		// walk through list of startPoints. we do this first for the same
		// reason the Reset address is added first (see commentary above)
		//
		// this relies on the cartridge being in the correct state. for the
		// startAddress to be valid
		for _, s := range startAddress {
			bank := dsm.cart.GetBank(s)
			if !bank.NonCart {
				dsm.blessSequence(bank.Number, s&memorymap.CartridgeBits)
			}
		}
	}

	// list of start points for every bank
	blessings := make([][]uint16, len(dsm.entries))
	for b := range dsm.entries {
		blessings[b] = make([]uint16, 0)
	}

	// loop through every bank in the cartridge and collate a list of blessings
	// for the bank. deliberately not using IterateCart for this.
	//
	// !!TODO: find blessing start point due to bank switch.
	//	for example: HeMan bank 7 addr $fa03 jumps to $f7e8 jumping to bank 5
	//	in the process
	for b := range dsm.entries {
		for i := 0; i < len(dsm.entries[b]); i++ {
			e := dsm.entries[b][i]

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

	// now that we have a list of blessings, we can bless each instruction in
	// sequence, starting at each blessing point accumulated above.
	for b := range blessings {
		for _, a := range blessings[b] {
			dsm.blessSequence(b, a)
		}
	}

	return nil
}

func (dsm *Disassembly) blessSequence(b int, a uint16) {
	// blessing can happen at the same time as iteration which is probably
	// being run from a different goroutine. acknowledge the critical section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// examine every entry from the starting point a. next entry determined in
	// program counter style (ie. address plus instruction byte count)
	//
	// sequence will stop if:
	//  . instruction is not defined
	//		(condition can be removed once all opcodes have been defined)
	//  . there is a blessed instruction between this and the next entry
	//		(we call this interleaving and we don't allow it)
	//  . a flow control instruction is encountered
	//		(this is normal and expected)
	//  . if the end of cartridge has been reached
	//		(execution could theoretically carry one but we don't allow it)
	//
	for a < uint16(len(dsm.entries[b])) {

		// if mnemonic is unknown than end the sequence.
		// !!TODO: remove this check once every opcode is defined/implemented
		mnemonic := dsm.entries[b][a].Result.Defn.Mnemonic
		if mnemonic == "??" {
			return
		}

		next := a + uint16(dsm.entries[b][a].Result.ByteCount)

		// break if address has looped around. while this is possible
		// I'm not allowing it unless I can find an example of it being
		// used in actuality.
		if next > next&memorymap.CartridgeBits {
			return
		}

		// if an entry between this entry and the next has already been
		// blessed then this track is probably not correct.
		for i := a + 1; i < next; i++ {
			if dsm.entries[b][i].Level == EntryLevelBlessed {
				return
			}
		}

		// promote the entry
		dsm.entries[b][a].Level = EntryLevelBlessed

		// not breaking on JSR because the sequence will continue if
		// the jumped-to sequence has an RTS (which it probably will
		// have)
		if mnemonic == "JMP" || mnemonic == "RTS" || mnemonic == "BRK" {
			return
		}

		a = next
	}
}

func (dsm *Disassembly) decode(mc *cpu.CPU, mem *disasmMemory) error {
	// decoding can happen at the same time as iteration which is probably
	// being run from a different goroutine. acknowledge the critical section
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	bank, err := dsm.cart.IterateBanks(nil)
	if err != nil {
		return err
	}

	// count the number of times IterateBanks() has return valid bank
	// infomration. we'll use this to sanity check the decoding.
	bankCt := 0

	for bank != nil {
		bankCt++

		// point memory implementation to iterated bank
		mem.bank = bank

		for _, origin := range bank.Origins {

			// make sure origin address is rooted correctly. we'll convert all
			// addresses to the preferred mirror at the end of the disassembly
			mem.origin = (origin & memorymap.CartridgeBits) | memorymap.OriginCart
			memtop := origin + uint16(len(bank.Data)) - 1

			mc.Reset(false)

			// loop over entire address space for every bank. even then bank
			// sizes are smaller than the address space it makes things easier
			// later if we put a valid entry at every index. entries outside of
			// the bank space will be marked as Unused
			for address := memorymap.OriginCart; address <= memorymap.MemtopCart; address++ {

				// check that entry has not already been decoded. cartridge
				// segments should not be able to overlap
				e := dsm.entries[bank.Number][address&memorymap.CartridgeBits]
				if e != nil && e.Level > EntryLevelUnused {
					continue
				}

				// decide whether address is mappable or not. even if it isn't,
				// we'll still go through the decoding process so that we have
				// a usuable entry at all points in the disassembly. this
				// simplifies future iterations.
				entryLevel := EntryLevelUnused
				if address >= origin && address <= memtop {
					entryLevel = EntryLevelDecoded
				}

				// execute instruction at address
				mc.PC.Load(address)
				err := mc.ExecuteInstruction(nil)

				unimplementedInstruction := curated.Is(err, cpu.UnimplementedInstruction)

				// filter out the predictable errors
				if err != nil && !unimplementedInstruction {
					return err
				}

				// create a new disassembly entry using last result
				ent, err := dsm.formatResult(banks.Details{Number: bank.Number}, mc.LastResult, entryLevel)
				if err != nil {
					return err
				}

				// if it's a valid instruction then update the field width information
				if !unimplementedInstruction {
					if err = mc.LastResult.IsValid(); err != nil {
						return err
					}
					dsm.fields.updateWidths(ent)
				}

				// add entry to disassembly. we do this even if we've encountered a
				// unimplemented instruction or some other error
				dsm.entries[bank.Number][address&memorymap.CartridgeBits] = ent
			}
		}

		// onto the next bank
		bank, err = dsm.cart.IterateBanks(bank)
		if err != nil {
			return err
		}
	}

	// sanity checks
	if bankCt != dsm.cart.NumBanks() {
		return curated.Errorf("disassembly: number of banks in disassembly is different to NumBanks()")
	}
	for b := range dsm.entries {
		for _, a := range dsm.entries[b] {
			if a == nil {
				return curated.Errorf("disassembly: not every address has been decoded")
			}
		}
	}

	return nil
}
