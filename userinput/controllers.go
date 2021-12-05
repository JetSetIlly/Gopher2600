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
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// TV defines the television functions required by the Controllers type.
type TV interface {
	GetCoords() coords.TelevisionCoords
}

// Controllers keeps track of hardware userinput options.
type Controllers struct {
	tv            TV
	inputHandlers []HandleInput
	trigger       GamepadTrigger
	paddle        float32
}

// Controllers is the preferred method of initialisation for the Controllers type.
func NewControllers(tv TV) *Controllers {
	return &Controllers{
		tv:            tv,
		inputHandlers: make([]HandleInput, 0),
	}
}

// handleEvents sends the port/event information to each HandleInput
// implementation.
//
// returns True if event has been handled/recognised by at least one of the
// registered input handlers.
func (c *Controllers) handleEvents(id plugging.PortID, ev ports.Event, d ports.EventData) (bool, error) {
	var handled bool
	for _, h := range c.inputHandlers {
		v, err := h.HandleInputEvent(ports.InputEvent{Time: c.tv.GetCoords(), Port: id, Ev: ev, D: d})
		if err != nil {
			return handled, err
		}
		handled = handled || v
	}
	return handled, nil
}

func (c *Controllers) mouseMotion(ev EventMouseMotion) (bool, error) {
	return c.handleEvents(plugging.PortLeftPlayer, ports.PaddleSet, ev.X)
}

func (c *Controllers) mouseButton(ev EventMouseButton) (bool, error) {
	switch ev.Button {
	case MouseButtonLeft:
		if ev.Down {
			return c.handleEvents(plugging.PortLeftPlayer, ports.Fire, true)
		} else {
			return c.handleEvents(plugging.PortLeftPlayer, ports.Fire, false)
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

	var keyEv ports.Event
	var stickEvData ports.EventData
	var fireEvData bool
	if down {
		keyEv = ports.KeypadDown
		stickEvData = ports.DataStickTrue
		fireEvData = true
	} else {
		keyEv = ports.KeypadUp
		stickEvData = ports.DataStickFalse
		fireEvData = false
	}

	for _, h := range c.inputHandlers {
		switch key {
		case "Y":
			if h.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				ev = keyEv
				d = '6'
			} else {
				ev = ports.Up
				d = stickEvData
			}
		case "F":
			if h.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				ev = keyEv
				d = '7'
			} else {
				ev = ports.Fire
				d = fireEvData
			}
		case "G":
			if h.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				ev = keyEv
				d = '8'
			} else {
				ev = ports.Left
				d = stickEvData
			}
		case "H":
			if h.PeripheralID(plugging.PortRightPlayer) == plugging.PeriphKeypad {
				ev = keyEv
				d = '9'
			} else {
				ev = ports.Down
				d = stickEvData
			}
		default:
		}

		// all differentiated keyboard events go to the right player port
		v, err := h.HandleInputEvent(ports.InputEvent{Time: c.tv.GetCoords(), Port: plugging.PortRightPlayer, Ev: ev, D: d})
		if err != nil {
			return handled, err
		}
		handled = handled || v
	}

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
			return c.handleEvents(plugging.PortLeftPlayer, ports.Left, ports.DataStickTrue)
		case "Right":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Right, ports.DataStickTrue)
		case "Up":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Up, ports.DataStickTrue)
		case "Down":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Down, ports.DataStickTrue)
		case "Space":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Fire, true)

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			return c.handleEvents(plugging.PortRightPlayer, ports.Right, ports.DataStickTrue)

		// keypad (left player)
		case "1", "2", "3":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, rune(ev.Key[0]))
		case "Q":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '4')
		case "W":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '5')
		case "E":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '6')
		case "A":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '7')
		case "S":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '8')
		case "D":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '9')
		case "Z":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '*')
		case "X":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '0')
		case "C":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadDown, '#')

		// keypad (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '1')
		case "5":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '2')
		case "6":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '3')
		case "R":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '4')
		case "T":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '5')
		case "V":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '*')
		case "B":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '0')
		case "N":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadDown, '#')

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
			return c.handleEvents(plugging.PortLeftPlayer, ports.Left, ports.DataStickFalse)
		case "Right":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Right, ports.DataStickFalse)
		case "Up":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Up, ports.DataStickFalse)
		case "Down":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Down, ports.DataStickFalse)
		case "Space":
			return c.handleEvents(plugging.PortLeftPlayer, ports.Fire, false)

		// joystick (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "J":
			return c.handleEvents(plugging.PortRightPlayer, ports.Right, ports.DataStickFalse)

		// keyboard (left player)
		case "1", "2", "3", "Q", "W", "E", "A", "S", "D", "Z", "X", "C":
			return c.handleEvents(plugging.PortLeftPlayer, ports.KeypadUp, nil)

		// keyboard (right player)
		// * keypad and joystick share some keys (see below for other inputs)
		case "4", "5", "6", "R", "T", "V", "B", "N":
			return c.handleEvents(plugging.PortRightPlayer, ports.KeypadUp, nil)

		default:
			// keypad (right player) *OR* joystick (right player)
			// * keypad and joystick share some keys (see above for other inputs)
			return c.differentiateKeyboard(ev.Key, false)
		}
	}

	return false, nil
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
	}
	return false, nil
}

func (c *Controllers) gamepadThumbstick(ev EventGamepadThumbstick) (bool, error) {
	if ev.Thumbstick != GamepadThumbstickLeft {
		return false, nil
	}

	// quite a large deadzone for the thumbstick
	const deadzone = 10000

	if ev.Horiz > deadzone {
		if ev.Vert > deadzone {
			return c.handleEvents(ev.ID, ports.RightDown, ports.DataStickSet)
		} else if ev.Vert < -deadzone {
			return c.handleEvents(ev.ID, ports.RightUp, ports.DataStickSet)
		}
		return c.handleEvents(ev.ID, ports.Right, ports.DataStickSet)
	} else if ev.Horiz < -deadzone {
		if ev.Vert > deadzone {
			return c.handleEvents(ev.ID, ports.LeftDown, ports.DataStickSet)
		} else if ev.Vert < -deadzone {
			return c.handleEvents(ev.ID, ports.LeftUp, ports.DataStickSet)
		}
		return c.handleEvents(ev.ID, ports.Left, ports.DataStickSet)
	} else if ev.Vert > deadzone {
		return c.handleEvents(ev.ID, ports.Down, ports.DataStickSet)
	} else if ev.Vert < -deadzone {
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

func (c *Controllers) gamepadTriggers(ev EventGamepadTrigger) (bool, error) {
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
	return c.handleEvents(ev.ID, ports.PaddleSet, c.paddle)
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
		return c.gamepadTriggers(ev)
	default:
	}

	return false, nil
}

// Clear current list of input handlers.
func (c *Controllers) ClearInputHandlers() {
	c.inputHandlers = c.inputHandlers[0:]
}

// Add HandleInput implementation to list of input handlers. Each input handler
// will receive the ports.Event and ports.EventData.
//
// In many instances when running two parallel emulators that require the same
// user input it is better to add the "driver" and "passenger" emulations to
// RIOT.Ports
func (c *Controllers) AddInputHandler(h HandleInput) {
	c.inputHandlers = append(c.inputHandlers, h)
}
