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

package instructions

// AddressingMode describes the method of memory addressing used by an instruction
type AddressingMode int

func (m AddressingMode) String() string {
	switch m {
	case Implied:
		return "Implied"
	case Immediate:
		return "Immediate"
	case Relative:
		return "Relative"
	case Absolute:
		return "Absolute"
	case Indirect:
		return "Indirect"
	case PreIndexed:
		return "PreIndexed"
	case PostIndexed:
		return "PostIndexed"
	case AbsoluteX:
		return "AbsoluteX"
	case AbsoluteY:
		return "AbsoluteY"
	}
	return "unknown addressing mode"
}

const (
	Implied AddressingMode = iota
	Immediate
	Relative
	Absolute
	Indirect
	PreIndexed  // (ind,X)
	PostIndexed // (ind), Y
	AbsoluteX   // abs,X
	AbsoluteY   // abs,Y
)
