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
// Format information for fast-load binary rom mailing list post:
//
// Subject: Re: [stella] Supercharger BIN format
// From: Eckhard Stolberg
// Date: Fri, 08 Jan 1999.
type FastLoad struct {
	env   *environment.Environment
	state *state

	// fastload binaries have a header which controls how the binary is read
	blocks   []fastLoadBlock
	blockIdx int

	// value of loadCt on last successful load. we use this to prevent endless
	// rewinding and searching
	lastLoadCt int
}

// a fastload binary can have several blocks
const (
	fastLoadHeaderOffset = 0x2000
	fastLoadHeaderLen    = 0x100
	fastLoadBlockLen     = fastLoadHeaderOffset + fastLoadHeaderLen

	// the page table is part of the header
	fastLoadPageTableOffset = 0x10
	fastLoadPageTableLen    = 0x18
)

type fastLoadBlock struct {
	data []byte

	// remainder of block is the "header"

	// PC address to jump to once loading has finished
	startAddress uint16

	// RAM config to be set after tape load
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

func (b *fastLoadBlock) romdump(w io.Writer) error {
	n, err := w.Write(b.data)
	if err != nil {
		return err
	}
	if n != len(b.data) {
		return fmt.Errorf("data block is incomplete")
	}

	h := make([]byte, fastLoadHeaderLen)

	h[0] = byte(b.startAddress)
	h[1] = byte(b.startAddress >> 8)
	h[2] = b.configByte
	h[3] = b.numPages
	h[4] = b.checksum
	h[5] = b.multiload
	h[6] = byte(b.progressSpeed)
	h[7] = byte(b.progressSpeed >> 8)
	copy(h[fastLoadPageTableOffset:fastLoadPageTableOffset+fastLoadPageTableLen], b.pageTable)

	n, err = w.Write(h)
	if err != nil {
		return err
	}
	if n != len(h) {
		return fmt.Errorf("block header is incomplete")
	}

	return nil
}

// newFastLoad is the preferred method of initialisation for the FastLoad type.
func newFastLoad(env *environment.Environment, state *state) (tape, error) {
	if env.Loader.Size()%fastLoadBlockLen != 0 {
		return nil, fmt.Errorf("fastload: wrong number of bytes in cartridge data")
	}

	fl := &FastLoad{
		env:   env,
		state: state,
	}

	fl.blocks = make([]fastLoadBlock, env.Loader.Size()/fastLoadBlockLen)
	data, err := io.ReadAll(env.Loader)
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
		fl.blocks[i].numPages = gameHeader[3]
		fl.blocks[i].checksum = gameHeader[4]
		fl.blocks[i].multiload = gameHeader[5]
		fl.blocks[i].progressSpeed = (uint16(gameHeader[7]) << 8) | uint16(gameHeader[6])
		fl.blocks[i].pageTable = gameHeader[fastLoadPageTableOffset : fastLoadPageTableOffset+fastLoadPageTableLen]

		logger.Logf(fl.env, "supercharger: fastload", "block %d: start address: %#04x", i, fl.blocks[i].startAddress)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: config byte: %#08b", i, fl.blocks[i].configByte)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: num pages: %d", i, fl.blocks[i].numPages)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: checksum: %#02x", i, fl.blocks[i].checksum)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: multiload: %#02x", i, fl.blocks[i].multiload)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: progress speed: %#02x", i, fl.blocks[i].progressSpeed)

		for b := range 3 {
			logger.Logf(fl.env, "supercharger: fastload", "block %d: page-table: bank %d % 02x",
				i, b, fl.blocks[i].pageTable[8*b:8*(b+1)],
			)
		}
	}

	return fl, nil
}

// snapshot implements the tape interface.
func (fl *FastLoad) snapshot() tape {
	n := *fl
	return &n
}

// plumb implements the tape interface.
func (fl *FastLoad) plumb(state *state, env *environment.Environment) {
	fl.env = env
	fl.state = state
}

// load implements the tape interface.
func (fl *FastLoad) load() (uint8, error) {
	if fl.env.Label == environment.MainEmulation {
		// setup cartridge according to tape instructions
		fl.env.Notifications.Notify(notifications.NotifySuperchargerFastload)
	}
	return 0, nil
}

// step implements the Tape interface.
func (fl *FastLoad) step() {
}

// load implements the Tape interface.
func (tap *FastLoad) end() {
}

// Fastload implements the mapper.CartSuperChargerFastLoad interface.
func (fl *FastLoad) Fastload(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer) error {
	// look up requested multiload address
	m, err := ram.Peek(MutliloadByteAddress)
	if err != nil {
		return fmt.Errorf("fastload %w", err)
	}

	// find the requested fastload block, making sure we don't loop forever
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
		copy(fl.state.ram[bank][ramOffset:ramOffset+0x100], fl.blocks[fl.blockIdx].data[dataOffset:dataOffset+0x100])
	}

	// set the value to be used in the first instruction of the bootstrap program
	fl.state.registers.Value = fl.blocks[fl.blockIdx].configByte
	fl.state.registers.Delay = 0

	// the remainder of this function replicates the pertinent side-effects of the BIOS. it's not
	// certain if this is 100% of the side-effects we need to worry about

	// clear RIOT RAM. we don't clear all of it because that will wreck the persistent state. a
	// cursory examination of the BIOS shows that RAM $82 to $9d are cleared to zero
	//
	//  fd9f  ldx #$1b
	//  fda1  sty $82, x
	//  fda3  dex
	//  fda4  bpl fda1
	for i := uint16(0x0082); i <= 0x009d; i++ {
		_ = ram.Poke(i, 0x00)
	}

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
	tmr.PokeField("ticksRemaining", 0x1f)
	tmr.PokeField("intim", uint8(0x0a))
	tmr.PokeField("pa7", false)

	// jump to VCS RAM location 0x00fa. a short bootstrap program has been
	// poked there already
	err = mc.LoadPC(0x00fa)
	if err != nil {
		return fmt.Errorf("fastload: %w", err)
	}

	return nil
}

func (fl *FastLoad) romdump(w io.Writer) error {
	for i, b := range fl.blocks {
		err := b.romdump(w)
		if err != nil {
			return fmt.Errorf("block %d: %w", i, err)
		}
	}
	return nil
}
