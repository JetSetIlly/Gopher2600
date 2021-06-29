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

// CartCoProcDisasmEntry represents a single decoded instruction by the
// coprocessor. Generators of this type should nomalise the width of each
// field. For example, the maximum length of an Operator mnemonic might be 4
// characters, meaning that all Operator fields should be 4 characters and
// padded with spaces as required.
type CartCoProcDisasmEntry struct {
	Location string
	Address  string
	Operator string
	Operand  string

	// total cycles for this instruction
	Cycles float32

	// basic notes about the last execution of the entry
	ExecutionNotes string

	// ExecutionNotes field may be updated
	UpdateNotes bool

	// some coprocessors will have more detailed information and the last
	// execution othe entry.
	//
	// the contents of this field may change every execution
	ExecutionDetails CartCoProcExecutionDetails
}

// CartCoProcExecutionDetails represents more specific information about an execution
// of a coprocessor disassembly entry. At it's minimum it should report back a
// text summary.
//
// When coprocessor specific details are required by a consumer of the
// interface, the specific coprocessor being used should be known ahead of
// time. The interface can then be cast to the concrete type.
type CartCoProcExecutionDetails interface {
	String() string
}

// CartCoProcExecutionSummary represents a summary of a coprocessor execution.
//
// When coprocessor specific details are required by a consumer of the
// interface, the specific coprocessor being used should be known ahead of
// time. The interface can then be cast to the concrete type.
type CartCoProcExecutionSummary interface {
	String() string
}

func (e CartCoProcDisasmEntry) String() string {
	return fmt.Sprintf("%s %s %s (%.0f)", e.Address, e.Operator, e.Operand, e.Cycles)
}

// CartCoProcDisassembler defines the functions that must be defined for a
// disassembler to be attached to a coprocessor.
type CartCoProcDisassembler interface {
	// Start is called at the beginning of coprocessor program execution.
	Start()

	// Step called after every instruction in the coprocessor program.
	Step(CartCoProcDisasmEntry)

	// End is called when coprocessor program has finished.
	End(CartCoProcExecutionSummary)
}

// CartCoProcBus is implemented by cartridge mappers that have a coprocessor that
// functions independently from the VCS.
type CartCoProcBus interface {
	CoProcID() string
	SetDisassembler(CartCoProcDisassembler)
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
func (c *CartCoProcDisassemblerStdout) End(s CartCoProcExecutionSummary) {
	fmt.Println(s)
}
