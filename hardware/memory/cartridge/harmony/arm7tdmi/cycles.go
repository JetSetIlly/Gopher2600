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
	MergedN     bool
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
type busType int

const (
	prefetch busType = iota
	branch
	data
	write
)

// the type of cycle being executed
type cycleType int

const (
	I cycleType = iota
	S
	N
)

func (arm *ARM) mamBuffer(bus busType, addr uint32) bool {
	addr &= 0xffffff80

	switch bus {
	case prefetch:
		if addr != arm.mam.prefetchAddress {
			arm.mam.prefetchAddress = addr
			return false
		}
	case branch:
		if addr != arm.mam.lastBranchAddress {
			arm.branchTrail = BranchTrailFlushed
			arm.mam.lastBranchAddress = addr
			return false
		}
		arm.branchTrail = BranchTrailUsed
	case data:
		if addr != arm.mam.dataAddress {
			arm.mam.dataAddress = addr
			return false
		}
	}

	return true
}

func (arm *ARM) Icycle() {
	arm.I++
	arm.cycles++
	arm.prevCycles[1] = arm.prevCycles[0]
	arm.prevCycles[0] = I
}

func (arm *ARM) Scycle(bus busType, addr uint32) {
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
	if arm.prevCycles[1] == I {
		arm.cycles--
		arm.mergedIS = true
	}

	arm.S++
	arm.prevCycles[1] = arm.prevCycles[0]
	arm.prevCycles[0] = S

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
		// for MAM-1, we go to flash memory only if it's a program acess. which
		// means busType is "fetch" or "branch" (and not "write" or "data")
		if bus == write || bus == data {
			arm.cycles += clklenFlash
		} else {
			if arm.mamBuffer(bus, addr) {
				arm.cycles++
			} else {
				arm.cycles += clklenFlash
			}
		}
	case 2:
		if arm.mamBuffer(bus, addr) {
			arm.cycles++
		} else {
			arm.cycles += clklenFlash
		}
	}
}

func (arm *ARM) Ncycle(bus busType, addr uint32) {
	arm.N++
	arm.prevCycles[1] = arm.prevCycles[0]
	arm.prevCycles[0] = N

	if !arm.mam.mmap.isFlash(addr) {
		arm.cycles++
		return
	}

	switch arm.mam.mamcr {
	default:
		arm.cycles += clklenFlash
	case 0:
		arm.cycles += clklenFlash
	case 1:
		// for MAM-1, we always go to flash memory regardless of busType
		arm.cycles += clklenFlash
	case 2:
		if arm.mamBuffer(bus, addr) && bus != write {
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

func (arm *ARM) storeRegNCycle(addr uint32) {
	// "3.3.1 Nonsequential cycles" in "ARM7TDMI-S Technical Reference Manual r4p3"
	//
	// "The ARM7TDMI-S processor can perform back to back nonsequential memory cycles.
	// This happens, for example, when an STR instruction is executed, as shown in Figure 3-3.
	// If you are designing a memory controller for the ARM7TDMI-S processor, and your
	// memory system is unable to cope with this case, you must use the CLKEN signal to
	// extend the bus cycle to allow sufficient cycles for the memory system."
	//
	// How this actually works however is a matter of debate. But assuming the
	// N cycle is *always* merged seems to work out okay in all MAM/code-optimisation
	// combinations.
	arm.Ncycle(write, addr)
	arm.prefetchCycle = N
	arm.mergedN = true
}
