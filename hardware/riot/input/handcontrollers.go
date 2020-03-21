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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package input

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

// ControllerType keeps track of which controller type is being used at any
// given moment. we need this so that we don't ground/recharge the paddle if it
// is not being used. if we did then joystick input would be wrong.
//
// we default to the joystick type which should be fine. for non-joystick
// games, the paddle/keypad will be activated once the user starts using the
// corresponding controls.
//
// if a paddle/keypad ROM requires paddle/keypad probing from the instant
// the machine starts (are there any examples of this?) then we will need to
// initialise the hand controller accordingly, using the setup system.
type ControllerType int

// List of allowed ControllerTypes
const (
	JoystickType ControllerType = iota
	PaddleType
	KeypadType
)

// HandController represents the "joystick" port on the VCS. The different
// devices (joysticks, paddles, etc.) send events to the Handle() function.
//
// Note that handcontrollers need access to TIA memory as well as RIOT memory.
type HandController struct {
	port
	mem     *inputMemory
	control *VBlankBits

	// which controller type is currently being used
	which ControllerType

	// controller types
	stick  stick
	paddle paddle
	keypad keypad

	// data direction register. for simplicity, the bits should be normalised
	// such that only the upper nibble is used. in reality, player 0
	// controllers will use the upper nibble, and player 1 controller will use
	// the lower nibble.
	ddr uint8

	// the two hand controllers, for both joysticks and keypads, share
	// registers for certain values. In each instance where this is the case,
	// HandController0 uses the upper nibble and HandControll1er1 uses the
	// lower nibble.
	//
	// The normalise functions 'transform' the data to the correct nibble
	normaliseOnRead  func(uint8) uint8
	normaliseOnWrite func(uint8) uint8

	// when writing data with InputDeviceWrite() a mask is supplied to prevent
	// the bits in the 'other' nibble from being clobbered
	writeMask uint8
}

const stickButtonOn = uint8(0x00)
const stickButtonOff = uint8(0x80)

// the stick type implements the digital "joystick" controller
type stick struct {
	// the address in TIA memory for joystick fire button
	buttonReg addresses.ChipRegister

	// joysticks always write axis data to SWCHA and adjusted according to
	// normaliseOnWrite() in the HandController

	// values indicating joystick state
	axis   uint8
	button uint8
}

// the paddle type implements the "paddle" hand controller
type paddle struct {
	puckReg addresses.ChipRegister

	//
	buttonMask uint8

	// values indicating paddle state
	charge     uint8
	resistance float32

	// sensitivity governs the rate at which the controller capacitor fills.
	// the tick value is increased by the sensitivity value every cycle; once
	// it reaches or exceeds the resistance value, the charge value is
	// increased.
	sensitivity float32
	ticks       float32
}

// !!TODO: accurate paddle timings and sensitivity
//
// for now our, best guess is 0.01. no idea if this value is correct but it
// feels good during play so I'm going to go with it.
//
// justification: if the paddle resistor can take a value between 0.0 and 1.0
// then the maximum number of ticks required to increase the capacitor charge
// by 1 is 100. The maximum charge is 255 so it takes a maximum of 25500 ticks
// to fill the capacitor.
const bestGuessSensitivity = 0.01

// the keypad type implements the keypad or "keyboard" controller
type keypad struct {
	column [3]addresses.ChipRegister
	key    rune
}

// the value of keypad.key when nothing is being pressed
const noKey = ' '

// NewHandController0 is the preferred method of creating a new instance of
// HandController for representing hand controller zero
func NewHandController0(mem *inputMemory, control *VBlankBits) *HandController {
	hc := &HandController{
		mem:     mem,
		control: control,
		which:   JoystickType,
		stick: stick{
			buttonReg: addresses.INPT4,
			axis:      0xf0,
			button:    stickButtonOff,
		},
		paddle: paddle{
			puckReg:     addresses.INPT0,
			buttonMask:  0x7f,
			resistance:  0.0,
			sensitivity: bestGuessSensitivity,
		},
		keypad: keypad{
			column: [3]addresses.ChipRegister{addresses.INPT0, addresses.INPT1, addresses.INPT4},
			key:    noKey,
		},
		normaliseOnRead:  func(n uint8) uint8 { return n & 0xf0 },
		normaliseOnWrite: func(n uint8) uint8 { return n },
		writeMask:        0x0f,
		ddr:              0x00,
	}

	hc.port = port{
		id:     HandControllerZeroID,
		handle: hc.Handle,
	}

	// write initial joystick values
	hc.writeSWCHA(hc.stick.axis, hc.writeMask)
	hc.mem.tia.InputDeviceWrite(hc.stick.buttonReg, 0x80, 0x00)

	return hc
}

// NewHandController1 is the preferred method of creating a new instance of
// HandController for representing hand controller one
func NewHandController1(mem *inputMemory, control *VBlankBits) *HandController {
	hc := &HandController{
		mem:     mem,
		control: control,
		which:   JoystickType,
		stick: stick{
			buttonReg: addresses.INPT5,
			axis:      0xf0,
			button:    stickButtonOff,
		},
		paddle: paddle{
			puckReg:     addresses.INPT1,
			buttonMask:  0xbf,
			resistance:  0.0,
			sensitivity: bestGuessSensitivity,
		},
		keypad: keypad{
			column: [3]addresses.ChipRegister{addresses.INPT2, addresses.INPT3, addresses.INPT5},
			key:    noKey,
		},
		normaliseOnRead:  func(n uint8) uint8 { return (n & 0x0f) << 4 },
		normaliseOnWrite: func(n uint8) uint8 { return n >> 4 },
		writeMask:        0xf0,
		ddr:              0x00,
	}

	hc.port = port{
		id:     HandControllerOneID,
		handle: hc.Handle,
	}

	// write initial joystick values
	hc.writeSWCHA(hc.stick.axis, hc.writeMask)
	hc.mem.tia.InputDeviceWrite(hc.stick.buttonReg, hc.stick.button, 0x00)

	return hc
}

// String implements the Port interface
func (hc *HandController) String() string {
	return "nothing yet"
}

// SwitchType causes the HandController to swich controller type. If the type
// is switched or if the type is already of the requested type then true is
// returned.
func (hc *HandController) SwitchType(prospective ControllerType) bool {
	if hc.which == prospective {
		return true
	}

	switch prospective {
	case JoystickType:
		if hc.which != KeypadType {
			hc.which = JoystickType
			return true
		}
	case PaddleType:
		if hc.which != KeypadType {
			hc.which = PaddleType
			return true
		}
	case KeypadType:
		hc.which = KeypadType
		return true
	}

	return false
}

// Handle implements Port interface
func (hc *HandController) Handle(event Event, value EventValue) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case NoEvent:
		return nil

	case Left:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		if !hc.SwitchType(JoystickType) {
			return nil
		}

		if b {
			hc.stick.axis ^= 0x40
		} else {
			hc.stick.axis |= 0x40
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case Right:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		if !hc.SwitchType(JoystickType) {
			return nil
		}

		if b {
			hc.stick.axis ^= 0x80
		} else {
			hc.stick.axis |= 0x80
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case Up:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		if !hc.SwitchType(JoystickType) {
			return nil
		}

		if b {
			hc.stick.axis ^= 0x10
		} else {
			hc.stick.axis |= 0x10
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case Down:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		if !hc.SwitchType(JoystickType) {
			return nil
		}

		if b {
			hc.stick.axis ^= 0x20
		} else {
			hc.stick.axis |= 0x20
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case Fire:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		if !hc.SwitchType(JoystickType) {
			return nil
		}

		// record state of fire button regardless of latch bit. we need to know
		// the physical state for when the latch bit is unset
		if b {
			hc.stick.button = stickButtonOn
		} else {
			hc.stick.button = stickButtonOff
		}

		// write memory if button is pressed or it is not and the button latch
		// is false
		if hc.stick.button == stickButtonOn || !hc.control.latchFireButton {
			hc.mem.tia.InputDeviceWrite(hc.stick.buttonReg, hc.stick.button, 0x00)
		}

	case PaddleFire:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		if !hc.SwitchType(PaddleType) {
			return nil
		}

		var v uint8

		if b {
			v = 0x00
		} else {
			v = 0xff
		}
		hc.writeSWCHA(v, hc.paddle.buttonMask)

	case PaddleSet:
		f, ok := value.(float32)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "float32")
		}

		if !hc.SwitchType(PaddleType) {
			return nil
		}

		hc.paddle.resistance = 1.0 - f

	case KeypadDown:
		v, ok := value.(rune)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "rune")
		}

		// keypad switched to only when DDR is switched

		if v != '1' && v != '2' && v != '3' && v != '4' && v != '5' && v != '6' && v != '7' && v != '8' && v != '9' && v != '*' && v != '0' && v != '#' {
			return errors.New(errors.BadInputEventType, event, "numeric rune or '*' or '#'")
		}

		// note key for use by readKeypad()
		hc.keypad.key = v

	case KeypadUp:
		if value != nil {
			return errors.New(errors.BadInputEventType, event, "nil")
		}

		// keypad switched to only when DDR is switched

		hc.keypad.key = noKey

	case Unplug:
		return errors.New(errors.InputDeviceUnplugged, hc.id)

	// return now if there is no event to process
	default:
		return errors.New(errors.UnknownInputEvent, hc.id, event)
	}

	// record event with the EventRecorder
	if hc.recorder != nil {
		return hc.recorder.RecordEvent(hc.id, event, value)
	}

	return nil
}

// set DDR value. values should be normalised to the upper nibble before being
// passed to the function. this simplifies the implementation.
func (hc *HandController) setDDR(data uint8) {
	hc.ddr = hc.normaliseOnRead(data)

	// if the ddr value is being such so that SWCHA is input rather than output
	// the the expected controller is most probably a keypad. not sure what
	// we can say if ddr is only partially set to input.
	if hc.ddr == 0xf0 {
		hc.SwitchType(KeypadType)
	} else {
		// switch to Joystick if DDR is anything other than 0xf0
		hc.SwitchType(JoystickType)
	}
}

// readKeypad() is called whenever SWCHA is tickled by the CPU. the state of
// the ddr is of importance here.
func (hc *HandController) readKeypad(data uint8) {
	if hc.which != KeypadType {
		return
	}

	data = hc.normaliseOnRead(data)

	var column int

	switch hc.keypad.key {
	// row 0
	case '1':
		if data&0xe0 == 0xe0 && hc.ddr&0xe0 == 0xe0 {
			column = 1
		}
	case '2':
		if data&0xe0 == 0xe0 && hc.ddr&0xe0 == 0xe0 {
			column = 2
		}
	case '3':
		if data&0xe0 == 0xe0 && hc.ddr&0xe0 == 0xe0 {
			column = 3
		}

		// row 2
	case '4':
		if data&0xd0 == 0xd0 && hc.ddr&0xd0 == 0xd0 {
			column = 1
		}
	case '5':
		if data&0xd0 == 0xd0 && hc.ddr&0xd0 == 0xd0 {
			column = 2
		}
	case '6':
		if data&0xd0 == 0xd0 && hc.ddr&0xd0 == 0xd0 {
			column = 3
		}

		// row 3
	case '7':
		if data&0xb0 == 0xb0 && hc.ddr&0xb0 == 0xb0 {
			column = 1
		}
	case '8':
		if data&0xb0 == 0xb0 && hc.ddr&0xb0 == 0xb0 {
			column = 2
		}
	case '9':
		if data&0xb0 == 0xb0 && hc.ddr&0xb0 == 0xb0 {
			column = 3
		}

		// row 4
	case '*':
		if data&0x70 == 0x70 && hc.ddr&0x70 == 0x70 {
			column = 1
		}
	case '0':
		if data&0x70 == 0x70 && hc.ddr&0x70 == 0x70 {
			column = 2
		}
	case '#':
		if data&0x70 == 0x70 && hc.ddr&0x70 == 0x70 {
			column = 3
		}
	}

	// The Stella Programmer's Guide says that: "a delay of 400 microseconds is
	// necessary between writing to this port and reading the TIA input ports.".
	// We're not emulating this here because as far as I can tell there is no need
	// to. More over, I'm not sure what's supposed to happen if the 400ms is not
	// adhered to.
	//
	// !!TODO: Consider adding 400ms delay for DDR settings to take effect.
	switch column {
	case 1:
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[0], 0x00, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[1], 0x80, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[2], 0x80, 0x00)
	case 2:
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[0], 0x80, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[1], 0x00, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[2], 0x80, 0x00)
	case 3:
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[0], 0x80, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[1], 0x80, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[2], 0x00, 0x00)
	default:
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[0], 0x80, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[1], 0x80, 0x00)
		hc.mem.tia.InputDeviceWrite(hc.keypad.column[2], 0x80, 0x00)
	}
}

// VBLANK bit 6 has been set. joystick button will latch, meaning that
// releasing the fire button has no immediate effect
func (hc *HandController) unlatch() {
	if hc.which != JoystickType {
		return
	}

	// only unlatch if button is not pressed
	if hc.stick.button == stickButtonOff {
		hc.mem.tia.InputDeviceWrite(hc.stick.buttonReg, stickButtonOff, 0x00)
	}
}

// VBLANK bit 7 has been set. input capacitor is grounded.
func (hc *HandController) ground() {
	// don't allow grounding unless controller type is paddle type. if we don't
	// then it will play havoc with keyboard controllers.
	//
	// I'm not sure if this is correct. the keypad only seems to meddle with
	// the the high bit of INPT1 and I'm now wondering if the charge value ever
	// reaches the last bit (?) if it doesn't we can change the recharge
	// function and not worry about clobbering the high bit
	if hc.which != PaddleType {
		return
	}

	hc.paddle.charge = 0
	hc.mem.riot.InputDeviceWrite(hc.paddle.puckReg, hc.paddle.charge, 0x00)
}

// recharge() is called every video step via Input.Step()
func (hc *HandController) recharge() {
	// as in the case of ground() I'm not sure if restricting recharge() events
	// to the paddle type is strictly necessary.
	if hc.which != PaddleType {
		return
	}

	// from Stella Programmer's Guide:
	//
	// "B. Dumped Input Ports (I0 through I3)
	//
	// These 4 input ports are normally used to read paddle position from an
	// external potentiometer-capacitor circuit. In order to discharge these
	// capacitors each of these input ports has a large transistor, which may be
	// turned on (grounding the input ports) by writing into bit 7 of the register
	// VBLANK. When this control bit is cleared the potentiometers begin to
	// recharge the capacitors and the microprocessor measures the time required
	// to detect a logic 1 at each input port."
	if hc.paddle.charge < 255 {
		hc.paddle.ticks += hc.paddle.sensitivity
		if hc.paddle.ticks >= hc.paddle.resistance {
			hc.paddle.ticks = 0
			hc.paddle.charge++
			hc.mem.tia.InputDeviceWrite(hc.paddle.puckReg, hc.paddle.charge, 0x00)
		}
	}
}

// writing to SWCHA requires some filtering according to the data direction
// register (DDR). joysticks always write their axis data to SWCHA and paddles
// always write fire button data to SWCHA, according to a mask for which hand
// controller is issuing the call
func (hc *HandController) writeSWCHA(data uint8, mask uint8) {
	data = hc.normaliseOnWrite(data & (hc.ddr ^ 0xff))
	hc.mem.riot.InputDeviceWrite(addresses.SWCHA, data, mask)
}
