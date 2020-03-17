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

package disassembly

import "fmt"

type widths struct {
	location     int
	bytecode     int
	address      int
	mnemonic     int
	operand      int
	defnCycles   int
	defnNotes    int
	actualCycles int
	actualNotes  int
}

type format struct {
	location     string
	bytecode     string
	address      string
	mnemonic     string
	operand      string
	defnCycles   string
	defnNotes    string
	actualCycles string
	actualNotes  string
}

type fields struct {
	widths widths
	fmt    format
}

// Update width and formatting information for entry fields
func (fld *fields) updateWidths(d *Entry) {
	if len(d.Location) > fld.widths.location {
		fld.widths.location = len(d.Location)
	}
	if len(d.Bytecode) > fld.widths.bytecode {
		fld.widths.bytecode = len(d.Bytecode)
	}
	if len(d.Address) > fld.widths.address {
		fld.widths.address = len(d.Address)
	}
	if len(d.Mnemonic) > fld.widths.mnemonic {
		fld.widths.mnemonic = len(d.Mnemonic)
	}
	if len(d.Operand) > fld.widths.operand {
		fld.widths.operand = len(d.Operand)
	}
	if len(d.DefnCycles) > fld.widths.defnCycles {
		fld.widths.defnCycles = len(d.DefnCycles)
	}
	if len(d.DefnNotes) > fld.widths.defnNotes {
		fld.widths.defnNotes = len(d.DefnNotes)
	}
	if len(d.ActualCycles) > fld.widths.actualCycles {
		fld.widths.actualCycles = len(d.ActualCycles)
	}
	if len(d.ActualNotes) > fld.widths.actualNotes {
		fld.widths.actualNotes = len(d.ActualNotes)
	}

	fld.fmt.location = fmt.Sprintf("%%%ds", fld.widths.location)
	fld.fmt.bytecode = fmt.Sprintf("%%%ds", fld.widths.bytecode)
	fld.fmt.address = fmt.Sprintf("%%%ds", fld.widths.address)
	fld.fmt.mnemonic = fmt.Sprintf("%%%ds", fld.widths.mnemonic)
	fld.fmt.operand = fmt.Sprintf("%%%ds", fld.widths.operand)
	fld.fmt.defnCycles = fmt.Sprintf("%%%ds", fld.widths.defnCycles)
	fld.fmt.defnNotes = fmt.Sprintf("%%%ds", fld.widths.defnNotes)
	fld.fmt.actualCycles = fmt.Sprintf("%%%ds", fld.widths.actualCycles)
	fld.fmt.actualNotes = fmt.Sprintf("%%%ds", fld.widths.actualNotes)
}

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
)

// GetField returns the formatted field from the speficied Entry
func (dsm *Disassembly) GetField(field Field, e *Entry) string {
	switch field {
	case FldLocation:
		return fmt.Sprintf(dsm.fields.fmt.location, e.Location)
	case FldBytecode:
		return fmt.Sprintf(dsm.fields.fmt.bytecode, e.Bytecode)
	case FldAddress:
		return fmt.Sprintf(dsm.fields.fmt.address, e.Address)
	case FldMnemonic:
		return fmt.Sprintf(dsm.fields.fmt.mnemonic, e.Mnemonic)
	case FldOperand:
		return fmt.Sprintf(dsm.fields.fmt.operand, e.Operand)
	case FldDefnCycles:
		return fmt.Sprintf(dsm.fields.fmt.defnCycles, e.DefnCycles)
	case FldDefnNotes:
		return fmt.Sprintf(dsm.fields.fmt.defnNotes, e.DefnNotes)
	case FldActualCycles:
		return fmt.Sprintf(dsm.fields.fmt.actualCycles, e.ActualCycles)
	case FldActualNotes:
		return fmt.Sprintf(dsm.fields.fmt.actualNotes, e.ActualNotes)
	}
	return ""
}
