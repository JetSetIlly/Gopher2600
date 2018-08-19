package symbols

import (
	"gopher2600/errors"
	"strings"
)

// SearchSymbol return the address of the supplied symbol. search is
// case-insensitive
func (sym *Table) SearchSymbol(symbol string) (string, uint16, error) {
	symbolUpper := strings.ToUpper(symbol)

	for k, v := range sym.Locations {
		if strings.ToUpper(v) == symbolUpper {
			return symbol, k, nil
		}
	}

	for k, v := range sym.ReadSymbols {
		if strings.ToUpper(v) == symbolUpper {
			return v, k, nil
		}
	}

	for k, v := range sym.WriteSymbols {
		if strings.ToUpper(v) == symbolUpper {
			return v, k, nil
		}
	}

	return symbol, 0, errors.NewGopherError(errors.SymbolUnknown, symbol)
}
