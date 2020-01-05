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
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory/memorymap"
)

func (dsm *Disassembly) linearDisassembly(mc *cpu.CPU) error {
	for bank := 0; bank < len(dsm.linear); bank++ {
		for address := memorymap.OriginCart; address <= memorymap.MemtopCart; address++ {
			if err := dsm.cart.SetBank(address, bank); err != nil {
				return err
			}

			mc.PC.Load(address)

			// deliberately ignoring errors
			_ = mc.ExecuteInstruction(nil)

			// continue for loop on invalid results. we don't want to be as
			// discerning as in flowDisassembly(). the nature of
			// linearDisassembly() means that we're likely to try executing
			// invalid instructions. best just to ignore such errors.
			if mc.LastResult.IsValid() != nil {
				continue // for loop
			}

			ent, err := dsm.FormatResult(mc.LastResult)
			if err != nil {
				return err
			}

			dsm.linear[bank][address&disasmMask] = ent
		}
	}

	return nil
}
