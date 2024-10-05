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
	"errors"
	"fmt"
	"strconv"

	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// DbgMem is a front-end to the real VCS memory. it allows addressing by
// symbol name and uses the AddressInfo type for easier presentation.
type DbgMem struct {
	VCS *hardware.VCS
	Sym *symbols.Symbols
}

// GetAddressInfo allows addressing by symbols in addition to numerically.
func (dbgmem DbgMem) GetAddressInfo(address any, read bool) *AddressInfo {
	ai := &AddressInfo{Read: read}

	var searchTable symbols.SearchTable

	if read {
		searchTable = symbols.SearchRead
	} else {
		searchTable = symbols.SearchWrite
	}

	switch address := address.(type) {
	case uint16:
		ai.Address = address
		res := dbgmem.Sym.SearchByAddress(ai.Address, searchTable)
		if res == nil {
			ai.MappedAddress, ai.Area = memorymap.MapAddress(ai.Address, read)
			res := dbgmem.Sym.SearchByAddress(ai.MappedAddress, searchTable)
			if res != nil {
				ai.Symbol = res.Entry.Symbol
			}
		} else {
			ai.MappedAddress, ai.Area = memorymap.MapAddress(ai.Address, read)
			ai.Symbol = res.Entry.Symbol
		}
	case string:
		var err error

		res := dbgmem.Sym.SearchBySymbol(address, searchTable)
		if res != nil {
			ai.Address = res.Address
			ai.Symbol = res.Entry.Symbol
			ai.MappedAddress, ai.Area = memorymap.MapAddress(ai.Address, read)
		} else {
			// this may be a string representation of a numerical address
			var addr uint64

			addr, err = strconv.ParseUint(address, 0, 16)
			if err != nil {
				return nil
			}

			ai.Address = uint16(addr)
			res := dbgmem.Sym.SearchByAddress(ai.Address, searchTable)
			if res == nil {
				ai.MappedAddress, ai.Area = memorymap.MapAddress(ai.Address, read)
				res := dbgmem.Sym.SearchByAddress(ai.MappedAddress, searchTable)
				if res != nil {
					ai.Symbol = res.Entry.Symbol
				}
			} else {
				ai.MappedAddress, ai.Area = memorymap.MapAddress(ai.Address, read)
				ai.Symbol = res.Entry.Symbol
			}
		}
	default:
		panic(fmt.Sprintf("unsupported address type (%T)", address))
	}

	return ai
}

// sentinal errors returns by Peek() and Poke()
var PeekError = errors.New("cannot peek address")
var PokeError = errors.New("cannot poke address")

// Peek returns the contents of the memory address, without triggering any side
// effects. The supplied address can be numeric of symbolic.
func (dbgmem DbgMem) Peek(address any) (*AddressInfo, error) {
	ai := dbgmem.GetAddressInfo(address, true)
	if ai == nil {
		return nil, fmt.Errorf("%w: %v", PeekError, address)
	}

	area := dbgmem.VCS.Mem.GetArea(ai.Area)

	var err error
	ai.Data, err = area.Peek(ai.MappedAddress)
	if err != nil {
		if errors.Is(err, cpubus.AddressError) {
			return nil, fmt.Errorf("%w: %v", PeekError, address)
		}
		return nil, err
	}

	ai.Peeked = true

	return ai, nil
}

// Poke writes a value at the specified address. The supplied address be
// numeric or symbolic.
func (dbgmem DbgMem) Poke(address any, data uint8) (*AddressInfo, error) {
	// although the words "read" and "write" might lead us to think that we
	// "peek" from "read" addresses and "poke" to "write" addresses, it is in
	// fact necessary to treat "poke" addresses as "read" addresses
	//
	// on the surface this doesn't appear to be correct but on further thought
	// it is obviously true - we are in fact changing the value that is
	// subsequently read by the CPU, so that means poking to a read address
	ai := dbgmem.GetAddressInfo(address, true)
	if ai == nil {
		return nil, fmt.Errorf("%w: %v", PokeError, address)
	}

	area := dbgmem.VCS.Mem.GetArea(ai.Area)

	err := area.Poke(ai.MappedAddress, data)
	if err != nil {
		if errors.Is(err, cpubus.AddressError) {
			return nil, fmt.Errorf("%w: %v", PokeError, address)
		}
		return nil, err
	}

	ai.Data = data
	ai.Peeked = true

	return ai, err
}
