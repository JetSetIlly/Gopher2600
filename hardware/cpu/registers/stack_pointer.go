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

package registers

// StackPointer is a special purpose Register. It can be treated as a register
// if required through the Register field.
type StackPointer struct {
	Data
}

// NewStackPointer creates a new stack pointer register.
func NewStackPointer(val uint8) StackPointer {
	return StackPointer{
		Data: Data{
			value: val,
			label: "SP",
		},
	}
}

// The stack is hardwired to page one addresses. Note that the VCS stack
// actually appears in page zero but this is a consequence of how the memory
// bus is wired.
func (sp StackPointer) Address() uint16 {
	return 0x0100 | uint16(sp.value)
}
