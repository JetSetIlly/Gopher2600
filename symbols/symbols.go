package symbols

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory/addresses"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type keys []uint16

// Len is the number of elements in the collection
func (k keys) Len() int {
	return len(k)
}

// Less reports whether the element with index i should sort before the element
// with index j
func (k keys) Less(i, j int) bool {
	return k[i] < k[j]
}

// Swap swaps the elements with indexes i and j
func (k keys) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

// Table is the symbols table for the loaded programme
type Table struct {
	Locations    map[uint16]string
	ReadSymbols  map[uint16]string
	WriteSymbols map[uint16]string

	// sorted keys
	locations    keys
	readSymbols  keys
	writeSymbols keys

	MaxLocationWidth int
	MaxSymbolWidth   int
}

// StandardSymbolTable initialises a symbols table using the standard VCS symbols
func StandardSymbolTable() *Table {
	table := new(Table)
	table.ReadSymbols = addresses.Read
	table.WriteSymbols = addresses.Write
	table.genMaxWidths()
	return table
}

// ReadSymbolsFile initialises a symbols table from the symbols file for the
// specified cartridge
func ReadSymbolsFile(cartridgeFilename string) (*Table, error) {
	table := new(Table)
	table.Locations = make(map[uint16]string)
	table.ReadSymbols = make(map[uint16]string)
	table.WriteSymbols = make(map[uint16]string)
	table.locations = make(keys, 0)
	table.readSymbols = make(keys, 0)
	table.writeSymbols = make(keys, 0)

	// prioritise symbols with reference symbols for the VCS. do this in all
	// instances, even if there is an error with the symbols file
	defer func() {
		for k, v := range addresses.Read {
			table.ReadSymbols[k] = v
		}
		for k, v := range addresses.Write {
			table.WriteSymbols[k] = v
		}

		table.genMaxWidths()
	}()

	// try to open symbols file
	symFilename := cartridgeFilename
	ext := path.Ext(symFilename)

	// try to figure out the case of the file extension
	if ext == ".BIN" {
		symFilename = fmt.Sprintf("%s.SYM", symFilename[:len(symFilename)-len(ext)])
	} else {
		symFilename = fmt.Sprintf("%s.sym", symFilename[:len(symFilename)-len(ext)])
	}

	sf, err := os.Open(symFilename)
	if err != nil {
		// if this is the empty cartridge then this error is expected. return
		// the empty symbol table
		if cartridgeFilename == "" {
			return table, nil
		}
		return nil, errors.NewFormattedError(errors.SymbolsFileUnavailable, cartridgeFilename)
	}
	defer func() {
		_ = sf.Close()
	}()

	sym, err := ioutil.ReadAll(sf)
	if err != nil {
		return nil, errors.NewFormattedError(errors.SymbolsFileError, err)
	}
	lines := strings.Split(string(sym), "\n")

	// create new symbols table
	// loop over lines
	for _, ln := range lines {
		// ignore uninteresting lines
		p := strings.Fields(ln)
		if len(p) < 2 || p[0] == "---" {
			continue // for loop
		}

		// get address
		address, err := strconv.ParseUint(p[1], 16, 16)
		if err != nil {
			continue // for loop
		}

		// get symbol
		symbol := p[0]

		if unicode.IsDigit(rune(symbol[0])) {
			// if symbol begins with a number and a period then it is a location symbol
			i := strings.Index(symbol, ".")
			if i == -1 {
				continue // for loop
			}
			table.Locations[uint16(address)] = symbol[i:]
			table.locations = append(table.locations, uint16(address))
		} else {
			// put symbol in table
			table.ReadSymbols[uint16(address)] = symbol
			table.WriteSymbols[uint16(address)] = symbol
			table.readSymbols = append(table.locations, uint16(address))
			table.writeSymbols = append(table.locations, uint16(address))
		}
	}

	sort.Sort(table.locations)
	sort.Sort(table.readSymbols)
	sort.Sort(table.writeSymbols)

	return table, nil
}

// find the widest location and read/write symbol
func (tab *Table) genMaxWidths() {
	for _, s := range tab.Locations {
		if len(s) > tab.MaxLocationWidth {
			tab.MaxLocationWidth = len(s)
		}
	}
	for _, s := range tab.ReadSymbols {
		if len(s) > tab.MaxSymbolWidth {
			tab.MaxSymbolWidth = len(s)
		}
	}
	for _, s := range tab.WriteSymbols {
		if len(s) > tab.MaxSymbolWidth {
			tab.MaxSymbolWidth = len(s)
		}
	}
}
