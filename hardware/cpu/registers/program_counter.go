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

import "fmt"

// ProgramCounter represents the PC register in the 6507 CPU.
type ProgramCounter struct {
	value uint16
}

// NewProgramCounter is the preferred method of initialisation for ProgramCounter.
func NewProgramCounter(val uint16) ProgramCounter {
	return ProgramCounter{value: val}
}

// Label returns the program counter label (or ID).
func (pc ProgramCounter) Label() string {
	return "PC"
}

func (pc ProgramCounter) String() string {
	return fmt.Sprintf("%04x", pc.value)
}

// Value returns the current value of the register.
func (pc ProgramCounter) Value() uint16 {
	return pc.value
}

// BitWidth returns the number of bits used to store the program counter value.
func (pc ProgramCounter) BitWidth() int {
	return 16
}

// Address returns the current value of the PC as a a value of type uint16.
func (pc ProgramCounter) Address() uint16 {
	return pc.value
}

// Load a value into the PC.
func (pc *ProgramCounter) Load(val uint16) {
	pc.value = val
}

// Add a value to the PC.
func (pc *ProgramCounter) Add(val uint16) (carry, overflow bool) {
	v := pc.value
	pc.value += val
	return pc.value < v, false
}
