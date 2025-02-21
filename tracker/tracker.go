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
	"sync"

	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/rewind"
)

type Rewind interface {
	GetState(frame int) *rewind.State
}

type Television interface {
	GetCoords() coords.TelevisionCoords
	GetFrameInfo() frameinfo.Current
}

type Emulation interface {
	State() govern.State
}

// VolumeChange indicates whether the volume of the channel is rising or falling or
// staying steady
type VolumeChange int

// List of values for the Volume type
const (
	VolumeSteady VolumeChange = iota
	VolumeRising
	VolumeFalling
)

func (v VolumeChange) String() string {
	switch v {
	case VolumeSteady:
		return "volume is steady"
	case VolumeRising:
		return "volume is rising"
	case VolumeFalling:
		return "volume is falling"
	}
	panic("unknown VolumeChange value")
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
	Volume      VolumeChange
}

// IsMusical returns true if entry represents a musical note
func (e Entry) IsMusical() bool {
	return e.MusicalNote != Noise && e.MusicalNote != Silence && e.MusicalNote != Low
}

const maxTrackerEntries = 1024

type History struct {
	// critical sectioning
	section sync.Mutex

	// list of tracker Entries. length is capped to maxTrackerEntries
	Entries []Entry

	// the most recent information for each channel. the entries do no need to
	// have happened at the same time. ie. Recent[0] might refer to an audio
	// change on frame 10 and Recent[1] on frame 20
	Recent [2]Entry
}

// Tracker implements the audio.Tracker interface and keeps a history of the
// audio registers over time
type Tracker struct {
	emulation Emulation
	tv        Television
	rewind    Rewind

	// contentious fields are in the trackerCrit type
	crit History

	// previous register values so we can compare to see whether the registers
	// have change and thus worth recording
	prev [2]audio.Registers

	// emulation used for replaying tracker entries. it wil be created on demand
	// on the first call to Replay()
	replayEmulation *hardware.VCS
}

// NewTracker is the preferred method of initialisation for the Tracker type
func NewTracker(emulation Emulation, tv Television, rewind Rewind) *Tracker {
	return &Tracker{
		emulation: emulation,
		tv:        tv,
		rewind:    rewind,
		crit: History{
			Entries: make([]Entry, 0, maxTrackerEntries),
		},
	}
}

// Reset removes all entries from tracker list
func (tr *Tracker) Reset() {
	tr.crit.section.Lock()
	defer tr.crit.section.Unlock()

	tr.crit.Entries = tr.crit.Entries[:0]
}

// AudioTick implements the audio.Tracker interface
func (tr *Tracker) AudioTick(env audio.TrackerEnvironment, channel int, reg audio.Registers) {
	// do nothing if register hasn't changed
	match := audio.CmpRegisters(reg, tr.prev[channel])
	tr.prev[channel] = reg
	if match {
		return
	}

	tr.crit.section.Lock()
	defer tr.crit.section.Unlock()

	e := Entry{
		Coords:      tr.tv.GetCoords(),
		Channel:     channel,
		Registers:   reg,
		Distortion:  LookupDistortion(reg),
		MusicalNote: LookupMusicalNote(tr.tv, reg),
	}

	e.PianoKey = NoteToPianoKey(e.MusicalNote)

	if e.Registers.Volume > tr.crit.Recent[channel].Registers.Volume {
		e.Volume = VolumeRising
	} else if e.Registers.Volume < tr.crit.Recent[channel].Registers.Volume {
		e.Volume = VolumeFalling
	} else {
		e.Volume = VolumeSteady
	}

	// add entry to list of entries only if we're not in the tracker emulation
	if !env.IsEmulation(replayLabel) {
		if tr.emulation.State() != govern.Rewinding {
			// find splice point in tracker
			splice := len(tr.crit.Entries) - 1
			for splice > 0 && !coords.GreaterThan(e.Coords, tr.crit.Entries[splice].Coords) {
				splice--
			}
			tr.crit.Entries = tr.crit.Entries[:splice+1]

			// add new entry and limit number of entries
			tr.crit.Entries = append(tr.crit.Entries, e)
			if len(tr.crit.Entries) > maxTrackerEntries {
				tr.crit.Entries = tr.crit.Entries[1:]
			}
		}
	}

	// store entry in lastEntry reference
	tr.crit.Recent[channel] = e
}

// BorrowTracker will lock the Tracker history for the duration of the supplied
// function, which will be exectued with the History structure as an argument.
//
// Should not be called from the emulation goroutine.
func (tr *Tracker) BorrowTracker(f func(*History)) {
	tr.crit.section.Lock()
	defer tr.crit.section.Unlock()
	f(&tr.crit)
}
