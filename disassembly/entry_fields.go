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

package disassembly

import (
	"fmt"
)

// Field identifies which part of the disassmbly entry is of interest.
type Field int

// List of valid fields.
const (
	FldLabel Field = iota
	FldBytecode
	FldAddress
	FldOperator
	FldOperand
	FldCycles
	FldNotes
)

// required widths (in characters) of the various disassembly fields.
const (
	// the width of the label field is equal to the value returned by
	// Symbols.LabelWidth().

	// the widths of bytecode, address, and operator can be decided in advance.
	widthBytecode = 9
	widthAddress  = 6
	widthOperator = 3

	// the oeprand field width should be the value of Symbols.SymbolWidth()
	// plus widthOperandDecoration, which accounts for the maximum number of
	// additional symbols required to correctly display the addressing mode.
	// for example:
	//
	//	($6e), Y
	//
	// or
	//
	//  (OFFSET), Y
	widthOperandDecoration = 4

	// the width for the cycles column assumes that there will be at least one
	// branch instruction that has been executed. For example:
	//
	//		2/3 [3]
	//
	// see Entry.Cycles() function.
	widthCycles = 7

	// the width of the notes field is not needed.
)

// GetField returns the formatted field from the speficied Entry.
func (e *Entry) GetField(field Field) string {
	var s string
	var w int
	var rightJust bool

	switch field {
	case FldLabel:
		s = e.Label.String()
		if s == "" {
			w = 0
		} else {
			w = e.dsm.Sym.LabelWidth()
			rightJust = true
		}

	case FldBytecode:
		w = widthBytecode
		s = e.Bytecode

	case FldAddress:
		w = widthAddress
		s = e.Address

	case FldOperator:
		w = widthOperator
		s = e.Operator

	case FldOperand:
		s = e.Operand.String()
		w = e.dsm.Sym.SymbolWidth() + widthOperandDecoration

	case FldCycles:
		w = widthCycles
		s = e.Cycles()

	case FldNotes:
		return e.Notes()
	}

	if rightJust {
		s = fmt.Sprintf(fmt.Sprintf("%%-%ds", w), s)
	} else {
		s = fmt.Sprintf(fmt.Sprintf("%%%ds", w), s)
	}

	return s
}
