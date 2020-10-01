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

// Field identifies which part of the disassmbly entry is of interest
type Field int

// List of valid fields
const (
	FldLocation Field = iota
	FldBytecode
	FldAddress
	FldMnemonic
	FldOperand
	FldDefnCycles
	FldDefnNotes
	FldActualCycles
	FldActualNotes
	numFields
)

type fields struct {
	widths [numFields]int
	fmt    [numFields]string
}

// initialise with the minimum viable formatting string
func (fld *fields) initialise() {
	fld.widths[FldLocation] = 1
	fld.fmt[FldLocation] = fmt.Sprintf("%%%ds", fld.widths[FldLocation])
	fld.widths[FldBytecode] = 1
	fld.fmt[FldBytecode] = fmt.Sprintf("%%%ds", fld.widths[FldBytecode])
	fld.widths[FldAddress] = 1
	fld.fmt[FldAddress] = fmt.Sprintf("%%%ds", fld.widths[FldAddress])
	fld.widths[FldMnemonic] = 1
	fld.fmt[FldMnemonic] = fmt.Sprintf("%%%ds", fld.widths[FldMnemonic])
	fld.widths[FldOperand] = 1
	fld.fmt[FldOperand] = fmt.Sprintf("%%%ds", fld.widths[FldOperand])
	fld.widths[FldDefnCycles] = 1
	fld.fmt[FldDefnCycles] = fmt.Sprintf("%%%ds", fld.widths[FldDefnCycles])
	fld.widths[FldDefnNotes] = 1
	fld.fmt[FldDefnNotes] = fmt.Sprintf("%%%ds", fld.widths[FldDefnNotes])
	fld.widths[FldActualCycles] = 1
	fld.fmt[FldActualCycles] = fmt.Sprintf("%%%ds", fld.widths[FldActualCycles])
	fld.widths[FldActualNotes] = 1
	fld.fmt[FldActualNotes] = fmt.Sprintf("%%-%ds", fld.widths[FldActualNotes])
}

// update width and formatting information for entry fields. note that this
// doesn't update ActualCycles or ActualNotes
func (fld *fields) updateWidths(d *Entry) {
	if len(d.Location) > fld.widths[FldLocation] {
		fld.widths[FldLocation] = len(d.Location)
		fld.fmt[FldLocation] = fmt.Sprintf("%%%ds", fld.widths[FldLocation])
	}
	if len(d.Bytecode) > fld.widths[FldBytecode] {
		fld.widths[FldBytecode] = len(d.Bytecode)
		fld.fmt[FldBytecode] = fmt.Sprintf("%%%ds", fld.widths[FldBytecode])
	}
	if len(d.Address) > fld.widths[FldAddress] {
		fld.widths[FldAddress] = len(d.Address)
		fld.fmt[FldAddress] = fmt.Sprintf("%%%ds", fld.widths[FldAddress])
	}
	if len(d.Mnemonic) > fld.widths[FldMnemonic] {
		fld.widths[FldMnemonic] = len(d.Mnemonic)
		fld.fmt[FldMnemonic] = fmt.Sprintf("%%%ds", fld.widths[FldMnemonic])
	}
	if len(d.Operand) > fld.widths[FldOperand] {
		fld.widths[FldOperand] = len(d.Operand)
		fld.fmt[FldOperand] = fmt.Sprintf("%%%ds", fld.widths[FldOperand])
	}
	if len(d.DefnCycles) > fld.widths[FldDefnCycles] {
		fld.widths[FldDefnCycles] = len(d.DefnCycles)
		fld.fmt[FldDefnCycles] = fmt.Sprintf("%%%ds", fld.widths[FldDefnCycles])
	}
	if len(d.DefnNotes) > fld.widths[FldDefnNotes] {
		fld.widths[FldDefnNotes] = len(d.DefnNotes)
		fld.fmt[FldDefnNotes] = fmt.Sprintf("%%-%ds", fld.widths[FldDefnNotes])
	}
}

// update field widths for "actual" cycles and "actual" notes.
func (fld *fields) updateActual(d *Entry) {
	if len(d.ActualCycles) > fld.widths[FldActualCycles] {
		fld.widths[FldActualCycles] = len(d.ActualCycles)
		fld.fmt[FldActualCycles] = fmt.Sprintf("%%%ds", fld.widths[FldActualCycles])
	}
	if len(d.ActualNotes) > fld.widths[FldActualNotes] {
		fld.widths[FldActualNotes] = len(d.ActualNotes)
		fld.fmt[FldActualNotes] = fmt.Sprintf("%%-%ds", fld.widths[FldActualNotes])
	}
}

// GetField returns the formatted field from the speficied Entry
func (dsm *Disassembly) GetField(field Field, e *Entry) string {
	var s string

	switch field {
	case FldLocation:
		s = e.Location

	case FldBytecode:
		s = e.Bytecode

	case FldAddress:
		s = e.Address

	case FldMnemonic:
		s = e.Mnemonic

	case FldOperand:
		s = e.Operand

	case FldDefnCycles:
		s = e.DefnCycles

	case FldDefnNotes:
		s = e.DefnNotes

	case FldActualCycles:
		s = e.ActualCycles

	case FldActualNotes:
		s = e.ActualNotes
	}

	return fmt.Sprintf(dsm.fields.fmt[field], s)
}
