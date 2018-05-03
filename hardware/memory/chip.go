package memory

// ChipMemory defines the information for and operations allowed for those
// memory areas accessed by the VCS chips as well as the CPU
type ChipMemory struct {
	CPUBus
	ChipBus
	Area
	AreaInfo

	memory []uint8

	// read and write addresses from the perspective of the CPU
	// - links address locations to 'register' names
	// - must be the same length as ChipMemory.memory
	// - empty string means the address is not readable/writable
	readAddresses  []string
	writeAddresses []string

	// additional mask to further reduce address space when read from the CPU
	readMask uint16
}

// Label is an implementation of Area.Label
func (area ChipMemory) Label() string {
	return area.label
}

// Clear is an implementation of CPUBus.Clear
func (area *ChipMemory) Clear() {
	for i := range area.memory {
		area.memory[i] = 0
	}
}

// Implementation of CPUBus.Read
func (area ChipMemory) Read(address uint16) (uint8, error) {
	oa := address - area.origin
	oa &= area.readMask

	rl := area.readAddresses[oa]
	if rl == "" {
		// silently ignore illegal reads
		return 0, nil
	}

	return area.memory[oa], nil
}

// Implementation of CPUBus.Write
func (area *ChipMemory) Write(address uint16, data uint8) error {
	oa := address - area.origin

	rl := area.writeAddresses[oa]
	if rl == "" {
		// silently ignore illegal writes
		return nil
	}

	area.memory[oa] = data

	return nil
}

// NewRIOT is the preferred method of initialisation for the RIOT memory area
func NewRIOT() *ChipMemory {
	chip := new(ChipMemory)
	if chip == nil {
		return nil
	}
	chip.label = "RIOT"
	chip.origin = 0x0280
	chip.memtop = 0x0287
	chip.memory = make([]uint8, chip.memtop-chip.origin+1)
	chip.writeAddresses = []string{"SWCHA", "SWACNT", "", "", "TIM1T", "TIM8T", "TIM64T", "TIM1024"}
	chip.readAddresses = []string{"SWCHA", "SWACNT", "SWCHB", "SWBCNT", "INTIM", "", "", ""}
	chip.readMask = 0xffff
	return chip
}

// NewTIA is the preferred method of initialisation for the TIA memory area
func NewTIA() *ChipMemory {
	chip := new(ChipMemory)
	if chip == nil {
		return nil
	}
	chip.label = "TIA"
	chip.origin = 0x0000
	chip.memtop = 0x003f
	chip.memory = make([]uint8, chip.memtop-chip.origin+1)
	chip.writeAddresses = []string{"VSYNC", "VBLANK", "WSYNC", "RSYNC", "NUSIZ0", "NUSIZ1", "COLUP0", "COLUP1", "COLUPF", "COLUBK", "CTRLPF", "REFP0", "REFP1", "PF0", "PF1", "PF2", "RESP0", "RESP1", "RESM0", "RESM1", "RESBL", "AUDC0", "AUDC1", "AUDF0", "AUDF1", "AUDV0", "AUDV1", "GRP0", "GRP1", "ENAM0", "ENAM1", "ENABL", "HMP0", "HMP1", "HMM0", "HMM1", "HMBL", "VDELP0", "VDELP1", "VDELBL", "RESMP0", "RESMP1", "HMOVE", "HMCLR", "CXCLR", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", ""}
	chip.readAddresses = []string{"CXM0P", "CXM1P", "CXP0FB", "CXP1FB", "CXM0FB", "CXM1FB", "CXBLPF", "CXPPMM", "INPT0", "INPT1", "INPT2", "INPT3", "INPT4", "INPT5", "", ""}
	chip.readMask = 0x000f
	return chip
}
