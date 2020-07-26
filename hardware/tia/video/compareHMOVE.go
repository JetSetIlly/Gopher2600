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

package video

// CompareHMOVE tests two variables of type uint8 and checks to see if any of
// the bits in the lower nibble differ. returns false if no bits are the same,
// true otherwise
//
// returns true if any corresponding bits in the lower nibble are the same.
// from TIA_HW_Notes.txt:
//
// "When the comparator for a given object detects that none of the 4 bits
// match the bits in the counter state, it clears this latch"
//
func compareHMOVE(a uint8, b uint8) bool {
	return a&0x08 == b&0x08 || a&0x04 == b&0x04 || a&0x02 == b&0x02 || a&0x01 == b&0x01

	// at first flush the quotation above appears to be saying the following:
	//
	//	return a&b&0x0f != 0
	//
	// but it does not. this simpler construct does not check whether zero bits
	// are the same. the actual comparison, which we're using, compares one and
	// zero bits equally.
}
