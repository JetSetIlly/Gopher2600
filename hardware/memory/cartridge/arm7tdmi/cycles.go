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

package arm7tdmi

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

func (q cycleOrder) len() int {
	return q.idx
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

// returns false if address isn't latched. this means theat the bus access is
// subject to latency.
//
// dows not handle the decision about whether the MAM latches should be
// checked. for example, if MAMCR is zero than don't call this function at all.
// see Scycle() and Ncycle() for those decisions.
func (arm *ARM) isLatched(cycle cycleType, bus busAccess, addr uint32) bool {
	latch := addr & 0xffffff80

	switch bus {
	case prefetch:
		if latch == arm.mam.prefectchLatch {
			return true
		}

		arm.mam.prefectchLatch = latch

		// we'll assume MAMTIM is set adequately
		if cycle == S && !arm.mam.prefectchAborted {
			return true
		}

	case branch:
		if latch == arm.mam.branchLatch {
			arm.branchTrail = BranchTrailUsed
			return true
		}
		arm.mam.branchLatch = latch
		arm.branchTrail = BranchTrailFlushed

	case dataRead:
		if latch == arm.mam.dataLatch {
			return true
		}
		arm.mam.dataLatch = latch

	case dataWrite:
		// invalidate data latch
		arm.mam.dataLatch = 0x0
	}

	return false
}

func (arm *ARM) iCycle() {
	if arm.disasm != nil {
		arm.cycleOrder.add(I)
	}
	arm.stretchedCycles++
	arm.lastCycle = I
	arm.mam.prefectchAborted = false
}

func (arm *ARM) sCycle(bus busAccess, addr uint32) {
	arm.mam.prefectchAborted = bus.isDataAccess()

	// "Merged I-S cycles
	// Where possible, the ARM7TDMI-S processor performs an optimization on the bus to
	// allow extra time for memory decode. When this happens, the address of the next
	// memory cycle is broadcast during an internal cycle on this bus. This enables the
	// memory controller to decode the address, but it must not initiate a memory access
	// during this cycle. In a merged I-S cycle, the next cycle is a sequential cycle to the same
	// memory location. This commits to the access, and the memory controller must initiate
	// the memory access. This is shown in Figure 3-5 on page 3-9."
	//
	// page 3-8 of the "ARM7TDMI-S Technical Reference Manual r4p3"
	if arm.lastCycle == I {
		arm.stretchedCycles--
		arm.mergedIS = true
	}

	if arm.disasm != nil {
		arm.cycleOrder.add(S)
	}
	arm.lastCycle = S

	if !arm.mmap.IsFlash(addr) {
		arm.stretchedCycles++
		return
	}

	switch arm.mam.mamcr {
	default:
		arm.stretchedCycles += clklenFlash
	case 0:
		arm.stretchedCycles += clklenFlash
	case 1:
		// for MAM-1, we go to flash memory only if it's a program access (ie. not a data access)
		if bus.isDataAccess() {
			arm.stretchedCycles += clklenFlash
		} else if arm.isLatched(S, bus, addr) {
			arm.stretchedCycles++
		} else {
			arm.stretchedCycles += clklenFlash
		}
	case 2:
		if arm.isLatched(S, bus, addr) {
			arm.stretchedCycles++
		} else {
			arm.stretchedCycles += clklenFlash
		}
	}
}

func (arm *ARM) nCycle(bus busAccess, addr uint32) {
	arm.mam.prefectchAborted = bus.isDataAccess()

	// "3.3.1 Nonsequential cycles" in "ARM7TDMI-S Technical Reference Manual r4p3"
	//
	// "It is not uncommon for a memory system to require a longer access time
	// (extending the clock cycle) for nonsequential accesses. This is to allow
	// time for full address decoding or to latch both a row and column address
	// into DRAM."
	mclkFlash := 1.0
	mclkNonFlash := 1.0

	// "3.3.1 Nonsequential cycles" in "ARM7TDMI-S Technical Reference Manual r4p3"
	//
	// "The ARM7TDMI-S processor can perform back to back nonsequential memory cycles.
	// This happens, for example, when an STR instruction is executed, as shown in Figure 3-3.
	// If you are designing a memory controller for the ARM7TDMI-S processor, and your
	// memory system is unable to cope with this case, you must use the CLKEN signal to
	// extend the bus cycle to allow sufficient cycles for the memory system."
	if arm.lastCycle == N {
		mclkFlash = 1.3
		mclkNonFlash = 1.8
	}

	// the use of a fractional number for MCLK modulation is at odds with the
	// stretching required for flash access, which is a whole number. again, it
	// isn't clear if this is possible but again, the technical reference
	// points to the possibility of a difference. to be specific, there are two
	// methods of stretching access times: MCLK modulation and the use of nWait
	// to control bus cycles.
	//
	// on page 3-29 of r4p1 (but not in the equivalent section of r4p3,
	// curiously), nWait is described as allowing bus cycles to be extended in
	// "increments of complete MCLK cycles". MLCK itself meanwhile, is
	// described as being free-running. while not conclusive, this to me
	// suggests the modulation can be fractional.

	if arm.disasm != nil {
		arm.cycleOrder.add(N)
	}
	arm.lastCycle = N

	if !arm.mmap.IsFlash(addr) {
		arm.stretchedCycles += float32(mclkNonFlash)
		return
	}

	switch arm.mam.mamcr {
	default:
		arm.stretchedCycles += clklenFlash * float32(mclkFlash)
	case 0:
		arm.stretchedCycles += clklenFlash * float32(mclkFlash)
	case 1:
		arm.stretchedCycles += clklenFlash * float32(mclkFlash)
	case 2:
		if arm.isLatched(N, bus, addr) {
			arm.stretchedCycles += float32(mclkNonFlash)
		} else {
			arm.stretchedCycles += clklenFlash * float32(mclkFlash)
		}
	}
}

// called whenever PC changes unexpectedly (by a branch instruction for example).
func (arm *ARM) fillPipeline() {
	arm.Ncycle(branch, arm.registers[rPC])
	arm.Scycle(prefetch, arm.registers[rPC]+2)
}

// the cycle profile for store register type instructions is funky enough to
// need a specialist function.
func (arm *ARM) storeRegisterCycles(addr uint32) {
	arm.Ncycle(dataWrite, addr)
	arm.prefetchCycle = N
}

// add cycles accumulated during an BX to ARM code instruction. this is
// definitely only an estimate.
func (arm *ARM) armInterruptCycles(i ARMinterruptReturn) {
	// we'll assume all writes are to flash memory
	arm.stretchedCycles += float32(i.NumMemAccess) * clklenFlash
	arm.stretchedCycles += float32(i.NumAdditionalCycles)
}

// stub function for when the execution doesn't require cycle counting

func (arm *ARM) iCycleStub() {
}

func (arm *ARM) sCycleStub(bus busAccess, addr uint32) {
}

func (arm *ARM) nCycleStub(bus busAccess, addr uint32) {
}
