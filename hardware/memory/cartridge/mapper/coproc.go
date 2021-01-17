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

package mapper

// CartCoProcDisasmEntry summarises a single decoded instruction by the
// coprocessor. Implementations of this type should nomalise the width of each
// field. For example, the maximum length of an Operator mnemonic might be 4
// characters, meaning that all Operator fields should be 4 characters and
// padded with spaces as required.
type CartCoProcDisasmEntry struct {
	Location       string
	Address        string
	Operator       string
	Operand        string
	ExecutionNotes string
}

// CartCoProcDisassembler defines the functions that must be defined for a
// disassembler to be attached to a coprocessor.
type CartCoProcDisassembler interface {
	Reset()
	Instruction(CartCoProcDisasmEntry)
}

// CartCoProcBus is implemented by cartridge mappers that have a coprocessor that
// functions independently from the VCS.
type CartCoProcBus interface {
	CoProcID() string
	SetDisassembler(CartCoProcDisassembler)
}
