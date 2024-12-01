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

package rewind

import (
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// the number of frames we must preserve in the userinput queue
const userinputOverhead = 2

type userinput struct {
	queue    []ports.TimedInputEvent
	playback bool
	idx      int
}

func (u *userinput) reset() {
	u.queue = u.queue[:0]
	u.playback = false
	u.idx = 0
}

func (u *userinput) crop(earliest coords.TelevisionCoords) {
	if len(u.queue) == 0 {
		return
	}

	var i int

	for i < len(u.queue) {
		if coords.GreaterThanOrEqual(u.queue[i].Time, earliest) {
			break // for loop
		}
		i++
	}

	if i > 0 && i < len(u.queue) {
		u.queue = u.queue[i:]
	}
}

func (u *userinput) stopPlayback() {
	u.playback = false
}

func (u *userinput) startPlayback(start coords.TelevisionCoords) bool {
	// find first relevant playback entry. searching from the beginning. this
	// could probably be improved
	u.idx = 0
	for u.idx < len(u.queue) {
		if coords.GreaterThanOrEqual(u.queue[u.idx].Time, start) {
			u.playback = true
			return true
		}
		u.idx++
	}
	return false
}

// RecordEvent implements input.EventRecorder interface
func (r *Rewind) RecordEvent(ev ports.TimedInputEvent) error {
	if r.userinput.playback {
		return nil
	}
	r.userinput.queue = append(r.userinput.queue, ev)
	return nil
}

// GetPlayback implements input.EventPlayback interface
func (r *Rewind) GetPlayback() (ports.TimedInputEvent, error) {
	c := r.vcs.TV.GetCoords()

	if r.userinput.idx >= len(r.userinput.queue) {
		return ports.TimedInputEvent{
			Time: c,
			InputEvent: ports.InputEvent{
				Ev: ports.NoEvent,
			},
		}, nil
	}

	if coords.Equal(c, r.userinput.queue[r.userinput.idx].Time) {
		s := r.userinput.queue[r.userinput.idx]
		r.userinput.idx++
		return s, nil
	}

	return ports.TimedInputEvent{
		Time: c,
		InputEvent: ports.InputEvent{
			Ev: ports.NoEvent,
		},
	}, nil
}
