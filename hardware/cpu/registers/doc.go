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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// Package registers implements the three types of registers found in the 6507.
// The three types are the: program counter, status register and the 8 bit
// accumulator type used for A, X, Y.
//
// The 8 bit registers implemented as the Register type, define all the basic
// operations available to the 6507: load, add, subtract, logical operations and
// shifts/rotates. In addition it implements the tests required for status
// updates: is the value zero, is the number negative or is the overflow bit
// set.
//
// The program counter by comparison is 16 bits wide and defines only the load
// and add operations.
//
// The status register is implemented as a series of flags. Setting of flags
// is done directly. For instance, in the CPU, we might have this sequence of
// function calls:
//
//	a.Load(10)
//	a.Subtract(11)
//	sr.Zero = a.IsZero()
//
// In this case, the zero flag in the status register will be false.
package registers
