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
	FldMnemonic
	FldOperand
	FldDefnCycles
	FldActualCycles
	FldActualNotes
)

// required widths (in characters) of the various disassembly fields.
const (
	widthBytecode     = 9
	widthAddress      = 6
	widthMnemonic     = 3
	widthDefnCycles   = 3
	widthActualCycles = 1

	widthNoLabel = 0

	// the operand field can be numeric or symbolic. in the case of
	// non-symbolic the following value is used.
	widthNonSymbolicOperand = 3

	// the operand field is decorated according to the addressing mode of the
	// entry. the following value is added to the width value of both symbolic
	// and non-symbolic operand values, to give the minimum required width.
	widthAddressingModeDecoration = 4

	// the width of the notes field is not recorded.
)

// GetField returns the formatted field from the speficied Entry.
func (e *Entry) GetField(field Field) string {
	var s string
	var w int

	switch field {
	case FldLabel:
		o, ok := e.Label.checkString()
		w = widthNoLabel
		if ok {
			w = e.dsm.Symbols.LabelWidth()
		}
		return fmt.Sprintf(fmt.Sprintf("%%-%ds", w), o)

	case FldBytecode:
		w = widthBytecode
		s = e.Bytecode

	case FldAddress:
		w = widthAddress
		s = e.Address

	case FldMnemonic:
		w = widthMnemonic
		s = e.Mnemonic

	case FldOperand:
		o, ok := e.Operand.checkString()
		w = widthNonSymbolicOperand
		if ok && e.dsm.Symbols.SymbolWidth() > w {
			w = e.dsm.Symbols.SymbolWidth()
		}
		w += widthAddressingModeDecoration
		s = o

	case FldDefnCycles:
		w = widthDefnCycles
		s = e.DefnCycles

	case FldActualCycles:
		w = widthActualCycles
		s = e.Cycles

	case FldActualNotes:
		return e.ExecutionNotes
	}

	return fmt.Sprintf(fmt.Sprintf("%%%ds", w), s)
}
