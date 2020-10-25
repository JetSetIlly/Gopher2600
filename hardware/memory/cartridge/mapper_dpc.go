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

package cartridge

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// dpc implements the mapper.CartMapper interface.
//
// column, line number & figure references to US patent 4,644,495 are used to
// support coding decisions.
type dpc struct {
	mappingID   string
	description string

	// dpc cartridge have two banks of 4096 bytes
	bankSize int
	banks    [][]byte

	// static area of the cartridge. accessible outside of the cartridge
	// through GetStatic() and PutStatic()
	static []byte

	// rewindable state
	state *dpcState
}

func newDPC(data []byte) (mapper.CartMapper, error) {
	const staticSize = 2048

	cart := &dpc{
		description: "pitfall2 style",
		mappingID:   "DPC",
		bankSize:    4096,
		state:       newDPCState(),
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	if len(data) < cart.bankSize*cart.NumBanks()+staticSize {
		return nil, curated.Errorf("DPC: %v", "wrong number bytes in the cartridge data")
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		cart.banks[k] = data[offset : offset+cart.bankSize]
	}

	// copy some data into the static area of the DPC state structure
	staticStart := cart.NumBanks() * cart.bankSize
	cart.static = data[staticStart : staticStart+staticSize]

	return cart, nil
}

func (cart dpc) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.mappingID, cart.description, cart.state.bank)
}

// ID implements the mapper.CartMapper interface.
func (cart dpc) ID() string {
	return cart.mappingID
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *dpc) Snapshot() mapper.CartSnapshot {
	return cart.state.Snapshot()
}

// Plumb implements the mapper.CartMapper interface.
func (cart *dpc) Plumb(s mapper.CartSnapshot) {
	cart.state = s.(*dpcState)
}

// Reset implements the mapper.CartMapper interface.
func (cart *dpc) Reset(randSrc *rand.Rand) {
	cart.state.registers.reset(randSrc)
	cart.state.bank = len(cart.banks) - 1
}

// Read implements the mapper.CartMapper interface.
func (cart *dpc) Read(addr uint16, passive bool) (uint8, error) {
	var data uint8

	// chip select is active by definition when read() is called. pump RNG [col 7, ln 58-62, fig 8]
	cart.state.registers.RNG |= (cart.state.registers.RNG>>3)&0x01 ^ (cart.state.registers.RNG>>4)&0x01 ^ (cart.state.registers.RNG>>5)&0x01 ^ (cart.state.registers.RNG>>7)&0x01
	cart.state.registers.RNG <<= 1

	// bankswitch on hotspot access
	if cart.bankswitch(addr, passive) {
		// always return zero on hotspot - unlike the Atari multi-bank carts for example
		return 0, nil
	}

	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
	if addr > 0x003f {
		return cart.banks[cart.state.bank][addr], nil
	}

	// * the remaining addresses are function registers [col 4, ln 10-20]

	// the first eight registers do not touch the data fetchers and therefore
	// do not trigger any of the side-effects on the data fetchers's counter
	// registers [see below]
	if addr >= 0x0000 && addr <= 0x0003 {
		// RNG value
		return cart.state.registers.RNG, nil
	} else if addr >= 0x0004 && addr <= 0x0007 {
		// music value. mix music data-fetchers:

		// generate SIN signal which is the equivalent of the flag register
		// when in music mode [col 7, ln 30-31]

		// SIN signals are weighted and added together [col 7, ln 3-7, fig 12]

		if cart.state.registers.Fetcher[5].MusicMode && cart.state.registers.Fetcher[5].Flag {
			data += 4
		}

		if cart.state.registers.Fetcher[6].MusicMode && cart.state.registers.Fetcher[6].Flag {
			data += 5
		}

		if cart.state.registers.Fetcher[7].MusicMode && cart.state.registers.Fetcher[7].Flag {
			data += 6
		}

		return data, nil
	}

	// * the remaining functions all work on specific data fetchers

	// decide which data-fetcher to use. the three least-significant bits of
	// the address indicate the fetcher
	f := addr & 0x0007

	// most data-fetcher functions address gfx memory (only the flag registers
	// do not)
	gfxAddr := uint16(cart.state.registers.Fetcher[f].Hi)<<8 | uint16(cart.state.registers.Fetcher[f].Low)

	// only the 11 least-significant bits are used. gfx memory is also
	// addressed with reference from memtop so inverse the bits
	gfxAddr = gfxAddr&0x07ff ^ 0x07ff

	// set flag
	cart.state.registers.Fetcher[f].setFlag()

	if f >= 0x5 && cart.state.registers.Fetcher[f].MusicMode {
		// when in music mode return top register [col 7, ln 6-9]
		data = cart.state.registers.Fetcher[f].Top
	} else {
		if addr >= 0x0008 && addr <= 0x000f {
			// display data
			data = cart.static[gfxAddr]
		} else if addr >= 0x0010 && addr <= 0x0017 {
			// display data AND w/flag
			if cart.state.registers.Fetcher[f].Flag {
				data = cart.static[gfxAddr]
			}
		} else if addr >= 0x0018 && addr <= 0x001f {
			// display data AND w/flag, nibbles swapped

		} else if addr >= 0x0020 && addr <= 0x0027 {
			// display data AND w/flag, byte reversed

		} else if addr >= 0x0028 && addr <= 0x002f {
			// display data AND w/flag, ROR
			if cart.state.registers.Fetcher[f].Flag {
				data = cart.static[gfxAddr] >> 1
			}
		} else if addr >= 0x0030 && addr <= 0x0037 {
			// display data AND w/flag, ROL
			if cart.state.registers.Fetcher[f].Flag {
				data = cart.static[gfxAddr] << 1
			}
		} else if addr >= 0x0038 && addr <= 0x003f {
			// DFx flag
			if f >= 0x5 && cart.state.registers.Fetcher[f].Flag {
				data = 0xff
			}
		}
	}

	// clock signal is active whenever data fetcher is used
	cart.state.registers.Fetcher[f].clk()

	return data, nil
}

// Write implements the mapper.CartMapper interface.
func (cart *dpc) Write(addr uint16, data uint8, passive bool, poke bool) error {
	if cart.bankswitch(addr, passive) {
		return nil
	}

	// if the write address if a write address then the effect is on a
	// specific data-fetcher. the data-fetcher is specified by the three
	// least-significant bits
	f := addr & 0x0007

	if addr >= 0x0040 && addr <= 0x0047 {
		// set top register
		cart.state.registers.Fetcher[f].Top = data
		cart.state.registers.Fetcher[f].Flag = false
	} else if addr >= 0x0048 && addr <= 0x004f {
		// set bottom register
		cart.state.registers.Fetcher[f].Bottom = data
	} else if addr >= 0x0050 && addr <= 0x0057 {
		// set low register

		// treat music mode capable registers slightly differently
		if f >= 0x5 && cart.state.registers.Fetcher[f].MusicMode {
			// low is loaded with top value on low function [col 7, ln 12-14]
			cart.state.registers.Fetcher[f].Low = cart.state.registers.Fetcher[f].Top
		} else {
			cart.state.registers.Fetcher[f].Low = data
		}
	} else if addr >= 0x0058 && addr <= 0x005f {
		// set high register
		cart.state.registers.Fetcher[f].Hi = data

		// treat music mode capable registers slightly differently
		if f >= 0x5 && addr >= 0x005d { // && addr <= 0x00f5 is implied
			// set music mode [col 7, ln 1-6]
			cart.state.registers.Fetcher[f].MusicMode = data&0x10 == 0x10

			// set osc clock [col 7, ln 20-22]
			cart.state.registers.Fetcher[f].OSCclock = data&0x20 == 0x20
		}
	} else if addr >= 0x0070 && addr <= 0x0077 {
		// reset random number generator
		cart.state.registers.RNG = 0xff
	}

	// other addresses are not write registers and are ignored

	if poke {
		cart.banks[cart.state.bank][addr] = data
		return nil
	}

	return curated.Errorf("DPC: %v", curated.Errorf(bus.AddressError, addr))
}

// bank switch on hotspot access.
func (cart *dpc) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0x0ff8 && addr <= 0x0ff9 {
		if passive {
			return true
		}
		if addr == 0x0ff8 {
			cart.state.bank = 0
		} else if addr == 0x0ff9 {
			cart.state.bank = 1
		}
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface.
func (cart dpc) NumBanks() int {
	return 2
}

// GetBank implements the mapper.CartMapper interface.
func (cart dpc) GetBank(addr uint16) mapper.BankInfo {
	return mapper.BankInfo{Number: cart.state.bank, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface.
func (cart *dpc) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks)+len(cart.static) {
		return curated.Errorf("DPC: %v", fmt.Errorf("patch offset too high (%v)", offset))
	}

	staticStart := cart.NumBanks() * cart.bankSize
	if staticStart >= cart.NumBanks()*cart.bankSize {
		cart.static[offset] = data
	} else {
		bank := offset / cart.bankSize
		offset %= cart.bankSize
		cart.banks[bank][offset] = data
	}
	return nil
}

// Listen implements the mapper.CartMapper interface.
func (cart *dpc) Listen(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface.
func (cart *dpc) Step() {
	// clock music enabled data fetchers if oscClock is active [col 7, ln 25-27]

	// documented update rate is 42Khz [col 7, ln 25-27]

	// cpu rate 1.19Mhz. so:
	// 1.19Mhz / 42Khz
	// = 1190000 / 42000
	// = 28.33
	//
	// ie. cartridge clock ticks once every 28 ticks of the VCS clock
	//
	// however, comparison to how Stella sounds with the known DPC cartridge
	// (Pitfall 2) reveals that the tuning is wrong if this value is used. by
	// ear, a value of 59 is more accurate. by my reckoning this means that the
	// clock in the cartridge is 20Khz. I can find no supporting documentation
	// for this.

	cart.state.beats++
	if cart.state.beats%59 == 0 {
		cart.state.beats = 0
		for f := 5; f <= 7; f++ {
			if cart.state.registers.Fetcher[f].MusicMode && cart.state.registers.Fetcher[f].OSCclock {
				cart.state.registers.Fetcher[f].clk()
				cart.state.registers.Fetcher[f].setFlag()
			}
		}
	}
}

// GetRegisters implements the mapper.CartRegisters interface.
func (cart dpc) GetRegisters() mapper.CartRegisters {
	return mapper.CartRegisters(cart.state.registers)
}

// PutRegister implements the mapper.CartRegister interface
//
// Register specification is divided with the "::" string. The following table
// describes what the valid register strings and, after the = sign, the type to
// which the data argument will be converted.
//
//	fetcher::%int::hi = uint8
//	fetcher::%int::low = uint8
//	fetcher::%int::top = uint8
//	fetcher::%int::bottom = uint8
//	fetcher::%int::musicmode = bool
//	rng = uint8
//
// note that PutRegister() will panic() if the register or data string is invalid.
func (cart *dpc) PutRegister(register string, data string) {
	// most data is expected to an integer (a uint8 specifically) so we try
	// to convert it here. if it doesn't convert then it doesn't matter
	d, _ := strconv.ParseUint(data, 16, 8)

	r := strings.Split(register, "::")
	switch r[0] {
	case "fetcher":
		f, err := strconv.Atoi(r[1])
		if err != nil || f > len(cart.state.registers.Fetcher) {
			panic(fmt.Sprintf("unrecognised fetcher [%s]", register))
		}

		switch r[2] {
		case "hi":
			cart.state.registers.Fetcher[f].Hi = uint8(d)
		case "low":
			cart.state.registers.Fetcher[f].Low = uint8(d)
		case "top":
			cart.state.registers.Fetcher[f].Top = uint8(d)
		case "bottom":
			cart.state.registers.Fetcher[f].Bottom = uint8(d)
		case "musicmode":
			switch data {
			case "true":
				cart.state.registers.Fetcher[f].MusicMode = true
			case "false":
				cart.state.registers.Fetcher[f].MusicMode = false
			default:
				panic(fmt.Sprintf("unrecognised boolean state [%s]", data))
			}
		default:
			panic(fmt.Sprintf("unrecognised variable [%s]", register))
		}
	case "rng":
		cart.state.registers.RNG = uint8(d)
	default:
		panic(fmt.Sprintf("unrecognised variable [%s]", register))
	}
}

// GetStatic implements the mapper.CartDebugBus interface.
func (cart dpc) GetStatic() []mapper.CartStatic {
	s := make([]mapper.CartStatic, 1)
	s[0].Label = "Gfx"
	s[0].Data = make([]byte, len(cart.static))
	copy(s[0].Data, cart.static)
	return s
}

// PutStatic implements the mapper.CartDebugBus interface.
func (cart *dpc) PutStatic(label string, addr uint16, data uint8) error {
	if label == "Gfx" {
		if int(addr) >= len(cart.static) {
			return curated.Errorf("dpc: static: %v", fmt.Errorf("address too high (%#04x) for %s area", addr, label))
		}
		cart.static[addr] = data
	} else {
		return curated.Errorf("dpc: static: %v", fmt.Errorf("unknown static area (%s)", label))
	}

	return nil
}

// IterateBank implements the mapper.CartMapper interface.
func (cart dpc) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

// ReadHotspots implements the mapper.CartHotspotsBus interface.
func (cart dpc) ReadHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff8: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1000: {Symbol: "RNG", Action: mapper.HotspotRegister},
		0x1001: {Symbol: "RNG", Action: mapper.HotspotRegister},
		0x1002: {Symbol: "RNG", Action: mapper.HotspotRegister},
		0x1003: {Symbol: "RNG", Action: mapper.HotspotRegister},
		0x1004: {Symbol: "MUSIC", Action: mapper.HotspotRegister},
		0x1005: {Symbol: "MUSIC", Action: mapper.HotspotRegister},
		0x1006: {Symbol: "MUSIC", Action: mapper.HotspotRegister},
		0x1007: {Symbol: "MUSIC", Action: mapper.HotspotRegister},
		0x1008: {Symbol: "DF0", Action: mapper.HotspotRegister},
		0x1009: {Symbol: "DF1", Action: mapper.HotspotRegister},
		0x100a: {Symbol: "DF2", Action: mapper.HotspotRegister},
		0x100b: {Symbol: "DF3", Action: mapper.HotspotRegister},
		0x100c: {Symbol: "DF4", Action: mapper.HotspotRegister},
		0x100d: {Symbol: "DF5", Action: mapper.HotspotRegister},
		0x100e: {Symbol: "DF6", Action: mapper.HotspotRegister},
		0x100f: {Symbol: "DF7", Action: mapper.HotspotRegister},
		0x1010: {Symbol: "DF0f", Action: mapper.HotspotRegister},
		0x1011: {Symbol: "DF1f", Action: mapper.HotspotRegister},
		0x1012: {Symbol: "DF2f", Action: mapper.HotspotRegister},
		0x1013: {Symbol: "DF3f", Action: mapper.HotspotRegister},
		0x1014: {Symbol: "DF4f", Action: mapper.HotspotRegister},
		0x1015: {Symbol: "DF5f", Action: mapper.HotspotRegister},
		0x1016: {Symbol: "DF6f", Action: mapper.HotspotRegister},
		0x1017: {Symbol: "DF7f", Action: mapper.HotspotRegister},
		0x1018: {Symbol: "DF0f/swp", Action: mapper.HotspotRegister},
		0x1019: {Symbol: "DF1f/swp", Action: mapper.HotspotRegister},
		0x101a: {Symbol: "DF2f/swp", Action: mapper.HotspotRegister},
		0x101b: {Symbol: "DF3f/swp", Action: mapper.HotspotRegister},
		0x101c: {Symbol: "DF4f/swp", Action: mapper.HotspotRegister},
		0x101d: {Symbol: "DF5f/swp", Action: mapper.HotspotRegister},
		0x101e: {Symbol: "DF6f/swp", Action: mapper.HotspotRegister},
		0x101f: {Symbol: "DF7f/swp", Action: mapper.HotspotRegister},
		0x1020: {Symbol: "DF0f/rev", Action: mapper.HotspotRegister},
		0x1021: {Symbol: "DF1f/rev", Action: mapper.HotspotRegister},
		0x1022: {Symbol: "DF2f/rev", Action: mapper.HotspotRegister},
		0x1023: {Symbol: "DF3f/rev", Action: mapper.HotspotRegister},
		0x1024: {Symbol: "DF4f/rev", Action: mapper.HotspotRegister},
		0x1025: {Symbol: "DF5f/rev", Action: mapper.HotspotRegister},
		0x1026: {Symbol: "DF6f/rev", Action: mapper.HotspotRegister},
		0x1027: {Symbol: "DF7f/rev", Action: mapper.HotspotRegister},
		0x1028: {Symbol: "DF0f/ror", Action: mapper.HotspotRegister},
		0x1029: {Symbol: "DF1f/ror", Action: mapper.HotspotRegister},
		0x102a: {Symbol: "DF2f/ror", Action: mapper.HotspotRegister},
		0x102b: {Symbol: "DF3f/ror", Action: mapper.HotspotRegister},
		0x102c: {Symbol: "DF4f/ror", Action: mapper.HotspotRegister},
		0x102d: {Symbol: "DF5f/ror", Action: mapper.HotspotRegister},
		0x102e: {Symbol: "DF6f/ror", Action: mapper.HotspotRegister},
		0x102f: {Symbol: "DF7f/ror", Action: mapper.HotspotRegister},
		0x1030: {Symbol: "DF0f/rol", Action: mapper.HotspotRegister},
		0x1031: {Symbol: "DF1f/rol", Action: mapper.HotspotRegister},
		0x1032: {Symbol: "DF2f/rol", Action: mapper.HotspotRegister},
		0x1033: {Symbol: "DF3f/rol", Action: mapper.HotspotRegister},
		0x1034: {Symbol: "DF4f/rol", Action: mapper.HotspotRegister},
		0x1035: {Symbol: "DF5f/rol", Action: mapper.HotspotRegister},
		0x1036: {Symbol: "DF6f/rol", Action: mapper.HotspotRegister},
		0x1037: {Symbol: "DF7f/rol", Action: mapper.HotspotRegister},
		0x1038: {Symbol: "FLG0", Action: mapper.HotspotRegister},
		0x1039: {Symbol: "FLG1", Action: mapper.HotspotRegister},
		0x103a: {Symbol: "FLG2", Action: mapper.HotspotRegister},
		0x103b: {Symbol: "FLG3", Action: mapper.HotspotRegister},
		0x103c: {Symbol: "FLG4", Action: mapper.HotspotRegister},
		0x103d: {Symbol: "FLG5", Action: mapper.HotspotRegister},
		0x103e: {Symbol: "FLG6", Action: mapper.HotspotRegister},
		0x103f: {Symbol: "FLG7", Action: mapper.HotspotRegister},
	}
}

// WriteHotspots implements the mapper.CartHotspotsBus interface.
func (cart dpc) WriteHotspots() map[uint16]mapper.CartHotspotInfo {
	return map[uint16]mapper.CartHotspotInfo{
		0x1ff8: {Symbol: "BANK0", Action: mapper.HotspotBankSwitch},
		0x1ff9: {Symbol: "BANK1", Action: mapper.HotspotBankSwitch},
		0x1040: {Symbol: "DF0/top", Action: mapper.HotspotRegister},
		0x1041: {Symbol: "DF1/top", Action: mapper.HotspotRegister},
		0x1042: {Symbol: "DF2/top", Action: mapper.HotspotRegister},
		0x1043: {Symbol: "DF3/top", Action: mapper.HotspotRegister},
		0x1044: {Symbol: "DF4/top", Action: mapper.HotspotRegister},
		0x1045: {Symbol: "DF5/top", Action: mapper.HotspotRegister},
		0x1046: {Symbol: "DF6/top", Action: mapper.HotspotRegister},
		0x1047: {Symbol: "DF7/top", Action: mapper.HotspotRegister},
		0x1048: {Symbol: "DF0/bot", Action: mapper.HotspotRegister},
		0x1049: {Symbol: "DF1/bot", Action: mapper.HotspotRegister},
		0x104a: {Symbol: "DF2/bot", Action: mapper.HotspotRegister},
		0x104b: {Symbol: "DF3/bot", Action: mapper.HotspotRegister},
		0x104c: {Symbol: "DF4/bot", Action: mapper.HotspotRegister},
		0x104d: {Symbol: "DF5/bot", Action: mapper.HotspotRegister},
		0x104e: {Symbol: "DF6/bot", Action: mapper.HotspotRegister},
		0x104f: {Symbol: "DF7/bot", Action: mapper.HotspotRegister},
		0x1050: {Symbol: "DF0/low", Action: mapper.HotspotRegister},
		0x1051: {Symbol: "DF1/low", Action: mapper.HotspotRegister},
		0x1052: {Symbol: "DF2/low", Action: mapper.HotspotRegister},
		0x1053: {Symbol: "DF3/low", Action: mapper.HotspotRegister},
		0x1054: {Symbol: "DF4/low", Action: mapper.HotspotRegister},
		0x1055: {Symbol: "DF5/low", Action: mapper.HotspotRegister},
		0x1056: {Symbol: "DF6/low", Action: mapper.HotspotRegister},
		0x1057: {Symbol: "DF7/low", Action: mapper.HotspotRegister},
		0x1058: {Symbol: "DF0/hi", Action: mapper.HotspotRegister},
		0x1059: {Symbol: "DF1/hi", Action: mapper.HotspotRegister},
		0x105a: {Symbol: "DF2/hi", Action: mapper.HotspotRegister},
		0x105b: {Symbol: "DF3/hi", Action: mapper.HotspotRegister},
		0x105c: {Symbol: "DF4/hi", Action: mapper.HotspotRegister},
		0x105d: {Symbol: "DF5/hi", Action: mapper.HotspotRegister},
		0x105e: {Symbol: "DF6/hi", Action: mapper.HotspotRegister},
		0x105f: {Symbol: "DF7/hi", Action: mapper.HotspotRegister},
		0x1060: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1061: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1062: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1063: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1064: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1065: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1066: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1067: {Symbol: "LINE", Action: mapper.HotspotFunction},
		0x1068: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1069: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x106a: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x106b: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x106c: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x106d: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x106e: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x106f: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1070: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1071: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1072: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1073: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1074: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1075: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1076: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1077: {Symbol: "RNG/reset", Action: mapper.HotspotFunction},
		0x1078: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x1079: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x107a: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x107b: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x107c: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x107d: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x107e: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
		0x107f: {Symbol: "RESERVED", Action: mapper.HotspotReserved},
	}
}

// DPCregisters implements the mapper.CartRegisters interface.
type DPCregisters struct {
	Fetcher [8]DPCdataFetcher

	// the current random number value
	RNG uint8
}

func (r DPCregisters) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("RNG: %#02x\n", r.RNG))
	for f := 0; f < len(r.Fetcher); f++ {
		s.WriteString(fmt.Sprintf("F%d: l:%#02x h:%#02x t:%#02x b:%#02x mm:%v", f,
			r.Fetcher[f].Low,
			r.Fetcher[f].Hi,
			r.Fetcher[f].Top,
			r.Fetcher[f].Bottom,
			r.Fetcher[f].MusicMode,
		))
		s.WriteString("\n")
	}
	return s.String()
}

func (r *DPCregisters) reset(randSrc *rand.Rand) {
	for i := range r.Fetcher {
		if randSrc != nil {
			r.Fetcher[i].Low = byte(randSrc.Intn(0xff))
			r.Fetcher[i].Hi = byte(randSrc.Intn(0xff))
			r.Fetcher[i].Top = byte(randSrc.Intn(0xff))
			r.Fetcher[i].Bottom = byte(randSrc.Intn(0xff))
		} else {
			r.Fetcher[i].Low = 0
			r.Fetcher[i].Hi = 0
			r.Fetcher[i].Top = 0
			r.Fetcher[i].Bottom = 0
		}

		// not randomising state of the following
		r.Fetcher[i].Flag = false
		r.Fetcher[i].MusicMode = false
		r.Fetcher[i].OSCclock = false
	}

	if randSrc != nil {
		r.RNG = uint8(randSrc.Intn(0xff))
	} else {
		r.RNG = 0
	}
}

// DPCdataFetcher represents a single DPC data fetcher.
type DPCdataFetcher struct {
	Low    byte
	Hi     byte
	Top    byte
	Bottom byte

	// is the Low byte in the window between top and bottom
	Flag bool

	// music mode only used by data fetchers 4-7
	MusicMode bool
	OSCclock  bool
}

func (df *DPCdataFetcher) clk() {
	// decrease low byte [col 5, ln 65 - col 6, ln 3]
	df.Low--
	if df.Low == 0xff {
		// decrease hi-address byte on carry bit
		df.Hi--

		// reset low to top when in music mode [col7, ln 14-19]
		if df.MusicMode {
			df.Low = df.Top
		}
	}
}

func (df *DPCdataFetcher) setFlag() {
	// set flag register [col 6, ln 7-12]
	if df.Low == df.Top {
		df.Flag = true
	} else if df.Low == df.Bottom {
		df.Flag = false
	}
}

// rewindable state for the parker bros. cartridges.
type dpcState struct {
	// currently selected bank
	bank int

	// DPC registers are directly accessible by the VCS but have a special
	// meaning when written to and read. the DPCregisters type implements the
	// functionality of these special addresses and a copy of the field is
	// returned by the GetRegisters() function
	registers DPCregisters

	// the OSC clock found in DPC cartridges runs at slower than the VCS itself
	// to effectively emulate the slower clock therefore, we need to discount
	// the excess steps. see the step() function for details
	beats int
}

func newDPCState() *dpcState {
	return &dpcState{}
}

// Snapshot implements the mapper.CartSnapshot interface.
func (s *dpcState) Snapshot() mapper.CartSnapshot {
	n := *s
	return &n
}
