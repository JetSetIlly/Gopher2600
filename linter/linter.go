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

package linter

import (
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Lint the disassembly of the load ROM. Currently, the function looks for read
// instructions that target non-addressible TIA and RIOT addresses.
func Lint(dsm *disassembly.Disassembly, output io.Writer) error {
	// look at every bank in the disassembly
	citr := dsm.NewCartIteration()
	citr.Start()
	for b, ok := citr.Start(); ok; b, ok = citr.Next() {
		// create a new iteration for the bank
		bitr, err := dsm.NewBankIteration(disassembly.EntryLevelBlessed, b)
		if err != nil {
			return curated.Errorf("linter: %v", err)
		}

		// iterate through disassembled bank
		for _, e := bitr.Start(); e != nil; _, e = bitr.Next() {
			// if instruction has a read opcode, and the addressing mode seems
			// to be reading from non-read addresses in TIA or RIOT space then
			// create a lint warning

			if e.Result.Defn.Effect == instructions.Read {
				if e.Result.Defn.AddressingMode == instructions.Absolute ||
					e.Result.Defn.AddressingMode == instructions.ZeroPage {
					ma, area := memorymap.MapAddress(e.Result.InstructionData, true)

					switch area {
					case memorymap.TIA:
						_, isRead := addresses.TIAReadSymbols[ma]
						if !isRead {
							s := fmt.Sprintf("%#04x\tread TIA address [%#04x (%#04x)]\n", e.Result.Address, e.Result.InstructionData, ma)
							output.Write([]byte(s))
						}

					case memorymap.RIOT:
						_, isRead := addresses.RIOTReadSymbols[ma]
						if !isRead {
							s := fmt.Sprintf("%#04x\tread RIOT address [%#04x (%#04x)]\n", e.Result.Address, e.Result.InstructionData, ma)
							output.Write([]byte(s))
						}
					}
				}
			}
		}
	}

	return nil
}
