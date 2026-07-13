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

package arm

import (
	"strings"
)

type cycleOrder struct {
	queue [20]cycleType
	idx   int
}

func (q cycleOrder) String() string {
	s := strings.Builder{}
	for i := 0; i < q.idx; i++ {
		s.WriteRune(rune(q.queue[i]))
		s.WriteRune('+')
	}
	return strings.TrimRight(s.String(), "+")
}

func (q *cycleOrder) reset() {
	q.idx = 0
}

func (q *cycleOrder) add(c cycleType) {
	q.queue[q.idx] = c
	q.idx++
}

// BranchTrail indicates how the BrainTrail buffer was used for a cycle.
type BranchTrail int

// List of valid BranchTrail values.
const (
	BranchTrailNotUsed BranchTrail = iota
	BranchTrailUsed
	BranchTrailFlushed
)

// the bus activity during a cycle.
type busAccess int

const (
	prefetch busAccess = iota
	branch
	dataRead
	dataWrite
)

// is bus access an instruction or data read. equivalent in ARM terms, to
// asking if Prot0 is 0 or 1.
func (bt busAccess) isDataAccess() bool {
	return bt == dataRead || bt == dataWrite
}

// the type of cycle being executed.
type cycleType rune

const (
	N cycleType = 'N'
	I cycleType = 'I'
	S cycleType = 'S'
)

// called whenever PC changes unexpectedly (by a branch instruction for example).
func (arm *ARM) fillPipelineAfterBranch() {
	arm.Ncycle(branch, arm.state.registers[rPC])
	arm.Scycle(prefetch, arm.state.registers[rPC]+2)
}

// the cycle profile for store register type instructions is funky enough to
// need a specialist function.
func (arm *ARM) storeRegisterCycles(addr uint32) {
	arm.Ncycle(dataWrite, addr)
	arm.state.prefetchCycle = N
}

// add cycles accumulated during an BX to ARM code instruction. this is
// definitely only an estimate.
func (arm *ARM) armInterruptCycles(i ARMinterruptReturn) {
	// not taking into account latency of memory access
	arm.state.stretchedCycles += float32(i.NumMemAccess)
	arm.state.stretchedCycles += float32(i.NumAdditionalCycles)
}

// stub function for when the execution doesn't require cycle counting

func (arm *ARM) iCycle_Stub() {
}

func (arm *ARM) sCycle_Stub(bus busAccess, addr uint32) {
}

func (arm *ARM) nCycle_Stub(bus busAccess, addr uint32) {
}
