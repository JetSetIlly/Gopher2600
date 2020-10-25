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
	"github.com/jetsetilly/gopher2600/hardware/cpu"
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
}

// FastLoaded error is returned on success of FastLoad.load(). It must be
// honoured (ie. caught and the function called) by the driving emulator for
// the fastload process to complete.
//
// It is unusual to use an error in this way but this is the only effective way
// of sending a signal that something unusual has happened. In that sense, it
// is an error, the normal run of the emulator has been interrupted and must be
// handled.
//
// That there is nothing else like this in the 2600 emulation, using an error
// like this is justified. It is an exception to the rule, so to speak, and
// setting up another way of sending the "fast load completed" signal is
// inappropriate.
//
// !!TODO: is there a good way of handling FastLoading completion through the
// cartridgeloader.OnLoader() mechanism?
type FastLoaded func(*cpu.CPU, *vcs.RAM, *timer.Timer) error

func (er FastLoaded) Error() string {
	return "supercharger tape loaded, preparing VCS"
}

// newFastLoad is the preferred method of initialisation for the FastLoad type.
func newFastLoad(cart *Supercharger, loader cartridgeloader.Loader) (tape, error) {
	tap := &FastLoad{
		cart: cart,
		data: loader.Data,
	}

	l := len(tap.data)
	if l != 8448 && l != 25344 && l != 33792 {
		return nil, fmt.Errorf("fastload: %v", "wrong number of bytes in cartridge data")
	}

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
	gameData := tap.data[0:0x1eff]

	// only 8448 .bin format is supported currently
	gameHeader := tap.data[0x2000:0x2008]

	// PC address to jump to once loading has finished
	startAddress := (uint16(gameHeader[1]) << 8) | uint16(gameHeader[0])

	// RAM config to be set adter tape load
	configByte := gameHeader[2]

	// number of pages to load
	numPages := int(gameHeader[3])

	// not using the following in any meaningful way
	checksum := gameHeader[4]
	multiload := gameHeader[5]
	progressSpeed := (uint16(gameHeader[7]) << 8) | uint16(gameHeader[6])

	logger.Log("supercharger: fastload", fmt.Sprintf("start address: %#04x", startAddress))
	logger.Log("supercharger: fastload", fmt.Sprintf("config byte: %#08b", configByte))
	logger.Log("supercharger: fastload", fmt.Sprintf("num pages: %d", numPages))
	logger.Log("supercharger: fastload", fmt.Sprintf("checksum: %#02x", checksum))
	logger.Log("supercharger: fastload", fmt.Sprintf("multiload: %#02x", multiload))
	logger.Log("supercharger: fastload", fmt.Sprintf("progress speed: %#02x", progressSpeed))

	// data is loaded according to page table
	pageTable := tap.data[0x2010:0x2028]
	logger.Log("supercharger: fastload", fmt.Sprintf("page-table: %v", pageTable))

	// copy data to RAM banks
	for i := 0; i < numPages; i++ {
		bank := pageTable[i] & 0x3
		page := pageTable[i] >> 2
		bankOffset := int(page) * 0x100
		binOffset := i * 0x100

		data := gameData[binOffset : binOffset+0x100]
		copy(tap.cart.state.ram[bank][bankOffset:bankOffset+0x100], data)

		logger.Log("supercharger: fastload", fmt.Sprintf("copying %#04x:%#04x to bank %d page %d, offset %#04x", binOffset, binOffset+0x100, bank, page, bankOffset))
	}

	// setup cartridge according to tape instructions. we do this by returning
	// a function disguised as an error type. The VCS knows how to interpret
	// this error and will call the function
	return 0, FastLoaded(func(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer) error {
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
		err := mc.LoadPC(0x00fa)
		if err != nil {
			return fmt.Errorf("fastload: %v", err)
		}

		// set the value to be used in the first instruction of the bootstrap program
		tap.cart.state.registers.Value = configByte
		tap.cart.state.registers.Delay = 0

		return nil
	})
}

// step implements the Tape interface.
func (tap *FastLoad) step() {
}
