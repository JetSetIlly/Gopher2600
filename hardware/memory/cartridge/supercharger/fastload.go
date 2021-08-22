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

package supercharger

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
	"github.com/jetsetilly/gopher2600/logger"
)

// FastLoad implements the Tape interface. It loads data from a binary file
// rather than a sound file.
//
// On success it returns the FastLoaded error. This must be interpreted by the
// emulator driver and called with the arguments listed in the error type.
//
// Format information for fast-loca binary rom mailing list post:
//
// Subject: Re: [stella] Supercharger BIN format
// From: Eckhard Stolberg
// Date: Fri, 08 Jan 1999.
type FastLoad struct {
	cart *Supercharger
	data []byte

	// number of loads in the data
	numLoads int

	// which load is to be tried next
	loadCt int

	// value of loadCt on last successful load. we use this to prevent endless
	// rewinding and searching
	lastLoadCt int
}

// FastLoaded defines the callback function that is sent to the core emulation
// along with the HookActionFastloadEnded action. The core emulation in turn
// calls this function to complete the supercharger fastload process.
type FastLoaded func(*cpu.CPU, *vcs.RAM, *timer.Timer) error

// newFastLoad is the preferred method of initialisation for the FastLoad type.
func newFastLoad(cart *Supercharger, loader cartridgeloader.Loader) (tape, error) {
	tap := &FastLoad{
		cart: cart,
		data: loader.Data,
	}

	if len(tap.data)%8448 != 0 {
		return nil, fmt.Errorf("fastload: %v", "wrong number of bytes in cartridge data")
	}
	tap.numLoads = len(tap.data) / 8448

	return tap, nil
}

// snapshot implements the tape interface.
func (tap *FastLoad) snapshot() tape {
	// this function doesn't copy anything. data array in each snapshot will
	// point to the same data array
	return tap
}

// load implements the tape interface.
func (tap *FastLoad) load() (uint8, error) {
	// get data for the next multiload
	offset := tap.loadCt * 8448
	data := tap.data[offset : offset+8448]
	tap.loadCt++
	if tap.loadCt*8448 >= len(tap.data) {
		tap.loadCt = 0
		logger.Log("supercharger: fastload", "rewind")
	}

	// game header appears after main data
	gameHeader := data[0x2000:0x2008]

	// PC address to jump to once loading has finished
	startAddress := (uint16(gameHeader[1]) << 8) | uint16(gameHeader[0])

	// RAM config to be set adter tape load
	configByte := gameHeader[2]

	// number of pages to load
	numPages := int(gameHeader[3])

	// not using checksum in any meaningful way
	checksum := gameHeader[4]

	// we'll use this to check if the correct multiload is being read
	multiload := gameHeader[5]

	// not using progress speed in any meaningul way
	progressSpeed := (uint16(gameHeader[7]) << 8) | uint16(gameHeader[6])

	logger.Logf("supercharger: fastload", "header: start address: %#04x", startAddress)
	logger.Logf("supercharger: fastload", "header: config byte: %#08b", configByte)
	logger.Logf("supercharger: fastload", "header: num pages: %d", numPages)
	logger.Logf("supercharger: fastload", "header: checksum: %#02x", checksum)
	logger.Logf("supercharger: fastload", "header: multiload: %#02x", multiload)
	logger.Logf("supercharger: fastload", "header: progress speed: %#02x", progressSpeed)

	// data is loaded according to page table
	pageTable := data[0x2010:0x2028]
	logger.Logf("supercharger: fastload", "page-table: %v", pageTable)

	// setup cartridge according to tape instructions. this requires
	// cooperation from the core emulation so we use the
	// cartridgeloader.VCSHook mechanism.
	tap.cart.vcsHook(tap.cart, mapper.EventSuperchargerFastloadEnded, FastLoaded(func(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer) error {
		// look up requested multiload address
		m, err := ram.Peek(MutliloadByteAddress)
		if err != nil {
			return curated.Errorf("supercharger: fastload %v", err)
		}

		// this is not the mutliload we're looking for
		if m != multiload {
			logger.Logf("supercharger: fastload", "fastload header (%d) not matching multiload request (%d)", m, multiload)

			// test for whether the tape has looped. if it has just load the
			// first multiload
			if tap.loadCt == tap.lastLoadCt {
				logger.Logf("supercharger: fastload", "cannot find requested multiload (%d) loading mutliload 00", m)
				tap.loadCt = 0
				offset := tap.loadCt * 8448
				data = tap.data[offset : offset+8448]
				tap.loadCt++
			} else {
				return nil
			}
		}

		logger.Logf("supercharger: fastload", "loading multiload (%d)", multiload)

		// copy data to RAM banks
		for i := 0; i < numPages; i++ {
			bank := pageTable[i] & 0x3
			page := pageTable[i] >> 2
			bankOffset := int(page) * 0x100
			binOffset := i * 0x100

			data := data[binOffset : binOffset+0x100]
			copy(tap.cart.state.ram[bank][bankOffset:bankOffset+0x100], data)

			// logger.Logf("supercharger: fastload", "copying %#04x:%#04x to bank %d page %d, offset %#04x", binOffset, binOffset+0x100, bank, page, bankOffset)
		}

		// initialise VCS RAM with zeros
		for a := uint16(0x80); a <= 0xff; a++ {
			_ = ram.Poke(a, 0x00)
		}

		// poke values into RAM. these values would be the by-product of the
		// tape-loading process. because we are short-circuiting that process
		// however, by injecting the binary data into supercharger RAM
		// directly, the necessary code will not be run.

		// RAM address 0x80 contains the initial configbyte
		_ = ram.Poke(0x80, configByte)

		// CMP $fff8
		_ = ram.Poke(0xfa, 0xcd)
		_ = ram.Poke(0xfb, 0xf8)
		_ = ram.Poke(0xfc, 0xff)

		// JMP <absolute address>
		_ = ram.Poke(0xfd, 0x4c)
		_ = ram.Poke(0xfe, uint8(startAddress))
		_ = ram.Poke(0xff, uint8(startAddress>>8))

		// reset timer. in references to real tape loading, the number of ticks
		// is the value at the moment the PC reaches address 0x00fa
		tmr.SetInterval("TIM64T")
		tmr.SetValue(0x0a)
		tmr.SetTicks(0x1e)

		// jump to VCS RAM location 0x00fa. a short bootstrap program has been
		// poked there already
		err = mc.LoadPC(0x00fa)
		if err != nil {
			return curated.Errorf("supercharger: fastload: %v", err)
		}

		// set the value to be used in the first instruction of the bootstrap program
		tap.cart.state.registers.Value = configByte
		tap.cart.state.registers.Delay = 0

		// note the multiload request
		tap.lastLoadCt = tap.loadCt

		return nil
	}))

	return 0, nil
}

// step implements the Tape interface.
func (tap *FastLoad) step() {
}
