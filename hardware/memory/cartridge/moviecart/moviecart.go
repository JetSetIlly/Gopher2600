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
	"io"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/hardware/instance"
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
	offsetVersion      = 0 // nolint
	offsetFieldNumber  = 4
	offsetAudioData    = 7
	offsetGraphData    = 269
	offsetTimecodeData = 1229
	offsetColorData    = 1289
	offsetColorBkData  = 2249
	offsetEndData      = 2441
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

// the current control mode for the OSD.
type controlMode int

// list of control modes.
const (
	modeVolume     controlMode = 0
	modeBrightness controlMode = 1
	modeTime       controlMode = 2
	modeMax        controlMode = 2
)

// a frame consists of two interlaced fields.
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

	// data for current field and the indexes into it
	streamBuffer     [numFields][]byte
	streamField      int
	streamAudio      int
	streamGraph      int
	streamTimecode   int
	streamColor      int
	streamBackground int
	endOfStream      bool

	// the audio value to carry to the next frame
	audioCarry uint8

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

	blankLine bool

	// state machine
	state stateMachineCondition

	justReset bool

	// is playback paused. pauseStep is used to allow frame-by-frame stepping.
	paused    bool
	pauseStep int

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

	// whether the cartridge loader has indicated that it wants to shorten the
	// title card duration
	shortTitleCard bool
}

// what part of the OSD is currently being display.
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
	if s.shortTitleCard {
		// shorten title card. we don't want to eliminate it entirely so say
		// that it has run 75% of the normal duration already on initialisation
		s.totalCycles = titleCycles * 0.75
	} else {
		s.totalCycles = 0
	}
	copy(s.sram, coreData)
	s.state = stateMachineNewField
	s.paused = false
	s.streamChunk = 0
	s.volume = levelDefault
	s.brightness = levelDefault
	s.fieldAdv = 1
	s.a10Count = 0
	s.controlMode = modeMax
	s.justReset = true
}

type Moviecart struct {
	instance *instance.Instance

	mappingID   string
	description string

	loader io.ReadSeekCloser
	banks  []byte

	state *state
}

func NewMoviecart(ins *instance.Instance, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	cart := &Moviecart{
		instance:    ins,
		loader:      loader.StreamedData,
		mappingID:   "MC",
		description: "Moviecart",
	}

	cart.state = newState()
	cart.banks = make([]byte, 4096)

	// if the emulation has been labelled as a thumbnailer then shorten the
	// title card sequence
	if ins.Label == instance.Thumbnailer {
		cart.state.shortTitleCard = true
		cart.state.initialise()
	}

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

// MappedBanks implements the mapper.CartMapper interface.
func (cart *Moviecart) MappedBanks() string {
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
func (cart *Moviecart) Reset() {
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

	if cart.state.justReset {
		cart.state.directionLast = direction
		cart.state.justReset = false
		return
	}

	if direction.isUp() && !cart.state.directionLast.isUp() {
		cart.state.osdDuration = osdDuration
		if cart.state.controlMode == 0 {
			cart.state.controlMode = modeMax
		} else {
			cart.state.controlMode--
		}
	} else if direction.isDown() && !cart.state.directionLast.isDown() {
		cart.state.osdDuration = osdDuration
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
				if cart.state.paused {
					if direction.isLeft() {
						cart.state.pauseStep = -1
					} else if direction.isRight() {
						cart.state.pauseStep = 1
					}
				} else {
					if direction.isLeft() {
						cart.state.fieldAdv -= 4
					} else if direction.isRight() {
						cart.state.fieldAdv += 4
					}
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
		} else if cart.state.paused {
			if direction.isLeft() {
				if !cart.state.directionLast.isLeft() {
					cart.state.osdDuration = osdDuration
					cart.state.pauseStep = -1
				} else {
					cart.state.pauseStep = 0
				}
			} else if direction.isRight() {
				if !cart.state.directionLast.isRight() {
					cart.state.osdDuration = osdDuration
					cart.state.pauseStep = 1
				} else {
					cart.state.pauseStep = 0
				}
			}
		}
	} else {
		cart.state.controlRepeat = 0
		cart.state.fieldAdv = 1
		cart.state.pauseStep = 0
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

	// move movie stream if playback is not paused.
	//
	// frame-by-frame stepping when playback is paused is handled in the
	// readField() function
	if !cart.state.paused {
		if cart.state.endOfStream {
			if cart.state.fieldAdv < 0 {
				cart.state.streamChunk += cart.state.fieldAdv
				cart.state.endOfStream = false
			}
		} else {
			cart.state.streamChunk += cart.state.fieldAdv
		}

		// bounds check for stream chunk
		if cart.state.streamChunk < 0 {
			cart.state.streamChunk = 0
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
			cart.state.blankLine = cart.state.oddField

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
		cart.state.blankLine = false
	}
}

func (cart *Moviecart) fillAddrRightLine() {
	cart.writeAudio(addrSetAudRight + 1)

	cart.writeGraph(addrSetGData5 + 1)
	cart.writeGraph(addrSetGData6 + 1)
	cart.writeGraph(addrSetGData7 + 1)
	cart.writeGraph(addrSetGData8 + 1)
	cart.writeGraph(addrSetGData9 + 1)

	cart.writeColorStream(addrSetGCol5 + 1) // col 1/9
	cart.writeColorStream(addrSetGCol6 + 1) // col 3/9
	cart.writeColorStream(addrSetGCol7 + 1) // col 5/9
	cart.writeColorStream(addrSetGCol8 + 1) // col 7/9
	cart.writeColorStream(addrSetGCol9 + 1) // col 9/9

	cart.writeBackgroundStream(addrSetBkColR+1, true)
	// cart.writeBackgroundStream(addrSetPfColR+1, false)
}

func (cart *Moviecart) fillAddrLeftLine(again bool) {
	cart.writeAudio(addrSetAudLeft + 1)

	cart.writeGraph(addrSetGData0 + 1)
	cart.writeGraph(addrSetGData1 + 1)
	cart.writeGraph(addrSetGData2 + 1)
	cart.writeGraph(addrSetGData3 + 1)
	cart.writeGraph(addrSetGData4 + 1)

	cart.writeColorStream(addrSetGCol0 + 1) // col 0/9
	cart.writeColorStream(addrSetGCol1 + 1) // col 2/9
	cart.writeColorStream(addrSetGCol2 + 1) // col 4/9
	cart.writeColorStream(addrSetGCol3 + 1) // col 6/9
	cart.writeColorStream(addrSetGCol4 + 1) // col 8/9

	// cart.writeBackgroundStream(addrSetBkColL+1, false)
	cart.writeBackgroundStream(addrSetPfColL+1, true)

	if again {
		cart.writeJMPaddr(addrPickContinue+1, addrRightLine)
	} else {
		cart.writeJMPaddr(addrPickContinue+1, addrEndLines)
	}
}

func (cart *Moviecart) fillAddrEndLines() {
	cart.writeAudio(addrSetAudEndlines + 1)

	// different details for the end kernel every other frame
	if cart.state.oddField {
		cart.write8bit(addrSetOverscanSize+1, 29)
		cart.write8bit(addrSetVBlankSize+1, 36)
	} else {
		cart.state.audioCarry = cart.state.streamBuffer[cart.state.streamField][cart.state.streamAudio]
		cart.write8bit(addrSetOverscanSize+1, 30)
		cart.write8bit(addrSetVBlankSize+1, 37)
	}

	if cart.state.streamField == 0 {
		cart.writeJMPaddr(addrPickTransport+1, addrTransportDirection)
	} else {
		cart.writeJMPaddr(addrPickTransport+1, addrTransportButtons)
	}
}

func (cart *Moviecart) fillAddrBlankLines() {
	const blankLineSize = 69

	// slightly different number of trailing blank line every other frame
	if !cart.state.oddField {
		cart.writeAudioData(addrAudioBank, cart.state.audioCarry)
		for i := uint16(1); i < blankLineSize+1; i++ {
			cart.writeAudio(addrAudioBank + i)
		}
	} else {
		for i := uint16(0); i < blankLineSize-1; i++ {
			cart.writeAudio(addrAudioBank + i)
		}
	}
}

func (cart *Moviecart) writeAudio(addr uint16) {
	b := cart.state.streamBuffer[cart.state.streamField][cart.state.streamAudio]
	cart.state.streamAudio++
	cart.writeAudioData(addr, b)
}

func (cart *Moviecart) writeAudioData(addr uint16, data uint8) {
	b := volumeLevels[cart.state.volume][data]

	// output silence if playback is paused or we have reached the end of the stream
	if cart.state.paused || cart.state.endOfStream || cart.state.streamChunk == 0 {
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

func (cart *Moviecart) writeColorStream(addr uint16) {
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

	if cart.state.blankLine {
		b = 0x00
	}

	cart.write8bit(addr, b)
}

func (cart *Moviecart) writeBackgroundStream(addr uint16, readCol bool) {
	var b byte
	if readCol && cart.state.osdDisplay == osdNone {
		b = cart.state.streamBuffer[cart.state.streamField][cart.state.streamBackground]
		cart.state.streamBackground++
	}

	// adjust brightness
	brightIdx := int(b & 0x0f)
	brightIdx += cart.state.brightness
	if brightIdx >= len(brightLevels) {
		brightIdx = len(brightLevels) - 1
	}
	b = b&0xf0 | brightLevels[brightIdx]

	// best effort conversion of color to B&W
	if cart.state.forceBW {
		b &= 0x0f
	}

	cart.write8bit(addr, b)
}

const chunkSize = 8 * 512

func (cart *Moviecart) readField() {
	// the usual playback condition
	if !cart.state.paused && cart.state.streamChunk > 0 {
		dataOffset := cart.state.streamChunk * chunkSize
		_, err := cart.loader.Seek(int64(dataOffset), io.SeekStart)
		if err != nil {
			logger.Logf("MVC", "error reading field: %v", err)
		}
		n, err := cart.loader.Read(cart.state.streamBuffer[cart.state.streamField])
		if err != nil {
			logger.Logf("MVC", "error reading field: %v", err)
		}
		cart.state.endOfStream = n < fieldSize
	}

	// if playback is paused and pauseStep is not zero then handle
	// frame-by-frame stepping especially
	if cart.state.paused && cart.state.pauseStep != 0 {
		for fld := 0; fld < numFields; fld++ {
			cart.state.streamChunk += cart.state.pauseStep
			if cart.state.streamChunk < 0 {
				cart.state.streamChunk = 0
			}

			dataOffset := cart.state.streamChunk * chunkSize
			_, err := cart.loader.Seek(int64(dataOffset), io.SeekStart)
			if err != nil {
				logger.Logf("MVC", "error reading field: %v", err)
			}
			_, err = cart.loader.Read(cart.state.streamBuffer[fld])
			if err != nil {
				logger.Logf("MVC", "error reading field: %v", err)
			}

			// version string check
			if cart.state.streamBuffer[fld][0] != 'M' ||
				cart.state.streamBuffer[fld][1] != 'V' ||
				cart.state.streamBuffer[fld][2] != 'C' ||
				cart.state.streamBuffer[fld][3] != 0x00 {
				logger.Logf("MVC", "unrecognised version string in chunk %d", cart.state.streamChunk)
			}
		}
	}

	// frame number and odd parity check. we recalculate these every field
	// regardless of whether we've read new data in.
	cart.state.fieldNumber = int(cart.state.streamBuffer[cart.state.streamField][offsetFieldNumber]) << 16
	cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamField][offsetFieldNumber+1]) << 8
	cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamField][offsetFieldNumber+2])
	cart.state.oddField = cart.state.fieldNumber&0x01 == 0x01

	// reset stream indexes
	cart.state.streamAudio = offsetAudioData
	cart.state.streamGraph = offsetGraphData
	cart.state.streamTimecode = offsetTimecodeData
	cart.state.streamColor = offsetColorData
	cart.state.streamBackground = offsetColorBkData

	// cart.state.streamBackground++
	if cart.state.oddField {
		cart.state.streamBackground++
	}
}

// write 8bits of data to SRAM. the address in sram is made up of the current
// writePage value and the lo argument, which will for the lower 7bits of the
// address.
func (cart *Moviecart) write8bit(addr uint16, data uint8) {
	cart.state.sram[addr&0x3ff] = data
}

// write 16bits of data to SRAM. the address in sram is made up of the current
// writePage value and the lo argument, which will for the lower 7bits of the
// address.
func (cart *Moviecart) writeJMPaddr(addr uint16, jmpAddr uint16) {
	cart.state.sram[addr&0x3ff] = uint8(jmpAddr & 0xff)
	cart.state.sram[(addr+1)&0x3ff] = uint8(jmpAddr>>8) | 0x10
}
