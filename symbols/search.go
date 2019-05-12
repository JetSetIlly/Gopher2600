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
		for k, v := range tab.Locations {
			if strings.ToUpper(v) == symbolUpper {
				return LocationSymTable, symbol, k, nil
			}
		}
	}

	if tType == UnspecifiedSymTable || tType == ReadSymTable {
		for k, v := range tab.ReadSymbols {
			if strings.ToUpper(v) == symbolUpper {
				return ReadSymTable, v, k, nil
			}
		}
	}

	if tType == UnspecifiedSymTable || tType == WriteSymTable {
		for k, v := range tab.WriteSymbols {
			if strings.ToUpper(v) == symbolUpper {
				return WriteSymTable, v, k, nil
			}
		}
	}

	return UnspecifiedSymTable, symbol, 0, errors.NewFormattedError(errors.SymbolUnknown, symbol)
}
