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

// haltingCoordination ties all the mechanisms that can interrupt the normal
// running of the emulation.
//
// reset() and update() control the coordination itself. updating of the
// breakpoints, etc. need to be done directly on those fields
type haltCoordination struct {
	dbg *Debugger

	// has a halt condition been met since halt has been reset(). once halt
	// has been set to true it will remain set until explicitely set to fale
	// (via reset())
	halt bool

	// halt conditions
	breakpoints *breakpoints
	traps       *traps
	watches     *watches

	// volatile conditions. if these are non-empty they will take precedence
	// over the non-volatile conditions above.
	//
	// volatile conditions are always cleared in the input loop before
	// emulation continues after a halt
	volatileBreakpoints *breakpoints
	volatileTraps       *traps
}

func newHaltCoordination(dbg *Debugger) (*haltCoordination, error) {
	h := &haltCoordination{dbg: dbg}

	var err error

	// set up breakpoints/traps
	h.breakpoints, err = newBreakpoints(dbg)
	if err != nil {
		return nil, err
	}
	h.traps = newTraps(dbg)
	h.watches = newWatches(dbg)

	h.volatileBreakpoints, err = newBreakpoints(dbg)
	if err != nil {
		return nil, err
	}
	h.volatileTraps = newTraps(dbg)

	return h, nil
}

// reset halt condition.
func (h *haltCoordination) reset() {
	h.halt = false
}

// check for a halt condition and set the halt flag if found.
func (h *haltCoordination) check() {
	// we don't check for regular break/trap/wathes if there are volatileTraps in place
	if h.volatileTraps.isEmpty() && h.volatileBreakpoints.isEmpty() {
		breakMessage := h.breakpoints.check()
		trapMessage := h.traps.check()
		watchMessage := h.watches.check()

		if breakMessage != "" {
			h.dbg.printLine(terminal.StyleFeedback, breakMessage)
			h.halt = true
		}

		if trapMessage != "" {
			h.dbg.printLine(terminal.StyleFeedback, trapMessage)
			h.halt = true
		}

		if watchMessage != "" {
			h.dbg.printLine(terminal.StyleFeedback, watchMessage)
			h.halt = true
		}

		return
	}

	// check volatile conditions
	breakMessage := h.volatileBreakpoints.check()
	trapMessage := h.volatileTraps.check()
	h.halt = h.halt || breakMessage != "" || trapMessage != ""
}

// returns false if a breakpoint or trap target has the notInPlaymode flag set
func (h *haltCoordination) allowPlaymode() bool {
	for _, b := range h.breakpoints.breaks {
		if b.target.notInPlaymode {
			return false
		}
		n := b.next
		for n != nil {
			if n.target.notInPlaymode {
				return false
			}
			n = n.next
		}
	}

	for _, t := range h.traps.traps {
		if t.target.notInPlaymode {
			return false
		}
	}

	return true
}
