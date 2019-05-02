package debugger

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
	"gopher2600/symbols"
	"strconv"
	"strings"
)

// memoryDebug is a front-end to the real VCS memory. this additonal memory
// layer allows addressing by symbols.
type memoryDebug struct {
	mem *memory.VCSMemory

	// symbols.Table instance can change after we've initialised with
	// newMemoryDebug(), so we need a pointer to a pointer
	symtable **symbols.Table
}

// mapAddress allows addressing by symbols in addition to numerically
func (mem memoryDebug) mapAddress(address interface{}, cpuPerspective bool) (uint16, error) {
	var mapped bool
	var ma uint16
	var symbolTable map[uint16]string

	if cpuPerspective {
		symbolTable = (*mem.symtable).ReadSymbols
	} else {
		symbolTable = vcssymbols.WriteSymbols
	}

	switch address := address.(type) {
	case uint16:
		ma = mem.mem.MapAddress(address, true)
	case string:
		// search for symbolic address in standard vcs read symbols
		for a, sym := range symbolTable {
			if sym == address {
				ma = a
				mapped = true
				break // for loop
			}
		}
		if mapped {
			break // case switch
		}

		// try again with an uppercase label
		address = strings.ToUpper(address)
		for a, sym := range symbolTable {
			if sym == address {
				ma = a
				mapped = true
				break // for loop
			}
		}
		if mapped {
			break // case switch
		}

		// finally, this may be a string representation of a numerical address
		na, err := strconv.ParseUint(address, 0, 16)
		if err == nil {
			ma = uint16(na)
			ma = mem.mem.MapAddress(ma, true)
			mapped = true
		}

		if !mapped {
			return 0, errors.NewFormattedError(errors.UnrecognisedAddress, address)
		}
	}

	return ma, nil
}

// Peek returns the contents of the memory address, without triggering any side
// effects. returns:
//  o value
//  o mapped address
//  o area name
//  o address label
//  o error
func (mem memoryDebug) peek(address interface{}) (uint8, uint16, string, string, error) {
	ma, err := mem.mapAddress(address, true)
	if err != nil {
		return 0, 0, "", "", err
	}

	area, present := mem.mem.Memmap[ma]
	if !present {
		panic(fmt.Sprintf("%04x not mapped correctly", address))
	}

	return area.Peek(ma)
}

// Poke writes a value at the address
func (mem memoryDebug) poke(address interface{}, value uint8) error {
	ma, err := mem.mapAddress(address, true)
	if err != nil {
		return err
	}

	area, present := mem.mem.Memmap[ma]
	if !present {
		panic(fmt.Sprintf("%04x not mapped correctly", address))
	}

	return area.Poke(ma, value)
}
