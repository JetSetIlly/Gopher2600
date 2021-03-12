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
	"strings"
)

// SearchTable is used to select and identify a symbol table when searching.
type SearchTable int

func (t SearchTable) String() string {
	switch t {
	case SearchAll:
		return "unspecified"
	case SearchLabel:
		return "label"
	case SearchRead:
		return "read"
	case SearchWrite:
		return "write"
	}

	return ""
}

// List of valid symbol table identifiers.
const (
	SearchAll SearchTable = iota
	SearchLabel
	SearchRead
	SearchWrite
)

// SearchResults contains the normalised symbol info found in the SearchTable.
type SearchResults struct {
	Table   SearchTable
	Symbol  string
	Address uint16

	// Strict indicates that the address for this symbol is "strict" and should
	// not be further mapped by memorymap.MapAddress()
	Strict bool
}

// Search return the address of the supplied seartch string.
//
// Matching is case-insensitive and when TableType is SearchAll the
// search in order: locations > read > write.
func (sym *Symbols) Search(symbol string, target SearchTable) *SearchResults {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	symbolUpper := strings.ToUpper(symbol)

	if target == SearchAll || target == SearchLabel {
		for _, l := range sym.label {
			if norm, addr, ok := l.search(symbolUpper); ok {
				_, strict := l.strict[addr]
				return &SearchResults{
					Table:   SearchLabel,
					Symbol:  norm,
					Address: addr,
					Strict:  strict,
				}
			}
		}
	}

	if target == SearchAll || target == SearchRead {
		if norm, addr, ok := sym.read.search(symbolUpper); ok {
			_, strict := sym.read.strict[addr]
			return &SearchResults{
				Table:   SearchRead,
				Symbol:  norm,
				Address: addr,
				Strict:  strict,
			}
		}
	}

	if target == SearchAll || target == SearchWrite {
		if norm, addr, ok := sym.write.search(symbolUpper); ok {
			_, strict := sym.write.strict[addr]
			return &SearchResults{
				Table:   SearchWrite,
				Symbol:  norm,
				Address: addr,
				Strict:  strict,
			}
		}
	}

	return nil
}

// ReverseSearch returns the symbol for specified address.
//
// When TableType is SearchAll the search in order: locations > read > write.
func (sym *Symbols) ReverseSearch(addr uint16, target SearchTable) *SearchResults {
	if target == SearchAll || target == SearchLabel {
		for _, l := range sym.label {
			if s, ok := l.entries[addr]; ok {
				_, strict := l.strict[addr]
				return &SearchResults{
					Table:   SearchLabel,
					Symbol:  s,
					Address: addr,
					Strict:  strict,
				}
			}
		}
	}
	if target == SearchAll || target == SearchRead {
		if s, ok := sym.read.entries[addr]; ok {
			_, strict := sym.read.strict[addr]
			return &SearchResults{
				Table:   SearchRead,
				Symbol:  s,
				Address: addr,
				Strict:  strict,
			}
		}
	}
	if target == SearchAll || target == SearchWrite {
		if s, ok := sym.write.entries[addr]; ok {
			_, strict := sym.write.strict[addr]
			return &SearchResults{
				Table:   SearchWrite,
				Symbol:  s,
				Address: addr,
				Strict:  strict,
			}
		}
	}

	return nil
}
