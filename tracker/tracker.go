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
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/logger"
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

type analysis struct {
	entry Entry
	frame int
	audc  int
	audf  int
	audv  int
}

// Tracker implements the audio.Tracker interface and keeps a history of the audio registers over
// time
type Tracker struct {
	emulation Emulation
	tv        Television
	rewind    Rewind

	// contentious fields are in the trackerCrit type
	crit Listing

	analysis [2]analysis

	// previous register values so we can compare to see whether the registers have change and thus
	// worth recording
	prev [2]audio.Registers

	// emulation used for replaying tracker entries. it wil be created on demand on the first call
	// to Replay()
	replayEmulation *hardware.VCS

	// records whether the audio registers are updated more than once per frame. the tracker package
	// doesn't work well with multiple changes per frame
	moreThanOneChangePerPeriod bool
}

const maxTrackerEntries = 1024

// NewTracker is the preferred method of initialisation for the Tracker type
func NewTracker(emulation Emulation, tv Television, rewind Rewind) *Tracker {
	return &Tracker{
		emulation: emulation,
		tv:        tv,
		rewind:    rewind,
		crit: Listing{
			Entries: make([]Entry, 0, maxTrackerEntries),
		},
		analysis: [2]analysis{
			{entry: Entry{Channel: 0}},
			{entry: Entry{Channel: 1}},
		},
	}
}

// Reset removes all entries from tracker list
func (tr *Tracker) Reset() {
	tr.crit.section.Lock()
	defer tr.crit.section.Unlock()

	tr.crit.Entries = tr.crit.Entries[:0]
	tr.analysis[0].entry.Coords = coords.TelevisionCoords{}
	tr.analysis[1].entry.Coords = coords.TelevisionCoords{}
}

func (tr *Tracker) AUDCx(env audio.TrackerEnvironment, channel int, data uint8) {
	if !tr.tv.GetFrameInfo().Stable {
		return
	}

	tr.commit(env, channel)
	tr.analysis[channel].entry.Registers.Control = data
	tr.analysis[channel].audc++

	if tr.analysis[channel].audc > 1 {
		if !tr.moreThanOneChangePerPeriod {
			tr.moreThanOneChangePerPeriod = true
			logger.Logf(env, "tracker", "AUDC%d changed more than once in a frame", channel)
		}
	}
}

func (tr *Tracker) AUDFx(env audio.TrackerEnvironment, channel int, data uint8) {
	if !tr.tv.GetFrameInfo().Stable {
		return
	}

	tr.commit(env, channel)
	tr.analysis[channel].entry.Registers.Freq = data
	tr.analysis[channel].audf++

	// see AUDCx()
	if tr.analysis[channel].audf > 1 {
		if !tr.moreThanOneChangePerPeriod {
			tr.moreThanOneChangePerPeriod = true
			logger.Logf(env, "tracker", "AUDF%d changed more than once in a frame", channel)
		}
	}
}

func (tr *Tracker) AUDVx(env audio.TrackerEnvironment, channel int, data uint8) {
	if !tr.tv.GetFrameInfo().Stable {
		return
	}

	tr.commit(env, channel)
	tr.analysis[channel].entry.Registers.Volume = data
	tr.analysis[channel].audv++

	// see AUDCx()
	if tr.analysis[channel].audv > 1 {
		if !tr.moreThanOneChangePerPeriod {
			tr.moreThanOneChangePerPeriod = true
			logger.Logf(env, "tracker", "AUDV%d changed more than once in a frame. sampled audio playback?", channel)
		}
	}
}

// commit the current entry to the listing if appropriate
func (tr *Tracker) commit(env audio.TrackerEnvironment, channel int) {
	// add entry to list of entries only if we're not in the tracker emulation
	if env.IsEmulation(trackerReplayLabel) {
		return
	}

	// do nothing if frame hasn't changed
	c := tr.tv.GetCoords()
	if c.Frame <= tr.analysis[channel].entry.Coords.Frame {
		return
	}

	tr.analysis[channel].entry.Coords.Frame = c.Frame
	tr.analysis[channel].entry.Coords.Scanline = 0
	tr.analysis[channel].entry.Coords.Clock = 0
	tr.analysis[channel].audc = 0
	tr.analysis[channel].audf = 0
	tr.analysis[channel].audv = 0

	// do nothing if registers haven't changed
	if audio.CmpRegisters(tr.prev[channel], tr.analysis[channel].entry.Registers) {
		return
	}
	tr.prev[channel] = tr.analysis[channel].entry.Registers

	// add descriptive information
	tr.analysis[channel].entry.Distortion = lookupDistortion(tr.analysis[channel].entry.Registers)
	tr.analysis[channel].entry.MusicalNote = lookupMusicalNote(tr.tv, tr.analysis[channel].entry.Registers)
	tr.analysis[channel].entry.PianoKey = NoteToPianoKey(tr.analysis[channel].entry.MusicalNote)

	if tr.analysis[channel].entry.Registers.Volume > tr.crit.Current[channel].Registers.Volume {
		tr.analysis[channel].entry.Volume = VolumeRising
	} else if tr.analysis[channel].entry.Registers.Volume < tr.crit.Current[channel].Registers.Volume {
		tr.analysis[channel].entry.Volume = VolumeFalling
	} else {
		tr.analysis[channel].entry.Volume = VolumeSteady
	}

	tr.crit.section.Lock()
	defer tr.crit.section.Unlock()

	if tr.emulation.State() != govern.Rewinding {
		// find splice point in tracker
		splice := len(tr.crit.Entries) - 1
		for splice > 0 && !coords.GreaterThanOrEqual(tr.analysis[channel].entry.Coords, tr.crit.Entries[splice].Coords) {
			splice--
		}
		tr.crit.Entries = tr.crit.Entries[:splice+1]

		// add new entry and limit number of entries
		tr.crit.Entries = append(tr.crit.Entries, tr.analysis[channel].entry)
		if len(tr.crit.Entries) > maxTrackerEntries {
			tr.crit.Entries = tr.crit.Entries[1:]
		}
	}

	// store entry in lastEntry reference
	tr.crit.Current[channel] = tr.analysis[channel].entry
}

// BorrowTracker will lock the Tracker history for the duration of the supplied function, which will
// be exectued with the History structure as an argument.
//
// Should not be called from the emulation goroutine.
func (tr *Tracker) BorrowTracker(f func(*Listing)) {
	tr.crit.section.Lock()
	defer tr.crit.section.Unlock()
	f(&tr.crit)
}
