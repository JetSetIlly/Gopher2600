package symbols

import (
	"gopher2600/errors"
	"strings"
)

// TableType is used to select and identify a symbol table
// when searching
type TableType int

// list of valid symbol tables
const (
	UnspecifiedSymTable TableType = iota
	LocationSymTable
	ReadSymTable
	WriteSymTable
)

// SearchSymbol return the address of the supplied symbol. search is
// case-insensitive
func (tab *Table) SearchSymbol(symbol string, tType TableType) (TableType, string, uint16, error) {
	symbolUpper := strings.ToUpper(symbol)

	if tType == UnspecifiedSymTable || tType == LocationSymTable {
		if addr, ok := tab.Locations.search(symbolUpper); ok {
			return LocationSymTable, symbol, addr, nil
		}
	}

	if tType == UnspecifiedSymTable || tType == ReadSymTable {
		if addr, ok := tab.Read.search(symbolUpper); ok {
			return ReadSymTable, symbol, addr, nil
		}
	}

	if tType == UnspecifiedSymTable || tType == WriteSymTable {
		if addr, ok := tab.Write.search(symbolUpper); ok {
			return WriteSymTable, symbol, addr, nil
		}
	}

	return UnspecifiedSymTable, symbol, 0, errors.NewFormattedError(errors.SymbolUnknown, symbol)
}
