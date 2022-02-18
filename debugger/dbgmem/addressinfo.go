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

package dbgmem

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// AddressInfo is returned by dbgmem functions. This type contains everything
// you could possibly usefully know about an address. Most usefully perhaps,
// the String() function provides a normalised presentation of information.
type AddressInfo struct {
	Address       uint16
	MappedAddress uint16
	Symbol        string
	Area          memorymap.Area

	// addresses and symbols are mapped differently depending on whether
	// address is to be used for reading or writing
	Read bool

	// the data at the address. if peeked is false then data mays not be valid
	Peeked bool
	Data   uint8
}

func (ai AddressInfo) String() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("%#04x", ai.Address))

	if ai.Symbol != "" {
		s.WriteString(fmt.Sprintf(" (%s)", ai.Symbol))
	}

	if ai.Address != ai.MappedAddress {
		s.WriteString(fmt.Sprintf(" [mirror of %#04x]", ai.MappedAddress))
	}

	s.WriteString(fmt.Sprintf(" (%s)", ai.Area.String()))

	if ai.Peeked {
		s.WriteString(fmt.Sprintf(" -> %#02x", ai.Data))
	}

	return s.String()
}

// StringNoSymbol is the same as String() but it does not print the
// AddressSymbol field. Useful in some contexts were the symbol is printed in
// some other way.
func (ai AddressInfo) StringNoSymbol() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("%#04x", ai.Address))

	if ai.Address != ai.MappedAddress {
		s.WriteString(fmt.Sprintf(" [mirror of %#04x]", ai.MappedAddress))
	}

	s.WriteString(fmt.Sprintf(" (%s)", ai.Area.String()))

	if ai.Peeked {
		s.WriteString(fmt.Sprintf(" -> %#02x", ai.Data))
	}

	return s.String()
}
