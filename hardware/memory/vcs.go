package memory

import (
	"fmt"
	"gopher2600/errors"
	"time"
)

// VCSMemory presents a monolithic representation of system memory to the CPU -
// the CPU only ever access memory through an instance of this structure.
// Other parts of the system access chip memory through the a ChipBus
type VCSMemory struct {
	CPUBus

	// memmap is a hash for every address in the VCS address space, returning
	// one of the four memory areas
	Memmap map[uint16]Area

	// the four memory areas
	RIOT *ChipMemory
	TIA  *ChipMemory
	PIA  *PIA
	Cart *Cartridge

	// the following are only used by the debugging interface. it would be
	// lovely to remove these for non-debugging emulation but there's not much
	// impact on performance so they can stay for now.
	//  * a note of the last (mapperd) memory address to be accessed
	//  * the value that was written/read from the last address accessed
	//  * whether the last addres accessed was written or read
	//  * the timestamp of the last memory access
	LastAccessAddress   uint16
	LastAccessValue     uint8
	LastAccessWrite     bool
	LastAccessTimeStamp time.Time
}

// NewVCSMemory is the preferred method of initialisation for VCSMemory
func NewVCSMemory() (*VCSMemory, error) {
	mem := new(VCSMemory)
	mem.Memmap = make(map[uint16]Area)

	mem.RIOT = newRIOT()
	mem.TIA = newTIA()
	mem.PIA = newPIA()
	mem.Cart = NewCartridge()
	if mem.RIOT == nil || mem.TIA == nil || mem.PIA == nil || mem.Cart == nil {
		return nil, errors.NewFormattedError(errors.MemoryError, "cannot create memory areas")
	}

	// create the memory map; each address in the memory map points to the
	// memory area it resides in. we only record 'primary' addresses; all
	// addresses should be passed through the MapAddress() function in order
	// to iron out any mirrors
	for i := mem.TIA.origin; i <= mem.TIA.memtop; i++ {
		mem.Memmap[i] = mem.TIA
	}
	for i := mem.PIA.origin; i <= mem.PIA.memtop; i++ {
		mem.Memmap[i] = mem.PIA
	}
	for i := mem.RIOT.origin; i <= mem.RIOT.memtop; i++ {
		mem.Memmap[i] = mem.RIOT
	}
	for i := mem.Cart.origin; i <= mem.Cart.memtop; i++ {
		mem.Memmap[i] = mem.Cart
	}

	return mem, nil
}

// MapAddress translates the quoted address from mirror space to primary space.
// Generally, all access to the different memory areas should be passed through
// this function. Any other information about an address can be accessed
// through mem.Memmap[mappedAddress]
func (mem VCSMemory) MapAddress(address uint16, cpuRead bool) uint16 {
	// note that the order of these filters is important

	// cartridge addresses
	if address&mem.Cart.origin == mem.Cart.origin {
		return address & mem.Cart.memtop
	}

	// RIOT addresses
	if address&mem.RIOT.origin == mem.RIOT.origin {
		if cpuRead {
			return address & mem.RIOT.memtop & mem.RIOT.cpuReadMask
		}
		return address & mem.RIOT.memtop
	}

	// PIA RAM addresses
	if address&mem.PIA.origin == mem.PIA.origin {
		return address & mem.PIA.memtop
	}

	// everything else is in TIA space
	if cpuRead {
		return address & mem.TIA.memtop & mem.TIA.cpuReadMask
	}
	return address & mem.TIA.memtop
}

// Implementation of CPUBus.Read
func (mem VCSMemory) Read(address uint16) (uint8, error) {
	ma := mem.MapAddress(address, true)
	area, present := mem.Memmap[ma]
	if !present {
		return 0, errors.NewFormattedError(errors.MemoryError, fmt.Sprintf("address %#04x not mapped correctly", address))
	}
	mem.LastAccessAddress = ma
	mem.LastAccessWrite = false
	data, err := area.(CPUBus).Read(ma)
	mem.LastAccessValue = data
	mem.LastAccessTimeStamp = time.Now()
	return data, err
}

// Implementation of CPUBus.Write
func (mem *VCSMemory) Write(address uint16, data uint8) error {
	ma := mem.MapAddress(address, false)
	area, present := mem.Memmap[ma]
	if !present {
		return errors.NewFormattedError(errors.MemoryError, fmt.Sprintf("address %#04x not mapped correctly", address))
	}
	mem.LastAccessAddress = ma
	mem.LastAccessWrite = true
	mem.LastAccessValue = data
	mem.LastAccessTimeStamp = time.Now()
	return area.(CPUBus).Write(ma, data)
}
