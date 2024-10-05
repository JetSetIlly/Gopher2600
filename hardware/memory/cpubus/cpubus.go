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

import (
	"errors"

	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// Register represents a named address in RIOT/TIA memory
type Register string

// List of Valid CPUBusRegister values
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

// TIAReadRegisters indexes all TIA read symbols by normalised address
var TIAReadRegisters = map[uint16]Register{
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

// TIAWriteRegisters indexes all TIA write symbols by normalised address
var TIAWriteRegisters = map[uint16]Register{
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
	0x2a: HMOVE,
	0x2b: HMCLR,
	0x2c: CXCLR,
}

// RIOTReadRegisters indexes all RIOT read symbols by normalised address
var RIOTReadRegisters = map[uint16]Register{
	// RIOT
	0x0280: SWCHA,
	0x0281: SWACNT,
	0x0282: SWCHB,
	0x0283: SWBCNT,
	0x0284: INTIM,
	0x0285: TIMINT,
}

// RIOTWriteRegisters indexes all RIOT write symbols by normalised address
var RIOTWriteRegisters = map[uint16]Register{
	0x0280: SWCHA,
	0x0281: SWACNT,
	0x0282: SWCHB,
	0x0283: SWBCNT,

	// standard documentation for the 2600 claim that the write addresses for
	// TIM1T etc. begin at 0x0294. but these addresses are in fact mirrors of
	// 0x0284 etc.
	0x0294: TIM1T,
	0x0295: TIM8T,
	0x0296: TIM64T,
	0x0297: T1024T,
}

// ReadAddressByRegister indexes all VCS read addresses by canonical symbol
var ReadAddressByRegister = map[Register]uint16{}

// WriteAddressByRegister indexes all VCS write addresses by canonical symbol
var WriteAddressByRegister = map[Register]uint16{}

// ReadAddress is a sparse array containing the canonical labels for all read addresses
//
// If the address has no symbol then the entry will contain NotACPUBusRegister
var ReadAddress []Register

// WriteAddress is a sparse array containing the canonical labels for all write addresses
//
// If the address has no symbol then the entry will contain NotACPUBusRegister
var WriteAddress []Register

// Address does not correspond with any known register
const UnnamedAddress Register = ""

// sentinal error returned by Peek() and Poke() functions
var AddressError = errors.New("address error")

// this init() function create the Read/Write arrays using the read/write maps
// as a source
func init() {
	ReadAddress = make([]Register, memorymap.MemtopChipRegisters+1)
	for k, v := range TIAReadRegisters {
		ReadAddress[k] = v
		ReadAddressByRegister[v] = k
	}
	for k, v := range RIOTReadRegisters {
		ReadAddress[k] = v
		ReadAddressByRegister[v] = k
	}

	WriteAddress = make([]Register, memorymap.MemtopChipRegisters+1)
	for k, v := range TIAWriteRegisters {
		WriteAddress[k] = v
		WriteAddressByRegister[v] = k
	}
	for k, v := range RIOTWriteRegisters {
		WriteAddress[k] = v
		WriteAddressByRegister[v] = k
	}
}
