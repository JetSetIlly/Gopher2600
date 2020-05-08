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

package linter

import (
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Lint the disassembly of the load ROM. Currently, the function looks for read
// instructions that target non-addressible TIA and RIOT addresses.
func Lint(dsm *disassembly.Disassembly, output io.Writer) error {

	// look at every bank in the disassembly
	for bank := 0; bank < dsm.NumBanks(); bank++ {

		// create a new iteration for the bank
		itr, _, err := dsm.NewIteration(disassembly.EntryLevelBlessed, bank)
		if err != nil {
			return errors.New(errors.Linter, err)
		}

		// iterate through disassembled bank
		for d := itr.Start(); d != nil; d = itr.Next() {

			// if instruction has a read opcode, and the addressing mode seems
			// to be reading from non-read addresses in TIA or RIOT space then
			// create a lint warning

			if d.Result.Defn.Effect == instructions.Read {
				if d.Result.Defn.AddressingMode == instructions.Absolute ||
					d.Result.Defn.AddressingMode == instructions.ZeroPage {
					ma, area := memorymap.MapAddress(d.Result.InstructionData, true)

					switch area {
					case memorymap.TIA:
						_, isRead := addresses.CanonicalTIAReadSymbols[ma]
						if !isRead {
							s := fmt.Sprintf("%#04x\tread TIA address [%#04x (%#04x)]\n", d.Result.Address, d.Result.InstructionData, ma)
							output.Write([]byte(s))
						}

					case memorymap.RIOT:
						_, isRead := addresses.CanonicalRIOTReadSymbols[ma]
						if !isRead {
							s := fmt.Sprintf("%#04x\tread RIOT address [%#04x (%#04x)]\n", d.Result.Address, d.Result.InstructionData, ma)
							output.Write([]byte(s))
						}
					}
				}
			}
		}
	}

	return nil
}
