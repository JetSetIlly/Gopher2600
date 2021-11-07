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
	"github.com/jetsetilly/gopher2600/hardware"
)

func (dbg *Debugger) playLoop() error {
	// run and handle events
	return dbg.vcs.Run(func() (emulation.State, error) {
		select {
		case <-dbg.eventCheckPulse.C:
			err := dbg.readEventsHandler()
			if err != nil {
				return emulation.Ending, nil
			}
		default:
		}

		if dbg.state.Load().(emulation.State) == emulation.Running {
			dbg.Rewind.RecordFrameState()
		}

		// breakpoint, trap, watch check. vcs.Run() call the continueCheck
		// funcion every CPU cycle and not every video cycle. this means some
		// halting conditions may miss or be late
		dbg.halting.check()
		if dbg.halting.halt {
			// set debugging mode. halting messages will be preserved and
			// shown when entering debugging mode
			dbg.setMode(emulation.ModeDebugger)
			return emulation.Ending, nil
		}

		if dbg.mode != emulation.ModePlay {
			return emulation.Ending, nil
		}

		if dbg.rewindKeyboardAccumulation != 0 {
			amount := 0
			if dbg.rewindKeyboardAccumulation < 0 {
				if dbg.rewindKeyboardAccumulation > -100 {
					dbg.rewindKeyboardAccumulation--
				}
				amount = (dbg.rewindKeyboardAccumulation / 10) - 1
			} else {
				if dbg.rewindKeyboardAccumulation < 100 {
					dbg.rewindKeyboardAccumulation++
				}
				amount = (dbg.rewindKeyboardAccumulation / 10) + 1
			}
			dbg.RewindByAmount(amount)
		}

		return dbg.State(), nil
	}, hardware.PerformanceBrake)
}
