package memory

// newRIOT is the preferred method of initialisation for the RIOT memory area
func newRIOT() *ChipMemory {
	area := new(ChipMemory)
	if area == nil {
		return nil
	}
	area.label = "RIOT"
	area.origin = 0x0280
	area.memtop = 0x0287
	area.memory = make([]uint8, area.memtop-area.origin+1)
	area.readMask = 0xffff

	area.cpuWriteRegisters = []string{"SWCHA", "SWACNT", "", "", "TIM1T", "TIM8T", "TIM64T", "TIM1024"}
	area.cpuReadRegisters = []string{"SWCHA", "SWACNT", "SWCHB", "SWBCNT", "INTIM", "", "", ""}

	// create chipWriteRegisters from cpuReadRegisters
	area.chipWriteRegisters = make(map[string]int)
	if area.chipWriteRegisters == nil {
		return nil
	}
	for i, k := range area.cpuReadRegisters {
		if k != "" {
			area.chipWriteRegisters[k] = i
		}
	}

	return area
}
