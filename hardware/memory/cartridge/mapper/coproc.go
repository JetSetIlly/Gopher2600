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

import "fmt"

// CartCoProcDisasmEntry represents a single decoded instruction by the coprocessor.
type CartCoProcDisasmEntry interface {
	Key() string
	String() string
}

// CartCoProcDisasmSummary represents a summary of a coprocessor execution.
type CartCoProcDisasmSummary interface {
	String() string
}

// CartCoProcDisassembler defines the functions that must be defined for a
// disassembler to be attached to a coprocessor.
type CartCoProcDisassembler interface {
	// Start is called at the beginning of coprocessor program execution.
	Start()

	// Step called after every instruction in the coprocessor program.
	Step(CartCoProcDisasmEntry)

	// End is called when coprocessor program has finished.
	End(CartCoProcDisasmSummary)
}

// CartCoProcDeveloper is used by the coprocessor to provide functions
// available to developers when the source code is available.
type CartCoProcDeveloper interface {
	LookupSource(addr uint32)
}

// CartCoProcBus is implemented by cartridge mappers that have a coprocessor that
// functions independently from the VCS.
type CartCoProcBus interface {
	CoProcID() string
	SetDisassembler(CartCoProcDisassembler)
	SetDeveloper(CartCoProcDeveloper)
}

// CartCoProcDisassemblerStdout is a minimial implementation of the CartCoProcDisassembler
// interface. It outputs entries to stdout immediately upon request.
type CartCoProcDisassemblerStdout struct {
}

// Start implements the CartCoProcDisassembler interface.
func (c *CartCoProcDisassemblerStdout) Start() {
}

// Instruction implements the CartCoProcDisassembler interface.
func (c *CartCoProcDisassemblerStdout) Step(e CartCoProcDisasmEntry) {
	fmt.Println(e)
}

// End implements the CartCoProcDisassembler interface.
func (c *CartCoProcDisassemblerStdout) End(s CartCoProcDisasmSummary) {
	fmt.Println(s)
}
