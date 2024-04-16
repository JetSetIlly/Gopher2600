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
	"errors"
	"fmt"
	"io"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

const (
	improveControls = false

	// both of these options are now effectively enabled in recent changes made
	// to the firmware and stella by lodef
	improveNeatStartImageAfterRewind = true
	improveMuteAudioAtStart          = true
)

// default preference values.
const (
	titleCycles = 1000000
	osdColor    = 0x9a // blue
	osdDuration = 180
)

// size of each video field in bytes.
const fieldSize = 4096

// levels for both volume and brightness.
const (
	levelDefault = 6
	levelMax     = 11
)

// colours from the color stream are ajdusted accordin the brightness level.
var brightLevels = [...]uint8{0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 15, 15, 15, 15}

// volume can take 11 levels. data from the audio stream can be one of 16
// levels, the actual value written to the VCS is looked up from the correct
// volume array.
var volumeLevels = [levelMax][16]uint8{
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
	modeVolume controlMode = iota
	modeBrightness
	modeTime
	modeMax
)

// a frame consists of two interlaced fields.
const numFields = 2

type format struct {
	// version   [4]byte // ('M', 'V', 'C', '0')
	// format    byte    // (1-------)
	// timecode  [4]byte // (hour, minute, second, frame)
	vsync    byte // eg 3
	vblank   byte // eg 37
	overscan byte // eg 30
	visible  byte // eg 192
	rate     byte // eg 60
}

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
	streamIndex      int
	streamAudio      int
	streamGraph      int
	streamTimecode   int
	streamColor      int
	streamBackground int
	endOfStream      bool

	// field number is part of the timecode stream. this is the extracted value
	fieldNumber int

	// odd field uses the frame component of the timecode stream to decide if
	// the field is an odd number or not
	oddField bool

	// format for each field. parsed data from streamBuffer
	format [numFields]format

	// the audio value to carry to the next frame
	audioCarry uint8

	// which chunk of the buffer to read next
	streamChunk int

	// total number of address cycles the movie has been playing for
	totalCycles int

	// how many lines remaining for the current TIA frame. ranges from
	// linesThisFrame to zero
	lines int

	// the number of lines int the TIA frame
	linesThisFrame int

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

	// mask bottom edge of the screen when the OSD is visible
	blankPartialLines bool

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

	// what part of the osdControl is being shown at the current line of the field
	osdControl osdControl

	// index into the static osd data
	osdIdx int

	// whether the cartridge loader has indicated that it wants to shorten the
	// title card duration
	shortTitleCard bool

	// the stream has failed because bad data has been encountered
	streamFail bool

	// the most recent rotation instruction. can only ever changes for version 2 streams
	rotation byte
}

// which control is currently being displayed on screen
type osdControl int

const (
	osdNone osdControl = iota
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
	s.justReset = true

	// starting control mode depends on alternativeControls preference
	if improveControls {
		s.controlMode = modeTime
	} else {
		s.controlMode = modeVolume
	}
}

type Moviecart struct {
	env *environment.Environment

	specID    string
	mappingID string

	data  io.ReadSeeker
	banks []byte

	state *state
}

func NewMoviecart(env *environment.Environment, loader cartridgeloader.Loader) (mapper.CartMapper, error) {
	cart := &Moviecart{
		env:       env,
		data:      loader,
		mappingID: "MVC",
	}

	cart.state = newState()
	cart.banks = make([]byte, 4096)

	// shorten title card sequence if this is not the main emulation
	if !env.IsEmulation(environment.MainEmulation) {
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

	// field starts off in the end position
	cart.state.streamIndex = 1

	// read next field straight away. this has the advantage of triggering the
	// first screen rotation in time for the attract screen
	cart.nextField()

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
func (cart *Moviecart) Plumb(env *environment.Environment) {
	cart.env = env
}

// Reset implements the mapper.CartMapper interface.
func (cart *Moviecart) Reset() {
	cart.state.initialise()
}

// Access implements the mapper.CartMapper interface.
func (cart *Moviecart) Access(addr uint16, _ bool) (data uint8, mask uint8, err error) {
	// TODO: this is called far too frequently but it'll do for now
	cart.setConsoleTiming()

	return cart.state.sram[addr&0x3ff], mapper.CartDrivenPins, nil
}

// AccessVolatile implements the mapper.CartMapper interface.
func (cart *Moviecart) AccessVolatile(addr uint16, data uint8, _ bool) error {
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

// AccessPassive implements the mapper.CartMapper interface.
func (cart *Moviecart) AccessPassive(addr uint16, data uint8) error {
	cart.processAddress(addr)
	return nil
}

// Step implements the mapper.CartMapper interface.
func (cart *Moviecart) Step(clock float32) {
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
	// it's possible for a moviecart stream to produce bad data. rather than
	// having bounds checks in the writeAudioData(), etc. we allow the program
	// to panic in this exceptional situation and recover from it with a log
	// entry
	//
	// once an error is encountered no more data is processed
	if cart.state.streamFail {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			cart.state.streamFail = true
			logger.Logf("MVC", "serious data error in moviecart stream")
		}
	}()

	if addr >= 0x1ffa {
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

		err := cart.env.Notifications.Notify(notifications.NotifyMovieCartStarted)
		if err != nil {
			logger.Logf("moviecart", err.Error())
		}

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

	// change control mode with the up/down direction of the stick
	//
	// if alternativeControls are active then the change only occurs when the
	// OSD is display
	if direction.isUp() && !cart.state.directionLast.isUp() {
		if cart.state.osdDuration > 0 || !improveControls {
			if cart.state.controlMode == 0 {
				cart.state.controlMode = modeMax - 1
			} else {
				cart.state.controlMode--
			}
		}
		cart.state.osdDuration = osdDuration
	} else if direction.isDown() && !cart.state.directionLast.isDown() {
		if cart.state.osdDuration > 0 || !improveControls {
			cart.state.controlMode++
			if cart.state.controlMode == modeMax {
				cart.state.controlMode = 0
			}
		}
		cart.state.osdDuration = osdDuration
	}

	if direction.isLeft() || direction.isRight() {
		cart.state.controlRepeat++
		if cart.state.controlRepeat > 10 {
			cart.state.controlRepeat = 0

			switch cart.state.controlMode {
			case modeTime:
				cart.state.osdDuration = osdDuration
				if cart.state.paused {
					// allow pause to step repeatedly if alternative control
					// method is activated
					if improveControls {
						if direction.isLeft() {
							cart.state.pauseStep = -1
						} else if direction.isRight() {
							cart.state.pauseStep = 1
						}
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
					if cart.state.volume < levelMax-1 {
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
					if cart.state.brightness < levelMax-1 {
						cart.state.brightness++
					}
				}
			}
		} else if cart.state.paused {
			// if the left/right direction is not being held and if movie is
			// paused then move one frame foreward/back only if the control mode
			// is in time mode
			switch cart.state.controlMode {
			case modeTime:
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
	if cart.state.streamIndex == 1 {
		cart.updateDirection()
	} else {
		cart.updateButtons()
	}

	// we're done with a10 count now so reset it
	cart.state.a10Count = 0

	// rewind/fast-forward movie stream unless playback is paused
	//
	// frame-by-frame stepping when playback is paused is handled in the
	// nextField() function
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
		if improveNeatStartImageAfterRewind {
			if cart.state.streamChunk < 0 {
				cart.state.streamChunk = cart.state.streamIndex
			}
		} else {
			if cart.state.streamChunk < -1 {
				cart.state.streamChunk = -1
			}
		}

		if cart.state.controlMode == modeMax {
			cart.state.controlMode = modeTime
		}
	}
}

func (cart *Moviecart) runStateMachine() {
	// set blankPartialLines flag
	switch cart.state.lines {
	case 0:
		cart.state.blankPartialLines = cart.state.oddField
	case int(cart.state.format[cart.state.streamIndex].visible - 1):
		cart.state.blankPartialLines = !cart.state.oddField
	default:
		cart.state.blankPartialLines = false
	}

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
					cart.state.osdControl = osdTime
					cart.state.osdIdx = cart.state.streamTimecode
				}

			default:
				if cart.state.lines == 21 {
					cart.state.osdDuration--
					cart.state.osdControl = osdLabel
					cart.state.osdIdx = 0
				}

				if cart.state.lines == 7 {
					cart.state.osdControl = osdLevels
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
			if cart.state.osdDuration > 0 {
				switch cart.state.controlMode {
				case modeTime:
					cart.state.blankPartialLines = cart.state.lines == 12 && cart.state.oddField
				default:
					cart.state.blankPartialLines = cart.state.lines == 22 && cart.state.oddField
				}
			}
			cart.fillAddrLeftLine(true)
			cart.state.lines--
			cart.state.state = stateMachineRight
		} else {
			cart.fillAddrLeftLine(false)
			cart.fillAddrEndLines()

			cart.fillAddrBlankLines()

			// swap stream indexes
			cart.state.streamIndex++
			if cart.state.streamIndex >= numFields {
				cart.state.streamIndex = 0
			}

			cart.updateTransport()
			cart.state.state = stateMachineNewField
		}
	case stateMachineNewField:
		if !cart.state.a7 {
			break // switch
		}

		cart.nextField()
		cart.state.lines = int(cart.state.format[cart.state.streamIndex].visible - 1)
		cart.state.state = stateMachineRight
		cart.state.osdControl = osdNone
	}
}

func (cart *Moviecart) fillAddrRightLine() {
	cart.writeAudio(addrSetAudRight + 1)

	cart.writeGraph(addrSetGData5 + 1)
	cart.writeGraph(addrSetGData6 + 1)
	cart.writeGraph(addrSetGData7 + 1)
	cart.writeGraph(addrSetGData8 + 1)
	cart.writeGraph(addrSetGData9 + 1)

	cart.writeColor(addrSetGCol5 + 1) // col 1/9
	cart.writeColor(addrSetGCol6 + 1) // col 3/9
	cart.writeColor(addrSetGCol7 + 1) // col 5/9
	cart.writeColor(addrSetGCol8 + 1) // col 7/9
	cart.writeColor(addrSetGCol9 + 1) // col 9/9

	cart.writeBackground(addrSetBkColR + 1)
}

func (cart *Moviecart) fillAddrLeftLine(again bool) {
	cart.writeAudio(addrSetAudLeft + 1)

	cart.writeGraph(addrSetGData0 + 1)
	cart.writeGraph(addrSetGData1 + 1)
	cart.writeGraph(addrSetGData2 + 1)
	cart.writeGraph(addrSetGData3 + 1)
	cart.writeGraph(addrSetGData4 + 1)

	cart.writeColor(addrSetGCol0 + 1) // col 0/9
	cart.writeColor(addrSetGCol1 + 1) // col 2/9
	cart.writeColor(addrSetGCol2 + 1) // col 4/9
	cart.writeColor(addrSetGCol3 + 1) // col 6/9
	cart.writeColor(addrSetGCol4 + 1) // col 8/9

	cart.writeBackground(addrSetPfColL + 1)

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
		cart.write8bit(addrSetOverscanSize+1, cart.state.format[cart.state.streamIndex].overscan-1)
		cart.write8bit(addrSetVBlankSize+1, cart.state.format[cart.state.streamIndex].vblank-1)
	} else {
		cart.state.audioCarry = cart.state.streamBuffer[cart.state.streamIndex][cart.state.streamAudio]
		cart.write8bit(addrSetOverscanSize+1, cart.state.format[cart.state.streamIndex].overscan)
		cart.write8bit(addrSetVBlankSize+1, cart.state.format[cart.state.streamIndex].vblank)
	}

	if cart.state.streamIndex == 0 {
		cart.writeJMPaddr(addrPickTransport+1, addrTransportDirection)
	} else {
		cart.writeJMPaddr(addrPickTransport+1, addrTransportButtons)
	}
}

func (cart *Moviecart) fillAddrBlankLines() {
	blankLineSize := int(cart.state.format[cart.state.streamIndex].overscan + cart.state.format[cart.state.streamIndex].vsync + cart.state.format[cart.state.streamIndex].vblank - 1)

	// slightly different number of trailing blank line every other frame
	if !cart.state.oddField {
		cart.writeAudioData(addrAudioBank, cart.state.audioCarry)
		for i := 1; i < blankLineSize+1; i++ {
			cart.writeAudio(addrAudioBank + uint16(i))
		}
	} else {
		for i := 0; i < blankLineSize-1; i++ {
			cart.writeAudio(addrAudioBank + uint16(i))
		}
	}
}

func (cart *Moviecart) writeAudio(addr uint16) {
	b := cart.state.streamBuffer[cart.state.streamIndex][cart.state.streamAudio]
	cart.state.streamAudio++
	cart.writeAudioData(addr, b)
}

func (cart *Moviecart) writeAudioData(addr uint16, data uint8) {
	// special handling of improveMuteAudioAtStart
	if improveMuteAudioAtStart && cart.state.fieldAdv < 0 {
		if improveNeatStartImageAfterRewind {
			if cart.state.streamChunk <= 1 {
				cart.write8bit(addr, 0)
				return
			}
		} else {
			if cart.state.streamChunk < 0 {
				cart.write8bit(addr, 0)
				return
			}
		}
	}

	// output silence if playback is paused or we have reached the end of the stream
	if cart.state.paused || cart.state.endOfStream {
		cart.write8bit(addr, 0)
	} else {
		b := volumeLevels[cart.state.volume][data]
		cart.write8bit(addr, b)
	}
}

func (cart *Moviecart) writeGraph(addr uint16) {
	var b byte

	// check if we need to draw OSD using stats graphics data
	switch cart.state.osdControl {
	case osdTime:
		b = cart.state.streamBuffer[cart.state.streamIndex][cart.state.osdIdx]
		cart.state.osdIdx++
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
		b = cart.state.streamBuffer[cart.state.streamIndex][cart.state.streamGraph]
		cart.state.streamGraph++
	}

	// emit black when blankPartialLines flag is true
	if cart.state.blankPartialLines {
		b = 0x00
	}

	cart.write8bit(addr, b)
}

func (cart *Moviecart) writeColor(addr uint16) {
	b := cart.state.streamBuffer[cart.state.streamIndex][cart.state.streamColor]
	cart.state.streamColor++

	// adjust brightness
	brightIdx := int(b & 0x0f)
	brightIdx += cart.state.brightness
	b = b&0xf0 | brightLevels[brightIdx]

	// forcing a particular color means we've been drawing pixels from a timecode
	// or OSD label or level meter.
	if cart.state.osdControl != osdNone {
		b = osdColor
	}

	// best effort conversion of color to B&W
	if cart.state.forceBW {
		b &= 0x0f
	}

	// emit black when blankPartialLines flag is true
	if cart.state.blankPartialLines {
		b = 0x00
	}

	cart.write8bit(addr, b)
}

func (cart *Moviecart) writeBackground(addr uint16) {
	var b byte

	// emit black when OSD is enabled
	if cart.state.osdControl != osdNone {
		b = 0x00
		cart.write8bit(addr, b)
		return
	}

	// stream next background byte
	if cart.state.osdControl == osdNone {
		b = cart.state.streamBuffer[cart.state.streamIndex][cart.state.streamBackground]
		cart.state.streamBackground++
	}

	// emit black when blankPartialLines flag is true. note that we've streamed
	// the background byte in this case and that we're discarding the streamed
	// value
	if cart.state.blankPartialLines {
		b = 0x00
		cart.write8bit(addr, b)
		return
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

// nextField is an amalgamation of the readField() and swapField() functions in
// the reference implementation by Rob Bairos. in the reference swapField() is
// called during the stateMachineLeft phase of the state machine and readField()
// is called during the stateMachineNewField phase. it was found that this
// division was unnecessary in this implementation
func (cart *Moviecart) nextField() {
	// the usual playback condition
	if !cart.state.paused && cart.state.streamChunk >= 0 {
		dataOffset := cart.state.streamChunk * chunkSize
		_, err := cart.data.Seek(int64(dataOffset), io.SeekStart)
		if err != nil {
			logger.Logf("MVC", "error seeking field: %v", err)
		}
		n, err := cart.data.Read(cart.state.streamBuffer[cart.state.streamIndex])
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Logf("MVC", "error reading field: %v", err)
			}
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
			_, err := cart.data.Seek(int64(dataOffset), io.SeekStart)
			if err != nil {
				logger.Logf("MVC", "error reading field: %v", err)
			}
			_, err = cart.data.Read(cart.state.streamBuffer[fld])
			if err != nil {
				logger.Logf("MVC", "error reading field: %v", err)
			}
		}
	}

	// magic string check
	if cart.state.streamBuffer[cart.state.streamIndex][0] != 'M' ||
		cart.state.streamBuffer[cart.state.streamIndex][1] != 'V' ||
		cart.state.streamBuffer[cart.state.streamIndex][2] != 'C' ||
		cart.state.streamBuffer[cart.state.streamIndex][3] != 0x00 {
		logger.Logf("MVC", "unrecognised version string in chunk %d", cart.state.streamChunk)
		return
	}

	// version number
	switch cart.state.streamBuffer[cart.state.streamIndex][4] & 0x80 {
	case 0x80:
		rotation := cart.state.streamBuffer[cart.state.streamIndex][4] & 0b11
		if rotation != cart.state.rotation {
			switch rotation {
			case 0b00:
				cart.env.TV.SetRotation(specification.NormalRotation)
			case 0b01:
				cart.env.TV.SetRotation(specification.RightRotation)
			case 0b10:
				cart.env.TV.SetRotation(specification.FlippedRotation)
			case 0b11:
				cart.env.TV.SetRotation(specification.LeftRotation)
			}
			cart.state.rotation = rotation
		}

		cart.state.format[cart.state.streamIndex].vsync = byte(cart.state.streamBuffer[cart.state.streamIndex][9])
		cart.state.format[cart.state.streamIndex].vblank = byte(cart.state.streamBuffer[cart.state.streamIndex][10])
		cart.state.format[cart.state.streamIndex].overscan = byte(cart.state.streamBuffer[cart.state.streamIndex][11])
		cart.state.format[cart.state.streamIndex].visible = byte(cart.state.streamBuffer[cart.state.streamIndex][12])
		cart.state.format[cart.state.streamIndex].rate = byte(cart.state.streamBuffer[cart.state.streamIndex][13])

		lines := int(cart.state.format[cart.state.streamIndex].vsync) +
			int(cart.state.format[cart.state.streamIndex].vblank) +
			int(cart.state.format[cart.state.streamIndex].visible) +
			int(cart.state.format[cart.state.streamIndex].overscan)
		cart.state.streamAudio = 14
		cart.state.streamGraph = cart.state.streamAudio + lines
		cart.state.streamColor = cart.state.streamGraph + 5*int(cart.state.format[cart.state.streamIndex].visible)
		cart.state.streamBackground = cart.state.streamColor + 5*int(cart.state.format[cart.state.streamIndex].visible)
		cart.state.streamTimecode = cart.state.streamBackground + int(cart.state.format[cart.state.streamIndex].visible)

		cart.state.fieldNumber = int(cart.state.streamBuffer[cart.state.streamIndex][6]) << 16
		cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamIndex][7]) << 8
		cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamIndex][8])
		cart.state.oddField = cart.state.fieldNumber&0x01 != 0x01

	case 0x00:
		// offsets into each part of the field (audio, color, etc.)
		const (
			offsetVersion      = 0
			offsetFieldNumber  = 4
			offsetAudioData    = 7
			offsetGraphData    = 269
			offsetTimecodeData = 1229
			offsetColorData    = 1289
			offsetColorBkData  = 2249
			offsetEndData      = 2441
		)

		cart.state.format[cart.state.streamIndex].vsync = 3
		cart.state.format[cart.state.streamIndex].vblank = 37
		cart.state.format[cart.state.streamIndex].overscan = 30
		cart.state.format[cart.state.streamIndex].visible = 192

		// reset stream indexes
		cart.state.streamAudio = offsetAudioData
		cart.state.streamGraph = offsetGraphData
		cart.state.streamTimecode = offsetTimecodeData
		cart.state.streamColor = offsetColorData
		cart.state.streamBackground = offsetColorBkData

		cart.state.fieldNumber = int(cart.state.streamBuffer[cart.state.streamIndex][offsetFieldNumber]) << 16
		cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamIndex][offsetFieldNumber+1]) << 8
		cart.state.fieldNumber |= int(cart.state.streamBuffer[cart.state.streamIndex][offsetFieldNumber+2])
		cart.state.oddField = cart.state.fieldNumber&0x01 == 0x01
	}

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

// adjust program to reflect console timing
func (cart *Moviecart) setConsoleTiming() {
	id := cart.env.TV.GetSpecID()

	// do nothing if specification hasn't changed
	if cart.specID == id {
		return
	}

	const rainbowHeight = 30
	const titleHeight = 12

	var lines uint8

	switch id {
	case "SECAM":
		lines = 242
	case "PAL":
		lines = 242
	case "PAL-60":
		lines = 192
	case "NTSC":
		lines = 192
	default:
		lines = 192
	}

	val := (lines - rainbowHeight - rainbowHeight - titleHeight*2) / 2

	cart.write8bit(addrTitleGap1+1, val)
	cart.write8bit(addrTitleGap2+1, val)
}
