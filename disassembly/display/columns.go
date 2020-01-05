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

package display

import "fmt"

// Widths of DisasmInstuction entry fields.
type Widths struct {
	Location int
	Bytecode int
	Address  int
	Mnemonic int
	Operand  int
	Cycles   int
	Notes    int
}

// Fmt strings for Instruction fields. For use with fmt.Printf() and fmt.Sprintf()
type Fmt struct {
	Location string
	Bytecode string
	Address  string
	Mnemonic string
	Operand  string
	Cycles   string
	Notes    string
}

// Columns information for groups of Instructions
type Columns struct {
	Widths Widths
	Fmt    Fmt
}

// Update width and formatting information
func (col *Columns) Update(d *Instruction) {
	if len(d.Location) > col.Widths.Location {
		col.Widths.Location = len(d.Location)
	}
	if len(d.Bytecode) > col.Widths.Bytecode {
		col.Widths.Bytecode = len(d.Bytecode)
	}
	if len(d.Address) > col.Widths.Address {
		col.Widths.Address = len(d.Address)
	}
	if len(d.Mnemonic) > col.Widths.Mnemonic {
		col.Widths.Mnemonic = len(d.Mnemonic)
	}
	if len(d.Operand) > col.Widths.Operand {
		col.Widths.Operand = len(d.Operand)
	}
	if len(d.Cycles) > col.Widths.Cycles {
		col.Widths.Cycles = len(d.Cycles)
	}
	if len(d.Notes) > col.Widths.Notes {
		col.Widths.Notes = len(d.Notes)
	}

	col.Fmt.Location = fmt.Sprintf("%%%ds", col.Widths.Location)
	col.Fmt.Bytecode = fmt.Sprintf("%%%ds", col.Widths.Bytecode)
	col.Fmt.Address = fmt.Sprintf("%%%ds", col.Widths.Address)
	col.Fmt.Mnemonic = fmt.Sprintf("%%%ds", col.Widths.Mnemonic)
	col.Fmt.Operand = fmt.Sprintf("%%%ds", col.Widths.Operand)
	col.Fmt.Cycles = fmt.Sprintf("%%%ds", col.Widths.Cycles)
	col.Fmt.Notes = fmt.Sprintf("%%%ds", col.Widths.Notes)
}
