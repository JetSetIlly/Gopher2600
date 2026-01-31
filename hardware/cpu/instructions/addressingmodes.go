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
	case ZeroPage:
		return "ZeroPage"
	case Indirect:
		return "Indirect"
	case IndexedIndirect:
		return "IndexedIndirect"
	case IndirectIndexed:
		return "IndirectIndexed"
	case AbsoluteIndexedX:
		return "AbsoluteIndexedX"
	case AbsoluteIndexedY:
		return "AbsoluteIndexedY"
	case ZeroPageIndexedX:
		return "ZeroPageIndexedX"
	case ZeroPageIndexedY:
		return "ZeroPageIndexedY"
	}
	return "unknown addressing mode"
}

const (
	Implied AddressingMode = iota
	Immediate
	Relative
	Absolute
	ZeroPage
	Indirect
	IndexedIndirect  // (ind,X)
	IndirectIndexed  // (ind), Y
	AbsoluteIndexedX // abs,X
	AbsoluteIndexedY // abs,Y
	ZeroPageIndexedX // zpg,X
	ZeroPageIndexedY // zpg,Y
)
