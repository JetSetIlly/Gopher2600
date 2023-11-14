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

// catchupContext is used to inform the loop of the context in which a
// post-rewind catchup is running in
type catchupContext int

// list of valid catchupContext values
const (
	catchupGotoCoords catchupContext = iota
	catchupRewindToFrame
	catrupRerunLastNFrames
	catchupStepBack
)

// CatchUpLoop implements the rewind.Runner interface.
//
// It is called from the rewind package and sets the functions that are
// required for catchupLoop().
func (dbg *Debugger) CatchUpLoop(tgt coords.TelevisionCoords) error {
	// emulation state assertion
	if dbg.State() != govern.Rewinding {
		panic("catchup loop must only be run in the rewinding state")
	}

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

		dbg.catchupContinue = func() bool {
			newCoords := dbg.vcs.TV.GetCoords()

			// returns true if we're to continue
			return !coords.GreaterThanOrEqual(newCoords, tgt)
		}

		// disable peripherals during catch-up. they will be reenabled in the
		// cathupEnd() function
		dbg.vcs.RIOT.Ports.DisablePeripherals(true)

		// we are *not* uninhibiting coprocessor disassembly. the performance
		// loss is still too great to do so

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
