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
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/banks"
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

	// currently selected bank
	bank int

	// DPC registers are directly accessible by the VCS but have a special
	// meaning when written to and read. the DPCregisters type implements the
	// functionality of these special addresses and a copy of the field is
	// returned by the GetRegisters() function
	registers DPCregisters

	// dpc specific areas of the cartridge, not accessible by the normal VCS bus
	static DPCstatic

	// the OSC clock found in DPC cartridges runs at slower than the VCS itself
	// to effectively emulate the slower clock therefore, we need to discount
	// the excess steps. see the step() function for details
	beats int
}

// DPCstatic implements the bus.CartStatic interface
type DPCstatic struct {
	Gfx []byte
}

// DPCregisters implements the bus.CartRegisters interface
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

// DPCdataFetcher represents a single DPC data fetcher
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

func newDPC(data []byte) (mapper.CartMapper, error) {
	const staticSize = 2048

	cart := &dpc{
		description: "pitfall2 style",
		mappingID:   "DPC",
		bankSize:    4096,
	}

	cart.banks = make([][]uint8, cart.NumBanks())

	if len(data) < cart.bankSize*cart.NumBanks()+staticSize {
		return nil, curated.Errorf("%s: wrong number of bytes in the cartridge data", cart.mappingID)
	}

	for k := 0; k < cart.NumBanks(); k++ {
		cart.banks[k] = make([]uint8, cart.bankSize)
		offset := k * cart.bankSize
		cart.banks[k] = data[offset : offset+cart.bankSize]
	}

	staticStart := cart.NumBanks() * cart.bankSize
	cart.static.Gfx = data[staticStart : staticStart+staticSize]

	cart.Initialise()

	return cart, nil
}

func (cart dpc) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.mappingID, cart.description, cart.bank)
}

// ID implements the mapper.CartMapper interface
func (cart dpc) ID() string {
	return cart.mappingID
}

// Initialise implements the mapper.CartMapper interface
func (cart *dpc) Initialise() {
	cart.bank = len(cart.banks) - 1
}

// Read implements the mapper.CartMapper interface
func (cart *dpc) Read(addr uint16, passive bool) (uint8, error) {
	var data uint8

	// chip select is active by definition when read() is called. pump RNG [col 7, ln 58-62, fig 8]
	cart.registers.RNG |= (cart.registers.RNG>>3)&0x01 ^ (cart.registers.RNG>>4)&0x01 ^ (cart.registers.RNG>>5)&0x01 ^ (cart.registers.RNG>>7)&0x01
	cart.registers.RNG <<= 1

	// bankswitch on hotspot access
	if cart.bankswitch(addr, passive) {
		// always return zero on hotspot - unlike the Atari multi-bank carts for example
		return 0, nil
	}

	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
	if addr > 0x003f {
		return cart.banks[cart.bank][addr], nil
	}

	// * the remaining addresses are function registers [col 4, ln 10-20]

	// the first eight registers do not touch the data fetchers and therefore
	// do not trigger any of the side-effects on the data fetchers's counter
	// registers [see below]
	if addr >= 0x0000 && addr <= 0x0003 {
		// RNG value
		return cart.registers.RNG, nil

	} else if addr >= 0x0004 && addr <= 0x0007 {
		// music value. mix music data-fetchers:

		// generate SIN signal which is the equivalent of the flag register
		// when in music mode [col 7, ln 30-31]

		// SIN signals are weighted and added together [col 7, ln 3-7, fig 12]

		if cart.registers.Fetcher[5].MusicMode && cart.registers.Fetcher[5].Flag {
			data += 4
		}

		if cart.registers.Fetcher[6].MusicMode && cart.registers.Fetcher[6].Flag {
			data += 5
		}

		if cart.registers.Fetcher[7].MusicMode && cart.registers.Fetcher[7].Flag {
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
	gfxAddr := uint16(cart.registers.Fetcher[f].Hi)<<8 | uint16(cart.registers.Fetcher[f].Low)

	// only the 11 least-significant bits are used. gfx memory is also
	// addressed with reference from memtop so inverse the bits
	gfxAddr = gfxAddr&0x07ff ^ 0x07ff

	// set flag
	cart.registers.Fetcher[f].setFlag()

	if f >= 0x5 && cart.registers.Fetcher[f].MusicMode {
		// when in music mode return top register [col 7, ln 6-9]
		data = cart.registers.Fetcher[f].Top

	} else {
		if addr >= 0x0008 && addr <= 0x000f {
			// display data
			data = cart.static.Gfx[gfxAddr]

		} else if addr >= 0x0010 && addr <= 0x0017 {
			// display data AND w/flag
			if cart.registers.Fetcher[f].Flag {
				data = cart.static.Gfx[gfxAddr]
			}

		} else if addr >= 0x0018 && addr <= 0x001f {
			// display data AND w/flag, nibbles swapped

		} else if addr >= 0x0020 && addr <= 0x0027 {
			// display data AND w/flag, byte reversed

		} else if addr >= 0x0028 && addr <= 0x002f {
			// display data AND w/flag, ROR
			if cart.registers.Fetcher[f].Flag {
				data = cart.static.Gfx[gfxAddr] >> 1
			}

		} else if addr >= 0x0030 && addr <= 0x0037 {
			// display data AND w/flag, ROL
			if cart.registers.Fetcher[f].Flag {
				data = cart.static.Gfx[gfxAddr] << 1
			}

		} else if addr >= 0x0038 && addr <= 0x003f {
			// DFx flag
			if f >= 0x5 && cart.registers.Fetcher[f].Flag {
				data = 0xff
			}
		}
	}

	// clock signal is active whenever data fetcher is used
	cart.registers.Fetcher[f].clk()

	return data, nil
}

// Write implements the mapper.CartMapper interface
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
		cart.registers.Fetcher[f].Top = data
		cart.registers.Fetcher[f].Flag = false

	} else if addr >= 0x0048 && addr <= 0x004f {
		// set bottom register
		cart.registers.Fetcher[f].Bottom = data

	} else if addr >= 0x0050 && addr <= 0x0057 {
		// set low register

		// treat music mode capable registers slightly differently
		if f >= 0x5 && cart.registers.Fetcher[f].MusicMode {
			// low is loaded with top value on low function [col 7, ln 12-14]
			cart.registers.Fetcher[f].Low = cart.registers.Fetcher[f].Top

		} else {
			cart.registers.Fetcher[f].Low = data

		}

	} else if addr >= 0x0058 && addr <= 0x005f {
		// set high register
		cart.registers.Fetcher[f].Hi = data

		// treat music mode capable registers slightly differently
		if f >= 0x5 && addr >= 0x005d { // && addr <= 0x00f5 is implied
			// set music mode [col 7, ln 1-6]
			cart.registers.Fetcher[f].MusicMode = data&0x10 == 0x10

			// set osc clock [col 7, ln 20-22]
			cart.registers.Fetcher[f].OSCclock = data&0x20 == 0x20
		}

	} else if addr >= 0x0070 && addr <= 0x0077 {
		// reset random number generator
		cart.registers.RNG = 0xff
	}

	// other addresses are not write registers and are ignored

	if poke {
		cart.banks[cart.bank][addr] = data
		return nil
	}

	return curated.Errorf(bus.AddressError, addr)
}

// bank switch on hotspot access
func (cart *dpc) bankswitch(addr uint16, passive bool) bool {
	if addr >= 0x0ff8 && addr <= 0x0ff9 {
		if passive {
			return true
		}
		if addr == 0x0ff8 {
			cart.bank = 0
		} else if addr == 0x0ff9 {
			cart.bank = 1
		}
		return true
	}
	return false
}

// NumBanks implements the mapper.CartMapper interface
func (cart dpc) NumBanks() int {
	return 2
}

// GetBank implements the mapper.CartMapper interface
func (cart dpc) GetBank(addr uint16) banks.Details {
	return banks.Details{Number: cart.bank, IsRAM: false}
}

// Patch implements the mapper.CartMapper interface
func (cart *dpc) Patch(offset int, data uint8) error {
	if offset >= cart.bankSize*len(cart.banks)+len(cart.static.Gfx) {
		return curated.Errorf("%s: patch offset too high (%v)", cart.ID(), offset)
	}

	staticStart := cart.NumBanks() * cart.bankSize
	if staticStart >= staticStart {
		cart.static.Gfx[offset] = data
	} else {
		bank := int(offset) / cart.bankSize
		offset = offset % cart.bankSize
		cart.banks[bank][offset] = data
	}
	return nil
}

// Listen implements the mapper.CartMapper interface
func (cart *dpc) Listen(_ uint16, _ uint8) {
}

// Step implements the mapper.CartMapper interface
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

	cart.beats++
	if cart.beats%59 == 0 {
		cart.beats = 0
		for f := 5; f <= 7; f++ {
			if cart.registers.Fetcher[f].MusicMode && cart.registers.Fetcher[f].OSCclock {
				cart.registers.Fetcher[f].clk()
				cart.registers.Fetcher[f].setFlag()
			}
		}
	}
}

// GetRegisters implements the bus.CartDebugBus interface
func (cart dpc) GetRegisters() bus.CartRegisters {
	return bus.CartRegisters(cart.registers)
}

// PutRegister implements the bus.CartDebugBus interface
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
		if err != nil || f > len(cart.registers.Fetcher) {
			panic(fmt.Sprintf("unrecognised fetcher [%s]", register))
		}

		switch r[2] {
		case "hi":
			cart.registers.Fetcher[f].Hi = uint8(d)
		case "low":
			cart.registers.Fetcher[f].Low = uint8(d)
		case "top":
			cart.registers.Fetcher[f].Top = uint8(d)
		case "bottom":
			cart.registers.Fetcher[f].Bottom = uint8(d)
		case "musicmode":
			switch data {
			case "true":
				cart.registers.Fetcher[f].MusicMode = true
			case "false":
				cart.registers.Fetcher[f].MusicMode = false
			default:
				panic(fmt.Sprintf("unrecognised boolean state [%s]", data))
			}
		default:
			panic(fmt.Sprintf("unrecognised variable [%s]", register))
		}
	case "rng":
		cart.registers.RNG = uint8(d)
	default:
		panic(fmt.Sprintf("unrecognised variable [%s]", register))
	}
}

// GetStatic implements the bus.CartDebugBus interface
func (cart dpc) GetStatic() []bus.CartStatic {
	s := make([]bus.CartStatic, 1)
	s[0].Label = "Gfx"
	s[0].Data = make([]byte, len(cart.static.Gfx))
	copy(s[0].Data, cart.static.Gfx)
	return s
}

// PutStatic implements the bus.CartDebugBus interface
func (cart *dpc) PutStatic(label string, addr uint16, data uint8) error {
	if label == "Gfx" {
		if int(addr) >= len(cart.static.Gfx) {
			return curated.Errorf("dpc: static: %v", fmt.Errorf("address too high (%#04x) for %s area", addr, label))
		}
		cart.static.Gfx[addr] = data
	} else {
		return curated.Errorf("dpc: static: %v", fmt.Errorf("unknown static area (%s)", label))
	}

	return nil
}

// IterateBank implemnts the disassemble interface
func (cart dpc) IterateBanks(prev *banks.Content) *banks.Content {
	b := prev.Number + 1
	if b < len(cart.banks) {
		return &banks.Content{Number: b,
			Data: cart.banks[b],
			Origins: []uint16{
				memorymap.OriginCart,
			},
		}
	}
	return nil
}
