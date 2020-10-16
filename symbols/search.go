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

// TableType is used to select and identify a symbol table
// when searching.
type TableType int

func (t TableType) String() string {
	switch t {
	case UnspecifiedSymTable:
		return "unspecified"
	case LabelTable:
		return "label"
	case ReadSymTable:
		return "read"
	case WriteSymTable:
		return "write"
	}

	return ""
}

// List of valid symbol table identifiers.
const (
	UnspecifiedSymTable TableType = iota
	LabelTable
	ReadSymTable
	WriteSymTable
)

// SearchSymbol return the address of the supplied symbol. Search is
// case-insensitive and is conducted on the subtables in order: locations >
// read > write.
func (sym *Symbols) Search(symbol string, target TableType) (bool, TableType, string, uint16) {
	symbolUpper := strings.ToUpper(symbol)

	if target == UnspecifiedSymTable || target == LabelTable {
		if addr, ok := sym.Label.search(symbolUpper); ok {
			return true, LabelTable, symbol, addr
		}
	}

	if target == UnspecifiedSymTable || target == ReadSymTable {
		if addr, ok := sym.Read.search(symbolUpper); ok {
			return true, ReadSymTable, symbol, addr
		}
	}

	if target == UnspecifiedSymTable || target == WriteSymTable {
		if addr, ok := sym.Write.search(symbolUpper); ok {
			return true, WriteSymTable, symbol, addr
		}
	}

	return false, UnspecifiedSymTable, symbol, 0
}
