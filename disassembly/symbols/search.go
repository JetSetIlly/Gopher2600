package symbols

import "gopher2600/errors"

// SearchLocation return the address of the supplied location label
func (sym *Table) SearchLocation(location string) (uint16, error) {
	if sym != nil {
		for k, v := range sym.Locations {
			if v == location {
				return k, nil
			}
		}
	}
	return 0, errors.GopherError{errors.UnknownSymbol, errors.Values{location}}
}
