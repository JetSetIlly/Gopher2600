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

	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// SearchTable is used to select and identify a symbol table when searching.
type SearchTable string

// List of valid symbol table identifiers.
const (
	SearchLabel SearchTable = "label"
	SearchRead  SearchTable = "read"
	SearchWrite SearchTable = "write"
)

// SearchResults contains the normalised symbol/address info found in the
// requested SearchTable.
type SearchResults struct {
	// the table the result was found in
	Table SearchTable

	// the symbol as it exists in the table
	Symbol string

	// the normalised address the symbol refers to
	Address uint16
}

// SearchBySymbol return the address of the supplied search string. Matching is
// case-insensitive.
func (sym *Symbols) SearchBySymbol(symbol string, table SearchTable) *SearchResults {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	symbolUpper := strings.TrimSpace(strings.ToUpper(symbol))

	switch table {
	case SearchLabel:
		for _, l := range sym.label {
			if norm, addr, ok := l.search(symbolUpper); ok {
				return &SearchResults{
					Table:   SearchLabel,
					Symbol:  norm,
					Address: addr,
				}
			}
		}
	case SearchRead:
		if norm, addr, ok := sym.read.search(symbolUpper); ok {
			return &SearchResults{
				Table:   SearchRead,
				Symbol:  norm,
				Address: addr,
			}
		}
	case SearchWrite:
		if norm, addr, ok := sym.write.search(symbolUpper); ok {
			return &SearchResults{
				Table:   SearchWrite,
				Symbol:  norm,
				Address: addr,
			}
		}
	}

	return nil
}

// SearchByAddress returns the symbol for specified address. Address will be
// normalised before search.
func (sym *Symbols) SearchByAddress(addr uint16, table SearchTable) *SearchResults {
	switch table {
	case SearchLabel:
		addr, _ = memorymap.MapAddress(addr, true)
		for _, l := range sym.label {
			if s, ok := l.byAddr[addr]; ok {
				return &SearchResults{
					Table:   SearchLabel,
					Symbol:  s,
					Address: addr,
				}
			}
		}
	case SearchRead:
		addr, _ = memorymap.MapAddress(addr, true)
		if s, ok := sym.read.byAddr[addr]; ok {
			return &SearchResults{
				Table:   SearchRead,
				Symbol:  s,
				Address: addr,
			}
		}
	case SearchWrite:
		addr, _ = memorymap.MapAddress(addr, false)
		if s, ok := sym.write.byAddr[addr]; ok {
			return &SearchResults{
				Table:   SearchWrite,
				Symbol:  s,
				Address: addr,
			}
		}
	}

	return nil
}
