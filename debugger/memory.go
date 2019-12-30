package debugger

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/memorymap"
	"gopher2600/symbols"
	"strconv"
	"strings"
)

// memoryDebug is a front-end to the real VCS memory. it allows addressing by
// symbol name and uses the addressInfo type for easier presentation
type memoryDebug struct {
	mem      *memory.VCSMemory
	symtable *symbols.Table
}

// memoryDebug functions all return an instance of addressInfo. this struct
// contains everything you could possibly usefully know about an address. most
// usefully perhaps, the String() function provides a normalised presentation
// of information
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

// mapAddress allows addressing by symbols in addition to numerically
func (dbgmem memoryDebug) mapAddress(address interface{}, read bool) *addressInfo {
	ai := &addressInfo{read: read}

	var symbolTable map[uint16]string

	if read {
		symbolTable = (dbgmem.symtable).Read.Symbols
	} else {
		symbolTable = (dbgmem.symtable).Write.Symbols
	}

	switch address := address.(type) {
	case uint16:
		ai.address = address
		ai.mappedAddress, ai.area = memorymap.MapAddress(address, read)
	case string:
		var found bool
		var err error
		var addr uint64

		// search for symbolic address in standard vcs read symbols
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

		// try again with an uppercase label
		address = strings.ToUpper(address)
		for a, sym := range symbolTable {
			if sym == address {
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
	}

	ai.addressLabel = symbolTable[ai.mappedAddress]

	return ai
}

// Peek returns the contents of the memory address, without triggering any side
// effects. address can be expressed numerically or symbolically.
func (dbgmem memoryDebug) peek(address interface{}) (*addressInfo, error) {
	ai := dbgmem.mapAddress(address, true)
	if ai == nil {
		return nil, errors.New(errors.DebuggerError, errors.New(errors.UnpeekableAddress, address))
	}

	ar, err := dbgmem.mem.GetArea(ai.area)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}

	ai.data, err = ar.Peek(ai.mappedAddress)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}

	ai.peeked = true

	return ai, err
}

// Poke writes a value at the specified address, which may be numeric or
// symbolic.
func (dbgmem memoryDebug) poke(address interface{}, data uint8) (*addressInfo, error) {
	ai := dbgmem.mapAddress(address, false)
	if ai == nil {
		return nil, errors.New(errors.DebuggerError, errors.New(errors.UnpokeableAddress, address))
	}

	ar, err := dbgmem.mem.GetArea(ai.area)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}

	err = ar.Poke(ai.mappedAddress, data)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}

	ai.data = data
	ai.peeked = true

	return ai, err
}
