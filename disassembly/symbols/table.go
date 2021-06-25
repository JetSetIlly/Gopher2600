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
)

// table maps a symbol to an address. it also keeps track of the widest symbol
// in the table.
type table struct {
	// symbols indexed by address. addresses should be mapped before indexing
	// takes place
	byAddr map[uint16]string

	// sorted array of keys to the byAddr map
	sortedIdx []uint16

	// the longest symbol in the Entries map
	maxWidth int
}

// newTable is the preferred method of initialisation for the table type.
func newTable() *table {
	t := &table{
		byAddr:    make(map[uint16]string),
		sortedIdx: make([]uint16, 0),
	}
	return t
}

func (t *table) calcMaxWidth() {
	t.maxWidth = 0
	for _, s := range t.byAddr {
		if len(s) > t.maxWidth {
			t.maxWidth = len(s)
		}
	}
}

func (t table) String() string {
	s := strings.Builder{}
	for i := range t.sortedIdx {
		s.WriteString(fmt.Sprintf("%#04x -> %s\n", t.sortedIdx[i], t.byAddr[t.sortedIdx[i]]))
	}
	return s.String()
}

// make sure symbols is normalised:
//	no leading or trailing space
//	internal space compressed and replaced with underscores
func (t *table) normaliseSymbol(symbol string) string {
	s := strings.Fields(symbol)
	return strings.Join(s, "_")
}

// make sure symbol is unique in the table.
func (t *table) uniqueSymbol(symbol string) string {
	unique := symbol

	add := 1
	_, _, ok := t.search(unique)
	for ok {
		unique = fmt.Sprintf("%s_%d", symbol, add)
		add++
		_, _, ok = t.search(unique)
	}
	return unique
}

// get entry. address should be mapped before calling according to the context
// of the table.
func (t *table) get(addr uint16) (string, bool) {
	v, ok := t.byAddr[addr]
	return v, ok
}

// add entry. address should be mapped before calling according to the context
// of the table.
func (t *table) add(addr uint16, symbol string) bool {
	symbol = t.normaliseSymbol(symbol)

	// check for duplicates
	for i := range t.sortedIdx {
		if t.sortedIdx[i] == addr {
			return false
		}
	}

	t.byAddr[addr] = t.uniqueSymbol(symbol)
	t.sortedIdx = append(t.sortedIdx, addr)
	sort.Sort(t)
	t.calcMaxWidth()
	return true
}

// remove entry. address should be mapped before calling according to the
// context of the table.
func (t *table) remove(addr uint16) bool {
	if _, ok := t.byAddr[addr]; ok {
		delete(t.byAddr, addr)
		for i := range t.sortedIdx {
			if t.sortedIdx[i] == addr {
				t.sortedIdx = append(t.sortedIdx[:i], t.sortedIdx[i+1:]...)
				sort.Sort(t)
				t.calcMaxWidth()
				return true
			}
		}
		panic("an entry was found in a symbols map but not in the index")
	}

	return false
}

// update entry. address should be mapped before calling according to the
// context of the table.
func (t *table) update(addr uint16, oldSymbol string, newSymbol string) bool {
	oldSymbol = t.normaliseSymbol(oldSymbol)
	newSymbol = t.normaliseSymbol(newSymbol)

	if len(oldSymbol) == 0 || len(newSymbol) == 0 {
		return false
	}

	if oldSymbol == newSymbol {
		return false
	}

	if s, ok := t.byAddr[addr]; ok {
		if s == oldSymbol {
			t.byAddr[addr] = t.uniqueSymbol(newSymbol)
			t.calcMaxWidth()
			return true
		}
	}

	return false
}

func (t table) search(symbol string) (string, uint16, bool) {
	symbol = t.normaliseSymbol(symbol)

	for k, v := range t.byAddr {
		if strings.ToUpper(v) == symbol {
			return v, k, true
		}
	}

	return "", 0, false
}

// Len implements the sort.Interface.
func (t table) Len() int {
	return len(t.sortedIdx)
}

// Less implements the sort.Interface.
func (t table) Less(i, j int) bool {
	return t.sortedIdx[i] < t.sortedIdx[j]
}

// Swap implements the sort.Interface.
func (t table) Swap(i, j int) {
	t.sortedIdx[i], t.sortedIdx[j] = t.sortedIdx[j], t.sortedIdx[i]
}
