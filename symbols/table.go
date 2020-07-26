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
	"sort"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

// Table is the master symbols table for the loaded programme
type Table struct {
	// the master table is made up of three sub-tables
	Locations *symTable
	Read      *symTable
	Write     *symTable

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
	for k, v := range addresses.CanonicalReadSymbols {
		tbl.Read.add(uint16(k), v, prefer)
	}
	for k, v := range addresses.CanonicalWriteSymbols {
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

type symTable struct {
	Symbols  map[uint16]string
	idx      []uint16
	maxWidth int
}

func newTable() *symTable {
	sym := &symTable{
		Symbols: make(map[uint16]string),
		idx:     make([]uint16, 0),
	}
	return sym
}

func (sym symTable) String() string {
	s := strings.Builder{}
	for i := range sym.idx {
		s.WriteString(fmt.Sprintf("%#04x -> %s\n", sym.idx[i], sym.Symbols[sym.idx[i]]))
	}
	return s.String()
}

func (sym *symTable) add(addr uint16, symbol string, prefer bool) {
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

func (sym symTable) search(symbol string) (uint16, bool) {
	for k, v := range sym.Symbols {
		if strings.ToUpper(v) == symbol {
			return k, true
		}
	}
	return 0, false
}

// Len implements the sort.Interface
func (sym symTable) Len() int {
	return len(sym.idx)
}

// Less implements the sort.Interface
func (sym symTable) Less(i, j int) bool {
	return sym.idx[i] < sym.idx[j]
}

// Swap implements the sort.Interface
func (sym symTable) Swap(i, j int) {
	sym.idx[i], sym.idx[j] = sym.idx[j], sym.idx[i]
}
