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
	"io"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
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
	env  *environment.Environment
	cart *Supercharger

	// fastload binaries have a header which controls how the binary is read
	blocks   []fastloadBlock
	blockIdx int

	// value of loadCt on last successful load. we use this to prevent endless
	// rewinding and searching
	lastLoadCt int
}

// a fastload binary can have several blocks
const fastLoadHeaderOffset = 0x2000
const fastLoadHeaderLen = 0x100
const fastLoadBlockLen = fastLoadHeaderOffset + fastLoadHeaderLen

type fastloadBlock struct {
	data []byte

	// remainder of block is the "header"

	// PC address to jump to once loading has finished
	startAddress uint16

	// RAM config to be set adter tape load
	configByte uint8

	// number of pages to load
	numPages uint8

	// not using checksum in any meaningful way
	checksum uint8

	// we'll use this to check if the correct multiload is being read
	multiload uint8

	// not using progress speed in any meaningul way
	progressSpeed uint16

	// data is loaded according to page table
	pageTable []byte
}

// newFastLoad is the preferred method of initialisation for the FastLoad type.
func newFastLoad(env *environment.Environment, cart *Supercharger, loader cartridgeloader.Loader) (tape, error) {
	if loader.Size()%fastLoadBlockLen != 0 {
		return nil, fmt.Errorf("fastload: wrong number of bytes in cartridge data")
	}

	fl := &FastLoad{
		env:  env,
		cart: cart,
	}

	fl.blocks = make([]fastloadBlock, loader.Size()/fastLoadBlockLen)
	data, err := io.ReadAll(loader)
	if err != nil {
		return nil, err
	}

	for i := range fl.blocks {
		offset := i * fastLoadBlockLen
		fl.blocks[i].data = data[offset : offset+fastLoadHeaderOffset]

		// game header appears after main data
		gameHeader := fl.blocks[i].data[fastLoadHeaderOffset : fastLoadHeaderOffset+fastLoadHeaderLen]
		fl.blocks[i].startAddress = (uint16(gameHeader[1]) << 8) | uint16(gameHeader[0])
		fl.blocks[i].configByte = gameHeader[2]
		fl.blocks[i].numPages = uint8(gameHeader[3])
		fl.blocks[i].checksum = gameHeader[4]
		fl.blocks[i].multiload = gameHeader[5]
		fl.blocks[i].progressSpeed = (uint16(gameHeader[7]) << 8) | uint16(gameHeader[6])
		fl.blocks[i].pageTable = gameHeader[0x10:0x28]

		logger.Logf(fl.env, "supercharger: fastload", "block %d: start address: %#04x", i, fl.blocks[i].startAddress)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: config byte: %#08b", i, fl.blocks[i].configByte)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: num pages: %d", i, fl.blocks[i].numPages)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: checksum: %#02x", i, fl.blocks[i].checksum)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: multiload: %#02x", i, fl.blocks[i].multiload)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: progress speed: %#02x", i, fl.blocks[i].progressSpeed)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: page-table: %v", i, fl.blocks[i].pageTable)

	}

	return fl, nil
}

// snapshot implements the tape interface.
func (fl *FastLoad) snapshot() tape {
	// this function doesn't copy anything. data array in each snapshot will
	// point to the same data array
	return fl
}

// load implements the tape interface.
func (fl *FastLoad) load() (uint8, error) {
	// setup cartridge according to tape instructions
	fl.cart.env.Notifications.Notify(notifications.NotifySuperchargerFastload)
	return 0, nil
}

// step implements the Tape interface.
func (fl *FastLoad) step() error {
	return nil
}

// load implements the Tape interface.
func (tap *FastLoad) end() error {
	return nil
}

// Fastload implements the mapper.CartSuperChargerFastLoad interface.
func (fl *FastLoad) Fastload(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer) error {
	// look up requested multiload address
	m, err := ram.Peek(MutliloadByteAddress)
	if err != nil {
		return fmt.Errorf("fastload %w", err)
	}

	// check whether the block is the one we want to load
	//
	// note the blockIdx we're starting off with so that we can prevent an
	// infinite loop
	startBlockIdx := fl.blockIdx
	for m != fl.blocks[fl.blockIdx].multiload {
		fl.blockIdx++
		if fl.blockIdx >= len(fl.blocks) {
			fl.blockIdx = 0
		}
		if fl.blockIdx == startBlockIdx {
			logger.Logf(fl.env, "supercharger: fastload", "cannot find multiload %d", m)
			fl.blockIdx = 0
			break // for loop
		}
	}

	// log loading of multiload for non-zero multiload values
	if m != 0 {
		logger.Logf(fl.env, "supercharger: fastload", "loading multiload %d", fl.blocks[fl.blockIdx].multiload)
	}

	// copy data to RAM banks
	for i := 0; i < int(fl.blocks[fl.blockIdx].numPages); i++ {
		bank := fl.blocks[fl.blockIdx].pageTable[i] & 0x3
		page := fl.blocks[fl.blockIdx].pageTable[i] >> 2
		ramOffset := int(page) * 0x100
		dataOffset := i * 0x100
		copy(fl.cart.state.ram[bank][ramOffset:ramOffset+0x100], fl.blocks[fl.blockIdx].data[dataOffset:dataOffset+0x100])
	}

	// poke values into RAM. these values would be the by-product of the
	// tape-loading process. because we are short-circuiting that process
	// however, by injecting the binary data into supercharger RAM
	// directly, the necessary code will not be run.

	// RAM address 0x80 contains the initial configbyte
	_ = ram.Poke(0x80, fl.blocks[fl.blockIdx].configByte)

	// CMP $fff8
	_ = ram.Poke(0xfa, 0xcd)
	_ = ram.Poke(0xfb, 0xf8)
	_ = ram.Poke(0xfc, 0xff)

	// JMP <absolute address>
	_ = ram.Poke(0xfd, 0x4c)
	_ = ram.Poke(0xfe, uint8(fl.blocks[fl.blockIdx].startAddress))
	_ = ram.Poke(0xff, uint8(fl.blocks[fl.blockIdx].startAddress>>8))

	// reset timer. in references to real tape loading, the number of ticks
	// is the value at the moment the PC reaches address 0x00fa
	tmr.PokeField("divider", timer.TIM64T)
	tmr.PokeField("ticksRemaining", 0x1e)
	tmr.PokeField("intim", uint8(0x0a))

	// jump to VCS RAM location 0x00fa. a short bootstrap program has been
	// poked there already
	err = mc.LoadPC(0x00fa)
	if err != nil {
		return fmt.Errorf("fastload: %w", err)
	}

	// set the value to be used in the first instruction of the bootstrap program
	fl.cart.state.registers.Value = fl.blocks[fl.blockIdx].configByte
	fl.cart.state.registers.Delay = 0

	return nil
}
