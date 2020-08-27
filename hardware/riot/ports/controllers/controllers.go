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

// ControllerTypeList is a list of all possible string representations of the Interval type
var ControllerTypeList = []string{"Joystick", "Paddle", "Keypad"}

func (c ControllerType) String() string {
	switch c {
	case JoystickType:
		return "Joystick"
	case PaddleType:
		return "Paddle"
	case KeypadType:
		return "Keypad"
	}
	panic("unknown controller type")
}
