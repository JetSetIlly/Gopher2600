package symbols

// MaxVCSSymbolWidth is the maxiumum number of characters in the standard
// symbol tables below. provided for convenience.
const MaxVCSSymbolWidth = 7

// VCSReadSymbols are the canonical names for VCS read addresses
var VCSReadSymbols = map[uint16]string{
	0x00:  "CXM0P",
	0x01:  "CXM1P",
	0x02:  "CXP0F",
	0x03:  "CXP1F",
	0x04:  "CXM0F",
	0x05:  "CXM1F",
	0x06:  "CXBLP",
	0x07:  "CXPPM",
	0x08:  "INPT0",
	0x09:  "INPT1",
	0x0A:  "INPT2",
	0x0B:  "INPT3",
	0x0C:  "INPT4",
	0x0D:  "INPT5",
	0x280: "SWCHA",
	0x281: "SWACNT",
	0x282: "SWCHB",
	0x283: "SWBCNT",
	0x284: "INTIM",
}

// VCSWriteSymbols are the canonical names for VCS write addresses
var VCSWriteSymbols = map[uint16]string{
	0x00:  "VSYNC",
	0x01:  "VBLANK",
	0x02:  "WSYNC",
	0x03:  "RSYNC",
	0x04:  "NUSIZ0",
	0x05:  "NUSIZ1",
	0x06:  "COLUP0",
	0x07:  "COLUP1",
	0x08:  "COLUPF",
	0x09:  "COLUBK",
	0x0A:  "CTRLPF",
	0x0B:  "REFP0",
	0x0C:  "REFP1",
	0x0D:  "PF0",
	0x0E:  "PF1",
	0x0F:  "PF2",
	0x10:  "RESP0",
	0x11:  "RESP1",
	0x12:  "RESM0",
	0x13:  "RESM1",
	0x14:  "RESBL",
	0x15:  "AUDC0",
	0x16:  "AUDC1",
	0x17:  "AUDF0",
	0x18:  "AUDF1",
	0x19:  "AUDV0",
	0x1A:  "AUDV1",
	0x1B:  "GRP0",
	0x1C:  "GRP1",
	0x1D:  "ENAM0",
	0x1E:  "ENAM1",
	0x1F:  "ENABL",
	0x20:  "HMP0",
	0x21:  "HMP1",
	0x22:  "HMM0",
	0x23:  "HMM1",
	0x24:  "HMBL",
	0x25:  "VDELP0",
	0x26:  "VDELP1",
	0x27:  "VDELBL",
	0x28:  "RESMP0",
	0x29:  "RESMP1",
	0x2A:  "HMOVE",
	0x2B:  "HMCLR",
	0x2C:  "CXCLR",
	0x280: "SWCHA",
	0x281: "SWACNT",
	0x284: "TIM1T",
	0x285: "TIM8T",
	0x286: "TIM64T",
	0x287: "TIM1024",
}

// tia write registers
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

// riot write registers
const (
	SWCHA uint16 = iota
	SWACNT
	SWCHB
	SWBCNT
	INTIM
)
