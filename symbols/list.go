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

package symbols

import (
	"fmt"
	"io"
)

// ListSymbols outputs every symbol used in the current ROM
func (tbl *Table) ListSymbols(output io.Writer) {
	tbl.ListLocations(output)
	tbl.ListReadSymbols(output)
	tbl.ListWriteSymbols(output)
}

// ListLocations outputs every location symbol used in the current ROM
func (tbl *Table) ListLocations(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("Locations\n---------\n")))
	output.Write([]byte(tbl.Locations.String()))
}

// ListReadSymbols outputs every read symbol used in the current ROM
func (tbl *Table) ListReadSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nRead Symbols\n-----------\n")))
	output.Write([]byte(tbl.Read.String()))
}

// ListWriteSymbols outputs every write symbol used in the current ROM
func (tbl *Table) ListWriteSymbols(output io.Writer) {
	output.Write([]byte(fmt.Sprintf("\nWrite Symbols\n------------\n")))
	output.Write([]byte(tbl.Write.String()))
}
