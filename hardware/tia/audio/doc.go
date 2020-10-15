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

// Package audio implements the audio generation of the TIA. The implementation
// is taken almost directly from Ron Fries' original implementation, found in
// TIASound.c (easily searchable). The bit patterns are taken from there and
// the channels are mixed in the same way.
//
// Unlike the Fries' implementation, the Mix() function is called every video
// cycle, returning a new sample every 114th video clock. The TIA_Process()
// function in Frie's implementation meanwhile is called to fill a buffer. The
// samepl buffer in this emulation must sit outside of the TIA emulation and
// somewhere inside the television implementation. TIASound.c is published under
// the GNU Library GPL v2.0
//
// Some modifications were made to Fries' alogorithm in accordance to similar
// modifications made to the TIASnd.cxx file of the Stella emulator v5.1.3.
// Stella is published under the GNU GPL v2.0
package audio
