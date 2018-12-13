package debugger

import (
	"fmt"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
)

// memoryDebug is a front-end to the real VCS memory
type memoryDebug struct {
	vcsmem      *memory.VCSMemory
	disassembly *disassembly.Disassembly
}

// mapAddress is like the MapAddress function in the VCS.memory package but
// this accepts symbols as well as numeric addresses
func (mem memoryDebug) mapAddress(address interface{}, readAddress bool) (uint16, error) {
	var mapped bool
	var ma uint16
	var symbolTable map[uint16]string

	if readAddress {
		symbolTable = vcssymbols.ReadSymbols
	} else {
		symbolTable = vcssymbols.WriteSymbols
	}

	switch address := address.(type) {
	case uint16:
		ma = mem.vcsmem.MapAddress(uint16(address), true)
		mapped = true
	case string:
		// search for symbolic address in standard vcs read symbols
		// TODO: peeking of cartridge specific symbols
		for a, sym := range symbolTable {
			if sym == address {
				ma = a
				mapped = true
				break // for loop
			}
		}
		mapped = true
	}

	if !mapped {
		return 0, errors.NewGopherError(errors.UnrecognisedAddress, address)
	}

	return ma, nil
}

func newMemoryDebug(dbg *Debugger) *memoryDebug {
	memdbg := new(memoryDebug)
	memdbg.vcsmem = dbg.vcs.Mem
	memdbg.disassembly = dbg.disasm
	return memdbg
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

	area, present := mem.vcsmem.Memmap[ma]
	if !present {
		panic(fmt.Errorf("%04x not mapped correctly", address))
	}

	return area.(memory.Area).Peek(ma)
}

// Poke writes a value at the address
func (mem memoryDebug) poke(address interface{}, value uint8) error {
	ma, err := mem.mapAddress(address, true)
	if err != nil {
		return err
	}

	return mem.vcsmem.Memmap[ma].Poke(ma, value)
}
