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
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

func (dsm *Disassembly) decode(mc *cpu.CPU) error {
	for b := 0; b < len(dsm.Entries); b++ {
		address := memorymap.OriginCart
		nextDecodePoint := address

		for dsm.Entries[b][address&memorymap.AddressMaskCart] == nil {
			// bump nextDecodePoint if we've gone past it
			if address > nextDecodePoint {
				nextDecodePoint = address
			}

			// set bank in case the cartridge read has triggered a bank switch
			if err := dsm.cart.SetBank(address, b); err != nil {
				return err
			}

			// execute instruction at address
			mc.PC.Load(address)
			err := mc.ExecuteInstruction(nil)

			// filter out the predictable errors
			if err != nil {
				if !errors.IsAny(err) {
					return err
				}

				switch err.(errors.AtariError).Message {
				case errors.ProgramCounterCycled:
					break // for loop
				case errors.UnimplementedInstruction:
					// try next byte
					address++
					continue // for loop
				default:
					return err
				}
			}

			// continue for loop on invalid results
			if mc.LastResult.IsValid() != nil {
				// try next byte
				address++
				continue // for loop
			}

			// create a new disassembly entry using last result
			ent, err := dsm.FormatResult(mc.LastResult)
			if err != nil {
				return err
			}

			// add bank information
			ent.Bank = b

			// set entry type depending on whether we're at an expected decode
			// point
			if address == nextDecodePoint {
				ent.Type = EntryTypeDecode

				// as far as we can tell this is a "real" instruction so
				// note the next expected decode point
				nextDecodePoint += uint16(mc.LastResult.Defn.Bytes)

				// update field formatting information
				dsm.fields.updateWidths(ent)
			} else {
				ent.Type = EntryTypeNaive
			}

			// insert into Entries array
			dsm.Entries[b][address&memorymap.AddressMaskCart] = ent

			// onto the next instruction
			address++
		}
	}

	return nil
}
