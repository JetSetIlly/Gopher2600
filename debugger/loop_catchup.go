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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// CatchUpLoop implements the rewind.Runner interface.
//
// It is called from the rewind package and sets the functions that are
// required for catchupLoop().
func (dbg *Debugger) CatchUpLoop(tgt coords.TelevisionCoords) error {
	switch dbg.Mode() {
	case emulation.ModePlay:
		fpscap := dbg.vcs.TV.SetFPSCap(false)
		defer dbg.vcs.TV.SetFPSCap(fpscap)

		dbg.vcs.Run(func() (emulation.State, error) {
			dbg.userInputHandler_catchUpLoop()

			coords := dbg.vcs.TV.GetCoords()
			if coords.Frame >= tgt.Frame {
				return emulation.Ending, nil
			}
			return emulation.Running, nil
		})

	case emulation.ModeDebugger:
		// turn off TV's fps frame limiter
		fpsCap := dbg.vcs.TV.SetFPSCap(false)

		// we've already set emulation state to emulation.Rewinding

		dbg.catchupContinue = func() bool {
			newCoords := dbg.vcs.TV.GetCoords()

			// returns true if we're to continue
			return !coords.GreaterThanOrEqual(newCoords, tgt)
		}

		// the catchup loop will cause the emulation to move out of the
		// rewinding state.
		//
		// however, we don't want necessarily want peripherals to be triggered
		// during the catchup event (it depends on the peripheral). we
		// therefore disable the peripherals now and then enable them at the
		// end of the catchupEnd function
		dbg.vcs.RIOT.Ports.DisablePeripherals(true)

		dbg.catchupEnd = func() {
			dbg.vcs.TV.SetFPSCap(fpsCap)
			dbg.catchupContinue = nil
			dbg.catchupEnd = nil
			dbg.setState(emulation.Paused)
			dbg.runUntilHalt = false
			dbg.continueEmulation = dbg.catchupEndAdj
			dbg.catchupEndAdj = false

			// make sure peripherals have been reenabled
			dbg.vcs.RIOT.Ports.DisablePeripherals(false)
		}
	}

	return nil
}
