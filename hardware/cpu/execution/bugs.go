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

package execution

// The 6507 has some known bugs which can catch people out
type Bug string

const (
	NoBug                        Bug = ""
	JmpIndirectAddressingBug     Bug = "indirect addressing bug"
	IndexedIndirectAddressingBug Bug = "indexed indirect addressing bug"
	ZeroPageIndexBug             Bug = "zero page index bug"
)
