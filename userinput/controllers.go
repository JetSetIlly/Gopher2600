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

// Controllers keeps track of hardware userinput options.
type Controllers struct {
	trigger GamepadTrigger
	paddle  float32
}

func (c *Controllers) mouseMotion(ev EventMouseMotion, handle HandleInput) (bool, error) {
	return handle.HandleEvent(plugging.PortLeftPlayer, ports.PaddleSet, ev.X)
}

func (c *Controllers) mouseButton(ev EventMouseButton, handle HandleInput) (bool, error) {
	switch ev.Button {
	case MouseButtonLeft:
		if ev.Down {
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, true)
		} else {
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, false)
		}
	}

	return false, nil
}

func (c *Controllers) keyboard(ev EventKeyboard, handle HandleInput) (bool, error) {
	if ev.Down && ev.Mod == KeyModNone {
		switch ev.Key {
		// panel
		case "F1":
			return handle.HandleEvent(plugging.PortPanel, ports.PanelSelect, true)
		case "F2":
			return handle.HandleEvent(plugging.PortPanel, ports.PanelReset, true)
		case "F3":
			return handle.HandleEvent(plugging.PortPanel, ports.PanelToggleColor, nil)
		case "F4":
			return handle.HandleEvent(plugging.PortPanel, ports.PanelTogglePlayer0Pro, nil)
		case "F5":
			return handle.HandleEvent(plugging.PortPanel, ports.PanelTogglePlayer1Pro, nil)

		// joystick (left player)
		case "Left":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Left, ports.DataStickTrue)
		case "Right":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Right, ports.DataStickTrue)
		case "Up":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Up, ports.DataStickTrue)
		case "Down":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Down, ports.DataStickTrue)
		case "Space":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, true)

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.Right, ports.DataStickTrue)

		// keypad (left player)
		case "1", "2", "3":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, rune(ev.Key[0]))
		case "Q":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '4')
		case "W":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '5')
		case "E":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '6')
		case "A":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '7')
		case "S":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '8')
		case "D":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '9')
		case "Z":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '*')
		case "X":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '0')
		case "C":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '#')

		// keypad (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '1')
		case "5":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '2')
		case "6":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '3')
		case "R":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '4')
		case "T":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '5')
		case "V":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '*')
		case "B":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '0')
		case "N":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '#')

		// keypad (right player) *OR* joystick (right player)
		// * keypad and joystick share some keys (see above for other inputs)
		case "Y":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '6')
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Up, ports.DataStickTrue)
			}
		case "F":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '7')
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Fire, true)
			}
		case "G":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '8')
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Left, ports.DataStickTrue)
			}
		case "H":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '9')
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Down, ports.DataStickTrue)
			}
		default:
		}
	} else {
		switch ev.Key {
		// panel
		case "F1":
			return handle.HandleEvent(plugging.PortPanel, ports.PanelSelect, false)
		case "F2":
			return handle.HandleEvent(plugging.PortPanel, ports.PanelReset, false)

		// josytick (left player)
		case "Left":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Left, ports.DataStickFalse)
		case "Right":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Right, ports.DataStickFalse)
		case "Up":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Up, ports.DataStickFalse)
		case "Down":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Down, ports.DataStickFalse)
		case "Space":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, false)

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.Right, ports.DataStickFalse)

		// keyboard (left player)
		case "1", "2", "3", "Q", "W", "E", "A", "S", "D", "Z", "X", "C":
			return handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadUp, nil)

		// keyboard (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4", "5", "6", "R", "T", "V", "B", "N":
			return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)

		// keypad (right player) *OR* joystick (right player)
		// * keypad and joystick share some keys (see above for other inputs)
		case "Y":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Up, ports.DataStickFalse)
			}
		case "F":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Fire, false)
			}
		case "G":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Left, ports.DataStickFalse)
			}
		case "H":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				return handle.HandleEvent(plugging.PortRightPlayer, ports.Down, ports.DataStickFalse)
			}
		default:
		}
	}

	return false, nil
}

func (c *Controllers) gamepadDPad(ev EventGamepadDPad, handle HandleInput) (bool, error) {
	switch ev.Direction {
	case DPadCentre:
		return handle.HandleEvent(ev.ID, ports.Centre, nil)

	case DPadUp:
		return handle.HandleEvent(ev.ID, ports.Up, ports.DataStickSet)

	case DPadDown:
		return handle.HandleEvent(ev.ID, ports.Down, ports.DataStickSet)

	case DPadLeft:
		return handle.HandleEvent(ev.ID, ports.Left, ports.DataStickSet)

	case DPadRight:
		return handle.HandleEvent(ev.ID, ports.Right, ports.DataStickSet)

	case DPadLeftUp:
		return handle.HandleEvent(ev.ID, ports.LeftUp, ports.DataStickSet)

	case DPadLeftDown:
		return handle.HandleEvent(ev.ID, ports.LeftDown, ports.DataStickSet)

	case DPadRightUp:
		return handle.HandleEvent(ev.ID, ports.RightUp, ports.DataStickSet)

	case DPadRightDown:
		return handle.HandleEvent(ev.ID, ports.RightDown, ports.DataStickSet)
	}

	return false, nil
}

func (c *Controllers) gamepadButton(ev EventGamepadButton, handle HandleInput) (bool, error) {
	switch ev.Button {
	case GamepadButtonStart:
		return handle.HandleEvent(plugging.PortPanel, ports.PanelReset, ev.Down)
	case GamepadButtonA:
		return handle.HandleEvent(ev.ID, ports.Fire, ev.Down)
	}
	return false, nil
}

func (c *Controllers) gamepadThumbstick(ev EventGamepadThumbstick, handle HandleInput) (bool, error) {
	if ev.Thumbstick != GamepadThumbstickLeft {
		return false, nil
	}

	// quite a large deadzone for the thumbstick
	const deadzone = 10000

	if ev.Horiz > deadzone {
		if ev.Vert > deadzone {
			return handle.HandleEvent(ev.ID, ports.RightDown, ports.DataStickSet)
		} else if ev.Vert < -deadzone {
			return handle.HandleEvent(ev.ID, ports.RightUp, ports.DataStickSet)
		}
		return handle.HandleEvent(ev.ID, ports.Right, ports.DataStickSet)
	} else if ev.Horiz < -deadzone {
		if ev.Vert > deadzone {
			return handle.HandleEvent(ev.ID, ports.LeftDown, ports.DataStickSet)
		} else if ev.Vert < -deadzone {
			return handle.HandleEvent(ev.ID, ports.LeftUp, ports.DataStickSet)
		}
		return handle.HandleEvent(ev.ID, ports.Left, ports.DataStickSet)
	} else if ev.Vert > deadzone {
		return handle.HandleEvent(ev.ID, ports.Down, ports.DataStickSet)
	} else if ev.Vert < -deadzone {
		return handle.HandleEvent(ev.ID, ports.Up, ports.DataStickSet)
	}

	// never report that the centre event has been handled by the emulated
	// machine.
	//
	// for example, it prevents deadzone signals causing the emulation to unpause
	//
	// this might be wrong behaviour in some situations.
	_, err := handle.HandleEvent(ev.ID, ports.Centre, nil)
	return false, err
}

func (c *Controllers) gamepadTriggers(ev EventGamepadTrigger, handle HandleInput) (bool, error) {
	if c.trigger != GamepadTriggerNone && c.trigger != ev.Trigger {
		return false, nil
	}

	// small deadzone for the triggers
	const deadzone = 10

	const min = 0.0
	const max = 65535.0
	const mid = 32768.0

	n := float32(ev.Amount)
	n += mid

	switch ev.Trigger {
	case GamepadTriggerLeft:

		// check deadzone
		if n >= -deadzone && n <= deadzone {
			c.trigger = GamepadTriggerNone
			n = min
		} else {
			c.trigger = GamepadTriggerLeft
			n = max - n
			n /= max
		}

		// left trigger can only move the paddle left
		if n > c.paddle {
			return false, nil
		}
	case GamepadTriggerRight:
		// check deadzone
		if n >= -deadzone && n <= deadzone {
			c.trigger = GamepadTriggerNone
			n = min
		} else {
			c.trigger = GamepadTriggerRight
			n /= max
		}

		// right trigger can only move the paddle right
		if n < c.paddle {
			return false, nil
		}
	default:
	}

	c.paddle = n
	return handle.HandleEvent(ev.ID, ports.PaddleSet, c.paddle)
}

// HandleUserInput deciphers the Event and forwards the input to the Atari 2600
// player ports. Returns True if event should cause emulation to quit; in
// addition to any error.
func (c *Controllers) HandleUserInput(ev Event, handle HandleInput) (bool, error) {
	switch ev := ev.(type) {
	case EventKeyboard:
		return c.keyboard(ev, handle)
	case EventMouseButton:
		return c.mouseButton(ev, handle)
	case EventMouseMotion:
		return c.mouseMotion(ev, handle)
	case EventGamepadDPad:
		return c.gamepadDPad(ev, handle)
	case EventGamepadButton:
		return c.gamepadButton(ev, handle)
	case EventGamepadThumbstick:
		return c.gamepadThumbstick(ev, handle)
	case EventGamepadTrigger:
		return c.gamepadTriggers(ev, handle)
	default:
	}

	return false, nil
}
