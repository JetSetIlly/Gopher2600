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

package debugger

import (
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/userinput"
)

func (dbg *Debugger) userInputHandler_catchUpLoop() {
	select {
	case ev := <-dbg.events.UserInput:
		switch ev := ev.(type) {
		case userinput.EventMouseWheel:
			dbg.rewindMouseWheelAccumulation += int(ev.Delta)
		default:
			dbg.events.UserInput <- ev
		}
	default:
	}
}

func (dbg *Debugger) userInputHandler(ev userinput.Event) error {
	// quit event handling is simple
	switch ev.(type) {
	case userinput.EventQuit:
		dbg.running = false
		return terminal.UserQuit
	}

	// if an event has been handled we return early and cut out later parts of the input handling
	var handled bool

	// handling of rewind events differs depending on which mode we're in
	switch dbg.Mode() {
	case govern.ModeDebugger:
		switch ev := ev.(type) {
		case userinput.EventMouseWheel:
			var amount int
			switch ev.Mod {
			case userinput.KeyModShift:
				amount = int(ev.Delta)
			default:
				amount = int(ev.Delta) * 5
			}
			dbg.RewindByAmount(dbg.rewindMouseWheelAccumulation + amount)
			handled = true
		}

	case govern.ModePlay:
		switch ev := ev.(type) {
		case userinput.EventMouseWheel:
			dbg.RewindByAmount(int(ev.Delta))
			handled = true

		case userinput.EventKeyboard:
			if ev.Down {
				switch ev.Key {
				case "Left":
					if ev.Mod == userinput.KeyModShift {
						if dbg.rewindKeyboardAccumulation >= 0 {
							dbg.rewindKeyboardAccumulation = -1
						}
						handled = true
					}
				case "Right":
					if ev.Mod == userinput.KeyModShift {
						if dbg.rewindKeyboardAccumulation <= 0 {
							dbg.rewindKeyboardAccumulation = 1
						}
						handled = true
					}
				}
			} else {
				dbg.rewindKeyboardAccumulation = 0
				if dbg.State() != govern.Running {
					handled = true
				}
			}
		case userinput.EventGamepadButton:
			if ev.Down {
				switch ev.Button {
				case userinput.GamepadButtonBumperLeft:
					if dbg.rewindKeyboardAccumulation >= 0 {
						dbg.rewindKeyboardAccumulation = -1
					}
					handled = true
				case userinput.GamepadButtonBumperRight:
					if dbg.rewindKeyboardAccumulation <= 0 {
						dbg.rewindKeyboardAccumulation = 1
					}
					handled = true
				}
			} else {
				dbg.rewindKeyboardAccumulation = 0
				if dbg.State() != govern.Running {
					handled = true
				}
			}
		}
	}

	// early return if event has been handled
	if handled {
		return nil
	}

	// applies to both playmode and debugger
	switch ev := ev.(type) {
	case userinput.EventGamepadButton:
		if ev.Down {
			switch ev.Button {
			case userinput.GamepadButtonBack:
				if dbg.State() != govern.Paused {
					dbg.PushSetPause(true)
				} else {
					dbg.PushSetPause(false)
				}
				handled = true
			case userinput.GamepadButtonGuide:
				switch dbg.Mode() {
				case govern.ModePlay:
					dbg.PushSetMode(govern.ModeDebugger)
				case govern.ModeDebugger:
					dbg.PushSetMode(govern.ModePlay)
				}
				handled = true
			}
		}
	}

	// early return if event has been handled
	if handled {
		return nil
	}

	// pass to VCS controller emulation via the userinput package
	handled, err := dbg.controllers.HandleUserInput(ev)
	if err != nil {
		return err
	}

	// the user input was something that controls the emulation (eg. a joystick direction). unpause
	// if the emulation is currently paused
	if dbg.Mode() == govern.ModePlay && dbg.State() == govern.Paused && handled {
		dbg.PushSetPause(false)
	}

	return nil
}
