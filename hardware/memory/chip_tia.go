package memory

import "gopher2600/hardware/memory/addresses"

// newTIA is the preferred method of initialisation for the TIA memory area
func newTIA() *ChipMemory {
	area := newChipMem()
	area.label = "TIA"
	area.origin = 0x0000
	area.memtop = 0x003f
	area.memory = make([]uint8, area.memtop-area.origin+1)
	area.cpuReadMask = 0x000f

	// initial values
	area.memory[addresses.INPT1] = 0x00
	area.memory[addresses.INPT2] = 0x00
	area.memory[addresses.INPT3] = 0x00
	area.memory[addresses.INPT4] = 0x80
	area.memory[addresses.INPT5] = 0x80

	return area
}
