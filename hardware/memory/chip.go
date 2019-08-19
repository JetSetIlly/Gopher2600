package memory

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory/addresses"
)

// ChipMemory defines the information for and operations allowed for those
// memory areas accessed by the VCS chips as well as the CPU
type ChipMemory struct {
	CPUBus
	ChipBus
	PeriphBus

	Area
	AreaInfo

	memory []uint8

	// additional mask to further reduce address space when read from the CPU
	cpuReadMask uint16

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

func newChipMem() *ChipMemory {
	area := new(ChipMemory)
	return area
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

// Peek is the implementation of Memory.Area.Peek. returns:
func (area ChipMemory) Peek(address uint16) (uint8, error) {
	sym := addresses.Read[address]
	if sym == "" {
		return 0, errors.NewFormattedError(errors.UnreadableAddress, address)
	}
	return area.memory[address-area.origin], nil
}

// Poke is the implementation of Memory.Area.Poke
func (area ChipMemory) Poke(address uint16, value uint8) error {
	return errors.NewFormattedError(errors.UnpokeableAddress, address)
}

// ChipRead is an implementation of ChipBus. returns:
// - whether a chip was last written to
// - the CPU name of the address that was written to
// - the written value
func (area *ChipMemory) ChipRead() (bool, string, uint8) {
	if area.writeSignal {
		area.writeSignal = false
		return true, addresses.Write[area.lastWriteAddress], area.writeData
	}

	return false, "", 0
}

// ChipWrite is an implementation of ChipBus. it writes the data to the memory
// area's address
func (area *ChipMemory) ChipWrite(address uint16, data uint8) {
	area.memory[address] = data
}

// LastReadRegister is an implementation of ChipBus. it returns the register
// name of the last memory location *read* by the CPU
func (area *ChipMemory) LastReadRegister() string {
	r := area.lastReadRegister
	area.lastReadRegister = ""
	return r
}

// Read is an implementation of CPUBus. returns the value and/or error
func (area *ChipMemory) Read(address uint16) (uint8, error) {
	// note the name of the register that we are reading
	area.lastReadRegister = addresses.Read[address]

	sym := addresses.Read[address]
	if sym == "" {
		return 0, errors.NewFormattedError(errors.UnreadableAddress, address)
	}

	return area.memory[area.origin|address^area.origin], nil
}

// Write is an implementation of CPUBus. it writes the data to the memory
// area's address
func (area *ChipMemory) Write(address uint16, data uint8) error {
	// check that the last write to this memory area has been serviced
	if area.writeSignal {
		return errors.NewFormattedError(errors.MemoryError, fmt.Sprintf("unserviced write to chip memory (%s)", addresses.Write[area.lastWriteAddress]))
	}

	sym := addresses.Write[address]
	if sym == "" {
		return errors.NewFormattedError(errors.UnwritableAddress, address)
	}

	// note address of write
	area.lastWriteAddress = address
	area.writeSignal = true
	area.writeData = data

	return nil
}

// PeriphWrite implements PeriphBus. it writes the data to the memory area's
// address
func (area *ChipMemory) PeriphWrite(address uint16, data uint8) {
	area.memory[address] = data
}
