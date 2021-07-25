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

package arm7tdmi

import "fmt"

// DisasmEntry implements the CartCoProcDisasmEntry interface.
type DisasmEntry struct {
	Location string
	Address  string
	Operator string
	Operand  string

	// total cycles for this instruction
	Cycles int

	// basic notes about the last execution of the entry
	ExecutionNotes string

	// execution notes should be updated on next execution
	updateNotes bool

	// details
	MAMCR          int
	BranchTrail    BranchTrail
	MergedIS       bool
	CyclesSequence string
}

// Key implements the CartCoProcDisasmEntry interface.
func (e DisasmEntry) Key() string {
	return e.Address
}

// String implements the CartCoProcDisasmEntry interface. Outputs CSV friendly
// entries, albeit seprated by semicolons rather than commas.
func (e DisasmEntry) String() string {
	mergedIS := ""
	if e.MergedIS {
		mergedIS = "merged IS"
	}

	return fmt.Sprintf("%s;%s;%s;%d;%s;%s;%s", e.Address, e.Operator, e.Operand, e.Cycles, e.ExecutionNotes, mergedIS, e.CyclesSequence)
}

// DisasmSummary implements the CartCoProcDisasmSummary interface.
type DisasmSummary struct {
	N int
	I int
	S int

	// whether this particular execution was run in immediate mode (ie. no
	// cycle counting)
	ImmediateMode bool
}

func (s DisasmSummary) String() string {
	return fmt.Sprintf("N: %d  I: %d  S: %d", s.N, s.I, s.S)
}

// add cycle order information to summary.
func (s *DisasmSummary) add(c cycleOrder) {
	for i := 0; i < c.idx; i++ {
		switch c.queue[i] {
		case N:
			s.N++
		case I:
			s.I++
		case S:
			s.S++
		}
	}
}
