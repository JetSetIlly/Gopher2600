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

	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
)

// Entry is represents a state of the audio
type Entry struct {
	// the time the change occurred
	Coords coords.TelevisionCoords

	// which channel the Registers field refers to
	Channel   int
	Registers audio.Registers

	// description of the current state. the Registers field contains the numeric information of the
	// audio change
	Distortion  Distortion
	MusicalNote MusicalNote
	Volume      VolumeChange

	// the piano key associated with the musical note
	PianoKey PianoKey
}

// IsMusical returns true if entry represents a musical note
func (e Entry) IsMusical() bool {
	return e.MusicalNote != Noise && e.MusicalNote != Silence && e.MusicalNote != Low
}

type Listing struct {
	// critical sectioning
	section sync.Mutex

	// list of tracker entries
	Entries []Entry

	// the most recent information for each channel. the entries do not need to have happened at the
	// same time
	Current [2]Entry
}
