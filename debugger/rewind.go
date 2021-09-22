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

// support functions for the rewind package that require more knowledge of
// the debugger than would otherwise be available.

import (
	"github.com/jetsetilly/gopher2600/emulation"
)

// PushRewind is a special case of PushRawEvent(). It prevents too many pushed
// rewind.GotoFrame function calls. Returns false if the rewind hasn't been
// pushed. The caller should try again.
//
// To be used from the GUI thread.
func (dbg *Debugger) PushRewind(fn int, last bool) bool {
	if dbg.State() == emulation.Rewinding {
		return false
	}

	// set state to emulation.Rewinding as soon as possible
	dbg.setState(emulation.Rewinding)

	// the function to push to the debugger/emulation routine
	doRewind := func() error {
		if last {
			err := dbg.Rewind.GotoLast()
			if err != nil {
				return err
			}
		} else {
			err := dbg.Rewind.GotoFrame(fn)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// how we push the doRewind() function depends on what kind of inputloop we
	// are currently in
	dbg.PushRawEventReturn(func() {
		dbg.unwindLoop(doRewind)
	})

	return true
}
