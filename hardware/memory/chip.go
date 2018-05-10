package memory

import "fmt"

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

	// when the CPU writes to chip memory it is not just writing to memory in the
	// way we might expect. instead we note the address that has been written to,
	// and a boolean true to indicate that a write has been performed by the CPU
	lastWriteAddress uint16 // normalised
	writeSignal      bool
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
		// silently ignore illegal reads (we're definitely reading from the correct
		// memory space but some registers are not readable)
		return 0, nil
	}

	return area.memory[oa], nil
}

// Implementation of CPUBus.Write
func (area *ChipMemory) Write(address uint16, data uint8) error {
	// check that the last write to this memory area has been serviced TODO:
	// we'll only be notified of an unserviced write signal if the chip memory is
	// written to again. byt the CPU theoretically, this may never happen so we
	// should consider implementing a "tick" function that is called every
	// machine cycle to perform the sanity check. on the other hand it does seem
	// unlikely for a program never to write to chip memory on a more-or-less
	// frequent basis
	if area.writeSignal != false {
		panic(fmt.Sprintf("chip memory write signal has not been serviced since previous write [%s]", area.writeAddresses[area.lastWriteAddress]))
	}

	oa := address - area.origin
	rl := area.writeAddresses[oa]
	if rl == "" {
		// silently ignore illegal reads (we're definitely writing to the correct
		// memory space but some registers are not writable)
		return nil
	}
	area.memory[oa] = data

	// note address of write
	area.lastWriteAddress = oa
	area.writeSignal = true

	return nil
}

// ChipRead is an implementation of ChipBus.ChipRead
func (area *ChipMemory) ChipRead() (bool, string, uint8) {
	if area.writeSignal == true {
		area.writeSignal = false
		return true, area.writeAddresses[area.lastWriteAddress], area.memory[area.lastWriteAddress]
	}
	return false, "", 0
}

// newRIOT is the preferred method of initialisation for the RIOT memory area
func newRIOT() *ChipMemory {
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

// newTIA is the preferred method of initialisation for the TIA memory area
func newTIA() *ChipMemory {
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
