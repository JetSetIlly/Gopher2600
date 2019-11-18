package memory

import (
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/memorymap"
)

// newRIOT is the preferred method of initialisation for the RIOT memory area
func newRIOT() *ChipMemory {
	area := &ChipMemory{
		origin:      memorymap.OriginRIOT,
		memtop:      memorymap.MemtopRIOT,
		cpuReadMask: memorymap.AddressMaskRIOT,
	}

	// allocation the minimal amount of memory
	area.memory = make([]uint8, area.memtop-area.origin+1)

	// initial values
	area.memory[addresses.SWCHA] = 0xff
	area.memory[addresses.SWCHB] = 0xff

	return area
}
