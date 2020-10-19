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

package debugger

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/symbols"
)

// memoryDebug is a front-end to the real VCS memory. it allows addressing by
// symbol name and uses the addressInfo type for easier presentation.
type memoryDebug struct {
	vcs     *hardware.VCS
	symbols *symbols.Symbols
}

// memoryDebug functions all return an instance of addressInfo. this struct
// contains everything you could possibly usefully know about an address. most
// usefully perhaps, the String() function provides a normalised presentation
// of information.
type addressInfo struct {
	address       uint16
	mappedAddress uint16
	addressLabel  string
	area          memorymap.Area

	// addresses and symbols are mapped differently depending on whether
	// address is to be used for reading or writing
	read bool

	// the data at the address. if peeked is false then data mays not be valid
	peeked bool
	data   uint8
}

func (ai addressInfo) String() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("%#04x", ai.address))

	if ai.addressLabel != "" {
		s.WriteString(fmt.Sprintf(" (%s)", ai.addressLabel))
	}

	if ai.address != ai.mappedAddress {
		s.WriteString(fmt.Sprintf(" [mirror of %#04x]", ai.mappedAddress))
	}

	s.WriteString(fmt.Sprintf(" (%s)", ai.area.String()))

	if ai.peeked {
		s.WriteString(fmt.Sprintf(" -> %#02x", ai.data))
	}

	return s.String()
}

// mapAddress allows addressing by symbols in addition to numerically.
func (dbgmem memoryDebug) mapAddress(address interface{}, read bool) *addressInfo {
	ai := &addressInfo{read: read}

	var symbolTable map[uint16]string

	if read {
		symbolTable = (dbgmem.symbols).Read.Entries
	} else {
		symbolTable = (dbgmem.symbols).Write.Entries
	}

	switch address := address.(type) {
	case uint16:
		ai.address = address
		ai.mappedAddress, ai.area = memorymap.MapAddress(address, read)
	case string:
		var found bool
		var err error
		var addr uint64

		// case sensitive
		for a, sym := range symbolTable {
			if sym == address {
				ai.address = a
				ai.mappedAddress, ai.area = memorymap.MapAddress(ai.address, read)
				found = true
				break // for loop
			}
		}
		if found {
			break // case switch
		}

		// case insensitive
		address = strings.ToUpper(address)
		for a, sym := range symbolTable {
			if strings.ToUpper(sym) == address {
				ai.address = a
				ai.mappedAddress, ai.area = memorymap.MapAddress(ai.address, read)
				found = true
				break // for loop
			}
		}

		if !found {
			// finally, this may be a string representation of a numerical address
			addr, err = strconv.ParseUint(address, 0, 16)
			if err != nil {
				return nil
			}

			ai.address = uint16(addr)
			ai.mappedAddress, ai.area = memorymap.MapAddress(ai.address, read)
		}
	default:
		panic(fmt.Sprintf("unsupported address type (%T)", address))
	}

	ai.addressLabel = symbolTable[ai.mappedAddress]

	return ai
}

// poke/peek error formatting, for consistency.
const (
	pokeError = "cannot poke address (%v)"
	peekError = "cannot peek address (%v)"
)

// Peek returns the contents of the memory address, without triggering any side
// effects. address can be expressed numerically or symbolically.
func (dbgmem memoryDebug) peek(address interface{}) (*addressInfo, error) {
	ai := dbgmem.mapAddress(address, true)
	if ai == nil {
		return nil, curated.Errorf(peekError, address)
	}

	area := dbgmem.vcs.Mem.GetArea(ai.area)

	var err error
	ai.data, err = area.Peek(ai.mappedAddress)
	if err != nil {
		if curated.Is(err, bus.AddressError) {
			return nil, curated.Errorf(peekError, address)
		}
		return nil, err
	}

	ai.peeked = true

	return ai, nil
}

// Poke writes a value at the specified address, which may be numeric or
// symbolic.
func (dbgmem memoryDebug) poke(address interface{}, data uint8) (*addressInfo, error) {
	ai := dbgmem.mapAddress(address, false)
	if ai == nil {
		return nil, curated.Errorf(pokeError, address)
	}

	area := dbgmem.vcs.Mem.GetArea(ai.area)

	err := area.Poke(ai.mappedAddress, data)
	if err != nil {
		if curated.Is(err, bus.AddressError) {
			return nil, curated.Errorf(pokeError, address)
		}
		return nil, err
	}

	ai.data = data
	ai.peeked = true

	return ai, err
}
