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

import (
	"fmt"
	"gopher2600/disassembly/display"
	"gopher2600/errors"
	"io"
)

// Write the entire disassembly to io.Writer
func (dsm *Disassembly) Write(output io.Writer, byteCode bool) error {
	var err error
	for bank := 0; bank < len(dsm.flow); bank++ {
		err = dsm.WriteBank(output, byteCode, bank)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteBank writes the disassembly of the selected bank to io.Writer
func (dsm *Disassembly) WriteBank(output io.Writer, byteCode bool, bank int) error {
	if bank < 0 || bank > len(dsm.flow)-1 {
		return errors.New(errors.DisasmError, fmt.Sprintf("no such bank (%d)", bank))
	}

	output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))

	for i := range dsm.flow[bank] {
		if d := dsm.flow[bank][i]; d != nil {
			dsm.WriteLine(output, byteCode, d)
		}
	}

	return nil
}

// WriteLine writes a single Instruction to io.Writer
func (dsm *Disassembly) WriteLine(output io.Writer, byteCode bool, d *display.Instruction) {
	if d.Location != "" {
		output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Location, d.Location)))
		output.Write([]byte("\n"))
	}

	if byteCode {
		output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Bytecode, d.Bytecode)))
		output.Write([]byte(" "))
	}

	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Address, d.Address)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Mnemonic, d.Mnemonic)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Operand, d.Operand)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Cycles, d.Cycles)))
	output.Write([]byte(" "))
	output.Write([]byte(fmt.Sprintf(dsm.Columns.Fmt.Notes, d.Notes)))

	output.Write([]byte("\n"))
}
