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

package cpubus

import "github.com/jetsetilly/gopher2600/hardware/memory/memorymap"

// Reset is the address where the reset address is stored
// - used by VCS.Reset() and Disassembly module.
const Reset = uint16(0xfffc)

// IRQ is the address where the interrupt address is stored.
const IRQ = uint16(0xfffe)

// Register represents a named address in RIOT/TIA memory.
type Register string

// List of Valid CPUBusRegister values.
const (
	CXM0P  Register = "CXM0P"
	CXM1P  Register = "CXM1P"
	CXP0FB Register = "CXP0FB"
	CXP1FB Register = "CXP1FB"
	CXM0FB Register = "CXM0FB"
	CXM1FB Register = "CXM1FB"
	CXBLPF Register = "CXBLPF"
	CXPPMM Register = "CXPPMM"
	INPT0  Register = "INPT0"
	INPT1  Register = "INPT1"
	INPT2  Register = "INPT2"
	INPT3  Register = "INPT3"
	INPT4  Register = "INPT4"
	INPT5  Register = "INPT5"
	SWCHA  Register = "SWCHA"
	SWACNT Register = "SWACNT"
	SWCHB  Register = "SWCHB"
	SWBCNT Register = "SWBCNT"
	INTIM  Register = "INTIM"
	TIMINT Register = "TIMINT"
	VSYNC  Register = "VSYNC"
	VBLANK Register = "VBLANK"
	WSYNC  Register = "WSYNC"
	RSYNC  Register = "RSYNC"
	NUSIZ0 Register = "NUSIZ0"
	NUSIZ1 Register = "NUSIZ1"
	COLUP0 Register = "COLUP0"
	COLUP1 Register = "COLUP1"
	COLUPF Register = "COLUPF"
	COLUBK Register = "COLUBK"
	CTRLPF Register = "CTRLPF"
	REFP0  Register = "REFP0"
	REFP1  Register = "REFP1"
	PF0    Register = "PF0"
	PF1    Register = "PF1"
	PF2    Register = "PF2"
	RESP0  Register = "RESP0"
	RESP1  Register = "RESP1"
	RESM0  Register = "RESM0"
	RESM1  Register = "RESM1"
	RESBL  Register = "RESBL"
	AUDC0  Register = "AUDC0"
	AUDC1  Register = "AUDC1"
	AUDF0  Register = "AUDF0"
	AUDF1  Register = "AUDF1"
	AUDV0  Register = "AUDV0"
	AUDV1  Register = "AUDV1"
	GRP0   Register = "GRP0"
	GRP1   Register = "GRP1"
	ENAM0  Register = "ENAM0"
	ENAM1  Register = "ENAM1"
	ENABL  Register = "ENABL"
	HMP0   Register = "HMP0"
	HMP1   Register = "HMP1"
	HMM0   Register = "HMM0"
	HMM1   Register = "HMM1"
	HMBL   Register = "HMBL"
	VDELP0 Register = "VDELP0"
	VDELP1 Register = "VDELP1"
	VDELBL Register = "VDELBL"
	RESMP0 Register = "RESMP0"
	RESMP1 Register = "RESMP1"
	HMOVE  Register = "HMOVE"
	HMCLR  Register = "HMCLR"
	CXCLR  Register = "CXCLR"
	TIM1T  Register = "TIM1T"
	TIM8T  Register = "TIM8T"
	TIM64T Register = "TIM64T"
	T1024T Register = "T1024T"
)

// TIAReadSymbols indexes all TIA read symbols by normalised address.
var TIAReadSymbols = map[uint16]Register{
	// TIA
	0x00: CXM0P,
	0x01: CXM1P,
	0x02: CXP0FB,
	0x03: CXP1FB,
	0x04: CXM0FB,
	0x05: CXM1FB,
	0x06: CXBLPF,
	0x07: CXPPMM,
	0x08: INPT0,
	0x09: INPT1,
	0x0a: INPT2,
	0x0b: INPT3,
	0x0c: INPT4,
	0x0d: INPT5,
}

// RIOTReadSymbols indexes all RIOT read symbols by normalised address.
var RIOTReadSymbols = map[uint16]Register{
	// RIOT
	0x0280: SWCHA,
	0x0281: SWACNT,
	0x0282: SWCHB,
	0x0283: SWBCNT,
	0x0284: INTIM,
	0x0285: TIMINT,
}

// TIAWriteSymbols indexes all TIA write symbols by normalised address.
var TIAWriteSymbols = map[uint16]Register{
	0x00: VSYNC,
	0x01: VBLANK,
	0x02: WSYNC,
	0x03: RSYNC,
	0x04: NUSIZ0,
	0x05: NUSIZ1,
	0x06: COLUP0,
	0x07: COLUP1,
	0x08: COLUPF,
	0x09: COLUBK,
	0x0a: CTRLPF,
	0x0b: REFP0,
	0x0c: REFP1,
	0x0d: PF0,
	0x0e: PF1,
	0x0f: PF2,
	0x10: RESP0,
	0x11: RESP1,
	0x12: RESM0,
	0x13: RESM1,
	0x14: RESBL,
	0x15: AUDC0,
	0x16: AUDC1,
	0x17: AUDF0,
	0x18: AUDF1,
	0x19: AUDV0,
	0x1a: AUDV1,
	0x1b: GRP0,
	0x1c: GRP1,
	0x1d: ENAM0,
	0x1e: ENAM1,
	0x1f: ENABL,
	0x20: HMP0,
	0x21: HMP1,
	0x22: HMM0,
	0x23: HMM1,
	0x24: HMBL,
	0x25: VDELP0,
	0x26: VDELP1,
	0x27: VDELBL,
	0x28: RESMP0,
	0x29: RESMP1,
	0x2A: HMOVE,
	0x2B: HMCLR,
	0x2C: CXCLR,
}

// RIOTWriteSymbols indexes all RIOT write symbols by normalised address.
var RIOTWriteSymbols = map[uint16]Register{
	0x0280: SWCHA,
	0x0281: SWACNT,
	0x0282: SWCHB,
	0x0283: SWBCNT,
	0x0294: TIM1T,
	0x0295: TIM8T,
	0x0296: TIM64T,
	0x0297: T1024T,
}

// ReadAddress indexes all VCS read addresses by canonical symbol.
var ReadAddress = map[Register]uint16{}

// WriteAddress indexes all VCS write addresses by canonical symbol.
var WriteAddress = map[Register]uint16{}

// Read is a sparse array containing the canonical labels for all read addresses.
//
// If the address has no symbol then the entry will contain NotACPUBusRegister.
var Read []Register

// Write is a sparse array containing the canonical labels for all write addresses.
//
// If the address has no symbol then the entry will contain NotACPUBusRegister.
var Write []Register

// Address does not correspond with any known symbol.
const NotACPUBusRegister Register = ""

// this init() function create the Read/Write arrays using the read/write maps
// as a source.
func init() {
	// we know that the maximum address either chip can read or write to is
	// 0x297, in RIOT memory space. we can say this is the extent of our Read
	// and Write sparse arrays
	const chipTop = memorymap.MemtopRIOT

	Read = make([]Register, chipTop+1)
	for k, v := range TIAReadSymbols {
		Read[k] = v
	}
	for k, v := range RIOTReadSymbols {
		Read[k] = v
	}

	Write = make([]Register, chipTop+1)
	for k, v := range TIAWriteSymbols {
		Write[k] = v
	}
	for k, v := range RIOTWriteSymbols {
		Write[k] = v
	}
}

// Sentinal error returned by memory package functions. Note that the error
// expects a numberic address, which will be formatted as four digit hex.
const (
	AddressError = "inaccessible address (%#04x)"
)
