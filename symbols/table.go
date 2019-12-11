package symbols

import (
	"fmt"
	"gopher2600/hardware/memory/addresses"
	"sort"
	"strings"
)

// Table is the master symbols table for the loaded programme
type Table struct {
	// the master table is made up of three sub-tables
	Locations *subtable
	Read      *subtable
	Write     *subtable

	// use max width values to help with formatting
	MaxLocationWidth int
	MaxSymbolWidth   int
}

// NewTable is the preferred method of initialisation for the Table type. In
// many instances however, ReadSymbolsFile() might be more appropriate. Naked
// initalisation of the Table type (ie. &Table{}) will rarely be useful.
func NewTable() *Table {
	tbl := &Table{}
	tbl.canoniseTable(true)
	return tbl
}

// put canonical symbols into table. prefer flag should be true if canonical
// names are to supercede any existing symbol.
func (tbl *Table) canoniseTable(prefer bool) {
	// loop through the array of canonical names.
	//
	// note that because Read and Write in the addresses package are sparse
	// arrays we need to filter out the empty entries. (the Read and Write
	// structures used to be maps and we didn't need to do this)
	for k, v := range addresses.CanonicalNamesReadAddresses {
		tbl.Read.add(uint16(k), v, prefer)
	}
	for k, v := range addresses.CanonicalNamesWriteAddresses {
		tbl.Write.add(uint16(k), v, prefer)
	}

	tbl.polishTable()
}

// polishTable() should be called whenever any of the sub-tables have been dirtied
func (tbl *Table) polishTable() {
	sort.Sort(tbl.Locations)
	sort.Sort(tbl.Read)
	sort.Sort(tbl.Write)

	// find max symbol width
	tbl.MaxLocationWidth = tbl.Locations.maxWidth
	if tbl.Read.maxWidth > tbl.Write.maxWidth {
		tbl.MaxSymbolWidth = tbl.Read.maxWidth
	} else {
		tbl.MaxSymbolWidth = tbl.Write.maxWidth
	}
}

type subtable struct {
	Symbols  map[uint16]string
	idx      []uint16
	maxWidth int
}

func newTable() *subtable {
	sym := &subtable{
		Symbols: make(map[uint16]string),
		idx:     make([]uint16, 0),
	}
	return sym
}

func (sym subtable) String() string {
	s := strings.Builder{}
	for i := range sym.idx {
		s.WriteString(fmt.Sprintf("%#04x -> %s\n", i, sym.Symbols[sym.idx[i]]))
	}
	return s.String()
}

func (sym *subtable) add(addr uint16, symbol string, prefer bool) {
	// end add procedure with check for max symbol width
	defer func() {
		for _, s := range sym.Symbols {
			if len(s) > sym.maxWidth {
				sym.maxWidth = len(s)
			}
		}
	}()

	// check for duplicates
	for i := range sym.idx {
		if sym.idx[i] == addr {
			// overwrite existing symbol with preferred symbol
			if prefer {
				sym.Symbols[addr] = symbol
			}
			return
		}
	}

	sym.Symbols[addr] = symbol
	sym.idx = append(sym.idx, addr)
	sort.Sort(sym)
}

func (sym subtable) search(symbol string) (uint16, bool) {
	for k, v := range sym.Symbols {
		if strings.ToUpper(v) == symbol {
			return k, true
		}
	}
	return 0, false
}

// Len implements the sort.Interface
func (sym subtable) Len() int {
	return len(sym.idx)
}

// Less implements the sort.Interface
func (sym subtable) Less(i, j int) bool {
	return sym.idx[i] < sym.idx[j]
}

// Swap implements the sort.Interface
func (sym subtable) Swap(i, j int) {
	sym.idx[i], sym.idx[j] = sym.idx[j], sym.idx[i]
}
