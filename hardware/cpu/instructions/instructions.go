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

import "fmt"

// AddressingMode describes the method data for the instruction should be received.
type AddressingMode int

// List of supported addressing modes.
const (
	Implied AddressingMode = iota
	Immediate
	Relative // relative addressing is used for branch instructions

	Absolute // abs
	ZeroPage // zpg
	Indirect // ind

	IndexedIndirect // (ind,X)
	IndirectIndexed // (ind), Y

	AbsoluteIndexedX // abs,X
	AbsoluteIndexedY // abs,Y

	ZeroPageIndexedX // zpg,X
	ZeroPageIndexedY // zpg,Y
)

// EffectCategory categorises an instruction by the effect it has.
type EffectCategory int

// List of effect categories.
const (
	Read EffectCategory = iota
	Write
	RMW

	// the following three effects have a variable effect on the program
	// counter, depending on the instruction's precise operand.

	// flow consists of the Branch and JMP instructions. Branch instructions
	// specifically can be distinguished by the AddressingMode.
	Flow

	Subroutine
	Interrupt
)

// Definition defines each instruction in the instruction set; one per instruction.
type Definition struct {
	OpCode         uint8
	Mnemonic       string
	Bytes          int
	Cycles         int
	AddressingMode AddressingMode
	PageSensitive  bool
	Effect         EffectCategory
}

// String returns a single instruction definition as a string.
func (defn Definition) String() string {
	if defn.Mnemonic == "" {
		return "undecoded instruction"
	}
	return fmt.Sprintf("%02x %s +%dbytes (%d cycles) [mode=%d pagesens=%t effect=%d]", defn.OpCode, defn.Mnemonic, defn.Bytes, defn.Cycles, defn.AddressingMode, defn.PageSensitive, defn.Effect)
}

// IsBranch returns true if instruction is a branch instruction.
func (defn Definition) IsBranch() bool {
	return defn.AddressingMode == Relative && defn.Effect == Flow
}
