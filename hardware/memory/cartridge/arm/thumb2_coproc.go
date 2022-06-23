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

// see thumb2.go for documentation information

package arm

func (arm *ARM) thumb2Coprocessor(opcode uint16) {
	// "3.3.7 Coprocessor instructions" of "Thumb-2 Supplement"
	//
	// and
	//
	// "A5.3.18 Coprocessor instructions" of "ARMv7-M"

	panic("coprocessor")
}
