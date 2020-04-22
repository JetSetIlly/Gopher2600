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

	"github.com/jetsetilly/gopher2600/errors"
)

// dpc implements the cartMapper interface.
//
// column, line number & figure references to US patent 4,644,495 are used to
// support coding decisions.
type dpc struct {
	formatID    string
	description string
	banks       [][]byte
	bank        int
	gfx         []byte

	fetcher [8]dataFetcher

	// the current random number value
	rng uint8

	// the OSC clock found in DPC cartridges runs at slower than the VCS itself
	// to effectively emulate the slower clock therefore, we need to discount
	// the excess steps. see the step() function for details
	beats int
}

type dataFetcher struct {
	top    byte
	bottom byte
	low    byte
	hi     byte
	flag   bool

	// music mode not used for all data fetcher instances
	musicMode bool
	oscClock  bool
}

func (df *dataFetcher) clk() {
	// decrease low byte [col 5, ln 65 - col 6, ln 3]
	df.low--
	if df.low == 0xff {
		// decrease hi-address byte on carry bit
		df.hi--

		// reset low to top when in music mode [col7, ln 14-19]
		if df.musicMode {
			df.low = df.top
		}
	}
}

func (df *dataFetcher) setFlag() {
	// set flag register [col 6, ln 7-12]

	if df.low == df.top {
		df.flag = true
	} else if df.low == df.bottom {
		df.flag = false
	}
}

func newDPC(data []byte) (*dpc, error) {
	const bankSize = 4096
	const gfxSize = 2048

	cart := &dpc{}
	cart.formatID = "DPC"
	cart.description = "DPC Pitfall2 style"
	cart.banks = make([][]uint8, cart.numBanks())

	if len(data) < bankSize*cart.numBanks()+gfxSize {
		return nil, errors.New(errors.CartridgeError, fmt.Sprintf("%s: wrong number of bytes in the cartridge file", cart.formatID))
	}

	for k := 0; k < cart.numBanks(); k++ {
		cart.banks[k] = make([]uint8, bankSize)
		offset := k * bankSize
		cart.banks[k] = data[offset : offset+bankSize]
	}

	gfxStart := cart.numBanks() * bankSize
	cart.gfx = data[gfxStart : gfxStart+gfxSize]

	cart.initialise()

	return cart, nil
}

func (cart dpc) String() string {
	return fmt.Sprintf("%s [%s] Bank: %d", cart.description, cart.formatID, cart.bank)
}

func (cart dpc) format() string {
	return cart.formatID
}

func (cart *dpc) initialise() {
	cart.bank = len(cart.banks) - 1
}

func (cart *dpc) read(addr uint16) (uint8, error) {
	var data uint8

	// chip select is active by definition when read() is called. pump RNG [col 7, ln 58-62, fig 8]
	cart.rng |= (cart.rng>>3)&0x01 ^ (cart.rng>>4)&0x01 ^ (cart.rng>>5)&0x01 ^ (cart.rng>>7)&0x01
	cart.rng <<= 1

	// if address is above register space then we only need to check for bank
	// switching before returning data at the quoted address
	if addr > 0x003f {
		if addr == 0x0ff8 {
			cart.bank = 0
		} else if addr == 0x0ff9 {
			cart.bank = 1
		} else {
			data = cart.banks[cart.bank][addr]
		}
		return data, nil
	}

	// * the remaining addresses are function registers [col 4, ln 10-20]

	// the first eight registers do not touch the data fetchers and therefore
	// do not trigger any of the side-effects on the data fetchers's counter
	// registers [see below]
	if addr >= 0x0000 && addr <= 0x0003 {
		// RNG value
		return cart.rng, nil

	} else if addr >= 0x0004 && addr <= 0x0007 {
		// music value. mix music data-fetchers:

		// generate SIN signal which is the equivalent of the flag register
		// when in music mode [col 7, ln 30-31]

		// SIN signals are weighted and added together [col 7, ln 3-7, fig 12]

		if cart.fetcher[5].musicMode && cart.fetcher[5].flag {
			data += 4
		}

		if cart.fetcher[6].musicMode && cart.fetcher[6].flag {
			data += 5
		}

		if cart.fetcher[7].musicMode && cart.fetcher[7].flag {
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
	gfxAddr := uint16(cart.fetcher[f].hi)<<8 | uint16(cart.fetcher[f].low)

	// only the 11 least-significant bits are used. gfx memory is also
	// addressed with reference from memtop so inverse the bits
	gfxAddr = gfxAddr&0x07ff ^ 0x07ff

	// set flag
	cart.fetcher[f].setFlag()

	if f >= 0x5 && cart.fetcher[f].musicMode {
		// when in music mode return top register [col 7, ln 6-9]
		data = cart.fetcher[f].top

	} else {
		if addr >= 0x0008 && addr <= 0x000f {
			// display data
			data = cart.gfx[gfxAddr]

		} else if addr >= 0x0010 && addr <= 0x0017 {
			// display data AND w/flag
			if cart.fetcher[f].flag {
				data = cart.gfx[gfxAddr]
			}

		} else if addr >= 0x0018 && addr <= 0x001f {
			// display data AND w/flag, nibbles swapped

		} else if addr >= 0x0020 && addr <= 0x0027 {
			// display data AND w/flag, byte reversed

		} else if addr >= 0x0028 && addr <= 0x002f {
			// display data AND w/flag, ROR
			if cart.fetcher[f].flag {
				data = cart.gfx[gfxAddr] >> 1
			}

		} else if addr >= 0x0030 && addr <= 0x0037 {
			// display data AND w/flag, ROL
			if cart.fetcher[f].flag {
				data = cart.gfx[gfxAddr] << 1
			}

		} else if addr >= 0x0038 && addr <= 0x003f {
			// DFx flag
			if f >= 0x5 && cart.fetcher[f].flag {
				data = 0xff
			}
		}
	}

	// clock signal is active whenever data fetcher is used
	cart.fetcher[f].clk()

	return data, nil
}

func (cart *dpc) write(addr uint16, data uint8) error {
	if addr == 0x0ff8 {
		cart.bank = 0
	} else if addr == 0x0ff9 {
		cart.bank = 1
	} else {

		// if the write address if a write address then the effect is on a
		// specific data-fetcher. the data-fetcher is specified by the three
		// least-significant bits
		f := addr & 0x0007

		if addr >= 0x0040 && addr <= 0x0047 {
			// set top register
			cart.fetcher[f].top = data
			cart.fetcher[f].flag = false

		} else if addr >= 0x0048 && addr <= 0x004f {
			// set bottom register
			cart.fetcher[f].bottom = data

		} else if addr >= 0x0050 && addr <= 0x0057 {
			// set low register

			// treat music mode capable registers slightly differently
			if f >= 0x5 && cart.fetcher[f].musicMode {
				// low is loaded with top value on low function [col 7, ln 12-14]
				cart.fetcher[f].low = cart.fetcher[f].top

			} else {
				cart.fetcher[f].low = data

			}

		} else if addr >= 0x0058 && addr <= 0x005f {
			// set high register
			cart.fetcher[f].hi = data

			// treat music mode capable registers slightly differently
			if f >= 0x5 && addr >= 0x005d { // && addr <= 0x00f5 is implied
				// set music mode [col 7, ln 1-6]
				cart.fetcher[f].musicMode = data&0x10 == 0x10

				// set osc clock [col 7, ln 20-22]
				cart.fetcher[f].oscClock = data&0x20 == 0x20
			}

		} else if addr >= 0x0070 && addr <= 0x0077 {
			// reset random number generator
			cart.rng = 0xff

		}

		// other addresses are not write registers and are ignored
	}

	return nil
}

func (cart dpc) numBanks() int {
	return 2
}

func (cart *dpc) setBank(addr uint16, bank int) error {
	cart.bank = bank
	return nil
}

func (cart dpc) getBank(addr uint16) int {
	return cart.bank
}

func (cart *dpc) saveState() interface{} {
	return nil
}

func (cart *dpc) restoreState(state interface{}) error {
	return nil
}

func (cart *dpc) listen(addr uint16, data uint8) {
}

func (cart *dpc) poke(addr uint16, data uint8) error {
	return errors.New(errors.UnpokeableAddress, addr)
}

func (cart *dpc) patch(addr uint16, data uint8) error {
	return errors.New(errors.UnpatchableCartType, cart.description)
}

func (cart dpc) getRAMinfo() []RAMinfo {
	return nil
}

func (cart *dpc) step() {
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
			if cart.fetcher[f].musicMode && cart.fetcher[f].oscClock {
				cart.fetcher[f].clk()
				cart.fetcher[f].setFlag()
			}
		}
	}
}
