package memory

import (
	"fmt"
)

// Bus defines the operations for a memory system
type Bus interface {
	Clear()
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

const (
	originTIA  = 0x0000
	memtopTIA  = 0x003f
	originRAM  = 0x0080
	memtopRAM  = 0x00ff
	originRIOT = 0x0280
	memtopRIOT = 0x0287
	originCart = 0x1000
	memtopCart = 0x1fff

	memtopInternal = 0x287
)

// VCSMemory is the implementation of Memory for the VCSMemory
type VCSMemory struct {
	internal  []uint8
	cartridge []uint8
}

// NewVCSMemory is the preferred method of initialisation for VCSMemory
func NewVCSMemory() *VCSMemory {
	mem := new(VCSMemory)
	mem.internal = make([]uint8, memtopInternal)
	return mem
}

// Clear sets all bytes in memory to zero
func (mem *VCSMemory) Clear() {
	for i := 0; i < len(mem.internal); i++ {
		a, _ := mem.MapAddress(uint16(i))
		mem.internal[a] = 0
	}
}

// MapAddress translates a "real" address from mirror space to primary space
func (mem *VCSMemory) MapAddress(address uint16) (uint16, string) {
	// cartridge addresses
	if address&originCart == originCart {
		address &= memtopCart
		return address, "Cartridge"
	}

	// RIOT addresses
	if address&originRIOT == originRIOT {
		address &= memtopRIOT
		return address, "RIOT"
	}

	// PIA addresses
	if address&originRAM == originRAM {
		address &= memtopRAM
		return address, "PIA RAM"
	}

	// everything else is in TIA space

	address &= memtopTIA
	return address, "TIA"
}

func (mem *VCSMemory) Read(address uint16) (uint8, error) {
	address, _ = mem.MapAddress(address)

	if int(address) > len(mem.internal) {
		return 0, fmt.Errorf("address out of range (%d)", address)
	}
	return mem.internal[address], nil
}

func (mem *VCSMemory) Write(address uint16, data uint8) error {
	address, _ = mem.MapAddress(address)

	if int(address) > len(mem.internal) {
		return fmt.Errorf("address out of range (%d)", address)
	}
	mem.internal[address] = data
	return nil
}

// AttachCartridge maps a file to the cartridge addresses
func (mem *VCSMemory) AttachCartridge() {
}
