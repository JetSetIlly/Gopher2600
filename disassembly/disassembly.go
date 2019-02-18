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

	// table of instruction results. indexed by bank and normalised address
	// -- use Get() and put() functions
	program [](map[uint16]*result.Instruction)
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
	err = dsm.ParseMemory(mem.Cart, symtable)
	if err != nil {
		return dsm, err
	}

	return dsm, nil
}

// Dump writes the entire disassembly to the write interface
func (dsm *Disassembly) Dump(output io.Writer) {
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))
		for a := dsm.Cart.Origin(); a <= dsm.Cart.Memtop(); a++ {
			if dsm.program[bank][a] != nil {
				output.Write([]byte(dsm.program[bank][a].GetString(dsm.Symtable, result.StyleFull)))
				output.Write([]byte("\n"))
			}
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
					output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))
					bankHeader = true
				}
				output.Write([]byte(s))
				output.Write([]byte("\n"))
			}
		}
	}
}

// Get returns the disassembled entry at the specified bank/address
func (dsm Disassembly) Get(bank int, address uint16) (*result.Instruction, bool) {
	v, ok := dsm.program[bank][address&dsm.Cart.Memtop()]
	return v, ok
}

// put stores a disassembled entry - returns false if entry already exists
func (dsm Disassembly) put(bank int, result *result.Instruction) bool {
	if _, ok := dsm.Get(bank, result.Address); ok {
		return false
	}
	dsm.program[bank][result.Address&dsm.Cart.Memtop()] = result
	return true
}
