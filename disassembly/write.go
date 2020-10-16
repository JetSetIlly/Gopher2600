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
	"io"

	"github.com/jetsetilly/gopher2600/curated"
)

// WriteAttr controls what is printed by the Write*() functions.
type WriteAttr struct {
	ByteCode bool
}

// Write the entire disassembly to io.Writer.
func (dsm *Disassembly) Write(output io.Writer, attr WriteAttr) error {
	var err error
	for b := 0; b < len(dsm.entries); b++ {
		err = dsm.WriteBank(output, attr, b)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteBank writes the disassembly of the selected bank to io.Writer.
func (dsm *Disassembly) WriteBank(output io.Writer, attr WriteAttr, bank int) error {
	if bank < 0 || bank > len(dsm.entries)-1 {
		return curated.Errorf("disassembly: no such bank (%d)", bank)
	}

	output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))

	for i := range dsm.entries[bank] {
		dsm.WriteEntry(output, attr, dsm.entries[bank][i])
	}

	return nil
}

// WriteEntry writes a single Instruction to io.Writer.
func (dsm *Disassembly) WriteEntry(output io.Writer, attr WriteAttr, e *Entry) {
	if e == nil {
		return
	}

	if e.Level < EntryLevelBlessed {
		return
	}

	if e.Label.String() != "" {
		output.Write([]byte(e.GetField(FldLabel)))
		output.Write([]byte("\n"))
	}

	if attr.ByteCode {
		output.Write([]byte(e.GetField(FldBytecode)))
		output.Write([]byte(" "))
	}

	output.Write([]byte(e.GetField(FldAddress)))
	output.Write([]byte(" "))
	output.Write([]byte(e.GetField(FldMnemonic)))
	output.Write([]byte(" "))
	output.Write([]byte(e.GetField(FldOperand)))
	output.Write([]byte(" "))
	output.Write([]byte(e.GetField(FldDefnCycles)))

	output.Write([]byte("\n"))
}
