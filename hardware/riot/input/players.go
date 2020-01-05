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

// Player represents the "joystick" port on the VCS. Different devices can be
// added to it through selective seding of events to the Handler() function.
type Player struct {
	device

	// reference to input instance associated with Player
	input *Input

	// address in RIOT memory for joystick direction input
	stickAddr uint16

	// value indicating joystick state
	stickValue uint8

	// player port 0 and 1 write the stickValue to different nibbles of the
	// stickAddr. stickFunc allows us to transform that value with the help of
	// stickMask
	stickFunc func(uint8) uint8

	// data direction register
	ddr uint8

	// the address in TIA memory for joystick fire button
	buttonAddr uint16
}

// NewPlayer0 is the preferred method of creating a new instance of Player for
// representing player zero
func NewPlayer0(inp *Input) *Player {
	pl := &Player{
		input:      inp,
		stickAddr:  addresses.SWCHA,
		stickValue: 0xf0,
		ddr:        0x00,
		stickFunc:  func(n uint8) uint8 { return n },

		buttonAddr: addresses.INPT4,
	}

	pl.device = device{
		id:     PlayerZeroID,
		handle: pl.Handle}

	return pl
}

// NewPlayer1 is the preferred method of creating a new instance of Player for
// representing player one
func NewPlayer1(inp *Input) *Player {
	pl := &Player{
		input:      inp,
		stickAddr:  addresses.SWCHA,
		stickValue: 0xf0,
		ddr:        0x00,
		stickFunc:  func(n uint8) uint8 { return n << 4 },

		buttonAddr: addresses.INPT5,
	}

	pl.device = device{
		id:     PlayerOneID,
		handle: pl.Handle}

	return pl
}

// Handle translates the Event argument into the required memory-write
func (pl *Player) Handle(event Event) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case NoEvent:
		return nil

	case Left:
		pl.stickValue ^= 0x4f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case Right:
		pl.stickValue ^= 0x8f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case Up:
		pl.stickValue ^= 0x1f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case Down:
		pl.stickValue ^= 0x2f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case NoLeft:
		pl.stickValue |= 0x4f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case NoRight:
		pl.stickValue |= 0x8f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case NoUp:
		pl.stickValue |= 0x1f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case NoDown:
		pl.stickValue |= 0x2f
		pl.input.mem.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.ddr)
	case Fire:
		pl.input.tiaMem.InputDeviceWrite(pl.buttonAddr, 0x00, 0x00)
	case NoFire:
		pl.input.tiaMem.InputDeviceWrite(pl.buttonAddr, 0x80, 0x00)

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
