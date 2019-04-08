package disassembly

import (
	"fmt"
	"gopher2600/hardware/cpu/result"
	"io"
	"strings"
)

// Grep searches the disassembly dump for search string. case sensitive
func (dsm *Disassembly) Grep(search string, output io.Writer, caseSensitive bool, contextLines uint) {
	var s, m string

	ctx := make([]string, contextLines)

	if !caseSensitive {
		search = strings.ToUpper(search)
	}

	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		bankHeader := false
		for a := dsm.Cart.Origin(); a <= dsm.Cart.Memtop(); a++ {
			if dsm.program[bank][a] == nil {
				continue
			}

			s = dsm.program[bank][a].GetString(dsm.Symtable, result.StyleBrief)
			if !caseSensitive {
				m = strings.ToUpper(s)
			} else {
				m = s
			}

			if strings.Contains(m, search) {
				if !bankHeader {
					if bank > 0 {
						output.Write([]byte("\n"))
					}

					output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))
					bankHeader = true
				} else if contextLines > 0 {
					output.Write([]byte("\n"))
				}

				// print context
				for c := 0; c < len(ctx); c++ {
					output.Write([]byte(ctx[c]))
					output.Write([]byte("\n"))
				}

				// print match
				output.Write([]byte(s))
				output.Write([]byte("\n"))

				ctx = make([]string, contextLines)
			} else if contextLines > 0 {
				ctx = append(ctx[1:], s)
			}
		}
	}
}
