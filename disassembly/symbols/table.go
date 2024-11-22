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
	"slices"
	"strings"
)

// Entry records a symbol and the source of its definition.
type Entry struct {
	Address uint16
	Symbol  string
	Source  SymbolSource
}

// table maps a symbol to an address. it also keeps track of the widest symbol
// in the table.
type table struct {
	// symbols indexed by address. addresses should be mapped before indexing
	// takes place
	symbols map[uint16]Entry

	// addresses by symbol. useful when checking for duplicate symbols
	bySymbol map[string]uint16

	// a sorted list of addresses
	index []uint16

	// the longest symbol in the table
	maxWidth int
}

// newTable is the preferred method of initialisation for the table type.
func newTable() *table {
	t := &table{
		symbols:  make(map[uint16]Entry),
		bySymbol: make(map[string]uint16),
		index:    make([]uint16, 0),
	}
	return t
}

// should be called in critical section
func (t *table) sort() {
	// assertion check that byAddr and index are the same length
	if len(t.symbols) != len(t.index) {
		panic("symbol table is inconsistent")
	}
	if len(t.bySymbol) != len(t.index) {
		panic("symbol table is inconsistent")
	}

	slices.Sort(t.index)

	// calculate max width
	t.maxWidth = 0
	for _, e := range t.symbols {
		if len(e.Symbol) > t.maxWidth {
			t.maxWidth = len(e.Symbol)
		}
	}
}

// commandlineTemplate returns a
func (t *table) commandlineTemplate() string {
	var s strings.Builder
	for _, e := range t.symbols {
		s.WriteString(e.Symbol)
		s.WriteString("|")
	}
	return s.String()
}

func (t table) String() string {
	s := strings.Builder{}
	for _, addr := range t.index {
		e := t.symbols[addr]
		s.WriteString(fmt.Sprintf("%#04x -> %s [%s]\n", e.Address, e.Symbol, e.Source))
	}
	return s.String()
}

// make sure symbols is normalised:
//
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
func (t *table) get(addr uint16) (Entry, bool) {
	v, ok := t.symbols[addr]
	return v, ok
}

// add entry. address should be mapped before calling according to the context
// of the table.
func (t *table) add(source SymbolSource, addr uint16, symbol string) bool {
	symbol = t.normaliseSymbol(symbol)

	// check for duplicate
	if _, ok := t.symbols[addr]; ok {
		return false
	}
	if _, ok := t.bySymbol[symbol]; ok {
		return false
	}

	e := Entry{
		Address: addr,
		Source:  source,
		Symbol:  t.uniqueSymbol(symbol),
	}
	t.symbols[addr] = e
	t.bySymbol[e.Symbol] = addr
	t.index = append(t.index, addr)
	return true
}

// remove entry. address should be mapped before calling according to the
// context of the table.
func (t *table) remove(addr uint16) bool {
	if e, ok := t.symbols[addr]; ok {
		delete(t.symbols, addr)
		delete(t.bySymbol, e.Symbol)
		t.index = slices.DeleteFunc(t.index, func(a uint16) bool {
			return a == addr
		})
	}
	return false
}

// update entry. address should be mapped before calling according to the
// context of the table.
func (t *table) update(source SymbolSource, addr uint16, oldSymbol string, newSymbol string) bool {
	oldSymbol = t.normaliseSymbol(oldSymbol)
	newSymbol = t.normaliseSymbol(newSymbol)

	if len(oldSymbol) == 0 || len(newSymbol) == 0 {
		return false
	}

	if oldSymbol == newSymbol {
		return false
	}

	if t.symbols[addr].Symbol == oldSymbol {
		t.remove(addr)
		t.add(source, addr, newSymbol)
		return true
	}

	return false
}

// search is case-insenstiive.
func (t table) search(symbol string) (Entry, uint16, bool) {
	symbol = t.normaliseSymbol(symbol)

	if addr, ok := t.bySymbol[symbol]; ok {
		e := t.symbols[addr]
		return e, e.Address, true
	}

	return Entry{}, 0, false
}
