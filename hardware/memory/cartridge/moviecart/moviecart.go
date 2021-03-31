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

package moviecart

import (
	"fmt"
	"math/rand"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

const (
	titleCycles    = 1000000
	timeCodeHeight = 12
	blankLineSize  = 68
)

const (
	addrTitleLoop          = 0xb50
	addrRightLine          = 0x94c
	addrLeftLine           = 0x980
	addrPickContinue       = 0x9c2
	addrEndLines           = 0xa80
	addrEndLinesAudio      = 0xaa1
	addrSetOverscanSize    = 0xaad
	addrSetVBlankSize      = 0xac3
	addrPickTransport      = 0xacc
	addrTransportDirection = 0x897
	addrTransportButtons   = 0x880
	addrAudioBank          = 0xb80
	addrLastAudio          = 0xacf
)

const (
	fieldSize    = 2560
	sramMask     = 1023
	defaultLevel = 6
	maxLevel     = 11
)

const (
	offsetFrameData    = 1
	offsetAudioData    = 4
	offsetGraphData    = 266
	offsetTimecodeData = 1226
	offsetColorData    = 1286
	offsetEndData      = 2246
)

type state struct {
	a7       bool
	a10      bool
	a11      bool
	a12      bool
	a10Count int

	sram      []byte
	writePage int // 0 to 7

	streamBuffer   []byte
	streamData     int
	streamAudio    int
	streamGraph    int
	streamTimecode int
	streamColor    int

	totalCycles int
	lines       int
	frameNumber int
	play        bool
	state       int
	odd         bool
	bufferIndex bool

	// volume
	mainVolume    int
	levelBarsOdd  int
	levelBarsEven int

	drawTimecode  int
	drawLevelbars int

	forceColor int
	forceBW    bool
}

func newState() *state {
	s := &state{
		sram: make([]byte, sramMask+1),
	}
	s.streamBuffer = make([]byte, fieldSize)
	return s
}

// Snapshot implements the mapper.CartMapper interface.
func (s *state) Snapshot() *state {
	n := *s
	return &n
}

func (s *state) initialise() {
	s.totalCycles = 0

	copy(s.sram, coreData)

	s.state = 3
	s.play = true
	s.odd = false

	s.mainVolume = defaultLevel
	s.levelBarsOdd = 0
	s.levelBarsEven = 0

	s.a10Count = 0
}

type Moviecart struct {
	mappingID   string
	description string

	movieData []byte
	banks     [][]byte

	state *state
}

func NewMoviecart(data []byte) (mapper.CartMapper, error) {
	cart := &Moviecart{
		mappingID:   "MC",
		description: "Moviecart",
		movieData:   data,
	}

	cart.state = newState()
	cart.banks = make([][]byte, 1)
	cart.banks[0] = make([]byte, 4096)

	// put core data into bank ROM. this is so that we have something to
	// disassemble.
	//
	// TODO: get rid of the banks[] array and just use the state.sram array
	copy(cart.banks[0], coreData)
	copy(cart.banks[0][len(coreData):], coreData)
	copy(cart.banks[0][len(coreData)*2:], coreData)
	copy(cart.banks[0][len(coreData)*3:], coreData)

	return cart, nil
}

// Mapping implements the mapper.CartMapper interface.
func (cart *Moviecart) Mapping() string {
	return fmt.Sprintf("Frame: %d", cart.state.frameNumber)
}

// ID implements the mapper.CartMapper interface.
func (cart *Moviecart) ID() string {
	return "MC"
}

// Snapshot implements the mapper.CartMapper interface.
func (cart *Moviecart) Snapshot() mapper.CartMapper {
	n := *cart
	n.state = cart.state.Snapshot()
	return &n
}

// Plumb implements the mapper.CartMapper interface.
func (cart *Moviecart) Plumb() {
}

// Reset implements the mapper.CartMapper interface.
func (cart *Moviecart) Reset(randSrc *rand.Rand) {
}

// Read implements the mapper.CartMapper interface.
func (cart *Moviecart) Read(addr uint16, active bool) (data uint8, err error) {
	return cart.state.sram[addr&sramMask], nil
}

// Write implements the mapper.CartMapper interface.
func (cart *Moviecart) Write(addr uint16, data uint8, active bool, poke bool) error {
	return nil
}

// NumBanks implements the mapper.CartMapper interface.
func (cart *Moviecart) NumBanks() int {
	return 1
}

// GetBank implements the mapper.CartMapper interface.
func (cart *Moviecart) GetBank(addr uint16) mapper.BankInfo {
	return mapper.BankInfo{}
}

// Listen implements the mapper.CartMapper interface.
func (cart *Moviecart) Listen(addr uint16, data uint8) {
	cart.processAddress(addr)
}

// Step implements the mapper.CartMapper interface.
func (cart *Moviecart) Step(clock float32) {
}

// Patch implements the mapper.CartMapper interface.
func (cart *Moviecart) Patch(offset int, data uint8) error {
	return nil
}

// CopyBanks implements the mapper.CartMapper interface.
func (cart *Moviecart) CopyBanks() []mapper.BankContent {
	c := make([]mapper.BankContent, len(cart.banks))
	for b := 0; b < len(cart.banks); b++ {
		c[b] = mapper.BankContent{Number: b,
			Data:    cart.banks[b],
			Origins: []uint16{memorymap.OriginCart},
		}
	}
	return c
}

func (cart *Moviecart) processAddress(addr uint16) {
	if addr >= 0xfffa {
		cart.state.initialise()
	}

	cart.state.a12 = addr&(1<<12) == 1<<12
	cart.state.a11 = addr&(1<<11) == 1<<11

	if cart.state.a11 {
		cart.state.a7 = addr&(1<<7) == 1<<7
	}

	a10 := addr&(1<<10) == 1<<10
	if a10 && !cart.state.a10 {
		cart.state.a10Count++
	}
	cart.state.a10 = a10

	cart.state.totalCycles++
	if cart.state.totalCycles == titleCycles {
		// stop title screen
		cart.setWritePage(addrTitleLoop)
		cart.writeSRAM(addrTitleLoop, 0x18)
	} else if cart.state.totalCycles > titleCycles {
		cart.runStateMachine()
	}
}

func (cart *Moviecart) runStateMachine() {
	switch cart.state.state {
	case 1:
		if !cart.state.a7 {
			break // switch
		}

		// TODO: draw timecode and levelbars

		cart.fillAddrRightLine()
		cart.state.lines--
		cart.state.state = 2
	case 2:
		if cart.state.a7 {
			break // switch
		}

		if cart.state.lines >= 1 {
			cart.fillAddrLeftLine(true)
			cart.state.lines--
			cart.state.state = 1
		} else {
			cart.fillAddrLeftLine(false)
			cart.fillAddrEndLines()

			// TODO: update transport

			// frameNumber advancement should be in update transport where the
			// playback speed can be controlled
			cart.state.frameNumber++

			cart.fillAddrBlankLines()
			cart.state.state = 3
		}
	case 3:
		if !cart.state.a7 {
			break // switch
		}

		cart.readField()
		cart.state.forceColor = 0
		cart.state.lines = 191
		cart.state.state = 1
	}
}

func (cart *Moviecart) fillAddrRightLine() {
	cart.setWritePage(addrRightLine)
	cart.writeGraph(addrRightLine + 9)
	cart.writeGraph(addrRightLine + 13)
	cart.writeGraph(addrRightLine + 17)
	cart.writeGraph(addrRightLine + 21)
	cart.writeGraph(addrRightLine + 23)
	cart.writeColor(addrRightLine + 25)
	cart.writeColor(addrRightLine + 29)
	cart.writeColor(addrRightLine + 35)
	cart.writeColor(addrRightLine + 43)
	cart.writeColor(addrRightLine + 47)
}

func (cart *Moviecart) fillAddrLeftLine(again bool) {
	cart.setWritePage(addrLeftLine)

	cart.writeAudio(addrLeftLine + 5)
	cart.writeGraph(addrLeftLine + 15)
	cart.writeGraph(addrLeftLine + 19)
	cart.writeGraph(addrLeftLine + 23)
	cart.writeGraph(addrLeftLine + 27)
	cart.writeGraph(addrLeftLine + 29)
	cart.writeColor(addrLeftLine + 31)
	cart.writeColor(addrLeftLine + 35)
	cart.writeColor(addrLeftLine + 41)
	cart.writeColor(addrLeftLine + 49)
	cart.writeColor(addrLeftLine + 53)
	cart.writeAudio(addrLeftLine + 57)

	if again {
		cart.writeSRAM((addrPickContinue + 1), addrRightLine&0xff)
		cart.writeSRAM((addrPickContinue + 2), (addrRightLine>>8)|0x10)
	} else {
		cart.writeSRAM((addrPickContinue + 1), addrEndLines&0xff)
		cart.writeSRAM((addrPickContinue + 2), (addrEndLines>>8)|0x10)
	}
}

func (cart *Moviecart) fillAddrEndLines() {
	cart.setWritePage(addrEndLines)

	cart.writeAudio(addrEndLinesAudio + 1)

	if cart.state.odd {
		cart.writeSRAM((addrSetOverscanSize + 1), 28)
		cart.writeSRAM((addrSetVBlankSize + 1), 36)
		cart.writeSRAM((addrPickTransport + 1), addrTransportDirection&0xff)
		cart.writeSRAM((addrPickTransport + 2), (addrTransportDirection>>8)|0x10)
	} else {
		cart.writeSRAM((addrSetOverscanSize + 1), 29)
		cart.writeSRAM((addrSetVBlankSize + 1), 37)
		cart.writeSRAM((addrPickTransport + 1), addrTransportButtons&0xff)
		cart.writeSRAM((addrPickTransport + 2), (addrTransportButtons>>8)|0x10)
	}
}

func (cart *Moviecart) fillAddrBlankLines() {
	// version number
	cart.state.streamData++

	// frame number
	cart.state.streamData++
	cart.state.streamData++
	cart.state.odd = cart.state.streamBuffer[cart.state.streamData]&0x01 == 0x01
	cart.state.streamData++

	cart.setWritePage(addrAudioBank)

	if cart.state.odd {
		for i := uint16(0); i < blankLineSize; i++ {
			cart.writeAudio(addrAudioBank + i)
		}
	} else {
		for i := uint16(0); i < blankLineSize-1; i++ {
			cart.writeAudio(addrAudioBank + i)
		}
	}

	cart.setWritePage(addrEndLines)
	cart.writeAudio(addrLastAudio + 1)
}

func (cart *Moviecart) writeAudio(addr uint16) {
	b := cart.state.streamBuffer[cart.state.streamAudio]
	// TODO: adjust volume
	cart.state.streamAudio++
	cart.writeSRAM(addr, b)
}

func (cart *Moviecart) writeGraph(addr uint16) {
	b := cart.state.streamBuffer[cart.state.streamGraph]
	cart.state.streamGraph++
	cart.writeSRAM(addr, b)
}

func (cart *Moviecart) writeColor(addr uint16) {
	b := cart.state.streamBuffer[cart.state.streamColor]
	cart.state.streamColor++
	// TODO: brightness and force color and force B&W
	cart.writeSRAM(addr, b)
}

func (cart *Moviecart) readField() {
	dataOffset := cart.state.frameNumber * (8 * 512)

	// loop frames (not in the original code)
	if dataOffset > len(cart.movieData) || dataOffset+fieldSize > len(cart.movieData) {
		dataOffset = 0
	}

	copy(cart.state.streamBuffer, cart.movieData[dataOffset:dataOffset+len(cart.state.streamBuffer)])

	cart.state.streamData = 0
	cart.state.streamAudio = offsetAudioData
	cart.state.streamGraph = offsetGraphData
	cart.state.streamTimecode = offsetTimecodeData
	cart.state.streamColor = offsetColorData
}

func (cart *Moviecart) setWritePage(addr uint16) {
	cart.state.writePage = int(addr>>7) & 0x07
}

// lo is lower 7 bits of SRAM address
func (cart *Moviecart) writeSRAM(lo uint16, data uint8) {
	addr := uint16(cart.state.writePage<<7) | (lo & 0x7f)
	cart.state.sram[addr&sramMask] = data
}
