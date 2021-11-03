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

package dpcplus

import (
	"math/rand"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/harmony/callfn"
)

type State struct {
	registers Registers

	// currently selected bank
	bank int

	// was the last instruction read the opcode for "lda <immediate>"
	lda bool

	// music fetchers are clocked at a fixed (slower) rate than the reference
	// to the VCS's clock. see Step() function.
	beats int

	// parameters for next function call
	parameters []uint8

	// static area of the cartridge. accessible outside of the cartridge
	// through GetStatic() and PutStatic()
	static *Static

	// the callfn process is stateful
	callfn callfn.CallFn

	// the MAM mode to use when called arm7tdmi.Run(). Run() returns the value
	// the thumb code leaves it in so this can carry over between program
	// executions
	mamcr uint32
}

func newDPCPlusState() *State {
	s := &State{}
	s.parameters = make([]uint8, 0, 32)
	return s
}

// the ideal MAMCR value given to the arm7tdmi.Run() command.
const dpcPlusMAMCR = 2

func (s *State) initialise(randSrc *rand.Rand, bank int) {
	s.registers.reset(randSrc)
	s.bank = bank
	s.lda = false
	s.beats = 0
	s.parameters = []uint8{}

	// on reset the DPC+ driver is in mode 2
	s.mamcr = dpcPlusMAMCR
}

func (s *State) Snapshot() *State {
	n := *s
	n.static = s.static.Snapshot()
	n.parameters = make([]uint8, len(s.parameters))
	copy(n.parameters, s.parameters)
	return &n
}
