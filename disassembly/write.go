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

// Write the entire disassembly to io.Writer.
func (dsm *Disassembly) Write(output io.Writer, attr ColumnAttr) error {
	for b := range dsm.disasmEntries.Entries {
		for _, e := range dsm.disasmEntries.Entries[b] {
			if e.Level >= EntryLevelBlessed {
				output.Write([]byte(e.StringColumnated(attr)))
				output.Write([]byte("\n"))
			}
		}
	}

	return nil
}

// WriteBank writes the disassembly of the selected bank to io.Writer.
func (dsm *Disassembly) WriteBank(output io.Writer, attr ColumnAttr, bank int) error {
	if bank >= len(dsm.disasmEntries.Entries) {
		return nil
	}

	for _, e := range dsm.disasmEntries.Entries[bank] {
		if e.Level >= EntryLevelBlessed {
			output.Write([]byte(e.StringColumnated(attr)))
			output.Write([]byte("\n"))
		}
	}

	return nil
}

// WriteAddr writes the disassembly of the specified address to the io.Writer.
func (dsm *Disassembly) WriteAddr(output io.Writer, attr ColumnAttr, addr uint16) error {
	e, _ := dsm.GetEntryByAddress(addr)
	if e != nil && e.Level >= EntryLevelBlessed {
		output.Write([]byte(e.StringColumnated(attr)))
	} else {
		return curated.Errorf(fmt.Sprintf("no blessed disassembly at $%04x", addr))
	}
	return nil
}
