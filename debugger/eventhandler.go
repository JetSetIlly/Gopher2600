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
	"github.com/jetsetilly/gopher2600/debugger/terminal"
)

// readEventsHandler is called by inputLoop to make sure the program is
// handling pushed events and/or user input.
//
// used alongside TermReadCheck() it means the inputLoop can react without
// having to enter the TermRead() function. The TermRead() function is only
// used when the emulation is halted.
//
// used in both playmode and debugging mode
func (dbg *Debugger) readEventsHandler() error {
	for {
		select {
		case <-dbg.events.IntEvents:
			return terminal.UserInterrupt

		case ev := <-dbg.events.UserInput:
			err := dbg.events.UserInputHandler(ev)
			if err != nil {
				return err
			}

		case ev := <-dbg.events.PushedFunctions:
			ev()

		case ev := <-dbg.events.PushedFunctionsImmediate:
			ev()
			return nil

		default:
			return nil
		}
	}
}
