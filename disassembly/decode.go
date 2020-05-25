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
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

func (dsm *Disassembly) decode(mc *cpu.CPU) error {
	for b := 0; b < len(dsm.reference); b++ {
		mc.Reset(false)
		err := mc.LoadPCIndirect(addresses.Reset)
		if err != nil {
			return err
		}

		err = dsm.decodeBank(mc, b)
		if err != nil {
			return err
		}
	}

	return dsm.polish()
}

func (dsm *Disassembly) polish() error {
	for b := 0; b < dsm.NumBanks(); b++ {
		for i := 0; i < len(dsm.reference[b]); i++ {
			var p, d, n *Entry

			// reference to current, previous and next entry in the list
			d = dsm.reference[b][i]
			if i > 0 {
				p = dsm.reference[b][i-1]
			}
			if i < len(dsm.reference[b])-1 {
				n = dsm.reference[b][i+1]
			}

			// if this is a dead instruction then ignore it
			if d.Result.Defn == nil {
				continue // for loop
			}

			// very basic polish to get rid of cumulative instructions that
			// wouldn't be useful. this makes the assumption that real code
			// wouldn't do silly things like this.
			if d.Result.Defn.Mnemonic == "BRK" {
				if p != nil && p.Result.Defn != nil && p.Result.Defn.Mnemonic == "BRK" {
					if n != nil && n.Result.Defn != nil && n.Result.Defn.Mnemonic == "BRK" {
						d.Level = EntryLevelDecoded
					}
				}
			} else if d.Result.Defn.Effect == instructions.Flow {
				if p != nil && p.Result.Defn != nil && p.Result.Defn.Effect == instructions.Flow {
					d.Level = EntryLevelDecoded
				}
			}
		}
	}

	return nil
}

func (dsm *Disassembly) decodeBank(mc *cpu.CPU, b int) error {
	// address := mc.PC.Address()
	address := memorymap.OriginCart
	nextDecodePoint := address

	for dsm.reference[b][address&memorymap.AddressMaskCart] == nil {
		// set bank every iteration in case the cartridge read has triggered a
		// bank switch.
		// * at the front of the for loop
		if err := dsm.cart.SetBank(address, b); err != nil {
			return err
		}

		// execute instruction at address
		mc.PC.Load(address)
		err := mc.ExecuteInstruction(nil)

		unimplementedInstruction := errors.Is(err, errors.UnimplementedInstruction)

		// filter out the predictable errors
		if err != nil && !unimplementedInstruction {
			return err
		}

		// create a new disassembly entry using last result
		ent, err := dsm.FormatResult(b, mc.LastResult, EntryLevelDead)
		if err != nil {
			return err
		}

		// add bank information
		ent.Bank = b
		ent.BankDecorated = Bank(b)

		if !unimplementedInstruction {
			if err = mc.LastResult.IsValid(); err != nil {
				return err
			}

			ent.Level = EntryLevelDecoded

			// set entry type depending on whether we're at an expected decode
			// point
			if address == nextDecodePoint {
				// we're reasonably sure this is a real instruction
				ent.Level = EntryLevelBlessed

				// as far as we can tell this is a "real" instruction so
				// note the next expected decode point
				nextDecodePoint += uint16(mc.LastResult.Defn.Bytes)

			}

			// update field formatting information
			dsm.fields.updateWidths(ent)
		}

		// insert into Entries array
		dsm.reference[b][address&memorymap.AddressMaskCart] = ent

		// onto the next instruction
		address++

		// bump nextDecodePoint if we've gone past it
		if address > nextDecodePoint {
			nextDecodePoint = address
		}
	}

	return nil
}
