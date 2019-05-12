package debugger

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/symbols"
	"strconv"
	"strings"
)

// memoryDebug is a front-end to the real VCS memory. this additional layer
// allows addressing by symbols.
type memoryDebug struct {
	mem *memory.VCSMemory

	// symbols.Table instance can change after we've initialised with
	// newMemoryDebug(), so we need a pointer to a pointer
	symtable **symbols.Table
}

// memoryDebug functions all return an instance of addressInfo. this struct
// contains everything you could possibly usefully know about an address. most
// usefully perhaps, the String() function provides a normalised presentation
// of information.
type addressInfo struct {
	address       uint16
	mappedAddress uint16
	area          memory.Area
	addressLabel  string
	value         memoryValue
}

func (mem addressInfo) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%#04x", mem.address))
	if mem.addressLabel != "" {
		s.WriteString(fmt.Sprintf(" (%s)", mem.addressLabel))
	}
	if mem.address != mem.mappedAddress {
		s.WriteString(fmt.Sprintf(" [mirror of %#04x]", mem.mappedAddress))
	}
	if mem.area.Label() != "" {
		s.WriteString(fmt.Sprintf(" :: %s", mem.area.Label()))
	}
	s.WriteString(mem.value.String())
	return s.String()
}

// the value associated with an address may or may not be available depending
// on how the addressInfo was generated. the assign() and String() functions
// take care of the details.
type memoryValue struct {
	available bool
	value     uint8
}

func (val *memoryValue) assign(value uint8) {
	val.value = value
	val.available = true
}

func (val memoryValue) String() string {
	if !val.available {
		return ""
	}
	return fmt.Sprintf(" -> %#02x", val.value)
}

// mapAddress allows addressing by symbols in addition to numerically
func (mem memoryDebug) mapAddress(address interface{}, cpuRead bool) *addressInfo {
	ai := &addressInfo{}

	var symbolTable map[uint16]string

	if cpuRead {
		symbolTable = (*mem.symtable).ReadSymbols
	} else {
		symbolTable = addresses.Write
	}

	switch address := address.(type) {
	case uint16:
		ai.address = address
		ai.mappedAddress = mem.mem.MapAddress(address, cpuRead)
	case string:
		var found bool
		var err error
		var addr uint64

		// search for symbolic address in standard vcs read symbols
		for a, sym := range symbolTable {
			if sym == address {
				ai.address = a
				ai.mappedAddress = a
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
				ai.mappedAddress = a
				found = true
				break // for loop
			}
		}
		if found {
			break // case switch
		}

		// finally, this may be a string representation of a numerical address
		addr, err = strconv.ParseUint(address, 0, 16)
		if err != nil {
			return nil
		}

		ai.address = uint16(addr)
		ai.mappedAddress = uint16(addr)
		ai.mappedAddress = mem.mem.MapAddress(ai.address, cpuRead)
	}

	var present bool
	ai.area, present = mem.mem.Memmap[ai.mappedAddress]
	if !present {
		return nil
	}

	ai.addressLabel = symbolTable[ai.mappedAddress]

	return ai
}

// Peek returns the contents of the memory address, without triggering any side
// effects. address can be expressed numerically or symbolically.
func (mem memoryDebug) peek(address interface{}) (*addressInfo, error) {
	ai := mem.mapAddress(address, true)
	if ai == nil {
		return nil, errors.NewFormattedError(errors.MemoryError, fmt.Sprintf("%#04x not mapped correctly", address))
	}

	val, err := ai.area.Peek(ai.mappedAddress)
	ai.value.assign(val)
	return ai, err
}

// Poke writes a value at the specified address, which may be numeric or
// symbolic.
func (mem memoryDebug) poke(address interface{}, value uint8) (*addressInfo, error) {
	ai := mem.mapAddress(address, true)
	if ai == nil {
		return nil, errors.NewFormattedError(errors.MemoryError, fmt.Sprintf("%#04x not mapped correctly", address))
	}
	ai.value.assign(value)
	err := ai.area.Poke(ai.mappedAddress, value)
	return ai, err
}
