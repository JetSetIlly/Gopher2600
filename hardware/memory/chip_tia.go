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

	area.cpuWriteRegisters = []string{"VSYNC", "VBLANK", "WSYNC", "RSYNC", "NUSIZ0", "NUSIZ1", "COLUP0", "COLUP1", "COLUPF", "COLUBK", "CTRLPF", "REFP0", "REFP1", "PF0", "PF1", "PF2", "RESP0", "RESP1", "RESM0", "RESM1", "RESBL", "AUDC0", "AUDC1", "AUDF0", "AUDF1", "AUDV0", "AUDV1", "GRP0", "GRP1", "ENAM0", "ENAM1", "ENABL", "HMP0", "HMP1", "HMM0", "HMM1", "HMBL", "VDELP0", "VDELP1", "VDELBL", "RESMP0", "RESMP1", "HMOVE", "HMCLR", "CXCLR", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", ""}
	area.cpuReadRegisters = []string{"CXM0P", "CXM1P", "CXP0FB", "CXP1FB", "CXM0FB", "CXM1FB", "CXBLPF", "CXPPMM", "INPT0", "INPT1", "INPT2", "INPT3", "INPT4", "INPT5", "", ""}

	return area
}

// chip write registers
const (
	CXM0P uint16 = iota
	CXM1P
	CXP0FB
	CXP1FB
	CXM0FB
	CXM1FB
	CXBLPF
	CXPPMM
	INPT0
	INPT1
	INPT2
	INPT3
	INPT4
	INPT5
)
