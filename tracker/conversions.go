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

// Preset values that the MusicalNote can be. Other values should be musical
// notation. eg. "C4, D#4", etc.
const (
	Noise   = MusicalNote("*")
	Silence = "-"
	Low     = "L"
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
		case 1: // Buzzy
			switch reg.Freq {
			case 0:
				return MusicalNote("C7")
			case 1:
				return MusicalNote("C6")
			case 2:
				return MusicalNote("F5")
			case 3:
				return MusicalNote("C5")
			case 4:
				return MusicalNote("G#4")
			case 5:
				return MusicalNote("F4")
			case 6:
				return MusicalNote("D4")
			case 7:
				return MusicalNote("C4")
			case 8:
				return MusicalNote("A#3")
			case 9:
				return MusicalNote("G#3")
			case 10:
				return MusicalNote("F#3")
			case 11:
				return MusicalNote("F3")
			case 12:
				return MusicalNote("E3")
			case 13:
				return MusicalNote("D3")
			case 14:
				return MusicalNote("C#3")
			case 15:
				return MusicalNote("C3")
			case 16:
				return MusicalNote("B2")
			case 17:
				return MusicalNote("A#2")
			case 18:
				return MusicalNote("A2")
			case 19:
				return MusicalNote("G#2")
			case 20:
				return MusicalNote("G2")
			case 21:
				return MusicalNote("G2")
			case 22:
				return MusicalNote("F#2")
			case 23:
				return MusicalNote("F2")
			case 24:
				return MusicalNote("E2")
			case 25:
				return MusicalNote("E2")
			case 26:
				return MusicalNote("D#2")
			case 27:
				return MusicalNote("D2")
			case 28:
				return MusicalNote("D2")
			case 29:
				return MusicalNote("C#2")
			case 30:
				return MusicalNote("C#2")
			case 31:
				return MusicalNote("C2")
			}

		case 2: // Rumble, Flangy
			fallthrough
		case 3:
			switch reg.Freq {
			case 0:
				return MusicalNote("C#2")
			case 1:
				return MusicalNote("C#1")
			case 2:
				return MusicalNote("F#0")
			case 3:
				return MusicalNote("C#0")
			case 4:
				return Low
			case 5:
				return Low
			case 6:
				return Low
			case 7:
				return Low
			case 8:
				return Low
			case 9:
				return Low
			case 10:
				return Low
			case 11:
				return Low
			case 12:
				return Low
			case 13:
				return Low
			case 14:
				return Low
			case 15:
				return Low
			case 16:
				return Low
			case 17:
				return Low
			case 18:
				return Low
			case 19:
				return Low
			case 20:
				return Low
			case 21:
				return Low
			case 22:
				return Low
			case 23:
				return Low
			case 24:
				return Low
			case 25:
				return Low
			case 26:
				return Low
			case 27:
				return Low
			case 28:
				return Low
			case 29:
				return Low
			case 30:
				return Low
			case 31:
				return Low
			}

		case 4: // Pure
			fallthrough
		case 5:
			switch reg.Freq {
			case 0:
				return Silence
			case 1:
				return MusicalNote("B8")
			case 2:
				return MusicalNote("E8")
			case 3:
				return MusicalNote("B7")
			case 4:
				return MusicalNote("G7")
			case 5:
				return MusicalNote("E7")
			case 6:
				return MusicalNote("C#7")
			case 7:
				return MusicalNote("B6")
			case 8:
				return MusicalNote("A6")
			case 9:
				return MusicalNote("G6")
			case 10:
				return MusicalNote("F6")
			case 11:
				return MusicalNote("E6")
			case 12:
				return MusicalNote("D6")
			case 13:
				return MusicalNote("C#6")
			case 14:
				return MusicalNote("C6")
			case 15:
				return MusicalNote("B5")
			case 16:
				return MusicalNote("A#5")
			case 17:
				return MusicalNote("A5")
			case 18:
				return MusicalNote("G#5")
			case 19:
				return MusicalNote("G5")
			case 20:
				return MusicalNote("F#5")
			case 21:
				return MusicalNote("F5")
			case 22:
				return MusicalNote("F5")
			case 23:
				return MusicalNote("E5")
			case 24:
				return MusicalNote("D#5")
			case 25:
				return MusicalNote("D5")
			case 26:
				return MusicalNote("D5")
			case 27:
				return MusicalNote("C#5")
			case 28:
				return MusicalNote("C#5")
			case 29:
				return MusicalNote("C5")
			case 30:
				return MusicalNote("B4")
			case 31:
				return MusicalNote("B4")
			}

		case 6:
			fallthrough
		case 7:
			fallthrough
		case 9:
			fallthrough
		case 10: // Puzzy, Reedy
			switch reg.Freq {
			case 0:
				return MusicalNote("B4")
			case 1:
				return MusicalNote("E4")
			case 2:
				return MusicalNote("B3")
			case 3:
				return MusicalNote("G#3")
			case 4:
				return MusicalNote("E3")
			case 5:
				return MusicalNote("D3")
			case 6:
				return MusicalNote("B2")
			case 7:
				return MusicalNote("A2")
			case 8:
				return MusicalNote("G#2")
			case 9:
				return MusicalNote("F#2")
			case 10:
				return MusicalNote("E2")
			case 11:
				return MusicalNote("D#2")
			case 12:
				return MusicalNote("D2")
			case 13:
				return MusicalNote("C#2")
			case 14:
				return MusicalNote("B1")
			case 15:
				return MusicalNote("A#1")
			case 16:
				return MusicalNote("A1")
			case 17:
				return MusicalNote("G#1")
			case 18:
				return MusicalNote("G#1")
			case 19:
				return MusicalNote("G1")
			case 20:
				return MusicalNote("F#1")
			case 21:
				return MusicalNote("F1")
			case 22:
				return MusicalNote("E1")
			case 23:
				return MusicalNote("E1")
			case 24:
				return MusicalNote("D#1")
			case 25:
				return MusicalNote("D1")
			case 26:
				return MusicalNote("D1")
			case 27:
				return MusicalNote("C#1")
			case 28:
				return MusicalNote("C#1")
			case 29:
				return MusicalNote("C1")
			case 30:
				return MusicalNote("B0")
			case 31:
			}

		case 8: // White Noise
			switch reg.Freq {
			case 0:
				return MusicalNote("B1")
			case 1:
				return MusicalNote("B0")
			case 2:
				return MusicalNote("E0")
			case 3:
				return Low
			case 4:
				return Low
			case 5:
				return Low
			case 6:
				return Low
			case 7:
				return Low
			case 8:
				return Low
			case 9:
				return Low
			case 10:
				return Low
			case 11:
				return Low
			case 12:
				return Low
			case 13:
				return Low
			case 14:
				return Low
			case 15:
				return Low
			case 16:
				return Low
			case 17:
				return Low
			case 18:
				return Low
			case 19:
				return Low
			case 20:
				return Low
			case 21:
				return Low
			case 22:
				return Low
			case 23:
				return Low
			case 24:
				return Low
			case 25:
				return Low
			case 26:
				return Low
			case 27:
				return Low
			case 28:
				return Low
			case 29:
				return Low
			case 30:
				return Low
			case 31:
				return Low
			}

		case 12:
			fallthrough
		case 13: // Pure (low)
			switch reg.Freq {
			case 0:
				return MusicalNote("E8")
			case 1:
				return MusicalNote("E7")
			case 2:
				return MusicalNote("A6")
			case 3:
				return MusicalNote("E6")
			case 4:
				return MusicalNote("C6")
			case 5:
				return MusicalNote("A5")
			case 6:
				return MusicalNote("F#5")
			case 7:
				return MusicalNote("E5")
			case 8:
				return MusicalNote("D5")
			case 9:
				return MusicalNote("C5")
			case 10:
				return MusicalNote("A#4")
			case 11:
				return MusicalNote("A4")
			case 12:
				return MusicalNote("G4")
			case 13:
				return MusicalNote("F#4")
			case 14:
				return MusicalNote("F4")
			case 15:
				return MusicalNote("E4")
			case 16:
				return MusicalNote("D#4")
			case 17:
				return MusicalNote("D4")
			case 18:
				return MusicalNote("C#4")
			case 19:
				return MusicalNote("C4")
			case 20:
				return MusicalNote("B3")
			case 21:
				return MusicalNote("A#3")
			case 22:
				return MusicalNote("A#3")
			case 23:
				return MusicalNote("A3")
			case 24:
				return MusicalNote("G#3")
			case 25:
				return MusicalNote("G3")
			case 26:
				return MusicalNote("G3")
			case 27:
				return MusicalNote("F#3")
			case 28:
				return MusicalNote("F#3")
			case 29:
				return MusicalNote("F3")
			case 30:
				return MusicalNote("E3")
			case 31:
				return MusicalNote("E3")
			}

		case 14:
			fallthrough
		case 15: // Electronic
			switch reg.Freq {
			case 0:
				return MusicalNote("E4")
			case 1:
				return MusicalNote("E3")
			case 2:
				return MusicalNote("A2")
			case 3:
				return MusicalNote("E2")
			case 4:
				return MusicalNote("C#2")
			case 5:
				return MusicalNote("A1")
			case 6:
				return MusicalNote("G1")
			case 7:
				return MusicalNote("E1")
			case 8:
				return MusicalNote("D1")
			case 9:
				return MusicalNote("C#1")
			case 10:
				return MusicalNote("B0")
			case 11:
				return MusicalNote("A0")
			case 12:
				return MusicalNote("G#0")
			case 13:
				return MusicalNote("G0")
			case 14:
				return MusicalNote("F#0")
			case 15:
				return MusicalNote("E0")
			case 16:
				return MusicalNote("D#0")
			case 17:
				return MusicalNote("D0")
			case 18:
				return MusicalNote("C#0")
			case 19:
				return MusicalNote("C#0")
			case 20:
				return MusicalNote("C0")
			case 21:
				return Low
			case 22:
				return Low
			case 23:
				return Low
			case 24:
				return Low
			case 25:
				return Low
			case 26:
				return Low
			case 27:
				return Low
			case 28:
				return Low
			case 29:
				return Low
			case 30:
				return Low
			case 31:
				return Low
			}
		}
	case "PAL":
		switch reg.Control {
		case 1: // Buzzy
			switch reg.Freq {
			case 0:
				return MusicalNote("C7")
			case 1:
				return MusicalNote("C6")
			case 2:
				return MusicalNote("F5")
			case 3:
				return MusicalNote("C5")
			case 4:
				return MusicalNote("G#4")
			case 5:
				return MusicalNote("F4")
			case 6:
				return MusicalNote("D4")
			case 7:
				return MusicalNote("C4")
			case 8:
				return MusicalNote("A#3")
			case 9:
				return MusicalNote("G#3")
			case 10:
				return MusicalNote("F#3")
			case 11:
				return MusicalNote("F3")
			case 12:
				return MusicalNote("D#3")
			case 13:
				return MusicalNote("D3")
			case 14:
				return MusicalNote("C#3")
			case 15:
				return MusicalNote("C3")
			case 16:
				return MusicalNote("B2")
			case 17:
				return MusicalNote("A#2")
			case 18:
				return MusicalNote("A2")
			case 19:
				return MusicalNote("G#2")
			case 20:
				return MusicalNote("G2")
			case 21:
				return MusicalNote("F#2")
			case 22:
				return MusicalNote("F#2")
			case 23:
				return MusicalNote("F2")
			case 24:
				return MusicalNote("E2")
			case 25:
				return MusicalNote("D#2")
			case 26:
				return MusicalNote("D#2")
			case 27:
				return MusicalNote("D2")
			case 28:
				return MusicalNote("D2")
			case 29:
				return MusicalNote("C#2")
			case 30:
				return MusicalNote("C2")
			case 31:
				return MusicalNote("C2")
			}

		case 2:
			fallthrough
		case 3: // Rumble, Flangy
			switch reg.Freq {
			case 0:
				return MusicalNote("C2")
			case 1:
				return MusicalNote("C1")
			case 2:
				return MusicalNote("F0")
			case 3:
				return MusicalNote("C0")
			case 4:
				return Low
			case 5:
				return Low
			case 6:
				return Low
			case 7:
				return Low
			case 8:
				return Low
			case 9:
				return Low
			case 10:
				return Low
			case 11:
				return Low
			case 12:
				return Low
			case 13:
				return Low
			case 14:
				return Low
			case 15:
				return Low
			case 16:
				return Low
			case 17:
				return Low
			case 18:
				return Low
			case 19:
				return Low
			case 20:
				return Low
			case 21:
				return Low
			case 22:
				return Low
			case 23:
				return Low
			case 24:
				return Low
			case 25:
				return Low
			case 26:
				return Low
			case 27:
				return Low
			case 28:
				return Low
			case 29:
				return Low
			case 30:
				return Low
			case 31:
				return Low
			}

		case 4:
			fallthrough
		case 5: // Pure
			switch reg.Freq {
			case 0:
				return Silence
			case 1:
				return MusicalNote("B8")
			case 2:
				return MusicalNote("E8")
			case 3:
				return MusicalNote("B7")
			case 4:
				return MusicalNote("G7")
			case 5:
				return MusicalNote("E7")
			case 6:
				return MusicalNote("C#7")
			case 7:
				return MusicalNote("B6")
			case 8:
				return MusicalNote("A6")
			case 9:
				return MusicalNote("G6")
			case 10:
				return MusicalNote("F6")
			case 11:
				return MusicalNote("E6")
			case 12:
				return MusicalNote("D6")
			case 13:
				return MusicalNote("C#6")
			case 14:
				return MusicalNote("C6")
			case 15:
				return MusicalNote("B5")
			case 16:
				return MusicalNote("A#5")
			case 17:
				return MusicalNote("A5")
			case 18:
				return MusicalNote("G#5")
			case 19:
				return MusicalNote("G5")
			case 20:
				return MusicalNote("F#5")
			case 21:
				return MusicalNote("F5")
			case 22:
				return MusicalNote("E5")
			case 23:
				return MusicalNote("E5")
			case 24:
				return MusicalNote("D#5")
			case 25:
				return MusicalNote("D5")
			case 26:
				return MusicalNote("D5")
			case 27:
				return MusicalNote("C#5")
			case 28:
				return MusicalNote("C5")
			case 29:
				return MusicalNote("C5")
			case 30:
				return MusicalNote("B4")
			case 31:
				return MusicalNote("B4")
			}

		case 6:
			fallthrough
		case 7:
			fallthrough
		case 9:
			fallthrough
		case 10: // Puzzy, Reedy
			switch reg.Freq {
			case 0:
				return MusicalNote("B5")
			case 1:
				return MusicalNote("B4")
			case 2:
				return MusicalNote("E4")
			case 3:
				return MusicalNote("B3")
			case 4:
				return MusicalNote("G3")
			case 5:
				return MusicalNote("E3")
			case 6:
				return MusicalNote("D3")
			case 7:
				return MusicalNote("B2")
			case 8:
				return MusicalNote("A2")
			case 9:
				return MusicalNote("G2")
			case 10:
				return MusicalNote("F#2")
			case 11:
				return MusicalNote("E2")
			case 12:
				return MusicalNote("D#2")
			case 13:
				return MusicalNote("D2")
			case 14:
				return MusicalNote("C2")
			case 15:
				return MusicalNote("B1")
			case 16:
				return MusicalNote("A#1")
			case 17:
				return MusicalNote("A1")
			case 18:
				return MusicalNote("G#1")
			case 19:
				return MusicalNote("G1")
			case 20:
				return MusicalNote("G1")
			case 21:
				return MusicalNote("F#1")
			case 22:
				return MusicalNote("F1")
			case 23:
				return MusicalNote("E1")
			case 24:
				return MusicalNote("E1")
			case 25:
				return MusicalNote("D#1")
			case 26:
				return MusicalNote("D1")
			case 27:
				return MusicalNote("D1")
			case 28:
				return MusicalNote("C#1")
			case 29:
				return MusicalNote("C1")
			case 30:
				return MusicalNote("C1")
			case 31:
				return MusicalNote("B0")
			}

		case 8: // White Noise
			switch reg.Freq {
			case 0:
				return MusicalNote("B1")
			case 1:
				return MusicalNote("B0")
			case 2:
				return MusicalNote("E0")
			case 3:
				return Low
			case 4:
				return Low
			case 5:
				return Low
			case 6:
				return Low
			case 7:
				return Low
			case 8:
				return Low
			case 9:
				return Low
			case 10:
				return Low
			case 11:
				return Low
			case 12:
				return Low
			case 13:
				return Low
			case 14:
				return Low
			case 15:
				return Low
			case 16:
				return Low
			case 17:
				return Low
			case 18:
				return Low
			case 19:
				return Low
			case 20:
				return Low
			case 21:
				return Low
			case 22:
				return Low
			case 23:
				return Low
			case 24:
				return Low
			case 25:
				return Low
			case 26:
				return Low
			case 27:
				return Low
			case 28:
				return Low
			case 29:
				return Low
			case 30:
				return Low
			case 31:
				return Low
			}

		case 12:
			fallthrough
		case 13: // Pure (low)
			switch reg.Freq {
			case 0:
				return MusicalNote("E8")
			case 1:
				return MusicalNote("E7")
			case 2:
				return MusicalNote("A6")
			case 3:
				return MusicalNote("E6")
			case 4:
				return MusicalNote("C6")
			case 5:
				return MusicalNote("A5")
			case 6:
				return MusicalNote("F#5")
			case 7:
				return MusicalNote("E5")
			case 8:
				return MusicalNote("D5")
			case 9:
				return MusicalNote("C5")
			case 10:
				return MusicalNote("A#4")
			case 11:
				return MusicalNote("A4")
			case 12:
				return MusicalNote("G4")
			case 13:
				return MusicalNote("F#4")
			case 14:
				return MusicalNote("F4")
			case 15:
				return MusicalNote("E4")
			case 16:
				return MusicalNote("D#4")
			case 17:
				return MusicalNote("D4")
			case 18:
				return MusicalNote("C#4")
			case 19:
				return MusicalNote("C4")
			case 20:
				return MusicalNote("B3")
			case 21:
				return MusicalNote("A#3")
			case 22:
				return MusicalNote("A3")
			case 23:
				return MusicalNote("A3")
			case 24:
				return MusicalNote("G#3")
			case 25:
				return MusicalNote("G3")
			case 26:
				return MusicalNote("G3")
			case 27:
				return MusicalNote("F#3")
			case 28:
				return MusicalNote("F3")
			case 29:
				return MusicalNote("F3")
			case 30:
				return MusicalNote("E3")
			case 31:
				return MusicalNote("E3")
			}

		case 14:
			fallthrough
		case 15: // Electronic
			switch reg.Freq {
			case 0:
				return MusicalNote("E4")
			case 1:
				return MusicalNote("E3")
			case 2:
				return MusicalNote("A2")
			case 3:
				return MusicalNote("E2")
			case 4:
				return MusicalNote("C2")
			case 5:
				return MusicalNote("A1")
			case 6:
				return MusicalNote("G1")
			case 7:
				return MusicalNote("E1")
			case 8:
				return MusicalNote("D1")
			case 9:
				return MusicalNote("C1")
			case 10:
				return MusicalNote("B0")
			case 11:
				return MusicalNote("A0")
			case 12:
				return MusicalNote("G#0")
			case 13:
				return MusicalNote("G0")
			case 14:
				return MusicalNote("F0")
			case 15:
				return MusicalNote("E0")
			case 16:
				return MusicalNote("D#0")
			case 17:
				return MusicalNote("D0")
			case 18:
				return MusicalNote("C#0")
			case 19:
				return MusicalNote("C0")
			case 20:
				return MusicalNote("C0")
			case 21:
				return Low
			case 22:
				return Low
			case 23:
				return Low
			case 24:
				return Low
			case 25:
				return Low
			case 26:
				return Low
			case 27:
				return Low
			case 28:
				return Low
			case 29:
				return Low
			case 30:
				return Low
			case 31:
				return Low
			}
		}
	}

	// control of 0 and 11
	return Noise
}
