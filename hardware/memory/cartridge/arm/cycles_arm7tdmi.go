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

func (arm *ARM) iCycle_ARM7TDMI() {
	if arm.disasm != nil {
		arm.state.cycleOrder.add(I)
	}
	arm.state.stretchedCycles++
	arm.state.lastCycle = I
	arm.state.mam.prefectchAborted = false
}

func (arm *ARM) sCycle_ARM7TDMI(bus busAccess, addr uint32) {
	arm.state.mam.prefectchAborted = bus.isDataAccess()

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
	if arm.state.lastCycle == I {
		arm.state.stretchedCycles--
		arm.state.mergedIS = true
	}

	if arm.disasm != nil {
		arm.state.cycleOrder.add(S)
	}
	arm.state.lastCycle = S

	if !arm.mmap.IsFlash(addr) {
		arm.state.stretchedCycles++
		return
	}

	switch arm.state.mam.mamcr {
	default:
		arm.state.stretchedCycles += arm.clklenFlash
	case 0:
		arm.state.stretchedCycles += arm.clklenFlash
	case 1:
		// for MAM-1, we go to flash memory only if it's a program access (ie. not a data access)
		if bus.isDataAccess() {
			arm.state.stretchedCycles += arm.clklenFlash
		} else if arm.isLatched(S, bus, addr) {
			arm.state.stretchedCycles++
		} else {
			arm.state.stretchedCycles += arm.clklenFlash
		}
	case 2:
		if arm.isLatched(S, bus, addr) {
			arm.state.stretchedCycles++
		} else {
			arm.state.stretchedCycles += arm.clklenFlash
		}
	}
}

func (arm *ARM) nCycle_ARM7TDMI(bus busAccess, addr uint32) {
	arm.state.mam.prefectchAborted = bus.isDataAccess()

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
	if arm.state.lastCycle == N {
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
		arm.state.cycleOrder.add(N)
	}
	arm.state.lastCycle = N

	if !arm.mmap.IsFlash(addr) {
		arm.state.stretchedCycles += float32(mclkNonFlash)
		return
	}

	switch arm.state.mam.mamcr {
	default:
		arm.state.stretchedCycles += arm.clklenFlash * float32(mclkFlash)
	case 0:
		arm.state.stretchedCycles += arm.clklenFlash * float32(mclkFlash)
	case 1:
		arm.state.stretchedCycles += arm.clklenFlash * float32(mclkFlash)
	case 2:
		if arm.isLatched(N, bus, addr) {
			arm.state.stretchedCycles += float32(mclkNonFlash)
		} else {
			arm.state.stretchedCycles += arm.clklenFlash * float32(mclkFlash)
		}
	}
}
