package disassembly

import (
	"fmt"
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

	for bank := 0; bank < len(dsm.flow); bank++ {
		bankHeader := false
		for a := 0; a < len(dsm.flow[bank]); a++ {
			entry := dsm.flow[bank][a]

			if entry.instructionDefinition != nil {
				s = entry.instruction
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
						// only write actual content. note that there is more often
						// than not, valid context. the main reason as far I can
						// see, for empty context are mistakes in disassembly
						if ctx[c] != "" {
							output.Write([]byte(ctx[c]))
							output.Write([]byte("\n"))
						}
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
}
