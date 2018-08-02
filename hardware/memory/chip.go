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
	PeriphBus

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

	// the periphQueue is used to write to chip memory in a goroutine friendly
	// manner. peripherals can be implemented with goroutines and so we need to
	// be careful when accessing memory.
	periphQueue chan *periphPayload
}

func newChipMem() *ChipMemory {
	area := new(ChipMemory)
	area.periphQueue = make(chan *periphPayload, periphQueueLen)
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

// Peek is the implementation of Area.Peek. returns:
func (area ChipMemory) Peek(address uint16) (uint8, uint16, string, string, error) {
	sym := vcssymbols.ReadSymbols[address&area.readMask]
	if sym == "" {
		return 0, 0, "", "", errors.GopherError{Errno: errors.UnreadableAddress, Values: nil}
	}
	return area.memory[address-area.origin], address & area.readMask, area.Label(), sym, nil
}
