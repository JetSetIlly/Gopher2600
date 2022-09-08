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

package developer

import (
	"sort"
)

// DisasmRange is used to dynamically create a list of SourceDisasm entries.
type DisasmRange struct {
	Disasm []*SourceDisasm
}

// Len implements the sort.Sort interface.
func (rng *DisasmRange) Len() int {
	return len(rng.Disasm)
}

// Less implements the sort.Sort interface.
func (rng *DisasmRange) Less(i int, j int) bool {
	return rng.Disasm[i].Addr < rng.Disasm[j].Addr
}

// Swap implements the sort.Sort interface.
func (rng *DisasmRange) Swap(i int, j int) {
	rng.Disasm[i], rng.Disasm[j] = rng.Disasm[j], rng.Disasm[i]
}

// Clear all disassembly entires from the range.
func (rng *DisasmRange) Clear() {
	rng.Disasm = rng.Disasm[:0]
}

// Add the disassembly entries for a SourceLine to the range.
func (rng *DisasmRange) Add(line *SourceLine) {
	for _, d := range line.Disassembly {
		rng.Disasm = append(rng.Disasm, d)
	}

	sort.Sort(rng)
}

// IsEmpty returns true if there are now SourceDisasm entries in the range.
func (rng *DisasmRange) IsEmpty() bool {
	return len(rng.Disasm) == 0
}
