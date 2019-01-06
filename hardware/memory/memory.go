package memory

import (
	"fmt"
)

// VCSMemory presents a monolithic representation of system memory to the CPU -
// the CPU only ever access memory through an instance of this structure.
// Other parts of the system access ChipMemory directly
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

	// a note of the last memory location to be accessed
	// this address is the mapped address
	LastAddressAccessFlag  bool
	LastAddressAccessed    uint16
	LastAddressAccessWrite bool
	LastAddressAccessValue uint8
}

// TODO: allow reading only when 02 clock is high and writing when it is low
// ??

// NewVCSMemory is the preferred method of initialisation for VCSMemory
func NewVCSMemory() (*VCSMemory, error) {
	mem := new(VCSMemory)
	mem.Memmap = make(map[uint16]Area)

	mem.RIOT = newRIOT()
	mem.TIA = newTIA()
	mem.PIA = newPIA()
	mem.Cart = newCart()
	if mem.RIOT == nil || mem.TIA == nil || mem.PIA == nil || mem.Cart == nil {
		return nil, fmt.Errorf("error creating memory areas")
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
func (mem VCSMemory) MapAddress(address uint16, cpuPerspective bool) uint16 {
	// note that the order of these filters is important

	// cartridge addresses
	if address&mem.Cart.origin == mem.Cart.origin {
		return address & mem.Cart.memtop
	}

	// RIOT addresses
	if address&mem.RIOT.origin == mem.RIOT.origin {
		if cpuPerspective {
			return address & mem.RIOT.memtop & mem.RIOT.readMask
		}
		return address & mem.RIOT.memtop
	}

	// PIA RAM addresses
	if address&mem.PIA.origin == mem.PIA.origin {
		return address & mem.PIA.memtop
	}

	// everything else is in TIA space
	if cpuPerspective {
		return address & mem.TIA.memtop & mem.TIA.readMask
	}
	return address & mem.TIA.memtop
}

// Implementation of CPUBus.Read
func (mem VCSMemory) Read(address uint16) (uint8, error) {
	ma := mem.MapAddress(address, true)
	area, present := mem.Memmap[ma]
	if !present {
		panic(fmt.Errorf("%04x not mapped correctly", address))
	}
	mem.LastAddressAccessFlag = true
	mem.LastAddressAccessed = ma
	mem.LastAddressAccessWrite = false
	data, err := area.(CPUBus).Read(ma)
	mem.LastAddressAccessValue = data
	return data, err
}

// Implementation of CPUBus.Write
func (mem *VCSMemory) Write(address uint16, data uint8) error {
	ma := mem.MapAddress(address, false)
	area, present := mem.Memmap[ma]
	if !present {
		return fmt.Errorf("%04x not mapped correctly", address)
	}
	mem.LastAddressAccessFlag = true
	mem.LastAddressAccessed = ma
	mem.LastAddressAccessWrite = true
	mem.LastAddressAccessValue = data
	return area.(CPUBus).Write(ma, data)
}
