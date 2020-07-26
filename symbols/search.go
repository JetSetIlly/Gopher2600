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

	"github.com/jetsetilly/gopher2600/errors"
)

// TableType is used to select and identify a symbol table
// when searching
type TableType int

func (t TableType) String() string {
	switch t {
	case UnspecifiedSymTable:
		return "unspecified"
	case LocationSymTable:
		return "location"
	case ReadSymTable:
		return "read"
	case WriteSymTable:
		return "write"
	}

	return ""
}

// List of valid symbol table identifiers
const (
	UnspecifiedSymTable TableType = iota
	LocationSymTable
	ReadSymTable
	WriteSymTable
)

// SearchSymbol return the address of the supplied symbol. Search is
// case-insensitive and is conducted on the subtables in order: locations >
// read > write.
func (tbl *Table) SearchSymbol(symbol string, target TableType) (TableType, string, uint16, error) {
	symbolUpper := strings.ToUpper(symbol)

	if target == UnspecifiedSymTable || target == LocationSymTable {
		if addr, ok := tbl.Locations.search(symbolUpper); ok {
			return LocationSymTable, symbol, addr, nil
		}
	}

	if target == UnspecifiedSymTable || target == ReadSymTable {
		if addr, ok := tbl.Read.search(symbolUpper); ok {
			return ReadSymTable, symbol, addr, nil
		}
	}

	if target == UnspecifiedSymTable || target == WriteSymTable {
		if addr, ok := tbl.Write.search(symbolUpper); ok {
			return WriteSymTable, symbol, addr, nil
		}
	}

	return UnspecifiedSymTable, symbol, 0, errors.New(errors.SymbolUnknown, symbol)
}
