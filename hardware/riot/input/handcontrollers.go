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
	"gopher2600/errors"
	"gopher2600/hardware/memory/addresses"
)

// HandController represents the "joystick" port on the VCS. The different
// devices (joysticks, paddles, etc.) send events to the Handle() function.
//
// Note that handcontrollers need access to TIA memory as well as RIOT memory.
type HandController struct {
	port
	mem     *inputMemory
	control *ControlBits

	// controller types
	stick  stick
	paddle paddle

	// data direction register
	ddr uint8
}

// the stick type handles the "joystick" hand controller type
type stick struct {
	// address in RIOT memory for joystick direction input
	addr uint16

	// the address in TIA memory for joystick fire button
	buttonAddr uint16

	// values indicating joystick state
	axis   uint8
	button bool

	// hand controllers 0 and 1 write the axis value to different nibbles of the
	// addr. transform allows us to transform that value with the help of
	// stickMask
	transform func(uint8) uint8

	// because the two hand controllers share the same stick address, each
	// controller needs to mask off the other hand controller's bits, or put
	// another way, the bits we need to preserve during the write
	preserveBits uint8
}

// the paddle type handles the "paddle" hand controller type
type paddle struct {
	addr       uint16
	buttonAddr uint16
	buttonMask uint8
}

// NewHandController0 is the preferred method of creating a new instance of
// HandController for representing hand controller zero
func NewHandController0(mem *inputMemory, control *ControlBits) *HandController {
	pl := &HandController{
		mem:     mem,
		control: control,
		stick: stick{
			addr:         addresses.SWCHA,
			buttonAddr:   addresses.INPT4,
			axis:         0xf0,
			transform:    func(n uint8) uint8 { return n },
			preserveBits: 0x0f,
		},
		paddle: paddle{
			addr:       addresses.INPT0,
			buttonAddr: addresses.SWCHA,
			buttonMask: 0x7f,
		},
		ddr: 0x00,
	}

	pl.port = port{
		id:     HandControllerZeroID,
		handle: pl.Handle,
	}

	return pl
}

// NewHandController1 is the preferred method of creating a new instance of
// HandController for representing hand controller one
func NewHandController1(mem *inputMemory, control *ControlBits) *HandController {
	pl := &HandController{
		mem:     mem,
		control: control,
		stick: stick{
			addr:         addresses.SWCHA,
			buttonAddr:   addresses.INPT5,
			axis:         0xf0,
			transform:    func(n uint8) uint8 { return n >> 4 },
			preserveBits: 0xf0,
		},
		paddle: paddle{
			addr:       addresses.INPT0,
			buttonAddr: addresses.SWCHA,
			buttonMask: 0xbf,
		},
		ddr: 0x00,
	}

	pl.port = port{
		id:     HandControllerOneID,
		handle: pl.Handle,
	}

	return pl
}

// String implements the Port interface
func (pl *HandController) String() string {
	return ""
}

// Handle translates the Event argument into the required memory-write
func (pl *HandController) Handle(event Event) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case NoEvent:
		return nil

	case Left:
		pl.stick.axis ^= 0x40
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)
	case Right:
		pl.stick.axis ^= 0x80
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)
	case Up:
		pl.stick.axis ^= 0x10
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)
	case Down:
		pl.stick.axis ^= 0x20
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)
	case NoLeft:
		pl.stick.axis |= 0x40
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)
	case NoRight:
		pl.stick.axis |= 0x80
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)
	case NoUp:
		pl.stick.axis |= 0x10
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)
	case NoDown:
		pl.stick.axis |= 0x20
		pl.mem.riot.InputDeviceWrite(pl.stick.addr, pl.stick.transform(pl.stick.axis), pl.stick.preserveBits)

	case Fire:
		pl.mem.tia.InputDeviceWrite(pl.stick.buttonAddr, 0x00, 0x00)
		pl.stick.button = true
	case NoFire:
		pl.stick.button = false
		if !pl.control.latchFireButton {
			pl.mem.tia.InputDeviceWrite(pl.stick.buttonAddr, 0x80, 0x00)
		}

	case PaddleFire:
		pl.mem.riot.InputDeviceWrite(pl.paddle.buttonAddr, 0x00, pl.paddle.buttonMask)
	case PaddleNoFire:
		pl.mem.riot.InputDeviceWrite(pl.paddle.buttonAddr, 0xff, pl.paddle.buttonMask)

	case Unplug:
		return errors.New(errors.InputDeviceUnplugged, pl.id)

	// return now if there is no event to process
	default:
		return errors.New(errors.UnknownInputEvent, pl.id, event)
	}

	// record event with the EventRecorder
	if pl.recorder != nil {
		return pl.recorder.RecordEvent(pl.id, event)
	}

	return nil
}

func (pl *HandController) unlatch() {
	// only unlatch if button is not pressed
	if !pl.stick.button {
		pl.mem.tia.InputDeviceWrite(pl.stick.buttonAddr, 0x80, 0x00)
	}
}
