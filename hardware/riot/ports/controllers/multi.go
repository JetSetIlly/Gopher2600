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

package controllers

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
)

// Multi is an implementation of the ports.Peripheral interface and attempts to
// handle automatic switching between the three main controller types
type Multi struct {
	ports.Recordable
	mem *ports.MemoryAccess

	// the current controller type. use SwitchType() to set.
	ControllerType ControllerType

	// whether SwitchType() should respond to automatic switching. use
	// SetAuto() to set.
	AutoControllerType bool

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
	// Multi0 uses the upper nibble and HandControll1er1 uses the
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
	// normaliseOnWrite() in the Multi

	// values indicating joystick state
	axis   uint8
	button uint8

	latchFireButton bool
}

// the number of times the paddle has to be waggled to the extremes before
// the controller mode switches to paddle type
const paddleTouchReq = 3

// the value used to write to the paddle fire button. the value is mased by the
// paddle.buttonMask value before writing to SWCHA
const paddleFire = 0xff

// as above but for when the first button is released
const paddleNoFire = 0x00

// sensitivity of the paddle puck
const paddleSensitivity = 0.009

// the paddle type implements the "paddle" hand controller
type paddle struct {
	puckReg addresses.ChipRegister

	buttonMask uint8

	ground bool

	// values indicating paddle state
	charge     uint8
	resistance float32

	// sensitivity governs the rate at which the controller capacitor fills.
	// the tick value is increased by the sensitivity value every cycle; once
	// it reaches or exceeds the resistance value, the charge value is
	// increased.
	sensitivity float32
	ticks       float32

	// count of how many times the paddle has touched the extreme values. we
	// use this to help decide whether to switch controller types
	touchLeft     int
	touchRight    int
	touchingLeft  bool
	touchingRight bool
}

// the keypad type implements the keypad or "keyboard" controller
type keypad struct {
	column [3]addresses.ChipRegister
	key    rune
}

// the value of keypad.key when nothing is being pressed
const noKey = ' '

// NewMulti0 is the preferred method of creating a new instance of
// Multi for representing hand controller zero
func NewMultiController0(mem *ports.MemoryAccess) ports.Peripheral {
	hc := &Multi{
		mem:                mem,
		ControllerType:     JoystickType,
		AutoControllerType: true,
		stick: stick{
			buttonReg: addresses.INPT4,
			axis:      0xf0,
			button:    stickButtonOff,
		},
		paddle: paddle{
			puckReg:     addresses.INPT0,
			buttonMask:  0x7f,
			resistance:  0.0,
			sensitivity: paddleSensitivity,
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

	hc.Recordable = ports.Recordable{
		ID:          ports.PlayerZeroID,
		HandleEvent: hc.HandleEvent,
	}

	// write initial joystick values
	hc.writeSWCHA(hc.stick.axis, hc.writeMask)
	hc.mem.TIA.InputDeviceWrite(hc.stick.buttonReg, 0x80, 0x00)

	return hc
}

// NewMulti1 is the preferred method of creating a new instance of
// Multi for representing hand controller one
func NewMultiController1(mem *ports.MemoryAccess) ports.Peripheral {
	hc := &Multi{
		mem:                mem,
		ControllerType:     JoystickType,
		AutoControllerType: true,
		stick: stick{
			buttonReg: addresses.INPT5,
			axis:      0xf0,
			button:    stickButtonOff,
		},
		paddle: paddle{
			puckReg:     addresses.INPT1,
			buttonMask:  0xbf,
			resistance:  0.0,
			sensitivity: paddleSensitivity,
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

	hc.Recordable = ports.Recordable{
		ID:          ports.PlayerOneID,
		HandleEvent: hc.HandleEvent,
	}

	// write initial joystick values
	hc.writeSWCHA(hc.stick.axis, hc.writeMask)
	hc.mem.TIA.InputDeviceWrite(hc.stick.buttonReg, hc.stick.button, 0x00)

	return hc
}

// String implements the Peripheral interface
func (hc *Multi) String() string {
	return "nothing yet"
}

// Reset implements the Peripheral interface
func (hc *Multi) Reset() {
	hc.updateSWACNT(0x00)
}

// SetAuto turns automatic controller switching on or off. Note that calling
// SwitchType() with a different type to what has been automatically selected
// will also turn auto-switching off.
func (hc *Multi) SetAuto(auto bool) {
	hc.AutoControllerType = auto

	// reset detection variables
	hc.paddle.touchLeft = 0
	hc.paddle.touchRight = 0
}

// SwitchType causes the Multi to swich controller type. If the type
// is switched or if the type is already of the requested type then true is
// returned.
func (hc *Multi) SwitchType(newType ControllerType) error {
	// reset detection variables
	hc.paddle.touchLeft = 0
	hc.paddle.touchRight = 0

	switch newType {
	case JoystickType:
		hc.ControllerType = JoystickType
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)
		hc.mem.TIA.InputDeviceWrite(hc.stick.buttonReg, hc.stick.button, 0x00)
	case PaddleType:
		hc.ControllerType = PaddleType
		hc.writeSWCHA(paddleFire, hc.writeMask)
	case KeypadType:
		hc.ControllerType = KeypadType

	default:
		return errors.New(errors.UnknownControllerType, newType)
	}

	return nil
}

// HandleEvent implements Peripheral interface
func (hc *Multi) HandleEvent(event ports.Event, value ports.EventData) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case ports.NoEvent:
		return nil

	case ports.Left:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		// smart switch to joystick type
		if hc.ControllerType != JoystickType {
			if hc.AutoControllerType {
				if err := hc.SwitchType(JoystickType); err != nil {
					return err
				}
			} else {
				return nil
			}
		}

		if b {
			hc.stick.axis ^= 0x40
		} else {
			hc.stick.axis |= 0x40
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case ports.Right:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		// smart switch to joystick type
		if hc.ControllerType != JoystickType {
			if hc.AutoControllerType {
				if err := hc.SwitchType(JoystickType); err != nil {
					return err
				}
			} else {
				return nil
			}
		}

		if b {
			hc.stick.axis ^= 0x80
		} else {
			hc.stick.axis |= 0x80
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case ports.Up:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		// smart switch to joystick type
		if hc.ControllerType != JoystickType {
			if hc.AutoControllerType {
				if err := hc.SwitchType(JoystickType); err != nil {
					return err
				}
			} else {
				return nil
			}
		}

		if b {
			hc.stick.axis ^= 0x10
		} else {
			hc.stick.axis |= 0x10
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case ports.Down:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		// smart switch to joystick type
		if hc.ControllerType != JoystickType {
			if hc.AutoControllerType {
				if err := hc.SwitchType(JoystickType); err != nil {
					return err
				}
			} else {
				return nil
			}
		}

		if b {
			hc.stick.axis ^= 0x20
		} else {
			hc.stick.axis |= 0x20
		}
		hc.writeSWCHA(hc.stick.axis, hc.writeMask)

	case ports.Fire:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		// smart switch to joystick type
		if hc.ControllerType != JoystickType {
			if hc.AutoControllerType {
				if err := hc.SwitchType(JoystickType); err != nil {
					return err
				}
			} else {
				return nil
			}
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
		if hc.stick.button == stickButtonOn || !hc.stick.latchFireButton {
			hc.mem.TIA.InputDeviceWrite(hc.stick.buttonReg, hc.stick.button, 0x00)
		}

	case ports.PaddleFire:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}

		// no smart switch on paddle fire
		if hc.ControllerType != PaddleType {
			return nil
		}

		var v uint8

		if b {
			v = paddleNoFire
		} else {
			v = paddleFire
		}
		hc.writeSWCHA(v, hc.paddle.buttonMask)

	case ports.PaddleSet:
		f, ok := value.(float32)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "float32")
		}

		// smart-switch to paddle type. because the paddle is more likely to be
		// triggered by accident (paddle is emulated with the mouse) we're a
		// lot more careful than with joystick smart-switching
		if hc.ControllerType != PaddleType {
			if hc.AutoControllerType {
				if hc.paddle.touchLeft < paddleTouchReq {
					if f < 0.1 {
						if !hc.paddle.touchingLeft {
							hc.paddle.touchLeft++
						}
						hc.paddle.touchingLeft = true
					} else {
						hc.paddle.touchingLeft = false
					}
				}
				if hc.paddle.touchRight < paddleTouchReq {
					if f > 0.9 {
						if !hc.paddle.touchingRight {
							hc.paddle.touchRight++
						}
						hc.paddle.touchingRight = true
					} else {
						hc.paddle.touchingRight = false
					}
				}
				if hc.paddle.touchLeft >= paddleTouchReq && hc.paddle.touchRight >= paddleTouchReq {
					if err := hc.SwitchType(PaddleType); err != nil {
						return err
					}
				}
			} else {
				return nil
			}
		}

		hc.paddle.resistance = 1.0 - f

	case ports.KeypadDown:
		v, ok := value.(rune)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "rune")
		}

		// keypad is smart selected when DDR is switched
		if hc.ControllerType != KeypadType {
			return nil
		}

		if v != '1' && v != '2' && v != '3' && v != '4' && v != '5' && v != '6' && v != '7' && v != '8' && v != '9' && v != '*' && v != '0' && v != '#' {
			return errors.New(errors.BadInputEventType, event, "numeric rune or '*' or '#'")
		}

		// note key for use by readKeypad()
		hc.keypad.key = v

	case ports.KeypadUp:
		if value != nil {
			return errors.New(errors.BadInputEventType, event, "nil")
		}

		// keypad is smart selected when DDR is switched
		if hc.ControllerType != KeypadType {
			return nil
		}

		hc.keypad.key = noKey

	case ports.Unplug:
		return errors.New(errors.InputDeviceUnplugged, hc.ID)

	// return now if there is no event to process
	default:
		return errors.New(errors.UnknownInputEvent, hc.ID, event)
	}

	// record event with the EventRecorder
	if hc.Recorder != nil {
		return hc.Recorder.RecordEvent(hc.ID, event, value)
	}

	return nil
}

// Update implements the Peripheral interface
func (hc *Multi) Update(data bus.ChipData) bool {
	switch data.Name {
	case "VBLANK":
		// dump paddle capacitors to ground
		hc.paddle.ground = data.Value&0x80 == 0x80
		// !!TODO: surely whether we acutally ground should be based on the
		// state of the ground bit
		hc.ground()

		hc.stick.latchFireButton = data.Value&0x40 == 0x40
		if !hc.stick.latchFireButton {
			hc.unlatch()
		}

	case "SWCHA":
		hc.updateSWCHA(data.Value)
		hc.mem.RIOT.InputDeviceWrite(addresses.SWCHA, data.Value, 0x00)

	case "SWACNT":
		hc.updateSWACNT(data.Value)
		hc.mem.RIOT.InputDeviceWrite(addresses.SWACNT, data.Value, 0x00)

	default:
		return true
	}

	return false
}

// updateSWACNT values should be normalised to the upper nibble before being
// passed to the function. this simplifies the implementation.
func (hc *Multi) updateSWACNT(data uint8) {
	hc.ddr = hc.normaliseOnRead(data)

	// if the ddr value is being such so that SWCHA is input rather than output
	// the the expected controller is most probably a keypad. not sure what
	// we can say if ddr is only partially set to input.
	if hc.ddr == 0xf0 {
		if hc.AutoControllerType {
			hc.SwitchType(KeypadType)
		}
	} else {
		if hc.AutoControllerType {
			// switch to Joystick if DDR is anything other than 0xf0
			hc.SwitchType(JoystickType)
		}
	}
}

// updateSWCHA() is called whenever SWCHA is tickled by the CPU. the state of
// the ddr is of importance here.
func (hc *Multi) updateSWCHA(data uint8) {
	if hc.ControllerType != KeypadType {
		return
	}

	data = hc.normaliseOnRead(data)

	var column int

	switch hc.keypad.key {
	// row 0
	case '1':
		if data&0xe0 == data && hc.ddr&0xe0 == 0xe0 {
			column = 1
		}
	case '2':
		if data&0xe0 == data && hc.ddr&0xe0 == 0xe0 {
			column = 2
		}
	case '3':
		if data&0xe0 == data && hc.ddr&0xe0 == 0xe0 {
			column = 3
		}

		// row 2
	case '4':
		if data&0xd0 == data && hc.ddr&0xd0 == 0xd0 {
			column = 1
		}
	case '5':
		if data&0xd0 == data && hc.ddr&0xd0 == 0xd0 {
			column = 2
		}
	case '6':
		if data&0xd0 == data && hc.ddr&0xd0 == 0xd0 {
			column = 3
		}

		// row 3
	case '7':
		if data&0xb0 == data && hc.ddr&0xb0 == 0xb0 {
			column = 1
		}
	case '8':
		if data&0xb0 == data && hc.ddr&0xb0 == 0xb0 {
			column = 2
		}
	case '9':
		if data&0xb0 == data && hc.ddr&0xb0 == 0xb0 {
			column = 3
		}

		// row 4
	case '*':
		if data&0x70 == data && hc.ddr&0x70 == 0x70 {
			column = 1
		}
	case '0':
		if data&0x70 == data && hc.ddr&0x70 == 0x70 {
			column = 2
		}
	case '#':
		if data&0x70 == data && hc.ddr&0x70 == 0x70 {
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
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[0], 0x00, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[1], 0x80, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[2], 0x80, 0x00)
	case 2:
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[0], 0x80, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[1], 0x00, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[2], 0x80, 0x00)
	case 3:
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[0], 0x80, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[1], 0x80, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[2], 0x00, 0x00)
	default:
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[0], 0x80, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[1], 0x80, 0x00)
		hc.mem.TIA.InputDeviceWrite(hc.keypad.column[2], 0x80, 0x00)
	}
}

// VBLANK bit 6 has been set. joystick button will latch, meaning that
// releasing the fire button has no immediate effect
func (hc *Multi) unlatch() {
	if hc.ControllerType != JoystickType {
		return
	}

	// only unlatch if button is not pressed
	if hc.stick.button == stickButtonOff {
		hc.mem.TIA.InputDeviceWrite(hc.stick.buttonReg, stickButtonOff, 0x00)
	}
}

// VBLANK bit 7 has been set. input capacitor is grounded.
func (hc *Multi) ground() {
	// don't allow grounding unless controller type is paddle type. if we don't
	// then it will play havoc with keyboard controllers.
	//
	// I'm not sure if this is correct. the keypad only seems to meddle with
	// the the high bit of INPT1 and I'm now wondering if the charge value ever
	// reaches the last bit (?) if it doesn't we can change the recharge
	// function and not worry about clobbering the high bit
	if hc.ControllerType != PaddleType {
		return
	}

	hc.paddle.charge = 0
	hc.mem.RIOT.InputDeviceWrite(hc.paddle.puckReg, hc.paddle.charge, 0x00)
}

// Step implements the Peripheral interface. It is called every video step via
// Input.Step()
func (hc *Multi) Step() {
	// as in the case of ground() I'm not sure if restricting recharge() events
	// to the paddle type is strictly necessary.
	if hc.ControllerType != PaddleType {
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
			hc.mem.TIA.InputDeviceWrite(hc.paddle.puckReg, hc.paddle.charge, 0x00)
		}
	}
}

// writing to SWCHA requires some filtering according to the data direction
// register (DDR). joysticks always write their axis data to SWCHA and paddles
// always write fire button data to SWCHA, according to a mask for which hand
// controller is issuing the call
func (hc *Multi) writeSWCHA(data uint8, mask uint8) {
	data = hc.normaliseOnWrite(data & (hc.ddr ^ 0xff))
	hc.mem.RIOT.InputDeviceWrite(addresses.SWCHA, data, mask)
}
