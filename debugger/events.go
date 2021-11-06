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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/userinput"
)

func (dbg *Debugger) userInputHandler(ev userinput.Event) error {
	// quite event
	switch ev.(type) {
	case userinput.EventQuit:
		dbg.running = false
		return curated.Errorf(terminal.UserInterrupt)
	}

	// mode specific special input (not passed to the VCS as controller input)
	switch dbg.mode {
	case emulation.ModePlay:
		switch ev := ev.(type) {
		case userinput.EventMouseWheel:
			dbg.playmodeRewind(int(ev.Delta))
			return nil

		case userinput.EventKeyboard:
			if ev.Down {
				switch ev.Key {
				case "Left":
					if ev.Mod == userinput.KeyModShift {
						dbg.playmodeRewind(-1)
					}
				case "Right":
					if ev.Mod == userinput.KeyModShift {
						dbg.playmodeRewind(1)
					}
				}
			} else if ev.Mod != userinput.KeyModNone {
				return nil
			}
		case userinput.EventGamepadButton:
			if ev.Down {
				switch ev.Button {
				case userinput.GamepadButtonBumperLeft:
					dbg.playmodeRewind(-1)
					return nil
				case userinput.GamepadButtonBumperRight:
					dbg.playmodeRewind(1)
					return nil
				}
			}
		}
	}

	// applies to both playmode and debugger
	switch ev := ev.(type) {
	case userinput.EventGamepadButton:
		if ev.Down {
			switch ev.Button {
			case userinput.GamepadButtonBack:
				if dbg.State() != emulation.Paused {
					dbg.SetFeature(emulation.ReqSetPause, true)
				} else {
					dbg.SetFeature(emulation.ReqSetPause, false)
				}
				return nil
			case userinput.GamepadButtonGuide:
				switch dbg.mode {
				case emulation.ModePlay:
					dbg.SetFeature(emulation.ReqSetMode, emulation.ModeDebugger)
				case emulation.ModeDebugger:
					dbg.SetFeature(emulation.ReqSetMode, emulation.ModePlay)
				}
			}
		}
	}

	// pass to VCS controller emulation via the userinput package
	handled, err := dbg.controllers.HandleUserInput(ev, dbg.vcs.RIOT.Ports)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	// the user input was something that controls the emulation (eg. a joystick
	// direction). unpause if the emulation is currently paused
	//
	// * we're only allowing this for playmode
	if dbg.mode == emulation.ModePlay && dbg.State() == emulation.Paused && handled {
		dbg.SetFeature(emulation.ReqSetPause, false)
	}

	return nil
}

// readEventsHandler is called by inputLoop to make sure the program is
// handling pushed events and/or user input.
//
// used alongside TermReadCheck() it means the inputLoop can react without
// having to enter the TermRead() function. The TermRead() function is only
// used when the emulation is halted.
func (dbg *Debugger) readEventsHandler() error {
	for {
		select {
		case <-dbg.events.IntEvents:
			return curated.Errorf(terminal.UserInterrupt)

		case ev := <-dbg.events.UserInput:
			err := dbg.events.UserInputHandler(ev)
			if err != nil {
				return err
			}

		case ev := <-dbg.events.RawEvents:
			ev()

		case ev := <-dbg.events.RawEventsReturn:
			ev()
			return nil

		default:
			return nil
		}
	}
}
