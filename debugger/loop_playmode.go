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
	"github.com/jetsetilly/gopher2600/gui"
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

		state := dbg.State()
		if dbg.mode != emulation.ModePlay {
			state = emulation.Ending
		}

		return state, nil
	}, hardware.PerformanceBrake)
}

func (dbg *Debugger) playmodeRewind(amount int) {
	coords := dbg.vcs.TV.GetCoords()
	tl := dbg.Rewind.GetTimeline()

	if amount < 0 && coords.Frame-1 <= tl.AvailableStart {
		dbg.setState(emulation.Paused)
		return
	}
	if amount > 0 && coords.Frame+1 >= tl.AvailableEnd {
		dbg.setStateQuiet(emulation.Paused, true)
		dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindAtEnd)
		return
	}

	dbg.setStateQuiet(emulation.Rewinding, true)
	dbg.Rewind.GotoFrame(coords.Frame + amount)
	dbg.setStateQuiet(emulation.Paused, true)

	if amount < 0 {
		dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindBack)
	} else {
		dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindFoward)
	}
}
