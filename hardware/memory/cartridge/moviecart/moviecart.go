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

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/logger"
)

// default preference values.
const (
	titleCycles = 1000000
	osdColor    = 0x9a // blue
	osdDuration = 180
)

// size of each video field in bytes.
const fieldSize = 2560

// offsets into each part of the field (audio, color, etc.)
const (
	offsetVersion      = 0
	offsetFieldNumber  = 4
	offsetAudioData    = 7
	offsetGraphData    = 269
	offsetTimecodeData = 1229
	offsetColorData    = 1289
	offsetEndData      = 2249
)

// levels for both volume and brightness.
const (
	levelDefault = 6
)

// colours from the color stream are ajdusted accordin the brightness level.
var brightLevels = [...]uint8{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 15, 15, 15, 15}

// volume can take 11 levels. data from the audio stream can be one of 16
// levels, the actual value written to the VCS is looked up from the correct
// volume array.
var volumeLevels = [11][16]uint8{
	/* 0.0000 */
	{8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8},
	/* 0.1667 */
	{6, 6, 7, 7, 7, 7, 7, 7, 8, 8, 8, 8, 8, 8, 9, 9},
	/* 0.3333 */
	{5, 5, 6, 6, 6, 7, 7, 7, 8, 8, 8, 9, 9, 9, 10, 10},
	/* 0.5000 */
	{4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11},
	/* 0.6667 */
	{3, 3, 4, 5, 5, 6, 7, 7, 8, 9, 9, 10, 11, 11, 12, 13},
	/* 0.8333 */
	{1, 2, 3, 4, 5, 5, 6, 7, 8, 9, 10, 10, 11, 12, 13, 14},
	/* 1.0000 */
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	/* 1.3611 */
	{0, 0, 0, 1, 3, 4, 5, 7, 8, 10, 11, 12, 14, 15, 15, 15},
	/* 1.7778 */
	{0, 0, 0, 0, 1, 3, 5, 7, 8, 10, 12, 14, 15, 15, 15, 15},
	/* 2.2500 */
	{0, 0, 0, 0, 0, 2, 4, 6, 9, 11, 13, 15, 15, 15, 15, 15},
	/* 2.7778 */
	{0, 0, 0, 0, 0, 1, 3, 6, 9, 12, 14, 15, 15, 15, 15, 15},
}

// the state machine moves from condition to condition processing the movie
// stream.
type stateMachineCondition int

// list of valid state machine conditionss.
const (
	stateMachineRight stateMachineCondition = iota
	stateMachineLeft
	stateMachineNewField
)

const noForceColor uint8 = 0x00

// the current control mode for the OSD.
type controlMode int

// list of control modes.
const (
	modeVolume     controlMode = 0
	modeBrightness controlMode = 1
	modeTime       controlMode = 2
	modeMax        controlMode = 2
)

// a frame consists of two interlaced fields
const numFields = 2

type state struct {
	// the state of the address pins
	a7       bool
	a10      bool
	a11      bool
	a12      bool
	a10Count uint8

	// memory in the moviecart hardware. the core is copied to this and
	// modified as required during movie playback.
	sram []byte

	// which page of SRAM we will be writing to
	writePage int // 0 to 7

	// data for current field and the indexes into it
	streamBuffer   [numFields][]byte
	streamField    int
	streamAudio    int
	streamGraph    int
	streamTimecode int
	streamColor    int

	// which chunk of the buffer to read next
	streamChunk int

	// total number of address cycles the movie has been playing for
	totalCycles int

	// how many lines remaining for the current TIA frame
	lines int

	// field number. a frame is made up of two fields. when playback is paused
	// the fieldNumber will flip between two consecutive values producing one
	// frame of two fields.
	fieldNumber int
	oddField    bool

	// state machine
	state stateMachineCondition

	// is playback paused
	paused bool

	// volume of audio
	volume int

	// brightness of image
	brightness int

	// force of color or B&W
	forceBW bool

	// transport
	directionLast transportDirection
	buttonsLast   transportButtons

	// the amount to move the stream. usually one.
	fieldAdv int

	// the mode being edited with the stick
	controlMode controlMode

	// the length of time the joystick has been help left or right
	controlRepeat int

	// the OSD is to be displayed for the remaining duration
	osdDuration int

	// what part of the osdDisplay is being shown at the current line of the field
	osdDisplay osdDisplay

	// index into the static osd data
	osdIdx int
}

// what part of the OSD is currently being display
type osdDisplay int

const (
	osdNone osdDisplay = iota
	osdLabel
	osdLevels
	osdTime
)

func newState() *state {
	s := &state{
		sram: make([]byte, 0x400),
	}
	s.streamBuffer[0] = make([]byte, fieldSize)
	s.streamBuffer[1] = make([]byte, fieldSize)
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
	s.state = stateMachineNewField
	s.paused = false
	s.streamChunk = 0
	s.volume = levelDefault
	s.brightness = levelDefault
	s.fieldAdv = 1
	s.a10Count = 0
}

type Moviecart struct {
	mappingID   string
	description string

	loader    cartridgeloader.Streamer
	numChunks int

	banks []byte

	state *state
}

func NewMoviecart(loader cartridgeloader.Streamer) (mapper.CartMapper, error) {
	cart := &Moviecart{
		loader:      loader,
		mappingID:   "MC",
		description: "Moviecart",
	}

	var err error
	cart.numChunks, err = loader.NumChunks(fieldSize)
	if err != nil {
		return nil, curated.Errorf("MVC: %v", err)
	}

	cart.state = newState()
	cart.banks = make([]byte, 4096)

	// put core data into bank ROM. this is so that we have something to
	// disassemble.
	//
	// TODO: get rid of the banks[] array and just use the state.sram array
	copy(cart.banks, coreData)
	copy(cart.banks[len(coreData):], coreData)
	copy(cart.banks[len(coreData)*2:], coreData)
	copy(cart.banks[len(coreData)*3:], coreData)

	return cart, nil
}

// Mapping implements the mapper.CartMapper interface.
func (cart *Moviecart) Mapping() string {
	return fmt.Sprintf("Field: %d", cart.state.fieldNumber)
}

// ID implements the mapper.CartMapper interface.
func (cart *Moviecart) ID() string {
	return "MVC"
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
	return cart.state.sram[addr&0x3ff], nil
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
	// note that as it is, this will be a copy of the core program without any
	// of the updates that happen as a result of playing a movie
	c := make([]mapper.BankContent, len(cart.banks))
	c[0] = mapper.BankContent{Number: 0,
		Data:    cart.banks,
		Origins: []uint16{memorymap.OriginCart},
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
		cart.write8bit(addrTitleLoop, 0x18)
	} else if cart.state.totalCycles > titleCycles {
		cart.runStateMachine()
	}
}

func (cart *Moviecart) updateDirection() {
	var direction transportDirection
	t := transportDirection(cart.state.a10Count)
	t = ^(t & 0x1e)
	t &= 0x1e
	direction = t

	if direction.isUp() && !cart.state.directionLast.isUp() {
		if cart.state.controlMode == 0 {
			cart.state.controlMode = modeMax
		} else {
			cart.state.controlMode--
		}
	} else if direction.isDown() && !cart.state.directionLast.isDown() {
		if cart.state.controlMode == modeMax {
			cart.state.controlMode = 0
		} else {
			cart.state.controlMode++
		}
	}

	if direction.isLeft() || direction.isRight() {
		cart.state.controlRepeat++
		if cart.state.controlRepeat > 16 {
			cart.state.controlRepeat = 0

			switch cart.state.controlMode {
			case modeTime:
				cart.state.osdDuration = osdDuration
				if direction.isLeft() {
					cart.state.fieldAdv -= 4
				} else if direction.isRight() {
					cart.state.fieldAdv += 4
				}
			case modeVolume:
				cart.state.osdDuration = osdDuration
				if direction.isLeft() {
					if cart.state.volume > 0 {
						cart.state.volume--
					}
				} else if direction.isRight() {
					if cart.state.volume < len(volumeLevels) {
						cart.state.volume++
					}
				}
			case modeBrightness:
				cart.state.osdDuration = osdDuration
				if direction.isLeft() {
					if cart.state.brightness > 0 {
						cart.state.brightness--
					}
				} else if direction.isRight() {
					if cart.state.brightness < len(brightLevels) {
						cart.state.brightness++
					}
				}
			}
		}

		// if playback paused then single step frame when joystick is moveed left/right
		if cart.state.paused {
			if direction.isLeft() {
				cart.state.streamChunk--
				if cart.state.streamChunk < 0 {
					cart.state.streamChunk = 0
				}
			}
			if direction.isRight() {
				cart.state.streamChunk++
			}
		}
	} else {
		cart.state.controlRepeat = 0
		cart.state.fieldAdv = 1
	}

	cart.state.directionLast = direction
}

func (cart *Moviecart) updateButtons() {
	var buttons transportButtons

	t := transportButtons(cart.state.a10Count)
	t = ^(t & 0x17)
	t &= 0x17
	buttons = t

	// B&W switch
	cart.state.forceBW = buttons.isBW()

	// reset switch
	if buttons.isReset() {
		cart.state.streamChunk = 0
		cart.state.paused = false
		return
	}

	// pause on button release
	if buttons.isButton() && !cart.state.buttonsLast.isButton() {
		cart.state.paused = !cart.state.paused
	}

	cart.state.buttonsLast = buttons
}

func (cart *Moviecart) updateTransport() {
	// alternate between direction and button servicing
	if cart.state.streamField == 1 {
		cart.updateDirection()
	} else {
		cart.updateButtons()
	}

	// we're done with a10 count now so reset it
	cart.state.a10Count = 0

	// move movie stream
	if !cart.state.paused {
		cart.state.streamChunk += cart.state.fieldAdv

		// bounds check for stream chunk
		if cart.state.streamChunk < 0 {
			cart.state.streamChunk = 0
		}
		if cart.state.streamChunk > cart.numChunks {
			cart.state.streamChunk = cart.numChunks
		}
	}
}

func (cart *Moviecart) runStateMachine() {
	switch cart.state.state {
	case stateMachineRight:
		if !cart.state.a7 {
			break // switch
		}

		if cart.state.osdDuration > 0 {
			switch cart.state.controlMode {
			case modeTime:
				if cart.state.lines == 11 {
					cart.state.osdDuration--
					cart.state.osdDisplay = osdTime
					cart.state.osdIdx = offsetTimecodeData
				}

			default:
				if cart.state.lines == 21 {
					cart.state.osdDuration--
					cart.state.osdDisplay = osdLabel
					cart.state.osdIdx = 0
				}

				if cart.state.lines == 7 {
					cart.state.osdDisplay = osdLevels
					switch cart.state.controlMode {
					case modeBrightness:
						cart.state.osdIdx = cart.state.brightness * 40
					case modeVolume:
						cart.state.osdIdx = cart.state.volume * 40
					}
				}
			}
		}

		cart.fillAddrRightLine()
		cart.state.lines--
		cart.state.state = stateMachineLeft
	case stateMachineLeft:
		if cart.state.a7 {
			break // switch
		}

		if cart.state.lines >= 1 {
			cart.fillAddrLeftLine(true)
			cart.state.lines--
			cart.state.state = stateMachineRight
		} else {
			cart.fillAddrLeftLine(false)
			cart.fillAddrEndLines()
			cart.fillAddrBlankLines()

			// swap stream indexes
			cart.state.streamField++
			if cart.state.streamField >= numFields {
				cart.state.streamField = 0
			}

			cart.updateTransport()
			cart.state.state = stateMachineNewField
		}
	case stateMachineNewField:
		if !cart.state.a7 {
			break // switch
		}

		cart.readField()
		cart.state.lines = 191
		cart.state.state = stateMachineRight
		cart.state.osdDisplay = osdNone
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
		cart.write16bit(addrPickContinue+1, addrRightLine)
	} else {
		cart.write16bit(addrPickContinue+1, addrEndLines)
	}
}

func (cart *Moviecart) fillAddrEndLines() {
	cart.setWritePage(addrEndLines)

	cart.writeAudio(addrEndLinesAudio + 1)

	// different details for the end kernel every other frame
	if cart.state.oddField {
		cart.write8bit(addrSetOverscanSize+1, 28)
		cart.write8bit(addrSetVBlankSize+1, 36)
	} else {
		cart.write8bit(addrSetOverscanSize+1, 29)
		cart.write8bit(addrSetVBlankSize+1, 37)
	}

	if cart.state.streamField == 0 {
		cart.write16bit(addrPickTransport+1, addrTransportDirection)
	} else {
		cart.write16bit(addrPickTransport+1, addrTransportButtons)
	}
}

func (cart *Moviecart) fillAddrBlankLines() {
	cart.setWritePage(addrAudioBank)

	const blankLineSize = 68

	// slightly different number of trailing blank line every other frame
	if cart.state.oddField {
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
	b := cart.state.streamBuffer[cart.state.streamField][cart.state.streamAudio]
	cart.state.streamAudio++
	b = volumeLevels[cart.state.volume][b]

	if cart.state.paused {
		cart.write8bit(addr, 0)
	} else {
		cart.write8bit(addr, b)
	}
}

func (cart *Moviecart) writeGraph(addr uint16) {
	var b byte

	// check if we need to draw OSD using stats graphics data
	switch cart.state.osdDisplay {
	case osdTime:
		if cart.state.osdIdx < offsetColorData {
			b = cart.state.streamBuffer[cart.state.streamField][cart.state.osdIdx]
			cart.state.osdIdx++
		}
	case osdLevels:
		if cart.state.osdIdx < levelBarsLen {
			if cart.state.oddField {
				b = levelBarsOddData[cart.state.osdIdx]
			} else {
				b = levelBarsEvenData[cart.state.osdIdx]
			}
			cart.state.osdIdx++
		}
	case osdLabel:
		switch cart.state.controlMode {
		case modeBrightness:
			if cart.state.osdIdx < brightLabelLen {
				if cart.state.oddField {
					b = brightLabelOdd[cart.state.osdIdx]
				} else {
					b = brightLabelEven[cart.state.osdIdx]
				}
				cart.state.osdIdx++
			}

		case modeVolume:
			if cart.state.osdIdx < volumeLabelLen {
				if cart.state.oddField {
					b = volumeLabelOdd[cart.state.osdIdx]
				} else {
					b = volumeLabelEven[cart.state.osdIdx]
				}
				cart.state.osdIdx++
			}
		}
	case osdNone:
		// use graphics from current field
		b = cart.state.streamBuffer[cart.state.streamField][cart.state.streamGraph]
		cart.state.streamGraph++
	}

	cart.write8bit(addr, b)
}

func (cart *Moviecart) writeColor(addr uint16) {
	b := cart.state.streamBuffer[cart.state.streamField][cart.state.streamColor]
	cart.state.streamColor++

	// adjust brightness
	brightIdx := int(b & 0x0f)
	brightIdx += cart.state.brightness
	if brightIdx >= len(brightLevels) {
		brightIdx = len(brightLevels) - 1
	}
	b = b&0xf0 | brightLevels[brightIdx]

	// forcing a particular color means we've been drawing pixels from a timecode
	// or OSD label or level meter.
	if cart.state.osdDisplay != osdNone {
		b = osdColor
	}

	// best effort conversion of color to B&W
	if cart.state.forceBW {
		b &= 0x0f
	}

	cart.write8bit(addr, b)
}

const chunkSize = 8 * 512

func (cart *Moviecart) readField() {
	// reset stream indexes
	defer func() {
		cart.state.streamAudio = offsetAudioData
		cart.state.streamGraph = offsetGraphData
		cart.state.streamTimecode = offsetTimecodeData
		cart.state.streamColor = offsetColorData
	}()

	// do not read more data if playback is paused or this is the first stream
	// chunk - the second part of the condition handles the condition when the
	// user has searched back to the beginning of the movie and is holding the
	// stick left and trying to go back more. in that instance the movie is
	// essentially paused.
	if !cart.state.paused && cart.state.streamChunk > 0 {
		dataOffset := cart.state.streamChunk * chunkSize
		err := cart.loader.Stream(int64(dataOffset), cart.state.streamBuffer[cart.state.streamField])
		if err != nil {
			logger.Logf("MVC", "error reading field: %v", err)
		}
	}

	// frame number and odd parity check. we recalculate these every field
	// regardless of whether we've read new data in.
	cart.state.fieldNumber = int(cart.state.streamBuffer[cart.state.streamField][offsetFieldNumber]) << 16
	cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamField][offsetFieldNumber+1]) << 8
	cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamField][offsetFieldNumber+2])
	cart.state.oddField = cart.state.fieldNumber&0x01 == 0x01
}

// the address used when writing to SRAM is made up of the writePage and the lo
// bits specified when calling write8bit() or write16bit()
//
// for a 16 bit value, the X bits are unused, P bits is the writePage and L
// bits the lo value.
//
//   XXXXXX PPP LLLLLLL
//          \_________/
//              |
//          0 - 1023 (maximum address in SRAM)
//
// for convenience the write page is taken from a complete reference address.
func (cart *Moviecart) setWritePage(addr uint16) {
	cart.state.writePage = int(addr>>7) & 0x07
}

// write 8bits of data to SRAM. the address in sram is made up of the current
// writePage value and the lo argument, which will for the lower 7bits of the
// address.
func (cart *Moviecart) write8bit(lo uint16, data uint8) {
	addr := uint16(cart.state.writePage<<7) | (lo & 0x7f)
	cart.state.sram[addr&0x3ff] = data
}

// write 16bits of data to SRAM. the address in sram is made up of the current
// writePage value and the lo argument, which will for the lower 7bits of the
// address.
func (cart *Moviecart) write16bit(lo uint16, data uint16) {
	addr := uint16(cart.state.writePage<<7) | (lo & 0x7f)
	cart.state.sram[addr&0x3ff] = uint8(data & 0xff)
	cart.state.sram[(addr+1)&0x3ff] = uint8(data>>8) | 0x10
}
