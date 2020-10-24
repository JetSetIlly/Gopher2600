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

package supercharger

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"

type state struct {
	tape      tape
	registers Registers
	ram       [3][]uint8
}

func newState() *state {
	s := &state{}
	for i := range s.ram {
		s.ram[i] = make([]uint8, bankSize)
	}
	return s
}

// Snapshot implements the mapper.CartSnapshot interface.
func (s *state) Snapshot() mapper.CartSnapshot {
	n := *s
	n.tape = s.tape.snapshot()
	for i := range n.ram {
		n.ram[i] = make([]uint8, len(s.ram[i]))
		copy(n.ram[i], s.ram[i])
	}
	return &n
}
