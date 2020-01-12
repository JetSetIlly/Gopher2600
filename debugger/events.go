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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/errors"
	"gopher2600/gui"
	"gopher2600/playmode"
)

func (dbg *Debugger) guiEventHandler(event gui.Event) error {
	var err error

	switch event.ID {
	case gui.EventWindowClose:
		err = dbg.scr.SetFeature(gui.ReqSetVisibility, false)
		if err != nil {
			return errors.New(errors.GUIEventError, err)
		}

	case gui.EventKeyboard:
		data := event.Data.(gui.EventDataKeyboard)

		// check playmode key presses first
		err = playmode.KeyboardEventHandler(data, dbg.scr, dbg.vcs)
		if err != nil {
			break // switch event.ID
		}

		if data.Down && data.Mod == gui.KeyModNone {
			switch data.Key {
			case "`":
				// back-tick: toggle masking
				err = dbg.scr.SetFeature(gui.ReqToggleMasking)

			case "1":
				// toggle debugging colours
				err = dbg.scr.SetFeature(gui.ReqToggleAltColors)
			case "2":
				// toggle overlay
				err = dbg.scr.SetFeature(gui.ReqToggleOverlay)

			case "=":
				fallthrough // equal sign is the same as plus, for convenience
			case "+":
				// increase scaling
				err = dbg.scr.SetFeature(gui.ReqIncScale)
			case "-":
				// decrease window scanling
				err = dbg.scr.SetFeature(gui.ReqDecScale)
			}
		}

	case gui.EventMouseRight:
		data := event.Data.(gui.EventDataMouse)
		if !data.Down {
			_, err = dbg.parseInput(fmt.Sprintf("%s sl %d & hp %d", cmdBreak, data.Scanline, data.HorizPos), false, false)
			if err == nil {
				dbg.printLine(terminal.StyleFeedback, "mouse break on sl->%d and hp->%d", data.Scanline, data.HorizPos)
			}
		}
	}

	// wrap error in GUIEventError
	if err != nil {
		err = errors.New(errors.GUIEventError, err)
	}

	return err

}

func (dbg *Debugger) checkInterruptsAndEvents() error {
	var err error

	// check interrupt channel and run any functions we find in there
	select {
	case <-dbg.intChan:
		// #ctrlc halt emulation
		if dbg.runUntilHalt {
			// stop emulation at the next step
			dbg.runUntilHalt = false

			// !!TODO: rather than halting immediately set a flag that says to
			// halt at the next manual-break point. if there is no manual break
			// point then stop immediately (or end of current frame might be
			// better)

		} else {
			// runUntilHalt is false which means that the emulation is
			// not running. at this point, an input loop is probably
			// running.
			//
			// note that ctrl-c signals do not always reach
			// this far into the program.  for instance, the colorterm
			// implementation of UserRead() puts the terminal into raw
			// mode and so must handle ctrl-c events differently.

			if dbg.scriptScribe.IsActive() {
				// unlike in the equivalent code in the QUIT command, there's
				// no need to call Rollback() here because the ctrl-c event
				// will not be recorded to the script
				dbg.scriptScribe.EndSession()
			} else {
				dbg.running = false
			}
		}
	case ev := <-dbg.guiChan:
		err = dbg.guiEventHandler(ev)
	default:
		// pro-tip: default case required otherwise the select will block
		// indefinately.
	}

	return err
}
