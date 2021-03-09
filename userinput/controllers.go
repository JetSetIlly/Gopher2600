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
)

// Controllers keeps track of hardware userinput options.
type Controllers struct {
	// keep track of which axis is being used. stops axes interfering with one
	// another
	axis GamepadAxis
}

// mouseMotion handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func (c *Controllers) mouseMotion(ev EventMouseMotion, handle HandleInput) error {
	return handle.HandleEvent(ports.Player0ID, ports.PaddleSet, ev.X)
}

// mouseButton handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func (c *Controllers) mouseButton(ev EventMouseButton, handle HandleInput) error {
	var err error

	switch ev.Button {
	case MouseButtonLeft:
		if ev.Down {
			err = handle.HandleEvent(ports.Player0ID, ports.Fire, true)
		} else {
			err = handle.HandleEvent(ports.Player0ID, ports.Fire, false)
		}
	}

	return err
}

// keyboard handles keypresses sent from a GUI. Returns true if
// key has been handled, false otherwise.
//
// For reasons of consistency, this handler is used by the debugger too.
func (c *Controllers) keyboard(ev EventKeyboard, handle HandleInput) error {
	var err error

	if ev.Down && ev.Mod == KeyModNone {
		switch ev.Key {
		// panel
		case "F1":
			err = handle.HandleEvent(ports.PanelID, ports.PanelSelect, true)
		case "F2":
			err = handle.HandleEvent(ports.PanelID, ports.PanelReset, true)
		case "F3":
			err = handle.HandleEvent(ports.PanelID, ports.PanelToggleColor, nil)
		case "F4":
			err = handle.HandleEvent(ports.PanelID, ports.PanelTogglePlayer0Pro, nil)
		case "F5":
			err = handle.HandleEvent(ports.PanelID, ports.PanelTogglePlayer1Pro, nil)

		// joystick
		case "Left":
			err = handle.HandleEvent(ports.Player0ID, ports.Left, ports.DataStickTrue)
		case "Right":
			err = handle.HandleEvent(ports.Player0ID, ports.Right, ports.DataStickTrue)
		case "Up":
			err = handle.HandleEvent(ports.Player0ID, ports.Up, ports.DataStickTrue)
		case "Down":
			err = handle.HandleEvent(ports.Player0ID, ports.Down, ports.DataStickTrue)
		case "Space":
			err = handle.HandleEvent(ports.Player0ID, ports.Fire, true)

		// keypad (left player)
		case "1", "2", "3":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, rune(ev.Key[0]))
		case "Q":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '4')
		case "W":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '5')
		case "E":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '6')
		case "A":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '7')
		case "S":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '8')
		case "D":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '9')
		case "Z":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '*')
		case "X":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '0')
		case "C":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardDown, '#')

		// keypad (right player)
		case "4":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '1')
		case "5":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '2')
		case "6":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '3')
		case "R":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '4')
		case "T":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '5')
		case "Y":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '6')
		case "F":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '7')
		case "G":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '8')
		case "H":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '9')
		case "V":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '*')
		case "B":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '0')
		case "N":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardDown, '#')
		}
	} else {
		switch ev.Key {
		// panel
		case "F1":
			err = handle.HandleEvent(ports.PanelID, ports.PanelSelect, false)
		case "F2":
			err = handle.HandleEvent(ports.PanelID, ports.PanelReset, false)

		// josytick
		case "Left":
			err = handle.HandleEvent(ports.Player0ID, ports.Left, ports.DataStickFalse)
		case "Right":
			err = handle.HandleEvent(ports.Player0ID, ports.Right, ports.DataStickFalse)
		case "Up":
			err = handle.HandleEvent(ports.Player0ID, ports.Up, ports.DataStickFalse)
		case "Down":
			err = handle.HandleEvent(ports.Player0ID, ports.Down, ports.DataStickFalse)
		case "Space":
			err = handle.HandleEvent(ports.Player0ID, ports.Fire, false)

		// keyboard (left player)
		case "1", "2", "3", "Q", "W", "E", "A", "S", "D", "Z", "X", "C":
			err = handle.HandleEvent(ports.Player0ID, ports.KeyboardUp, nil)

		// keyboard (right player)
		case "4", "5", "6", "R", "T", "Y", "F", "G", "H", "V", "B", "N":
			err = handle.HandleEvent(ports.Player1ID, ports.KeyboardUp, nil)
		}
	}

	return err
}

func (c *Controllers) gamepadDPad(ev EventGamepadDPad, handle HandleInput) error {
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

	return nil
}

func (c *Controllers) gamepadButton(ev EventGamepadButton, handle HandleInput) error {
	switch ev.Button {
	case GamepadButtonStart:
		return handle.HandleEvent(ports.PanelID, ports.PanelReset, ev.Down)
	case GamepadButtonA:
		return handle.HandleEvent(ev.ID, ports.Fire, ev.Down)
	}
	return nil
}

func (c *Controllers) gamepadAnalogue(ev EventGamepadAnalogue, handle HandleInput) error {
	// do nothing if something has been happening recently and this event is
	// not for that axis
	if c.axis != GamepadAxisNone && ev.Axis != c.axis {
		return nil
	}

	const deadzone = 10000
	const center = 0.4
	const centerRatio = center / 0.5

	n := float32(ev.Amount)

	switch ev.Axis {
	case GamepadAxisLeftHoriz:
	case GamepadAxisLeftTrigger:
		// left trigger uses full range for leftward  movement of the paddle
		n += 32768
		n *= -1
	case GamepadAxisRightTrigger:
		// right trigger uses full range for rightward movement of the paddle
		n += 32768
	default:
		return nil
	}

	// check deadzone
	if n >= -deadzone && n <= deadzone {
		c.axis = GamepadAxisNone
		return handle.HandleEvent(ev.ID, ports.PaddleSet, float32(center))
	}

	// note which axis this event is on
	c.axis = ev.Axis

	if n < -deadzone {
		n += deadzone
	} else if n > deadzone {
		n -= deadzone
	}

	n = (n + 32768.0) / 65535.0
	n *= centerRatio

	return handle.HandleEvent(ev.ID, ports.PaddleSet, n)
}

// HandleUserInput deciphers the Event and forwards the input to the Atari 2600
// player ports. Returns True if event is a Quit event and False otherwise.
func (c *Controllers) HandleUserInput(ev Event, handle HandleInput) (bool, error) {
	var err error
	switch ev := ev.(type) {
	case EventQuit:
		return true, nil
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
	case EventGamepadAnalogue:
		err = c.gamepadAnalogue(ev, handle)
	default:
	}

	return false, err
}
