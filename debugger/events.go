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
	"github.com/jetsetilly/gopher2600/userinput"
)

func (dbg *Debugger) userInputHandler(ev userinput.Event) error {
	quit, err := dbg.controllers.HandleUserInput(ev, dbg.VCS.RIOT.Ports)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	// on quit set running to false and return a UserInterrupt to make sure we
	// loop and check the dbg.running flag as soon as possible.
	if quit {
		dbg.running = false
		return curated.Errorf(terminal.UserInterrupt)
	}

	return nil
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

		case ev := <-dbg.events.UserInput:
			err := dbg.userInputHandler(ev)
			if err != nil {
				return err
			}

		case ev := <-dbg.events.RawEvents:
			ev()

		case ev := <-dbg.events.RawEventsImm:
			ev()
			return nil

		default:
			return nil
		}
	}
}
