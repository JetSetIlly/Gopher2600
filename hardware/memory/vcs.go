package memory

import "fmt"

// VCSMemory presents a monolithic representation of system memory to the CPU -
// the CPU only ever access memory through an instance of this structure.
// Other parts of the system access ChipMemory directly
type VCSMemory struct {
	CPUBus
	memmap map[uint16]Area
	riot   *ChipMemory
	tia    *ChipMemory
	pia    *PIA
	Cart   *Cartridge
}

// NewVCSMemory is the preferred method of initialisation for VCSMemory
func NewVCSMemory() *VCSMemory {
	mem := new(VCSMemory)
	mem.memmap = make(map[uint16]Area)
	mem.riot = NewRIOT()
	mem.tia = NewTIA()
	mem.pia = NewPIA()
	mem.Cart = NewCart()

	// create the memory map; each address in the memory map points to the
	// memory area it resides in. we only record 'primary' addresses; all
	// addresses should be  passed through the MapAddress() function in order
	// to iron out any mirrors

	var i uint16

	for i = mem.tia.origin; i <= mem.tia.memtop; i++ {
		mem.memmap[i] = mem.tia
	}

	for i = mem.pia.origin; i <= mem.pia.memtop; i++ {
		mem.memmap[i] = mem.pia
	}

	for i = mem.riot.origin; i <= mem.riot.memtop; i++ {
		mem.memmap[i] = mem.riot
	}

	for i = mem.Cart.origin; i <= mem.Cart.memtop; i++ {
		mem.memmap[i] = mem.Cart
	}

	return mem
}

func (mem VCSMemory) String() string {
	return mem.MemoryMap()
}

// MapAddress translates the quoted address from mirror space to primary space.
// Generally, all access to the different memory areas should be passed through
// this function. Any other information about an address can be accessed
// through mem.memmap[mappedAddress]
func (mem VCSMemory) MapAddress(address uint16) uint16 {

	// note that the order of these filters is important

	// cartridge addresses
	if address&mem.Cart.origin == mem.Cart.origin {
		return address & mem.Cart.memtop
	}

	// RIOT addresses
	if address&mem.riot.origin == mem.riot.origin {
		return address & mem.riot.memtop
	}

	// PIA RAM addresses
	if address&mem.pia.origin == mem.pia.origin {
		return address & mem.pia.memtop
	}

	// everything else is in TIA space
	return address & mem.tia.memtop
}

// Clear is an implementation of CPUBus.Clear
func (mem *VCSMemory) Clear() {
	mem.riot.Clear()
	mem.tia.Clear()
	mem.pia.Clear()
	mem.Cart.Clear()
}

// Implementation of CPUBus.Read
func (mem VCSMemory) Read(address uint16) (uint8, error) {
	ma := mem.MapAddress(address)
	area, present := mem.memmap[ma]
	if !present {
		return 0, fmt.Errorf("%04x not mapped correctly", address)
	}
	return area.(CPUBus).Read(ma)
}

// Implementation of CPUBus.Write
func (mem *VCSMemory) Write(address uint16, data uint8) error {
	ma := mem.MapAddress(address)
	area, present := mem.memmap[ma]
	if !present {
		return fmt.Errorf("%04x not mapped correctly", address)
	}
	return area.(CPUBus).Write(ma, data)
}
