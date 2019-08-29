package memory

import "gopher2600/hardware/memory/addresses"

// newRIOT is the preferred method of initialisation for the RIOT memory area
func newRIOT() *ChipMemory {
	area := newChipMem()
	area.label = "RIOT"
	area.origin = 0x0280
	area.memtop = 0x0297
	area.memory = make([]uint8, area.memtop-area.origin+1)
	area.cpuReadMask = 0x02f7

	// initial values
	area.memory[addresses.SWCHA] = 0xff
	area.memory[addresses.SWCHB] = 0xff

	return area
}
