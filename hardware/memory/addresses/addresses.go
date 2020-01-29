// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

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
