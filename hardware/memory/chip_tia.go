package memory

import (
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/memorymap"
)

// newTIA is the preferred method of initialisation for the TIA memory area
func newTIA() *ChipMemory {
	area := &ChipMemory{
		origin:      memorymap.OriginTIA,
		memtop:      memorymap.MemtopTIA,
		cpuReadMask: memorymap.AddressMaskTIA,
	}

	// allocation the minimal amount of memory
	area.memory = make([]uint8, area.memtop-area.origin+1)

	// initial values
	area.memory[addresses.INPT1] = 0x00
	area.memory[addresses.INPT2] = 0x00
	area.memory[addresses.INPT3] = 0x00
	area.memory[addresses.INPT4] = 0x80
	area.memory[addresses.INPT5] = 0x80

	return area
}
