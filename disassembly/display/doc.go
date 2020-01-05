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

// Package display facilitates the presentation of disassembled ROMs.
//
// The Instruction type stores the formatted parts of an individual
// disassembled instruction. Instruction should be instantiated with the
// Format command(). The Format() command takes an instance of execution.Result
// and annotates it for easy reading.
//
// The actual presentation of formatted results to the user is outside of the
// scope of this package but the Columns type is intended to help. The Update()
// function should be used to ensure that column widths are enough for all
// instances in a group of Instructions.
package display
