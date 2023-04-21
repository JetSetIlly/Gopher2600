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

package tracker

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/rewind"
)

// create replay emulation if it has not been created already
func (tr *Tracker) createReplayEmulation(mixer television.AudioMixer) error {
	if tr.replayEmulation != nil {
		return nil
	}

	tv, err := television.NewTelevision("AUTO")
	if err != nil {
		return fmt.Errorf("tracker: create replay emulation: %w", err)
	}
	tv.AddAudioMixer(mixer)

	tr.replayEmulation, err = hardware.NewVCS(tv, nil)
	if err != nil {
		return fmt.Errorf("tracker: create replay emulation: %w", err)
	}
	tr.replayEmulation.Env.Label = replayEnv
	tr.replayEmulation.TIA.Audio.SetTracker(tr)

	return nil
}

const replayEnv = environment.Label("tracker_replay")

// Replay audio from start to end indexes
func (tr *Tracker) Replay(start int, end int, mixer television.AudioMixer) {
	// the replay will run even if the master emulation is running. this may
	// cause audible issues with the hardware audio mixing

	tr.createReplayEmulation(mixer)

	tr.crit.section.Lock()
	defer tr.crit.section.Unlock()

	startState := tr.rewind.GetState(tr.crit.Entries[start].Coords.Frame)
	if startState == nil {
		logger.Logf("tracker", "replay: can't find rewind state for frame %d", tr.crit.Entries[start].Coords.Frame)
		return
	}

	rewind.Plumb(tr.replayEmulation, startState, true)

	// get copy of end frame number so that we don't have to acquire the
	// criticial section in the replay emulation
	endFrame := tr.crit.Entries[end].Coords.Frame

	go func() {
		err := tr.replayEmulation.Run(func() (govern.State, error) {
			if tr.replayEmulation.TV.GetCoords().Frame > endFrame {
				return govern.Ending, nil
			}
			return govern.Running, nil
		})
		if err != nil {
			logger.Logf("tracker", "replay: %s", err.Error())
		}
	}()
}
