package disassembly

import (
	"fmt"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
	"io"
	"strings"
)

// Disassembly represents the annotated disassembly of a 6502 binary
type Disassembly struct {
	Cart *memory.Cartridge

	// symbols used to build disassembly output
	Symtable *symbols.Table

	// sequencePoints contains the list of program counter values. listed in
	// order so can be used to index program map to produce complete
	// disassembly
	sequencePoints [][]uint16

	// table of instruction results. index with contents of sequencePoints
	Program [](map[uint16]*result.Instruction)
}

// NewDisassembly initialises a new partial emulation and returns a
// disassembly from the supplied cartridge filename. - useful for one-shot
// disassemblies, like the gopher2600 "disasm" mode
func NewDisassembly(cartridgeFilename string) (*Disassembly, error) {
	// ignore errors caused by loading of symbols table
	symtable, err := symbols.ReadSymbolsFile(cartridgeFilename)
	if err != nil {
		fmt.Println(err)
		symtable = symbols.StandardSymbolTable()
	}

	mem, err := memory.NewVCSMemory()
	if err != nil {
		return nil, err
	}

	err = mem.Cart.Attach(cartridgeFilename)
	if err != nil {
		return nil, err
	}

	dsm := new(Disassembly)
	err = dsm.ParseMemory(mem, symtable)
	if err != nil {
		return dsm, err
	}

	return dsm, nil
}

// Dump writes the entire disassembly to the write interface
func (dsm *Disassembly) Dump(output io.Writer) {
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))
		for _, pc := range dsm.sequencePoints[bank] {
			output.Write([]byte(dsm.Program[bank][pc].GetString(dsm.Symtable, result.StyleFull)))
			output.Write([]byte("\n"))
		}
	}
}

// Grep searches the disassembly dump for search string. case sensitive
func (dsm *Disassembly) Grep(search string, output io.Writer, caseSensitive bool) {
	var s, m string

	if !caseSensitive {
		search = strings.ToUpper(search)
	}

	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		bankHeader := false
		for _, pc := range dsm.sequencePoints[bank] {
			s = dsm.Program[bank][pc].GetString(dsm.Symtable, result.StyleBrief)
			if !caseSensitive {
				m = strings.ToUpper(s)
			} else {
				m = s
			}

			if strings.Contains(m, search) {
				if !bankHeader {
					output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))
					bankHeader = true
				}
				output.Write([]byte(s))
				output.Write([]byte("\n"))
			}
		}
	}
}
