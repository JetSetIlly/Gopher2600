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

type state struct {
	tape      tape
	registers Registers
	ram       [3][]uint8

	// is the tape currently in the process of being loaded
	isLoading bool
}

func newState() *state {
	s := &state{}
	for i := range s.ram {
		s.ram[i] = make([]uint8, bankSize)
	}
	return s
}

// Snapshot implements the mapper.CartMapper interface.
func (s *state) Snapshot() *state {
	n := *s
	n.tape = s.tape.snapshot()
	for i := range n.ram {
		n.ram[i] = make([]uint8, len(s.ram[i]))
		copy(n.ram[i], s.ram[i])
	}
	return &n
}
