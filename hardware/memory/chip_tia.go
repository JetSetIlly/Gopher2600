package memory

// newTIA is the preferred method of initialisation for the TIA memory area
func newTIA() *ChipMemory {
	area := new(ChipMemory)
	if area == nil {
		return nil
	}
	area.label = "TIA"
	area.origin = 0x0000
	area.memtop = 0x003f
	area.memory = make([]uint8, area.memtop-area.origin+1)
	area.readMask = 0x000f

	return area
}
