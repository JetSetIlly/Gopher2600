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

func (dsm *Disassembly) decode(mc *cpu.CPU) error {
	for b := 0; b < len(dsm.reference); b++ {
		mc.Reset()
		err := mc.LoadPCIndirect(addresses.Reset)
		if err != nil {
			return err
		}

		err = dsm.decodeBank(mc, b)
		if err != nil {
			return err
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
			err = mc.LastResult.IsValid()
			if err != nil {
				return err
			}

			// set entry type depending on whether we're at an expected decode
			// point
			if address == nextDecodePoint {
				ent.Level = EntryLevelBlessed

				// as far as we can tell this is a "real" instruction so
				// note the next expected decode point
				nextDecodePoint += uint16(mc.LastResult.Defn.Bytes)

				// update field formatting information
				dsm.fields.updateWidths(ent)
			} else {
				ent.Level = EntryLevelDecoded
			}
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
