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
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/callfn"
	"github.com/jetsetilly/gopher2600/random"
)

type State struct {
	registers Registers

	// frac fetcher count is *sometimes* reset when the low byte is set.
	// depending on the specific version of the DPC+ driver being used
	//
	// the value is set depending on the content of the first 3k of the DPC+
	// file. the first 3k is the where the Harmony driver resides
	//
	// use setDriverSpecificOptions() to set according to the driver
	resetFracFetcherCounterWhenLowFieldIsSet bool

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

	// most recent yield from the coprocessor
	yield coprocessor.CoProcYield
}

func newDPCPlusState() *State {
	s := &State{}
	s.parameters = make([]uint8, 0, 32)
	return s
}

func (s *State) initialise(version mmap, rand *random.Random) {
	s.registers.reset(version, rand)
	s.lda = false
	s.beats = 0
	s.parameters = []uint8{}
}

func (s *State) Snapshot() *State {
	n := *s
	n.static = s.static.Snapshot()
	n.parameters = make([]uint8, len(s.parameters))
	copy(n.parameters, s.parameters)
	return &n
}

// set options specific to harmony version. md5sum should be a hash of the first
// 3k of the binary file
//
// returns false if the driver is not recognised
func (s *State) setDriverSpecificOptions(md5sum string) bool {
	var knownDriverMD5 = map[string]bool{
		"17884ec14f9b1d06fe8d617a1fbdcf47": false,
		"5f80b5a5adbe483addc3f6e6f1b472f8": true,
		"8dd73b44fd11c488326ce507cbeb19d1": true,
		"b328dbdf787400c0f0e2b88b425872a5": false,
	}

	if v, ok := knownDriverMD5[md5sum]; ok {
		s.resetFracFetcherCounterWhenLowFieldIsSet = v
		return true
	}

	return false
}
