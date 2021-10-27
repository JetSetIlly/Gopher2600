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

package playmode

import (
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/rewind"
)

// CatchUpLoop implements the rewind.Runner interface.
func (pl *playmode) CatchUpLoop(tgt coords.TelevisionCoords, callback rewind.CatchUpLoopCallback) error {
	fpscap := pl.vcs.TV.SetFPSCap(false)
	defer pl.vcs.TV.SetFPSCap(fpscap)

	pl.vcs.Run(func() (emulation.State, error) {
		coords := pl.vcs.TV.GetCoords()
		if coords.Frame >= tgt.Frame {
			return emulation.Ending, nil
		}
		return emulation.Running, nil
	}, 1)

	return nil
}

func (pl *playmode) doRewind(amount int) {
	coords := pl.vcs.TV.GetCoords()
	tl := pl.rewind.GetTimeline()

	if amount < 0 && coords.Frame-1 <= tl.AvailableStart {
		pl.scr.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindAtStart)
		pl.setState(emulation.Paused)
		return
	}
	if amount > 0 && coords.Frame+1 >= tl.AvailableEnd {
		pl.scr.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindAtEnd)
		pl.setState(emulation.Paused)
		return
	}

	pl.setState(emulation.Rewinding)
	pl.rewind.GotoFrame(coords.Frame + amount)
	pl.setState(emulation.Paused)

	if amount < 0 {
		pl.scr.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindBack)
	} else {
		pl.scr.SetFeature(gui.ReqEmulationEvent, emulation.EventRewindFoward)
	}
}
