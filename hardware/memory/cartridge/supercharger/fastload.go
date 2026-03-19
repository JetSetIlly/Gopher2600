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
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

// FastLoad implements the tape interface. It loads data from a binary file
// rather than a sound file.
//
// On success it returns the FastLoaded error. This must be interpreted by the
// emulator driver and called with the arguments listed in the error type.
type FastLoad struct {
	env *environment.Environment

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

	fastLoadPageLen   = 0x100
	fastLoadPageCount = 0x18

	// the page table is part of the header
	fastLoadPageTableOffset = 0x10

	// the checksum table is part of the header
	fastLoadPageChecksumTableOffset = 0x40
)

type fastLoadBlock struct {
	data []byte

	// remainder of block is the "header"

	// PC address to jump to once loading has finished
	startAddressLo uint8
	startAddressHi uint8

	// RAM config to be set after tape load
	configByte uint8

	// number of pages to load
	numPages uint8

	// checksum of fields in header (excluding pageTable and pageChecksums)
	headerChecksum uint8

	// we'll use this to check if the correct multiload is being read
	multiload uint8

	// not using progress speed in any meaningul way
	progressSpeed uint16

	// data is loaded according to page table
	pageTable [fastLoadPageCount]byte

	// pageChecksums of the pages in the data
	pageChecksums [fastLoadPageCount]byte
}

// from 'Stolberg': "checksum (the sum over all 8 game header bytes must be $55)"
const fastloadChecksumBase = 0x55

func (b *fastLoadBlock) setChecksums() {
	b.headerChecksum = fastloadChecksumBase
	b.headerChecksum -= b.startAddressLo + b.startAddressHi +
		b.configByte + b.numPages + b.multiload +
		uint8(b.progressSpeed) + uint8(b.progressSpeed>>8)

	for c, p := range b.pageChecksums {
		p = fastloadChecksumBase
		for _, d := range b.data[c*fastLoadPageLen : (c+1)*fastLoadHeaderLen] {
			p -= d
		}
		p -= b.pageTable[c]
		b.pageChecksums[c] = p
	}

	if !b.verifyChecksum() {
		panic("error in supercharger/fastload checksums. this is a programming error in the setChecksum() function")
	}
}

func (b *fastLoadBlock) verifyChecksum() bool {
	var verified bool

	headerChecksum := b.headerChecksum + b.startAddressLo + b.startAddressHi +
		b.configByte + b.numPages + b.multiload +
		uint8(b.progressSpeed) + uint8(b.progressSpeed>>8)

	verified = headerChecksum == fastloadChecksumBase

	for c, p := range b.pageChecksums {
		for _, d := range b.data[c*fastLoadPageLen : (c+1)*fastLoadHeaderLen] {
			p += d
		}
		p += b.pageTable[c]
		verified = verified && p == fastloadChecksumBase
	}

	return verified
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

	h[0] = b.startAddressLo
	h[1] = b.startAddressHi
	h[2] = b.configByte
	h[3] = b.numPages
	h[4] = b.headerChecksum
	h[5] = b.multiload
	h[6] = byte(b.progressSpeed)
	h[7] = byte(b.progressSpeed >> 8)
	copy(h[fastLoadPageTableOffset:fastLoadPageTableOffset+len(b.pageTable)], b.pageTable[:])
	copy(h[fastLoadPageChecksumTableOffset:fastLoadPageChecksumTableOffset+len(b.pageChecksums)], b.pageChecksums[:])

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
func newFastLoad(env *environment.Environment) (tape, error) {
	if env.Loader.Size()%fastLoadBlockLen != 0 {
		return nil, fmt.Errorf("fastload: wrong number of bytes in cartridge data")
	}

	fl := &FastLoad{
		env: env,
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
		fl.blocks[i].startAddressLo = gameHeader[0]
		fl.blocks[i].startAddressHi = gameHeader[1]
		fl.blocks[i].configByte = gameHeader[2]
		fl.blocks[i].numPages = gameHeader[3]
		fl.blocks[i].headerChecksum = gameHeader[4]
		fl.blocks[i].multiload = gameHeader[5]
		fl.blocks[i].progressSpeed = (uint16(gameHeader[7]) << 8) | uint16(gameHeader[6])
		copy(fl.blocks[i].pageTable[:], gameHeader[fastLoadPageTableOffset:fastLoadPageTableOffset+len(fl.blocks[i].pageTable)])
		copy(fl.blocks[i].pageChecksums[:], gameHeader[fastLoadPageChecksumTableOffset:fastLoadPageChecksumTableOffset+len(fl.blocks[i].pageChecksums)])

		logger.Logf(fl.env, "supercharger: fastload", "block %d: start address: %#04x", i,
			uint16(fl.blocks[i].startAddressLo)|(uint16(fl.blocks[fl.blockIdx].startAddressHi)<<8),
		)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: config byte: %#08b", i, fl.blocks[i].configByte)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: num pages: %d", i, fl.blocks[i].numPages)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: checksum: %#02x", i, fl.blocks[i].headerChecksum)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: multiload: %#02x", i, fl.blocks[i].multiload)
		logger.Logf(fl.env, "supercharger: fastload", "block %d: progress speed: %#02x", i, fl.blocks[i].progressSpeed)

		for b := range 3 {
			logger.Logf(fl.env, "supercharger: fastload", "block %d: page-table: bank %d % 02x",
				i, b, fl.blocks[i].pageTable[8*b:8*(b+1)],
			)
		}

		for b := range 3 {
			logger.Logf(fl.env, "supercharger: fastload", "block %d: page checksums: bank %d % 02x",
				i, b, fl.blocks[i].pageChecksums[8*b:8*(b+1)],
			)
		}

		if fl.blocks[i].verifyChecksum() == false {
			logger.Logf(fl.env, "supercharger: fastload", "block %d: checksums incorrect (will now correct)", i)
			fl.blocks[i].setChecksums()
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
func (fl *FastLoad) plumb(env *environment.Environment) {
	fl.env = env
}

// load implements the tape interface.
func (fl *FastLoad) load() (uint8, error) {
	err := fl.env.Notifications.Notify(notifications.NotifySuperchargerFastLoad)
	if err != nil {
		return 0x00, fmt.Errorf("fastload: %w", err)
	}

	return 0x00, nil
}

// step implements the tape interface.
func (fl *FastLoad) step() {
}

// load implements the tape interface.
func (tap *FastLoad) end() {
}

// bootstrap implements the tape interface
func (fl *FastLoad) bootstrap(state *state, mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer, tia *tia.TIA) error {
	// look up requested multiload address
	m, err := ram.Peek(MutliloadByteAddr)
	if err != nil {
		return fmt.Errorf("fastload: %w", err)
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
		copy(state.ram[bank][ramOffset:ramOffset+0x100], fl.blocks[fl.blockIdx].data[dataOffset:dataOffset+0x100])
	}

	// set the value to be used in the first instruction of the bootstrap program
	state.registers.Value = fl.blocks[fl.blockIdx].configByte
	state.registers.Delay = 0

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
	_ = ram.Poke(jmpAddrLo, fl.blocks[fl.blockIdx].startAddressLo)
	_ = ram.Poke(jmpAddrHi, fl.blocks[fl.blockIdx].startAddressHi)

	// reset timer. in references to real tape loading, the number of ticks
	// is the value at the moment the PC reaches address 0x00fa
	tmr.PokeField("divider", timer.TIM64T)
	tmr.PokeField("ticksRemaining", 0x1f)
	tmr.PokeField("intim", uint8(0x0a))
	tmr.PokeField("pa7", false)

	// suicide mission does not reset the vertical delay registers which will have been enabled at the
	// start of the BIOS load routine and disabled somewhere in the part of the BIOS we skip
	//
	// even though this only affects suicide mission, it's still correct that we do this
	tia.Video.Player0.SetVerticalDelay(false)
	tia.Video.Player1.SetVerticalDelay(false)

	// the other tia video components should also be reset to match the state at the end of the BIOS
	// load routine. as far as I can tell none of these affect any starpath game but it may affect a
	// non-original supercharger binary that's been converted to wav and then dumped to an AR file
	tia.Video.Player0.SetNUSIZ(0)
	tia.Video.Player1.SetNUSIZ(0)
	tia.Video.Ball.Hmove = 8

	// we should also set the positions of the tia video components but that's not convenient to do
	// at the moment. but it's an improvement to consider for the future

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
			return fmt.Errorf("fastload: romdump: block %d: %w", i, err)
		}
	}
	return nil
}
