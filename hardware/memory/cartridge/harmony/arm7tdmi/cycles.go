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
	"fmt"
	"strings"
)

// ExecutionDetails implements CartCoProcExecutionDetails interface.
type ExecutionDetails struct {
	N           int
	I           int
	S           int
	MAMCR       int
	BranchTrail BranchTrail
	MergedIS    bool
}

func (es ExecutionDetails) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("N: %d\n", es.N))
	s.WriteString(fmt.Sprintf("I: %d\n", es.I))
	s.WriteString(fmt.Sprintf("S: %d\n", es.S))
	return s.String()
}

// BranchTrail indicates how the BrainTrail buffer was used for a cycle.
type BranchTrail int

// List of valid BranchTrail values.
const (
	BranchTrailNotUsed BranchTrail = iota
	BranchTrailUsed
	BranchTrailFlushed
)

// the bus activity during a cycle
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

// the type of cycle being executed
type cycleType int

const (
	I cycleType = iota
	S
	N
)

type cycleEvent struct {
	cycle cycleType

	// bus and addr are undefined if cycle is I
	bus  busAccess
	addr uint32
}

// returns false if address isn't latched. this means theat the bus access is
// subject to latency.
//
// dows not handle the decision about whether the MAM latches should be
// checked. for example, if MAMCR is zero than don't call this function at all.
// see Scycle() and Ncycle() for those decisions.
func (arm *ARM) isLatched(bus busAccess, addr uint32) bool {
	addr &= 0xffffff80

	switch bus {
	case prefetch:
		if addr == arm.mam.prefetchAddress {
			return true
		}
		arm.mam.prefetchAddress = addr

		// From UM10161, page 16:
		//
		// "Timing of Flash read operations is programmable and is described
		// later in this section. In this manner, there is no code fetch
		// penalty for sequential instruction execution when the CPU clock
		// period is greater than or equal to one fourth of the Flash access
		// time."
		if arm.mam.mamcr == 1 && arm.mam.mamtim >= 4 {
			arm.mam.prefetchAddress = addr
			return true
		}

	case branch:
		if addr == arm.mam.lastBranchAddress {
			arm.branchTrail = BranchTrailUsed
			return true
		}
		arm.mam.lastBranchAddress = addr
		arm.branchTrail = BranchTrailFlushed

	case dataRead:
		if addr == arm.mam.dataAddress {
			return true
		}
		arm.mam.dataAddress = addr
	}

	return false
}

func (arm *ARM) Icycle() {
	arm.I++
	arm.cycles++
	arm.prevCycles[1] = arm.prevCycles[0]
	arm.prevCycles[0] = cycleEvent{cycle: I}
}

func (arm *ARM) Scycle(bus busAccess, addr uint32) {
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
	if arm.prevCycles[1].cycle == I && arm.prevCycles[0].cycle == S {
		arm.cycles--
		arm.mergedIS = true
	}

	arm.S++
	arm.prevCycles[1] = arm.prevCycles[0]
	arm.prevCycles[0] = cycleEvent{cycle: S, bus: bus, addr: addr}

	if !arm.mmap.isFlash(addr) {
		arm.cycles++
		return
	}

	switch arm.mam.mamcr {
	default:
		arm.cycles += clklenFlash
	case 0:
		arm.cycles += clklenFlash
	case 1:
		// for MAM-1, we go to flash memory only if it's a program access (ie. not a data access)
		if bus.isDataAccess() {
			arm.cycles += clklenFlash
		} else if arm.isLatched(bus, addr) {
			arm.cycles++
		} else {
			arm.cycles += clklenFlash
		}
	case 2:
		if arm.isLatched(bus, addr) {
			arm.cycles++
		} else {
			arm.cycles += clklenFlash
		}
	}
}

func (arm *ARM) Ncycle(bus busAccess, addr uint32) {
	arm.N++
	arm.prevCycles[1] = arm.prevCycles[0]
	arm.prevCycles[0] = cycleEvent{cycle: N, bus: bus, addr: addr}

	if !arm.mmap.isFlash(addr) {
		arm.cycles++
		return
	}

	switch arm.mam.mamcr {
	default:
		arm.cycles += clklenFlash
	case 0:
		arm.cycles += clklenFlash
	case 1:
		arm.cycles += clklenFlash
	case 2:
		if arm.isLatched(bus, addr) {
			arm.cycles++
		} else {
			arm.cycles += clklenFlash
		}
	}
}

func (arm *ARM) pcCycle() {
	// assume PC change is in the same memory area
	arm.Ncycle(prefetch, arm.registers[rPC])
	arm.Scycle(prefetch, arm.registers[rPC])
}

// the cycle profile for store register type instructions is funky enough to
// need a specialist function
func (arm *ARM) storeRegisterCycles(addr uint32) {
	// "3.3.1 Nonsequential cycles" in "ARM7TDMI-S Technical Reference Manual r4p3"
	//
	// "The ARM7TDMI-S processor can perform back to back nonsequential memory cycles.
	// This happens, for example, when an STR instruction is executed, as shown in Figure 3-3.
	// If you are designing a memory controller for the ARM7TDMI-S processor, and your
	// memory system is unable to cope with this case, you must use the CLKEN signal to
	// extend the bus cycle to allow sufficient cycles for the memory system."
	//
	// in practice, I've reasoned that this means that the next prefetch is an
	// N cycle rather than the normal S cycle. meaning that there's a regular N
	// cycle followed by an N cycle prefetch
	arm.Ncycle(dataWrite, addr)
	arm.prefetchCycle = N
}
