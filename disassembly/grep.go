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
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

// GrepScope limits the scope of the search.
type GrepScope int

// List of available scopes.
const (
	GrepAll GrepScope = iota
	GrepMnemonic
	GrepOperand
)

// Grep searches the disassembly for the specified search string.
func (dsm *Disassembly) Grep(output io.Writer, scope GrepScope, search string, caseSensitive bool) error {
	var s, m string

	if !caseSensitive {
		search = strings.ToUpper(search)
	}

	hasOutput := false

	citr := dsm.NewCartIteration()
	citr.Start()
	for b, ok := citr.Start(); ok; b, ok = citr.Next() {
		bankHeader := false

		bitr, err := dsm.NewBankIteration(EntryLevelBlessed, b)
		if err != nil {
			return curated.Errorf("grep: %v", err)
		}

		for _, e := bitr.Start(); e != nil; _, e = bitr.Next() {
			// string representation of disasm entry
			l := &bytes.Buffer{}
			dsm.WriteEntry(l, WriteAttr{}, e)

			// limit scope of grep to the correct Instruction field
			switch scope {
			case GrepMnemonic:
				s = e.Mnemonic
			case GrepOperand:
				s = e.String()
			case GrepAll:
				s = l.String()
			}

			if !caseSensitive {
				m = strings.ToUpper(s)
			} else {
				m = s
			}

			if strings.Contains(m, search) {
				// if we've not yet printed head for the current bank then
				// print it now
				if !bankHeader {
					if hasOutput {
						output.Write([]byte("\n"))
					}

					output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", b)))
					bankHeader = true
					hasOutput = true
				}

				// we've matched so print entire l
				output.Write(l.Bytes())
			}
		}
	}

	return nil
}
