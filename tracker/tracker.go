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
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/rewind"
)

type Rewind interface {
	GetState(frame int) *rewind.State
}

type Emulation interface {
	State() govern.State
	TV() *television.Television
}

// Entry represents a single change of audio for a channel
type Entry struct {
	// the (TV) time the change occurred
	Coords coords.TelevisionCoords

	// which channel the Registers field refers to
	Channel   int
	Registers audio.Registers

	// description of the change. the Registers field by comparison contains the
	// numeric information of the audio change
	Distortion  string
	MusicalNote MusicalNote
	PianoKey    PianoKey
}

// IsMusical returns true if entry represents a musical note
func (e Entry) IsMusical() bool {
	return e.MusicalNote != Noise && e.MusicalNote != Silence && e.MusicalNote != Low
}

const maxTrackerEntries = 1024

// Tracker implements the audio.Tracker interface and keeps a history of the
// audio registers over time
type Tracker struct {
	emulation Emulation
	rewind    Rewind

	// list of tracker entries. length is capped to maxTrackerEntries
	entries []Entry

	// previous register values so we can compare to see whether the registers
	// have change and thus worth recording
	prevRegister [2]audio.Registers

	// the most recent information for each channel. the entries do no need to
	// have happened at the same time. ie. lastEntry[0] might refer to an audio
	// change on frame 10 and lastEntry[1] on frame 20
	lastEntry [2]Entry

	// emulation used for replaying tracker entries. it wil be created on demand
	// on the first call to Replay()
	replayEmulation *hardware.VCS
}

// NewTracker is the preferred method of initialisation for the Tracker type
func NewTracker(emulation Emulation, rewind Rewind) *Tracker {
	return &Tracker{
		emulation: emulation,
		rewind:    rewind,
		entries:   make([]Entry, 0, maxTrackerEntries),
	}
}

// Reset removes all entries from tracker list
func (tr *Tracker) Reset() {
	tr.entries = tr.entries[:0]
}

// AudioTick implements the audio.Tracker interface
func (tr *Tracker) AudioTick(env audio.TrackerEnvironment, channel int, reg audio.Registers) {
	changed := !audio.CmpRegisters(reg, tr.prevRegister[channel])
	tr.prevRegister[channel] = reg

	if changed {
		tv := tr.emulation.TV()

		e := Entry{
			Coords:      tv.GetCoords(),
			Channel:     channel,
			Registers:   reg,
			Distortion:  LookupDistortion(reg),
			MusicalNote: LookupMusicalNote(tv, reg),
		}
		e.PianoKey = NoteToPianoKey(e.MusicalNote)

		// add entry to list of entries only if we're not in the tracker emulation
		if !env.IsEmulation(envLabel) {
			if tr.emulation.State() != govern.Rewinding {
				// find splice point in tracker
				splice := len(tr.entries) - 1
				for splice > 0 && !coords.GreaterThan(e.Coords, tr.entries[splice].Coords) {
					splice--
				}
				tr.entries = tr.entries[:splice+1]

				// add new entry and limit number of entries
				tr.entries = append(tr.entries, e)
				if len(tr.entries) > maxTrackerEntries {
					tr.entries = tr.entries[1:]
				}
			}
		}

		// store entry in lastEntry reference
		tr.lastEntry[channel] = e
	}
}

// Copy makes a copy of the Tracker entries
func (tr *Tracker) Copy() []Entry {
	return tr.entries
}

// GetLast entry for channel
func (tr *Tracker) GetLast(channel int) Entry {
	return tr.lastEntry[channel]
}
