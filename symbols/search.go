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

// Search return the address of the supplied symbol.
//
// Matching is case-insensitive and when TableType is UnspecifiedSymTable the
// search in order: locations > read > write.
//
// Returns success, the table in which it was found, the normalised symbol, and
// the normalised address.
func (sym *Symbols) Search(symbol string, target TableType) (bool, TableType, string, uint16) {
	sym.crit.Lock()
	defer sym.crit.Unlock()

	symbolUpper := strings.ToUpper(symbol)

	if target == UnspecifiedSymTable || target == LabelTable {
		for _, l := range sym.label {
			if symbolNorm, addr, ok := l.search(symbolUpper); ok {
				return true, LabelTable, symbolNorm, addr
			}
		}
	}

	if target == UnspecifiedSymTable || target == ReadSymTable {
		if symbolNorm, addr, ok := sym.read.search(symbolUpper); ok {
			return true, ReadSymTable, symbolNorm, addr
		}
	}

	if target == UnspecifiedSymTable || target == WriteSymTable {
		if symbolNorm, addr, ok := sym.write.search(symbolUpper); ok {
			return true, WriteSymTable, symbolNorm, addr
		}
	}

	return false, UnspecifiedSymTable, symbol, 0
}
