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

package symbols

import (
	"fmt"
	"io"
)

// ListSymbols outputs every symbol used in the current ROM.
func (sym *Symbols) ListSymbols(output io.Writer) {
	sym.ListLabels(output)
	sym.ListReadSymbols(output)
	sym.ListWriteSymbols(output)
}

// ListLabels outputs every label used in the current ROM.
func (sym *Symbols) ListLabels(output io.Writer) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	output.Write([]byte("Labels\n------\n"))
	for b, l := range sym.label {
		s := l.String()

		if len(s) > 0 {
			if len(sym.label) > 1 {
				output.Write(fmt.Appendf(nil, "Bank %d:\n", b))
			}
			output.Write([]byte(s))
		}
	}
}

// ListReadSymbols outputs every read symbol used in the current ROM.
func (sym *Symbols) ListReadSymbols(output io.Writer) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	output.Write([]byte("\nRead Symbols\n------------\n"))
	output.Write([]byte(sym.read.String()))
}

// ListWriteSymbols outputs every write symbol used in the current ROM.
func (sym *Symbols) ListWriteSymbols(output io.Writer) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	output.Write([]byte("\nWrite Symbols\n-------------\n"))
	output.Write([]byte(sym.write.String()))
}
