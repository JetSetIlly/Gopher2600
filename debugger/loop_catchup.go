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
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// CatchUpLoop implements the rewind.Runner interface.
//
// It is called from the rewind package and sets the functions that are
// required for catchupLoop().
func (dbg *Debugger) CatchUpLoop(tgt coords.TelevisionCoords) error {
	switch dbg.Mode() {
	case govern.ModePlay:
		fpscap := dbg.vcs.TV.SetFPSCap(false)
		defer dbg.vcs.TV.SetFPSCap(fpscap)

		dbg.vcs.Run(func() (govern.State, error) {
			dbg.userInputHandler_catchUpLoop()

			coords := dbg.vcs.TV.GetCoords()
			if coords.Frame >= tgt.Frame {
				return govern.Ending, nil
			}
			return govern.Running, nil
		})

	case govern.ModeDebugger:
		// turn off TV's fps frame limiter
		fpsCap := dbg.vcs.TV.SetFPSCap(false)

		// we've already set emulation state to govern.Rewinding

		dbg.catchupContinue = func() bool {
			newCoords := dbg.vcs.TV.GetCoords()

			// returns true if we're to continue
			return !coords.GreaterThanOrEqual(newCoords, tgt)
		}

		// disable peripherals during catch-up. they will be reenabled in the
		// cathupEnd() function
		dbg.vcs.RIOT.Ports.DisablePeripherals(true)

		// coprocessor developer features are disabled when entering the rewind
		// state and will be enabled when entering other states. however we do
		// what the features enabled during the catch-up state so we enable
		// them early here
		if dbg.CoProcDev != nil {
			dbg.CoProcDev.Disable(false)
		}

		dbg.catchupEnd = func() {
			dbg.vcs.TV.SetFPSCap(fpsCap)
			dbg.catchupContinue = nil
			dbg.catchupEnd = nil
			dbg.setState(govern.Paused)
			dbg.runUntilHalt = false
			dbg.continueEmulation = dbg.catchupEndAdj
			dbg.catchupEndAdj = false
			dbg.vcs.RIOT.Ports.DisablePeripherals(false)
		}
	}

	return nil
}
