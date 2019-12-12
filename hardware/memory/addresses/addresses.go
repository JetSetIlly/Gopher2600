package addresses

// Reset is the address where the reset address is stored
// - used by VCS.Reset() and Disassembly module
const Reset = uint16(0xfffc)

// IRQ is the address where the interrupt address is stored
const IRQ = uint16(0xfffe)

// CanonicalReadSymbols list all the writable addresses along with the
// canonical names for those addresses. We don't use this structure in the
// emulation because the map structure introduces an overhead that we'd like to
// avoid. We do however use it to create a more suitable structure for
// emulation.
var CanonicalReadSymbols = map[uint16]string{
	// TIA
	0x00: "CXM0P",
	0x01: "CXM1P",
	0x02: "CXP0FB",
	0x03: "CXP1FB",
	0x04: "CXM0FB",
	0x05: "CXM1FB",
	0x06: "CXBLPF",
	0x07: "CXPPMM",
	0x08: "INPT0",
	0x09: "INPT1",
	0x0a: "INPT2",
	0x0b: "INPT3",
	0x0c: "INPT4",
	0x0d: "INPT5",

	// RIOT
	0x0280: "SWCHA",
	0x0281: "SWACNT",
	0x0282: "SWCHB",
	0x0283: "SWBCNT",
	0x0284: "INTIM",
	0x0285: "TIMINT",
}

// CanonicalWriteSymbols list all the writable addresses along with the
// canonical names for those addresses. (see above for commentary)
var CanonicalWriteSymbols = map[uint16]string{
	// TIA
	0x00: "VSYNC",
	0x01: "VBLANK",
	0x02: "WSYNC",
	0x03: "RSYNC",
	0x04: "NUSIZ0",
	0x05: "NUSIZ1",
	0x06: "COLUP0",
	0x07: "COLUP1",
	0x08: "COLUPF",
	0x09: "COLUBK",
	0x0a: "CTRLPF",
	0x0b: "REFP0",
	0x0c: "REFP1",
	0x0d: "PF0",
	0x0e: "PF1",
	0x0f: "PF2",
	0x10: "RESP0",
	0x11: "RESP1",
	0x12: "RESM0",
	0x13: "RESM1",
	0x14: "RESBL",
	0x15: "AUDC0",
	0x16: "AUDC1",
	0x17: "AUDF0",
	0x18: "AUDF1",
	0x19: "AUDV0",
	0x1a: "AUDV1",
	0x1b: "GRP0",
	0x1c: "GRP1",
	0x1d: "ENAM0",
	0x1e: "ENAM1",
	0x1f: "ENABL",
	0x20: "HMP0",
	0x21: "HMP1",
	0x22: "HMM0",
	0x23: "HMM1",
	0x24: "HMBL",
	0x25: "VDELP0",
	0x26: "VDELP1",
	0x27: "VDELBL",
	0x28: "RESMP0",
	0x29: "RESMP1",
	0x2A: "HMOVE",
	0x2B: "HMCLR",
	0x2C: "CXCLR",

	// RIOT
	0x0280: "SWCHA",
	0x0281: "SWACNT",
	0x0294: "TIM1T",
	0x0295: "TIM8T",
	0x0296: "TIM64T",
	0x0297: "TIM1024",
}

// Read is a sparse array containing the canonical labels for VCS read
// addresses. If the address is not named (empty string) then the address is
// not reabable
var Read []string

// Write is a sparse array containing the canonical labels for VCS write
// addresses. If the address is not named (empty string) then the address is
// not writable
var Write []string

// this init() function create the Read/Write arrays using the read/write maps
// as a source
func init() {
	// we know that the maximum address either chip can read or write to is
	// 0x297, in RIOT memory space. we can say this is the extent of our Read
	// and Write sparse arrays
	const chipTop = 0x297

	Read = make([]string, chipTop+1)
	for k, v := range CanonicalReadSymbols {
		Read[k] = v
	}

	Write = make([]string, chipTop+1)
	for k, v := range CanonicalWriteSymbols {
		Write[k] = v
	}
}

// Named TIA registers
//
// These value are used by the emulator to specifiy known addresses. For
// example, when writing collision information we know we need the CXM0P
// register. these named values make the code more readable
//
// For simplicity values are enumerated from 0; value is added to the origin
// address of the TIA in ChipBus.ChipWrite implementation
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

// Named RIOT registers
//
// These value are used by the emulator to specifiy known addresses. For
// example, the timer updates itself every cycle and stores time remaining
// value in the INTIM register.
//
// For simplicity values are enumerated from 0; value is added to the origin
// address of the TIA in ChipBus.ChipWrite implementation
const (
	SWCHA uint16 = iota
	SWACNT
	SWCHB
	SWBCNT
	INTIM
	TIMINT
)

// DataMasks are applied to data read by the CPU from lowest 16 addresses. This
// requirement is a consequence of how the address/data bus works in the VCS.
//
// For example, if the CPU wants to read the contents of the CXM1P register, it
// can use the address 0x0d to do so.
//
//		LDA 0x01
//
// If there are no collisions (between missile 1 and either player, in this
// case) than the value of the most significant bits are zero. The lower six
// bits are not part of the CXM1P register and are left undefined by the TIA
// when the data is put on the bus. The lower bits of the LDA operation are in
// fact "left over" from the address. In our example, the lowest six bits are
//
//		0bxx000001
//
// meaning the the returned data is in fact 0x01 and not 0x00, as you might
// expect.  Things get interesting when we use mirrored addresses. If instead
// of 0x01 we used the mirror address 0x11, the lowest six bits are:
//
//		0bxx01001
//
// meaning that the returned value is 0x11 and not (again, as you might expect)
// 0x00 or even 0x01.
//
// So what happens if there is sprite collision information in the register?
// Meaning that the top bits are not necessarily zero. Let's say there is a
// collusion between missile 1 and player 0, the data before masking will be
//
//		0b01000000
//
// If we used address 0x11 to load this value, we would in fact, get this
// pattern (0x51 in hex):
//
//		0b01010001
//
// Now, if all ROMs read and interpreted chip registers only as they're
// supposed to (defails in the 2600 programmer's guide) then none of this would
// matter but some ROMs do make use of the extra bits, and so we must account
// for it in emulation.
//
// It's worth noting that the above is implicitely talking about zero-page
// addressing; but masking also occurs with regular two-byte addressing. The
// key to understanding is that the masking is applied to the most recent byte
// of the address to be put on the address bus*. In all cases, this is the
// most-significant byte. So, if the requested address is 0x171, the bit
// pattern for the address is:
//
//		0x0000000101110001
//
// the most significant byte in this pattern is 0x00000001 and so the data
// retreived is AND-ed with that. The mapped address for 0x171 incidentally, is
// 0x01, which is the CXM1P register also used in the examples above.
//
var DataMasks = []uint8{
	0b11000000, // CXM0P
	0b11000000, // CXM1P
	0b11000000, // CXP0FB
	0b11000000, // CXP1FB
	0b11000000, // CXM0FB
	0b11000000, // CXM1FB

	// event though legitimate usage of CXBLPF suggests only the most
	// significant bit is used, for the purposes of masking it acts just like
	// the other collision registers
	0b11000000, // CXBLPF

	0b11000000, // CXPPMM
	0b10000000, // INPT0
	0b10000000, // INPT1
	0b10000000, // INPT2
	0b10000000, // INPT3
	0b10000000, // INPT4
	0b10000000, // INPT5

	// the contents of the last two locations are "undefined" according to the
	// Stella Programmer's Guide but are readable anyway. we can see through
	// experiementation that the mask is as follows (details of what we
	// experimented with has been forgotten)
	0b11000000,
	0b11000000,
}
