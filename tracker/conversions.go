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
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
)

// LookupDistortion converts the control register value into a text
// description.
//
// Descriptions taken from Random Terrain's "The Atari 2600 Music and Sound
// Page"
//
// https://www.randomterrain.com/atari-2600-memories-music-and-sound.html
func LookupDistortion(reg audio.Registers) string {
	switch reg.Control {
	case 0:
		return "-"
	case 1:
		return "Buzzy"
	case 2:
		return "Rumble"
	case 3:
		return "Flangy"
	case 4:
		return "Pure"
	case 5:
		// same as 4
		return "Pure"
	case 6:
		return "Puzzy"
	case 7:
		return "Reedy"
	case 8:
		return "White Noise"
	case 9:
		// same as 7
		return "Reedy"
	case 10:
		// same as 6
		return "Puzzy"
	case 11:
		// same as 0
		return "-"
	case 12:
		return "Pure (low)"
	case 13:
		// same as 12
		return "Pure (low)"
	case 14:
		return "Electronic"
	case 15:
		return "Electronic"
	}

	return ""
}

// MusicalNote defines the musical note (C#, D, D#, etc.) of an TIA audio
// channel register group.
type MusicalNote string

// Values that the MusicalNote can be. For now we only have the two
// possibilities.
const (
	NoMusicalNote = MusicalNote("-")
	IsMusicalNote = MusicalNote("M")
)

// LookupMusicalNote converts the current register values for a channel into a
// musical note.
//
// Descriptions taken from Random Terrain's "The Atari 2600 Music and Sound
// Page"
//
// https://www.randomterrain.com/atari-2600-memories-music-and-sound.html
func LookupMusicalNote(tv *television.Television, reg audio.Registers) MusicalNote {
	switch tv.GetFrameInfo().Spec.ID {
	case "NTSC":
		switch reg.Control {
		case 1:
			return IsMusicalNote

		case 2:
			fallthrough
		case 3:
			return IsMusicalNote

		case 4:
			fallthrough
		case 5:
			return IsMusicalNote

		case 6:
			fallthrough
		case 7:
			fallthrough
		case 9:
			fallthrough
		case 10:
			return IsMusicalNote

		case 8:
			return IsMusicalNote

		case 12:
			fallthrough
		case 13:
			return IsMusicalNote

		case 14:
			fallthrough
		case 15:
		}
	case "PAL":
		switch reg.Control {
		case 1:
			return IsMusicalNote

		case 2:
			fallthrough
		case 3:
			return IsMusicalNote

		case 4:
			fallthrough
		case 5:
			return IsMusicalNote

		case 6:
			fallthrough
		case 7:
			fallthrough
		case 9:
			fallthrough
		case 10:
			return IsMusicalNote

		case 8:
			return IsMusicalNote

		case 12:
			fallthrough
		case 13:
			return IsMusicalNote

		case 14:
			fallthrough
		case 15:
			return IsMusicalNote
		}
	}

	// control of 0 and 11
	return NoMusicalNote
}
