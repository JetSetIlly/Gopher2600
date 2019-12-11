package symbols

import (
	"gopher2600/errors"
	"strings"
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
