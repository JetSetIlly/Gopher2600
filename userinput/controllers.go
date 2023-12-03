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

package userinput

import (
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// quite a large deadzone for the thumbstick
const ThumbstickDeadzone = 10000

// Controllers keeps track of hardware userinput options.
type Controllers struct {
	inputHandler HandleInput
}

// Controllers is the preferred method of initialisation for the Controllers type.
func NewControllers(h HandleInput) *Controllers {
	return &Controllers{inputHandler: h}
}

// handleEvents sends the port/event information to each HandleInput
// implementation.
//
// returns True if event has been handled/recognised by at least one of the
// registered input handlers.
func (c *Controllers) handleEvents(id plugging.PortID, ev ports.Event, d ports.EventData) (bool, error) {
	handled, err := c.inputHandler.HandleInputEvent(ports.InputEvent{Port: id, Ev: ev, D: d})
	if err != nil {
		return handled, err
	}
	return handled, nil
}

func (c *Controllers) mouseMotion(ev EventMouseMotion) (bool, error) {
	return c.handleEvents(plugging.PortLeft, ports.PaddleSet, ports.EventDataPaddle{
		A:        ev.X,
		Relative: true,
	})
}

func (c *Controllers) mouseButton(ev EventMouseButton) (bool, error) {
	switch ev.Button {
	case MouseButtonLeft:
		if ev.Down {
			return c.handleEvents(plugging.PortLeft, ports.Fire, true)
		} else {
			return c.handleEvents(plugging.PortLeft, ports.Fire, false)
		}
	}

	return false, nil
}

// differentiateKeyboard is called by the keyboard() function in order to
// disambiguate physical key presses and sending the correct event depending on
// the attached peripheral.
//
// like the handleEvent() function if sends the port/event information to each
// HandleInput implemtation, when appropriate.
//
// returns True if event has been handled/recognised by at least one of the
// registered input handlers.
func (c *Controllers) differentiateKeyboard(key string, down bool) (bool, error) {
	var handled bool

	var ev ports.Event
	var d ports.EventData

	var portID plugging.PortID
	var keyEv ports.Event
	var stickEvData ports.EventData
	var fireEvData ports.EventData

	if down {
		keyEv = ports.KeypadDown
		stickEvData = ports.DataStickTrue
		fireEvData = true
	} else {
		keyEv = ports.KeypadUp
		stickEvData = ports.DataStickFalse
		fireEvData = false
	}

	switch key {
	case "Y":
		if c.inputHandler.PeripheralID(plugging.PortRight) == plugging.PeriphKeypad {
			portID = plugging.PortRight
			ev = keyEv
			d = '6'
		} else {
			portID = plugging.PortRight
			ev = ports.Up
			d = stickEvData
		}
	case "F":
		if c.inputHandler.PeripheralID(plugging.PortRight) == plugging.PeriphKeypad {
			portID = plugging.PortRight
			ev = keyEv
			d = '7'
		} else {
			portID = plugging.PortRight
			ev = ports.Fire
			d = fireEvData
		}
	case "G":
		if c.inputHandler.PeripheralID(plugging.PortRight) == plugging.PeriphKeypad {
			portID = plugging.PortRight
			ev = keyEv
			d = '8'
		} else {
			portID = plugging.PortRight
			ev = ports.Left
			d = stickEvData
		}
	case "H":
		if c.inputHandler.PeripheralID(plugging.PortRight) == plugging.PeriphKeypad {
			portID = plugging.PortRight
			ev = keyEv
			d = '9'
		} else {
			portID = plugging.PortRight
			ev = ports.Down
			d = stickEvData
		}
	case "B":
		if c.inputHandler.PeripheralID(plugging.PortRight) == plugging.PeriphKeypad {
			portID = plugging.PortRight
			ev = keyEv
			d = '0'
		} else {
			portID = plugging.PortLeft
			ev = ports.SecondFire
			d = fireEvData
		}
	case "6":
		if c.inputHandler.PeripheralID(plugging.PortRight) == plugging.PeriphKeypad {
			portID = plugging.PortRight
			ev = keyEv
			d = '6'
		} else {
			portID = plugging.PortRight
			ev = ports.SecondFire
			d = fireEvData
		}
	default:
	}

	// all differentiated keyboard events go to the right player port
	v, err := c.inputHandler.HandleInputEvent(ports.InputEvent{Port: portID, Ev: ev, D: d})
	if err != nil {
		return handled, err
	}
	handled = handled || v

	return handled, nil
}

func (c *Controllers) keyboard(ev EventKeyboard) (bool, error) {
	if ev.Down && ev.Mod == KeyModNone {
		switch ev.Key {
		// panel
		case "F1":
			return c.handleEvents(plugging.PortPanel, ports.PanelSelect, true)
		case "F2":
			return c.handleEvents(plugging.PortPanel, ports.PanelReset, true)
		case "F3":
			return c.handleEvents(plugging.PortPanel, ports.PanelToggleColor, nil)
		case "F4":
			return c.handleEvents(plugging.PortPanel, ports.PanelTogglePlayer0Pro, nil)
		case "F5":
			return c.handleEvents(plugging.PortPanel, ports.PanelTogglePlayer1Pro, nil)

		// joystick (left player)
		case "Left":
			return c.handleEvents(plugging.PortLeft, ports.Left, ports.DataStickTrue)
		case "Right":
			return c.handleEvents(plugging.PortLeft, ports.Right, ports.DataStickTrue)
		case "Up":
			return c.handleEvents(plugging.PortLeft, ports.Up, ports.DataStickTrue)
		case "Down":
			return c.handleEvents(plugging.PortLeft, ports.Down, ports.DataStickTrue)
		case "Space":
			return c.handleEvents(plugging.PortLeft, ports.Fire, true)

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			return c.handleEvents(plugging.PortRight, ports.Right, ports.DataStickTrue)

		// keypad (left player)
		case "1", "2", "3":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, rune(ev.Key[0]))
		case "Q":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '4')
		case "W":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '5')
		case "E":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '6')
		case "A":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '7')
		case "S":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '8')
		case "D":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '9')
		case "Z":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '*')
		case "X":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '0')
		case "C":
			return c.handleEvents(plugging.PortLeft, ports.KeypadDown, '#')

		// keypad (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4":
			return c.handleEvents(plugging.PortRight, ports.KeypadDown, '1')
		case "5":
			return c.handleEvents(plugging.PortRight, ports.KeypadDown, '2')
		case "R":
			return c.handleEvents(plugging.PortRight, ports.KeypadDown, '4')
		case "T":
			return c.handleEvents(plugging.PortRight, ports.KeypadDown, '5')
		case "V":
			return c.handleEvents(plugging.PortRight, ports.KeypadDown, '*')
		case "N":
			return c.handleEvents(plugging.PortRight, ports.KeypadDown, '#')

		default:
			// keypad (right player) *OR* joystick (right player)
			// * keypad and joystick share some keys (see above for other inputs)
			return c.differentiateKeyboard(ev.Key, true)
		}
	} else {
		switch ev.Key {
		// panel
		case "F1":
			return c.handleEvents(plugging.PortPanel, ports.PanelSelect, false)
		case "F2":
			return c.handleEvents(plugging.PortPanel, ports.PanelReset, false)

		// josytick (left player)
		case "Left":
			return c.handleEvents(plugging.PortLeft, ports.Left, ports.DataStickFalse)
		case "Right":
			return c.handleEvents(plugging.PortLeft, ports.Right, ports.DataStickFalse)
		case "Up":
			return c.handleEvents(plugging.PortLeft, ports.Up, ports.DataStickFalse)
		case "Down":
			return c.handleEvents(plugging.PortLeft, ports.Down, ports.DataStickFalse)
		case "Space":
			return c.handleEvents(plugging.PortLeft, ports.Fire, false)

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			return c.handleEvents(plugging.PortRight, ports.Right, ports.DataStickFalse)

		// keyboard (left player)
		case "1", "2", "3", "Q", "W", "E", "A", "S", "D", "Z", "X", "C":
			return c.handleEvents(plugging.PortLeft, ports.KeypadUp, nil)

		// keyboard (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4", "5", "R", "T", "V", "N":
			return c.handleEvents(plugging.PortRight, ports.KeypadUp, nil)

		default:
			// keypad (right player) *OR* joystick (right player)
			// * keypad and joystick share some keys (see above for other inputs)
			return c.differentiateKeyboard(ev.Key, false)
		}
	}
}

func (c *Controllers) gamepadDPad(ev EventGamepadDPad) (bool, error) {
	switch ev.Direction {
	case DPadCentre:
		return c.handleEvents(ev.ID, ports.Centre, nil)

	case DPadUp:
		return c.handleEvents(ev.ID, ports.Up, ports.DataStickSet)

	case DPadDown:
		return c.handleEvents(ev.ID, ports.Down, ports.DataStickSet)

	case DPadLeft:
		return c.handleEvents(ev.ID, ports.Left, ports.DataStickSet)

	case DPadRight:
		return c.handleEvents(ev.ID, ports.Right, ports.DataStickSet)

	case DPadLeftUp:
		return c.handleEvents(ev.ID, ports.LeftUp, ports.DataStickSet)

	case DPadLeftDown:
		return c.handleEvents(ev.ID, ports.LeftDown, ports.DataStickSet)

	case DPadRightUp:
		return c.handleEvents(ev.ID, ports.RightUp, ports.DataStickSet)

	case DPadRightDown:
		return c.handleEvents(ev.ID, ports.RightDown, ports.DataStickSet)
	}

	return false, nil
}

func (c *Controllers) gamepadButton(ev EventGamepadButton) (bool, error) {
	switch ev.Button {
	case GamepadButtonStart:
		return c.handleEvents(plugging.PortPanel, ports.PanelReset, ev.Down)
	case GamepadButtonA:
		return c.handleEvents(ev.ID, ports.Fire, ev.Down)
	case GamepadButtonB:
		return c.handleEvents(ev.ID, ports.SecondFire, ev.Down)
	}
	return false, nil
}

func (c *Controllers) gamepadThumbstick(ev EventGamepadThumbstick) (bool, error) {
	if ev.Thumbstick != GamepadThumbstickLeft {
		return false, nil
	}

	if ev.Horiz > ThumbstickDeadzone {
		if ev.Vert > ThumbstickDeadzone {
			return c.handleEvents(ev.ID, ports.RightDown, ports.DataStickSet)
		} else if ev.Vert < -ThumbstickDeadzone {
			return c.handleEvents(ev.ID, ports.RightUp, ports.DataStickSet)
		}
		return c.handleEvents(ev.ID, ports.Right, ports.DataStickSet)
	} else if ev.Horiz < -ThumbstickDeadzone {
		if ev.Vert > ThumbstickDeadzone {
			return c.handleEvents(ev.ID, ports.LeftDown, ports.DataStickSet)
		} else if ev.Vert < -ThumbstickDeadzone {
			return c.handleEvents(ev.ID, ports.LeftUp, ports.DataStickSet)
		}
		return c.handleEvents(ev.ID, ports.Left, ports.DataStickSet)
	} else if ev.Vert > ThumbstickDeadzone {
		return c.handleEvents(ev.ID, ports.Down, ports.DataStickSet)
	} else if ev.Vert < -ThumbstickDeadzone {
		return c.handleEvents(ev.ID, ports.Up, ports.DataStickSet)
	}

	// never report that the centre event has been handled by the emulated
	// machine.
	//
	// for example, it prevents deadzone signals causing the emulation to unpause
	//
	// this might be wrong behaviour in some situations.
	_, err := c.handleEvents(ev.ID, ports.Centre, nil)
	return false, err
}

func (c *Controllers) stelladaptor(ev EventStelladaptor) (bool, error) {
	switch c.inputHandler.PeripheralID(ev.ID) {
	case plugging.PeriphStick:
		// boundary value is compared again incoming axes values
		//
		// a better strategy might be just to switch on the values that we know
		// represent the state of a digital stick (ie. up/down, left/right, at rest)
		// but I'm not yet confident in the precise values thar are sent by the
		// stelladapter
		//
		// suspected values currently:
		// 0x007f = at rest
		// 0x7fff = down (vert axis) right (horiz axis)
		// 0x8000 = up (vert axis) left (horiz axis)
		const boundaryValue = 255

		if ev.Horiz > boundaryValue {
			if ev.Vert > boundaryValue {
				return c.handleEvents(ev.ID, ports.RightDown, ports.DataStickSet)
			} else if ev.Vert < -boundaryValue {
				return c.handleEvents(ev.ID, ports.RightUp, ports.DataStickSet)
			}
			return c.handleEvents(ev.ID, ports.Right, ports.DataStickSet)
		} else if ev.Horiz < -boundaryValue {
			if ev.Vert > boundaryValue {
				return c.handleEvents(ev.ID, ports.LeftDown, ports.DataStickSet)
			} else if ev.Vert < -boundaryValue {
				return c.handleEvents(ev.ID, ports.LeftUp, ports.DataStickSet)
			}
			return c.handleEvents(ev.ID, ports.Left, ports.DataStickSet)
		} else if ev.Vert > boundaryValue {
			return c.handleEvents(ev.ID, ports.Down, ports.DataStickSet)
		} else if ev.Vert < -boundaryValue {
			return c.handleEvents(ev.ID, ports.Up, ports.DataStickSet)
		}

		return c.handleEvents(ev.ID, ports.Centre, nil)

	case plugging.PeriphPaddles:
		return c.handleEvents(ev.ID, ports.PaddleSet, ports.EventDataPaddle{
			A: ev.Horiz,
			B: ev.Vert,
		})
	}

	return false, nil
}

// HandleUserInput deciphers the Event and forwards the input to the Atari 2600
// player ports.
//
// Returns True if event has been handled/recognised by at least one of the
// registered input handlers.
func (c *Controllers) HandleUserInput(ev Event) (bool, error) {
	switch ev := ev.(type) {
	case EventKeyboard:
		return c.keyboard(ev)
	case EventMouseButton:
		return c.mouseButton(ev)
	case EventMouseMotion:
		return c.mouseMotion(ev)
	case EventGamepadDPad:
		return c.gamepadDPad(ev)
	case EventGamepadButton:
		return c.gamepadButton(ev)
	case EventGamepadThumbstick:
		return c.gamepadThumbstick(ev)
	case EventGamepadTrigger:
		// not using trigger
	case EventStelladaptor:
		return c.stelladaptor(ev)
	default:
	}

	return false, nil
}
