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
	cycles       int
	notes        int
	actualCycles int
	actualNotes  int
}

type format struct {
	location     string
	bytecode     string
	address      string
	mnemonic     string
	operand      string
	cycles       string
	notes        string
	actualCycles string
	actualNotes  string
}

type fields struct {
	widths widths
	fmt    format
}

// Update width and formatting information for entry fields
func (fld *fields) update(d *Entry) {
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
	if len(d.Cycles) > fld.widths.cycles {
		fld.widths.cycles = len(d.Cycles)
	}
	if len(d.Notes) > fld.widths.notes {
		fld.widths.notes = len(d.Notes)
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
	fld.fmt.cycles = fmt.Sprintf("%%%ds", fld.widths.cycles)
	fld.fmt.notes = fmt.Sprintf("%%%ds", fld.widths.notes)
	fld.fmt.actualCycles = fmt.Sprintf("%%%ds", fld.widths.actualCycles)
	fld.fmt.actualNotes = fmt.Sprintf("%%%ds", fld.widths.actualNotes)
}

// Field identifies which part of the disassmbly entry is of interest
type Field int

// List of valid fields
const (
	Location Field = iota
	Bytecode
	Address
	Mnemonic
	Operand
	Cycles
	Notes
	ActualCycles
	ActualNotes
)

// GetField returns the formatted field from the speficied Entry
func (dsm *Disassembly) GetField(field Field, d *Entry) string {
	switch field {
	case Location:
		return fmt.Sprintf(dsm.fields.fmt.location, d.Location)
	case Bytecode:
		return fmt.Sprintf(dsm.fields.fmt.bytecode, d.Bytecode)
	case Address:
		return fmt.Sprintf(dsm.fields.fmt.address, d.Address)
	case Mnemonic:
		return fmt.Sprintf(dsm.fields.fmt.mnemonic, d.Mnemonic)
	case Operand:
		return fmt.Sprintf(dsm.fields.fmt.operand, d.Operand)
	case Cycles:
		return fmt.Sprintf(dsm.fields.fmt.cycles, d.Cycles)
	case Notes:
		return fmt.Sprintf(dsm.fields.fmt.notes, d.Notes)
	case ActualCycles:
		return fmt.Sprintf(dsm.fields.fmt.actualCycles, d.ActualCycles)
	case ActualNotes:
		return fmt.Sprintf(dsm.fields.fmt.actualNotes, d.ActualNotes)
	}
	return ""
}
