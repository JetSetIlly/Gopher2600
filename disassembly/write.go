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
	"gopher2600/errors"
	"io"
)

// WriteAttr controls what is printed by the Write*() functions
type WriteAttr struct {
	ByteCode bool
	FlowInfo bool
}

// Write the entire disassembly to io.Writer
func (dsm *Disassembly) Write(output io.Writer, attr WriteAttr) error {
	var err error
	for bank := 0; bank < len(dsm.Entries); bank++ {
		err = dsm.WriteBank(output, attr, bank)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteBank writes the disassembly of the selected bank to io.Writer
func (dsm *Disassembly) WriteBank(output io.Writer, attr WriteAttr, bank int) error {
	if bank < 0 || bank > len(dsm.Entries)-1 {
		return errors.New(errors.DisasmError, fmt.Sprintf("no such bank (%d)", bank))
	}

	output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))

	for i := range dsm.Entries[bank] {
		dsm.WriteLine(output, attr, dsm.Entries[bank][i])
	}

	return nil
}

// WriteLine writes a single Instruction to io.Writer
func (dsm *Disassembly) WriteLine(output io.Writer, attr WriteAttr, e *Entry) {
	if e == nil || e.Type < EntryTypeDecode {
		return
	}

	if e.Location != "" {
		output.Write([]byte(dsm.GetField(FldLocation, e)))
		output.Write([]byte("\n"))
	}

	if attr.ByteCode {
		output.Write([]byte(dsm.GetField(FldBytecode, e)))
		output.Write([]byte(" "))
	}

	output.Write([]byte(dsm.GetField(FldAddress, e)))
	output.Write([]byte(" "))
	output.Write([]byte(dsm.GetField(FldMnemonic, e)))
	output.Write([]byte(" "))
	output.Write([]byte(dsm.GetField(FldOperand, e)))
	output.Write([]byte(" "))
	output.Write([]byte(dsm.GetField(FldDefnCycles, e)))
	output.Write([]byte(" "))
	output.Write([]byte(dsm.GetField(FldDefnNotes, e)))

	if attr.FlowInfo {
		if len(e.Next) > 0 {
			output.Write([]byte(" -> "))
			for i := range e.Next {
				output.Write([]byte(fmt.Sprintf("%#04x ", e.Next[i])))
			}
		}

		if len(e.Prev) > 0 {
			output.Write([]byte(" <- "))
			for i := range e.Prev {
				output.Write([]byte(fmt.Sprintf("%#04x ", e.Prev[i])))
			}
		}
	}

	output.Write([]byte("\n"))
}
