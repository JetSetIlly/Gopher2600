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
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

// haltingCoordination ties all the mechanisms that can interrupt the normal
// running of the emulation.
//
// reset() and update() control the coordination itself. updating of the
// breakpoints, etc. need to be done directly on those fields
type haltCoordination struct {
	dbg *Debugger

	// has a halt condition been met since halt has been reset(). once halt
	// has been set to true it will remain set until explicitely set to false
	// (via reset())
	halt bool

	// the television has issued a yield
	televisionHalt television.HaltCondition

	// the cartridge has issued a yield signal that we should stop the debugger for
	cartridgeYield coprocessor.CoProcYield

	// the emulation must yield to the cartridge but it must be delayed until it
	// is in a better state
	//
	// this is an area that's likely to change. it's of particular interest to
	// ACE and ELF ROMs in which the coprocessor is run very early in order to
	// retrive the 6507 reset address
	deferredCartridgeYield bool

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

	// the reason why the emulation has halted
	haltReason string
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
	h.cartridgeYield = coprocessor.CoProcYield{
		Type: coprocessor.YieldProgramEnded,
	}
	h.televisionHalt = nil
}

// check for a halt condition and set the halt flag if found. returns true if
// emulation should continue and false if the emulation should halt
func (h *haltCoordination) check() bool {
	if h.cartridgeYield.Type != coprocessor.YieldProgramEnded {
		h.haltReason = string(h.cartridgeYield.Type)
		// if h.cartridgeYield.Error != nil && h.cartridgeYield.Error.Error() != "" {
		// 	h.haltReason = fmt.Sprintf("%s: %s", h.haltReason, h.cartridgeYield.Error.Error())
		// }
		h.halt = true
		return false
	}

	if h.televisionHalt != nil {
		h.haltReason = h.televisionHalt.Error()
		h.halt = true
		return false
	}

	// we don't check for regular break/trap/wathes if there are volatileTraps in place
	if h.volatileTraps.isEmpty() && h.volatileBreakpoints.isEmpty() {
		breakMessage := h.breakpoints.check()
		trapMessage := h.traps.check()
		watchMessage := h.watches.check()

		if breakMessage != "" {
			h.dbg.printLine(terminal.StyleFeedback, breakMessage)
			h.halt = true
			h.haltReason = breakMessage
		}

		if trapMessage != "" {
			h.dbg.printLine(terminal.StyleFeedback, trapMessage)
			h.halt = true
			h.haltReason = trapMessage
		}

		if watchMessage != "" {
			h.dbg.printLine(terminal.StyleFeedback, watchMessage)
			h.halt = true
			h.haltReason = watchMessage
		}

		return !h.halt
	}

	// check volatile conditions
	breakMessage := h.volatileBreakpoints.check()
	trapMessage := h.volatileTraps.check()
	h.halt = h.halt || breakMessage != "" || trapMessage != ""

	return !h.halt
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

// HaltFromTelevision implements television.Debugger interface
func (h *haltCoordination) HaltFromTelevision(halt television.HaltCondition) {
	h.televisionHalt = halt
}

// GetHaltReason returns the haltReason field from the haltCoordination type
func (dbg *Debugger) GetHaltReason() string {
	return dbg.halting.haltReason
}

// ClearHaltReason clears the haltReason field in the haltCoordination type
func (dbg *Debugger) ClearHaltReason() {
	dbg.halting.haltReason = ""
}
