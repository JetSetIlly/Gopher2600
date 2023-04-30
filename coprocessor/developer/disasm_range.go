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

// InstructionRange is used to dynamically create a list of SourceInstruction entries.
type InstructionRange struct {
	Instructions []*SourceInstruction
}

// Len implements the sort.Sort interface.
func (rng *InstructionRange) Len() int {
	return len(rng.Instructions)
}

// Less implements the sort.Sort interface.
func (rng *InstructionRange) Less(i int, j int) bool {
	return rng.Instructions[i].Addr < rng.Instructions[j].Addr
}

// Swap implements the sort.Sort interface.
func (rng *InstructionRange) Swap(i int, j int) {
	rng.Instructions[i], rng.Instructions[j] = rng.Instructions[j], rng.Instructions[i]
}

// Clear all instructions from the range.
func (rng *InstructionRange) Clear() {
	rng.Instructions = rng.Instructions[:0]
}

// Add the instructions for a SourceLine to the range.
func (rng *InstructionRange) Add(line *SourceLine) {
	for _, d := range line.Instruction {
		rng.Instructions = append(rng.Instructions, d)
	}
	sort.Sort(rng)
}

// IsEmpty returns true if there are no instructions entries in the range.
func (rng *InstructionRange) IsEmpty() bool {
	return len(rng.Instructions) == 0
}
