package memory

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory/vcssymbols"
)

// ChipMemory defines the information for and operations allowed for those
// memory areas accessed by the VCS chips as well as the CPU
type ChipMemory struct {
	CPUBus
	ChipBus
	Area
	AreaInfo

	memory []uint8

	// additional mask to further reduce address space when read from the CPU
	readMask uint16

	// when the CPU writes to chip memory it is not writing to memory in the
	// way we might expect. instead we note the address that has been written
	// to, and a boolean true to indicate that a write has been performed by
	// the CPU
	lastWriteAddress uint16 // mapped from 16bit to chip address length
	writeData        uint8
	writeSignal      bool

	// lastReadRegister works slightly different that lastWriteAddress. it stores
	// the register *name* of the last memory location *read* by the CPU
	lastReadRegister string
}

// note that all the symbols used are the standard VCS symbol names, as defined
// in the symbols package. this may be confusing if the cartridge has a symbols
// file that does not use standard names, but this seems unlikely.

// Label is an implementation of Area.Label
func (area ChipMemory) Label() string {
	return area.label
}

// Origin is an implementation of Area.Origin
func (area ChipMemory) Origin() uint16 {
	return area.origin
}

// Memtop is an implementation of Area.Memtop
func (area ChipMemory) Memtop() uint16 {
	return area.memtop
}

// Implementation of CPUBus.Read
func (area *ChipMemory) Read(address uint16) (uint8, error) {
	address &= area.readMask

	// note the name of the register that we are reading
	area.lastReadRegister = vcssymbols.ReadSymbols[address]

	sym := vcssymbols.ReadSymbols[address]
	if sym == "" {
		// silently ignore illegal reads (we're definitely reading from the correct
		// memory space but some registers are not readable)
		return 0, nil
	}

	return area.memory[address-area.origin], nil
}

// Implementation of CPUBus.Write
func (area *ChipMemory) Write(address uint16, data uint8) error {
	// check that the last write to this memory area has been serviced TODO:
	// we'll only be notified of an unserviced write signal if the chip memory is
	// written to again. byt the CPU theoretically, this may never happen so we
	// should consider implementing a "tick" function that is called every
	// machine cycle to perform the sanity check. on the other hand it does seem
	// unlikely for a program never to write to chip memory on a more-or-less
	// frequent basis
	if area.writeSignal {
		return errors.GopherError{errors.UnservicedChipWrite, errors.Values{vcssymbols.WriteSymbols[area.lastWriteAddress]}}
	}

	sym := vcssymbols.WriteSymbols[address]
	if sym == "" {
		// silently ignore illegal writes (we're definitely writing to the correct
		// memory space but some registers are not writable)
		return nil
	}

	// note address of write
	area.lastWriteAddress = address
	area.writeSignal = true
	area.writeData = data

	return nil
}

// ChipRead is an implementation of ChipBus.ChipRead. returns:
// - whether a chip was last written to
// - the CPU name of the address that was written to
// - the written value
func (area *ChipMemory) ChipRead() (bool, string, uint8) {
	if area.writeSignal {
		area.writeSignal = false
		return true, vcssymbols.WriteSymbols[area.lastWriteAddress], area.writeData
	}
	return false, "", 0
}

// ChipWrite writes the data to the memory area's address specified by
// registerName
func (area *ChipMemory) ChipWrite(address uint16, data uint8) {
	area.memory[address] = data
}

// LastReadRegister returns the register name of the last memory
// location *read* by the CPU
func (area ChipMemory) LastReadRegister() string {
	return area.lastReadRegister
}

// Peek is the implementation of Area.Peek. returns:
func (area ChipMemory) Peek(address uint16) (uint8, uint16, string, string, error) {
	sym := vcssymbols.ReadSymbols[address&area.readMask]
	if sym == "" {
		return 0, 0, "", "", errors.GopherError{errors.UnreadableAddress, nil}
	}
	return area.memory[address-area.origin], address & area.readMask, area.Label(), sym, nil
}
