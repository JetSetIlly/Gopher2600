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
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

func rules(e *disassembly.Entry) []*LintEntry {
	res := make([]*LintEntry, 0)

	// if instruction has a read opcode, and the addressing mode seems
	// to be reading from non-read addresses in TIA or RIOT space then
	// create a lint warning
	if e.Result.Defn.AddressingMode != instructions.Immediate {
		if e.Result.Defn.Effect == instructions.Read {
			ma, area := memorymap.MapAddress(e.Result.InstructionData, true)
			switch area {
			case memorymap.TIA:
				if _, ok := addresses.TIAReadSymbols[ma]; !ok {
					le := &LintEntry{
						DisasmEntry: e,
						Error:       "reading a write only TIA address",
						Details:     ma,
					}
					res = append(res, le)
				}

			case memorymap.RIOT:
				if _, ok := addresses.RIOTReadSymbols[ma]; !ok {
					le := &LintEntry{
						DisasmEntry: e,
						Error:       "reading a write only RIOT address",
						Details:     ma,
					}
					res = append(res, le)
				}
			}
		}

		if e.Result.Defn.Effect == instructions.Write || e.Result.Defn.Effect == instructions.RMW {
			ma, area := memorymap.MapAddress(e.Result.InstructionData, false)
			switch area {
			case memorymap.TIA:
				if _, ok := addresses.TIAWriteSymbols[ma]; !ok {
					le := &LintEntry{
						DisasmEntry: e,
						Error:       "writing a read only TIA address",
						Details:     ma,
					}
					res = append(res, le)
				}

			case memorymap.RIOT:
				if _, ok := addresses.RIOTWriteSymbols[ma]; !ok {
					le := &LintEntry{
						DisasmEntry: e,
						Error:       "writing a read only RIOT address",
						Details:     ma,
					}
					res = append(res, le)
				}
			}
		}
	}

	return res
}
