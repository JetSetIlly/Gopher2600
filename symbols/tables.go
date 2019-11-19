package symbols

import (
	"fmt"
	"sort"
	"strings"
)

// Table is the symbols table for the loaded programme
type Table struct {
	Locations *table
	Read      *table
	Write     *table

	MaxLocationWidth int
	MaxSymbolWidth   int
}

type table struct {
	Symbols  map[uint16]string
	idx      []uint16
	maxWidth int
}

func newTable() *table {
	tb := &table{
		Symbols: make(map[uint16]string),
		idx:     make([]uint16, 0),
	}
	return tb
}

func (tb table) String() string {
	s := strings.Builder{}
	for i := range tb.idx {
		s.WriteString(fmt.Sprintf("%#04x -> %s\n", i, tb.Symbols[tb.idx[i]]))
	}
	return s.String()
}

func (tb *table) add(addr uint16, symbol string, prefer bool) {
	// end add procedure with check for max symbol width
	defer func() {
		for _, s := range tb.Symbols {
			if len(s) > tb.maxWidth {
				tb.maxWidth = len(s)
			}
		}
	}()

	// check for duplicates
	for i := range tb.idx {
		if tb.idx[i] == addr {
			// overwrite existing symbol with preferred symbol
			if prefer {
				tb.Symbols[addr] = symbol
			}
			return
		}
	}

	tb.Symbols[addr] = symbol
	tb.idx = append(tb.idx, addr)
	sort.Sort(tb)
}

func (tb table) search(symbol string) (uint16, bool) {
	for k, v := range tb.Symbols {
		if strings.ToUpper(v) == symbol {
			return k, true
		}
	}
	return 0, false
}

// Len is the number of elements in the collection
func (tb table) Len() int {
	return len(tb.idx)
}

// Less reports whether the element with index i should sort before the element
// with index j
func (tb table) Less(i, j int) bool {
	return tb.idx[i] < tb.idx[j]
}

// Swap swaps the elements with indexes i and j
func (tb table) Swap(i, j int) {
	tb.idx[i], tb.idx[j] = tb.idx[j], tb.idx[i]
}
