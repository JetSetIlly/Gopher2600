package memory

// newRIOT is the preferred method of initialisation for the RIOT memory area
func newRIOT() *ChipMemory {
	area := newChipMem()
	if area == nil {
		return nil
	}
	area.label = "RIOT"
	area.origin = 0x0280
	area.memtop = 0x0287
	area.memory = make([]uint8, area.memtop-area.origin+1)
	area.readMask = 0xffff

	return area
}
