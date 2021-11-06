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

	// whether or not the last HandleUserInput() was for an event that was
	// consumed by the emulation as an input (controller or panel)
	LastKeyHandled bool

	// is true if last event was consumed/handled by an emulated controller
	HandledByController bool

	// is true if last event was a quit emulation event
	Quit bool
}

func (c *Controllers) mouseMotion(ev EventMouseMotion, handle HandleInput) error {
	return handle.HandleEvent(plugging.PortLeftPlayer, ports.PaddleSet, ev.X)
}

func (c *Controllers) mouseButton(ev EventMouseButton, handle HandleInput) error {
	var err error

	switch ev.Button {
	case MouseButtonLeft:
		if ev.Down {
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, true)
			c.HandledByController = true
		} else {
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, false)
		}
	}

	return err
}

func (c *Controllers) keyboard(ev EventKeyboard, handle HandleInput) error {
	var err error

	if ev.Repeat {
		c.LastKeyHandled = false
		return nil
	}

	// by default we'll say the key has been handled, unless specified otherwise
	c.LastKeyHandled = true

	if ev.Down && ev.Mod == KeyModNone {
		switch ev.Key {
		// panel
		case "F1":
			err = handle.HandleEvent(plugging.PortPanel, ports.PanelSelect, true)
			c.HandledByController = true
		case "F2":
			err = handle.HandleEvent(plugging.PortPanel, ports.PanelReset, true)
			c.HandledByController = true
		case "F3":
			err = handle.HandleEvent(plugging.PortPanel, ports.PanelToggleColor, nil)
			c.HandledByController = true
		case "F4":
			err = handle.HandleEvent(plugging.PortPanel, ports.PanelTogglePlayer0Pro, nil)
			c.HandledByController = true
		case "F5":
			err = handle.HandleEvent(plugging.PortPanel, ports.PanelTogglePlayer1Pro, nil)
			c.HandledByController = true

		// joystick (left player)
		case "Left":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Left, ports.DataStickTrue)
			c.HandledByController = true
		case "Right":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Right, ports.DataStickTrue)
			c.HandledByController = true
		case "Up":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Up, ports.DataStickTrue)
			c.HandledByController = true
		case "Down":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Down, ports.DataStickTrue)
			c.HandledByController = true
		case "Space":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, true)
			c.HandledByController = true

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.Right, ports.DataStickTrue)

		// keypad (left player)
		case "1", "2", "3":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, rune(ev.Key[0]))
			c.HandledByController = true
		case "Q":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '4')
			c.HandledByController = true
		case "W":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '5')
			c.HandledByController = true
		case "E":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '6')
			c.HandledByController = true
		case "A":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '7')
			c.HandledByController = true
		case "S":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '8')
			c.HandledByController = true
		case "D":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '9')
			c.HandledByController = true
		case "Z":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '*')
			c.HandledByController = true
		case "X":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '0')
			c.HandledByController = true
		case "C":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadDown, '#')
			c.HandledByController = true

		// keypad (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '1')
			c.HandledByController = true
		case "5":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '2')
			c.HandledByController = true
		case "6":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '3')
			c.HandledByController = true
		case "R":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '4')
			c.HandledByController = true
		case "T":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '5')
			c.HandledByController = true
		case "V":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '*')
			c.HandledByController = true
		case "B":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '0')
			c.HandledByController = true
		case "N":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '#')
			c.HandledByController = true

		// keypad (right player) *OR* joystick (right player)
		// * keypad and joystick share some keys (see above for other inputs)
		case "Y":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '6')
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Up, ports.DataStickTrue)
			}
			c.HandledByController = true
		case "F":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '7')
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Fire, true)
			}
			c.HandledByController = true
		case "G":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '8')
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Left, ports.DataStickTrue)
			}
			c.HandledByController = true
		case "H":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadDown, '9')
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Down, ports.DataStickTrue)
			}
			c.HandledByController = true
		default:
			c.LastKeyHandled = false
		}
	} else {
		switch ev.Key {
		// panel
		case "F1":
			err = handle.HandleEvent(plugging.PortPanel, ports.PanelSelect, false)
		case "F2":
			err = handle.HandleEvent(plugging.PortPanel, ports.PanelReset, false)

		// josytick (left player)
		case "Left":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Left, ports.DataStickFalse)
		case "Right":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Right, ports.DataStickFalse)
		case "Up":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Up, ports.DataStickFalse)
		case "Down":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Down, ports.DataStickFalse)
		case "Space":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.Fire, false)

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.Right, ports.DataStickFalse)

		// keyboard (left player)
		case "1", "2", "3", "Q", "W", "E", "A", "S", "D", "Z", "X", "C":
			err = handle.HandleEvent(plugging.PortLeftPlayer, ports.KeypadUp, nil)

		// keyboard (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4", "5", "6", "R", "T", "V", "B", "N":
			err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)

		// keypad (right player) *OR* joystick (right player)
		// * keypad and joystick share some keys (see above for other inputs)
		case "Y":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Up, ports.DataStickFalse)
			}
		case "F":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Fire, false)
			}
		case "G":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Left, ports.DataStickFalse)
			}
		case "H":
			if handle.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.KeypadUp, nil)
			} else {
				err = handle.HandleEvent(plugging.PortRightPlayer, ports.Down, ports.DataStickFalse)
			}
		default:
			c.LastKeyHandled = false
		}
	}

	return err
}

func (c *Controllers) gamepadDPad(ev EventGamepadDPad, handle HandleInput) error {
	switch ev.Direction {
	case DPadCentre:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Centre, nil)

	case DPadUp:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Up, ports.DataStickSet)

	case DPadDown:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Down, ports.DataStickSet)

	case DPadLeft:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Left, ports.DataStickSet)

	case DPadRight:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Right, ports.DataStickSet)

	case DPadLeftUp:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.LeftUp, ports.DataStickSet)

	case DPadLeftDown:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.LeftDown, ports.DataStickSet)

	case DPadRightUp:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.RightUp, ports.DataStickSet)

	case DPadRightDown:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.RightDown, ports.DataStickSet)
	}

	return nil
}

func (c *Controllers) gamepadButton(ev EventGamepadButton, handle HandleInput) error {
	switch ev.Button {
	case GamepadButtonStart:
		c.HandledByController = true
		return handle.HandleEvent(plugging.PortPanel, ports.PanelReset, ev.Down)
	case GamepadButtonA:
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Fire, ev.Down)
	}
	return nil
}

func (c *Controllers) gamepadThumbstick(ev EventGamepadThumbstick, handle HandleInput) error {
	if ev.Thumbstick != GamepadThumbstickLeft {
		return nil
	}

	// quite a large deadzone for the thumbstick
	const deadzone = 10000

	if ev.Horiz > deadzone {
		c.HandledByController = true
		if ev.Vert > deadzone {
			return handle.HandleEvent(ev.ID, ports.RightDown, ports.DataStickSet)
		} else if ev.Vert < -deadzone {
			return handle.HandleEvent(ev.ID, ports.RightUp, ports.DataStickSet)
		}
		return handle.HandleEvent(ev.ID, ports.Right, ports.DataStickSet)
	} else if ev.Horiz < -deadzone {
		c.HandledByController = true
		if ev.Vert > deadzone {
			return handle.HandleEvent(ev.ID, ports.LeftDown, ports.DataStickSet)
		} else if ev.Vert < -deadzone {
			return handle.HandleEvent(ev.ID, ports.LeftUp, ports.DataStickSet)
		}
		return handle.HandleEvent(ev.ID, ports.Left, ports.DataStickSet)
	} else if ev.Vert > deadzone {
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Down, ports.DataStickSet)
	} else if ev.Vert < -deadzone {
		c.HandledByController = true
		return handle.HandleEvent(ev.ID, ports.Up, ports.DataStickSet)
	}

	return handle.HandleEvent(ev.ID, ports.Centre, nil)
}

func (c *Controllers) gamepadTriggers(ev EventGamepadTrigger, handle HandleInput) error {
	if c.trigger != GamepadTriggerNone && c.trigger != ev.Trigger {
		return nil
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
			return nil
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
			return nil
		}
	default:
	}

	c.paddle = n
	return handle.HandleEvent(ev.ID, ports.PaddleSet, c.paddle)
}

// HandleUserInput deciphers the Event and forwards the input to the Atari 2600
// player ports. Returns True if event should cause emulation to quit; in
// addition to any error.
func (c *Controllers) HandleUserInput(ev Event, handle HandleInput) error {
	c.Quit = false
	c.HandledByController = false

	var err error
	switch ev := ev.(type) {
	case EventQuit:
		c.Quit = true
	case EventKeyboard:
		err = c.keyboard(ev, handle)
	case EventMouseButton:
		err = c.mouseButton(ev, handle)
	case EventMouseMotion:
		err = c.mouseMotion(ev, handle)
	case EventGamepadDPad:
		err = c.gamepadDPad(ev, handle)
	case EventGamepadButton:
		err = c.gamepadButton(ev, handle)
	case EventGamepadThumbstick:
		err = c.gamepadThumbstick(ev, handle)
	case EventGamepadTrigger:
		err = c.gamepadTriggers(ev, handle)
	default:
	}

	return err
}
