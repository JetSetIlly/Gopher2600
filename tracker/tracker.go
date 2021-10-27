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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
)

type Entry struct {
	Coords    coords.TelevisionCoords
	Channel   int
	Registers audio.Registers

	Distortion  string
	MusicalNote MusicalNote
}

// Tracker implements the audio.Tracker interface and keeps a history of the
// audio registers over time.
type Tracker struct {
	emulation emulation.Emulation

	entries []Entry

	// previous register values so we can compare to see whether the registers
	// have change and thus worth recording
	prevRegister0 audio.Registers
	prevRegister1 audio.Registers
}

// NewTracker is the preferred method of initialisation for the Tracker type.
func NewTracker(emulation emulation.Emulation) *Tracker {
	return &Tracker{
		emulation: emulation,
		entries:   make([]Entry, 0, 1024),
	}
}

// Tick implements the audio.Tracker interface
func (tr *Tracker) Tick(channel int, reg audio.Registers, changed bool) {
	if tr.emulation.State() == emulation.Rewinding {
		return
	}

	if changed {
		switch channel {
		case 0:
			changed = !audio.CmpRegisters(reg, tr.prevRegister0)
			tr.prevRegister0 = reg
		case 1:
			changed = !audio.CmpRegisters(reg, tr.prevRegister1)
			tr.prevRegister1 = reg
		}

		tv := tr.emulation.TV().(*television.Television)

		if changed {
			tr.entries = append(tr.entries, Entry{
				Coords:      tv.GetCoords(),
				Channel:     channel,
				Registers:   reg,
				Distortion:  LookupDistortion(reg),
				MusicalNote: LookupMusicalNote(tv, reg),
			})
			if len(tr.entries) > 1024 {
				tr.entries = tr.entries[1:]
			}
		}
	}
}

// Copy makes a copy of the Tracker entries.
func (tr *Tracker) Copy() []Entry {
	return tr.entries
}
