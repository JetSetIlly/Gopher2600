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

// PianoKey is the key number on a piano keyboard.
type PianoKey int

// NoPianoKey is pressed.
const NoPianoKey = 0

// NoteToPianoKey converts the musical note to the corresponding piano key.
//
// Handle sharps but not flats.
func NoteToPianoKey(note MusicalNote) PianoKey {
	switch note {
	case "A#0":
		return -1
	case "C#1":
		return -3
	case "D#1":
		return -4
	case "F#1":
		return -6
	case "G#1":
		return -7
	case "A#1":
		return -8
	case "C#2":
		return -10
	case "D#2":
		return -11
	case "F#2":
		return -13
	case "G#2":
		return -14
	case "A#2":
		return -15
	case "C#3":
		return -17
	case "D#3":
		return -18
	case "F#3":
		return -20
	case "G#3":
		return -21
	case "A#3":
		return -22
	case "C#4":
		return -24
	case "D#4":
		return -25
	case "F#4":
		return -27
	case "G#4":
		return -28
	case "A#4":
		return -29
	case "C#5":
		return -31
	case "D#5":
		return -32
	case "F#5":
		return -34
	case "G#5":
		return -35
	case "A#5":
		return -36
	case "C#6":
		return -38
	case "D#6":
		return -39
	case "F#6":
		return -41
	case "G#6":
		return -42
	case "A#6":
		return -43
	case "C#7":
		return -45
	case "D#7":
		return -46
	case "F#7":
		return -48
	case "G#7":
		return -49
	case "A#7":
		return -50
	case "C#8":
		return -52
	case "D#8":
		return -53
	case "F#8":
		return -55
	case "G#8":
		return -56
	case "A#8":
		return -57

	case "A0":
		return 1
	case "B0":
		return 2
	case "C1":
		return 3
	case "D1":
		return 4
	case "E1":
		return 5
	case "F1":
		return 6
	case "G1":
		return 7
	case "A1":
		return 8
	case "B1":
		return 9
	case "C2":
		return 10
	case "D2":
		return 11
	case "E2":
		return 12
	case "F2":
		return 13
	case "G2":
		return 14
	case "A2":
		return 15
	case "B2":
		return 16
	case "C3":
		return 17
	case "D3":
		return 18
	case "E3":
		return 19
	case "F3":
		return 20
	case "G3":
		return 21
	case "A3":
		return 22
	case "B3":
		return 23
	case "C4":
		return 24
	case "D4":
		return 25
	case "E4":
		return 26
	case "F4":
		return 27
	case "G4":
		return 28
	case "A4":
		return 29
	case "B4":
		return 30
	case "C5":
		return 31
	case "D5":
		return 32
	case "E5":
		return 33
	case "F5":
		return 34
	case "G5":
		return 35
	case "A5":
		return 36
	case "B5":
		return 37
	case "C6":
		return 38
	case "D6":
		return 39
	case "E6":
		return 40
	case "F6":
		return 41
	case "G6":
		return 42
	case "A6":
		return 43
	case "B6":
		return 44
	case "C7":
		return 45
	case "D7":
		return 46
	case "E7":
		return 47
	case "F7":
		return 48
	case "G7":
		return 49
	case "A7":
		return 50
	case "B7":
		return 51
	case "C8":
		return 52
	case "D8":
		return 53
	case "E8":
		return 54
	case "F8":
		return 55
	case "G8":
		return 56
	case "A8":
		return 57
	case "B8":
		return 58
	}
	return NoPianoKey
}
