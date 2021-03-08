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

// mouseMotion handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func mouseMotion(ev EventMouseMotion, handle HandleInput) error {
	return handle.HandleEvent(ports.Player0ID, ports.PaddleSet, ev.X)
}

// mouseButton handles mouse events sent from a GUI. Returns true if key
// has been handled, false otherwise.
func mouseButton(ev EventMouseButton, handle HandleInput) error {
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
func keyboard(ev EventKeyboard, handle HandleInput) error {
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

func gamepadDPad(ev EventGamepadDPad, handle HandleInput) error {
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

func gamepadButton(ev EventGamepadButton, handle HandleInput) error {
	switch ev.Button {
	case GamepadButtonStart:
		return handle.HandleEvent(ports.PanelID, ports.PanelReset, ev.Down)
	case GamepadButtonA:
		return handle.HandleEvent(ev.ID, ports.Fire, ev.Down)
	}
	return nil
}

func gamepadStick(ev EventGamepadStick, handle HandleInput) error {
	return handle.HandleEvent(ev.ID, ports.PaddleSet, ev.Amount)
}

// HandleUserInput deciphers the Event and forwards the input to the Atari 2600
// player ports. Returns True if event is a Quit event and False otherwise.
func HandleUserInput(ev Event, handle HandleInput) (bool, error) {
	var err error
	switch ev := ev.(type) {
	case EventQuit:
		return true, nil
	case EventKeyboard:
		err = keyboard(ev, handle)
	case EventMouseButton:
		err = mouseButton(ev, handle)
	case EventMouseMotion:
		err = mouseMotion(ev, handle)
	case EventGamepadDPad:
		err = gamepadDPad(ev, handle)
	case EventGamepadButton:
		err = gamepadButton(ev, handle)
	case EventGamepadStick:
		err = gamepadStick(ev, handle)
	default:
	}

	return false, err
}
