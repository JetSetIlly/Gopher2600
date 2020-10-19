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

package audio

import "math/rand"

// from TIASound.c:
//
// "Initialze the bit patterns for the polynomials.  The 4bit and 5bit patterns
// are the identical ones used in the tia chip.  Though the patterns could be
// packed with 8 bits per byte, using only a single bit per byte keeps the math
// simple, which is important for efficient processing".
var poly4bit = [15]uint8{1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 0}
var poly5bit = [31]uint8{0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1, 1, 0, 0,
	0, 1, 1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1}

// from TIASound.c:
//
// "I've treated the 'Div by 31' counter as another polynomial because of the
// way it operates.  It does not have a 50% duty cycle, but instead has a 13:18
// ratio (of course, 13+18 = 31).  This could also be implemented by using
// counters".
var div31 = [31]uint8{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

// from TIASound.c (referring to 9 bit polynomial table):
//
// "Rather than have a table with 511 entries, I use a random number
// generator".
var poly9bit [511]uint16

func init() {
	for i := 0; i < len(poly9bit); i++ {
		poly9bit[i] = uint16(rand.Int() & 0x01)
	}
}
