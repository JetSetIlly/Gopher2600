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

type haltCoordination struct {
	set bool

	breakMessage    string
	trapMessage     string
	watchMessage    string
	stepTrapMessage string
}

func (h *haltCoordination) reset() {
	h.set = false
	h.breakMessage = ""
	h.trapMessage = ""
	h.watchMessage = ""
	h.stepTrapMessage = ""
}

func (h *haltCoordination) update(dbg *Debugger) {
	// check for breakpoints and traps. for video cycle input loops we only
	// do this if the instruction has affected flow.
	h.breakMessage = dbg.breakpoints.check(h.breakMessage)
	h.trapMessage = dbg.traps.check(h.trapMessage)
	h.watchMessage = dbg.watches.check(h.watchMessage)
	h.stepTrapMessage = dbg.stepTraps.check("")

	// check for halt conditions
	h.set = h.stepTrapMessage != "" || h.breakMessage != "" || h.trapMessage != "" || h.watchMessage != ""
}
