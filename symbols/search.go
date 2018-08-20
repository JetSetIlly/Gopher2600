package symbols

import (
	"gopher2600/errors"
	"strings"
)

// TableID is used to select and identify a symbol table
type TableID int

// list of valid symbol tables
const (
	UnspecifiedSymTable TableID = iota
	LocationSymTable
	ReadSymTable
	WriteSymTable
)

// SearchSymbol return the address of the supplied symbol. search is
// case-insensitive
func (sym *Table) SearchSymbol(symbol string, table TableID) (TableID, string, uint16, error) {
	symbolUpper := strings.ToUpper(symbol)

	if table == UnspecifiedSymTable || table == LocationSymTable {
		for k, v := range sym.Locations {
			if strings.ToUpper(v) == symbolUpper {
				return LocationSymTable, symbol, k, nil
			}
		}
	}

	if table == UnspecifiedSymTable || table == ReadSymTable {
		for k, v := range sym.ReadSymbols {
			if strings.ToUpper(v) == symbolUpper {
				return ReadSymTable, v, k, nil
			}
		}
	}

	if table == UnspecifiedSymTable || table == WriteSymTable {
		for k, v := range sym.WriteSymbols {
			if strings.ToUpper(v) == symbolUpper {
				return WriteSymTable, v, k, nil
			}
		}
	}

	return UnspecifiedSymTable, symbol, 0, errors.NewGopherError(errors.SymbolUnknown, symbol)
}
