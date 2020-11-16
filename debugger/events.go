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
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/playmode"
)

func (dbg *Debugger) guiEventHandler(ev gui.Event) error {
	var err error

	switch ev := ev.(type) {
	case gui.EventQuit:
		dbg.running = false
		return curated.Errorf(terminal.UserInterrupt)

	case gui.EventKeyboard:
		var handled bool

		// check playmode key presses first
		handled, err = playmode.KeyboardEventHandler(ev, dbg.VCS)
		if err != nil {
			break // switch ev.(type)
		}

		if !handled {
			if ev.Down && ev.Mod == gui.KeyModNone {
				switch ev.Key {
				// debugging helpers
				case "F12":
					// toggle croppint
					err = dbg.scr.SetFeature(gui.ReqToggleCropping)
				case "F11":
					// toggle debugging colours
					err = dbg.scr.SetFeature(gui.ReqToggleDbgColors)
				case "F10":
					// toggle overlay
					err = dbg.scr.SetFeature(gui.ReqToggleOverlay)

				// screen scaling
				case "=":
					// equal sign is the same as plus, for convenience
					fallthrough
				case "+":
					// increase scaling
					err = dbg.scr.SetFeature(gui.ReqIncScale)
				case "-":
					// decrease window scanling
					err = dbg.scr.SetFeature(gui.ReqDecScale)
				}
			}
		}

	case gui.EventDbgMouseButton:
		switch ev.Button {
		case gui.MouseButtonRight:
			if ev.Down {
				err = dbg.parseInput(fmt.Sprintf("%s sl %d & hp %d", cmdBreak, ev.Scanline, ev.HorizPos), false, false)
				if err == nil {
					logger.Log("mouse break", fmt.Sprintf("on sl->%d and hp->%d", ev.Scanline, ev.HorizPos))
				}
			}
		}

	case gui.EventMouseButton:
		_, err := playmode.MouseButtonEventHandler(ev, dbg.VCS, dbg.scr)
		return err

	case gui.EventMouseMotion:
		_, err := playmode.MouseMotionEventHandler(ev, dbg.VCS)
		return err
	}

	return err
}

func (dbg *Debugger) checkEvents() error {
	for {
		select {
		case <-dbg.events.IntEvents:
			// note that ctrl-c signals do not always reach
			// this far into the program.  for instance, the colorterm
			// implementation of UserRead() puts the terminal into raw
			// mode and so must handle ctrl-c events differently.

			// if the emulation is running freely then stop emulation
			if dbg.runUntilHalt {
				dbg.runUntilHalt = false
				return nil
			}

			// stop script scribe if it one is active
			if dbg.scriptScribe.IsActive() {
				// unlike in the equivalent code in the QUIT command, there's
				// no need to call Rollback() here because the ctrl-c event
				// will not be recorded to the script
				return dbg.scriptScribe.EndSession()
			}

			// end debugger
			dbg.running = false

		case ev := <-dbg.events.GuiEvents:
			err := dbg.guiEventHandler(ev)
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
