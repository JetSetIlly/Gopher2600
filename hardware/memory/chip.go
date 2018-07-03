package memory

import (
	"gopher2600/errors"
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

	// read and write addresses from the perspective of the CPU
	// - links address locations to 'register' names
	// - must be the same length as ChipMemory.memory
	// - empty string means the address is not readable/writable
	cpuReadRegisters  []string
	cpuWriteRegisters []string

	// write addresses from the perspective of the VCS Chips should use the
	// chip area specific enumerations

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
	oa := address - area.origin
	oa &= area.readMask

	// note the name of the register that we are reading
	area.lastReadRegister = area.cpuReadRegisters[oa]

	rl := area.cpuReadRegisters[oa]
	if rl == "" {
		// silently ignore illegal reads (we're definitely reading from the correct
		// memory space but some registers are not readable)
		return 0, nil
	}

	return area.memory[oa], nil
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
		return errors.GopherError{errors.UnservicedChipWrite, errors.Values{area.cpuWriteRegisters[area.lastWriteAddress]}}
	}

	oa := address - area.origin
	rl := area.cpuWriteRegisters[oa]
	if rl == "" {
		// silently ignore illegal writes (we're definitely writing to the correct
		// memory space but some registers are not writable)
		return nil
	}

	// note address of write
	area.lastWriteAddress = oa
	area.writeSignal = true
	area.writeData = data

	//area.memory[oa] = data

	return nil
}

// ChipRead is an implementation of ChipBus.ChipRead. returns:
// - whether a chip was last written to
// - the CPU name of the address that was written to
// - the written value
func (area *ChipMemory) ChipRead() (bool, string, uint8) {
	if area.writeSignal {
		area.writeSignal = false
		return true, area.cpuWriteRegisters[area.lastWriteAddress], area.writeData
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
// - the value in memory
// - the register name of the address
// - any errors
func (area ChipMemory) Peek(address uint16) (uint8, string, error) {
	oa := address - area.origin
	oa &= area.readMask

	rl := area.cpuReadRegisters[oa]
	if rl == "" {
		return 0, "", errors.GopherError{errors.UnreadableAddress, nil}
	}
	return area.memory[oa], rl, nil
}
