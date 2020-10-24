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

package harmony

import "github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"

type dpcPlusState struct {
	registers DPCplusRegisters

	// was the last instruction read the opcode for "lda <immediate>"
	lda bool

	// music fetchers are clocked at a fixed (slower) rate than the reference
	// to the VCS's clock. see Step() function.
	beats int
}

func newDPCPlusState() *dpcPlusState {
	return &dpcPlusState{}
}

func (s *dpcPlusState) Snapshot() mapper.CartSnapshot {
	n := *s
	return &n
}
