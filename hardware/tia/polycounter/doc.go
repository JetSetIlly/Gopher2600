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

// Package polycounter implements the polynomial counters found in the TIA.
// Described by Andrew Towers in the "Atari 2600 TIA Hardware Notes"
// (TIA_HW_Notes.txt), polynomial counters are a predictably performative way
// of counting in simple electronics - performance of ripple counters can
// change due to carrying etc.
//
// In our emulation we are normal integers but for the purposes of debugging
// the TIA loop (HSYNC counter) we'd still like to know what the equivalent
// polycounter value is. We use a 6-bit polycounter for this.
//
//	hsync := polycounter.New(6)
//
// As the emulated polycounter is just an integer we can "tick" it along in the
// obvious way. We should take care to make sure it doesn't run past the end of
// the polycounter however. The accepted pattern is:
//
//	p++
//	if p >= polycounter.LenTable6Bit {
//		p = 0
//	}
//
// Whenever the polycounter is to be reset set it it polycounter.ResetValue.
//
// The polycounter bit pattern can be retrieved at any time with the ToBinary()
// function.
//
// # Additional Note
//
// In the 2600, polycounter logic is also used to generate the bit sequences
// required for TIA audio emulation. A real TIA variously uses 4-bit, 5-bit and
// 9-bit polycounters to generate the sound waves available to the 2600
// programmer. As of yet, this package doesn't support this functionality
// correctly. The bit sequences required are hard-coded into the tia/audio
// package as discovered by Ron Fries.
package polycounter
