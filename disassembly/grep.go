package disassembly

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// GrepScope limits the scope of the search
type GrepScope int

// List of available scopes
const (
	GrepMnemonic GrepScope = iota
	GrepOperand
	GrepAll
)

// Grep searches the disassembly for the specified search string.
func (dsm *Disassembly) Grep(output io.Writer, scope GrepScope, search string, caseSensitive bool) {
	var s, m string

	if !caseSensitive {
		search = strings.ToUpper(search)
	}

	for bank := 0; bank < len(dsm.flow); bank++ {
		bankHeader := false
		for a := 0; a < len(dsm.flow[bank]); a++ {
			d := dsm.flow[bank][a]

			if d != nil {

				// line representation of Instruction. we'll print this
				// in case of a match
				line := &bytes.Buffer{}
				dsm.WriteLine(line, false, d)

				// limit scope of grep to the correct Instruction field
				switch scope {
				case GrepMnemonic:
					s = d.Mnemonic
				case GrepOperand:
					s = d.Operand
				case GrepAll:
					s = line.String()
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
						if bank > 0 {
							output.Write([]byte("\n"))
						}

						output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))
						bankHeader = true
					}

					// we've matched so print entire line
					output.Write(line.Bytes())
				}
			}
		}
	}
}
