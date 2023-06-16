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
	"strings"
)

// GrepScope limits the scope of the search.
type GrepScope int

// List of available scopes.
const (
	GrepAll GrepScope = iota
	GrepOperator
	GrepOperand
)

// Grep searches the disassembly for the specified search string.
func (dsm *Disassembly) Grep(output io.Writer, scope GrepScope, search string, caseSensitive bool) error {
	if !caseSensitive {
		search = strings.ToUpper(search)
	}

	includeBankNumber := len(dsm.disasmEntries.Entries) > 0

	for b := range dsm.disasmEntries.Entries {
		bankNumberPrinted := false

		for _, e := range dsm.disasmEntries.Entries[b] {
			if e != nil && e.Level >= EntryLevelBlessed {
				var s, m string

				// limit scope of grep to the correct Instruction field
				switch scope {
				case GrepOperator:
					s = e.Operator
				case GrepOperand:
					s = e.Operand.String()
				case GrepAll:
					s = e.String()
				}

				if !caseSensitive {
					m = strings.ToUpper(s)
				} else {
					m = s
				}

				if strings.Contains(m, search) {
					if includeBankNumber && !bankNumberPrinted {
						bankNumberPrinted = true
						output.Write([]byte(fmt.Sprintf("Bank %d\n", b)))
					}
					output.Write([]byte(e.StringColumnated(ColumnAttr{})))
					output.Write([]byte("\n"))
				}
			}
		}
	}

	return nil
}
