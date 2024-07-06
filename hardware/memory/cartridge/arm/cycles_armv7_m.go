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

func (arm *ARM) iCycle_ARMv7_M() {
	// comments in cycles_arm7tdmi.go

	if arm.disasm != nil {
		arm.state.cycleOrder.add(I)
	}
	arm.state.stretchedCycles++
	arm.state.lastCycle = I
}

func (arm *ARM) sCycle_ARMv7_M(_ busAccess, addr uint32) {
	// comments in cycles_arm7tdmi.go

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

	arm.state.stretchedCycles += arm.clklenFlash
}

func (arm *ARM) nCycle_ARMv7_M(_ busAccess, addr uint32) {
	// comments in cycles_arm7tdmi.go

	mclkFlash := 1.0
	mclkNonFlash := 1.0

	if arm.state.lastCycle == N {
		mclkFlash = 1.3
		mclkNonFlash = 1.8
	}

	if arm.disasm != nil {
		arm.state.cycleOrder.add(N)
	}
	arm.state.lastCycle = N

	if !arm.mmap.IsFlash(addr) {
		arm.state.stretchedCycles += float32(mclkNonFlash)
		return
	}

	arm.state.stretchedCycles += arm.clklenFlash * float32(mclkFlash)
}
