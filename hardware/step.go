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

package hardware

func nullCallback(_ bool) error {
	return nil
}

// Step the emulator state one CPU instruction
func (vcs *VCS) Step(colorClockCallback func(isCycle bool) error) error {
	if colorClockCallback == nil {
		colorClockCallback = nullCallback
	}

	// the cycle function defines the order of operation for the rest of
	// the VCS for every CPU cycle. the function block represents the ϕ0 cycle
	//
	// the cpu calls the cycle function after every CPU cycle. this is a
	// bit backwards compared to the operation of a real VCS but I believe the
	// effect is the same:
	//
	// in the real machine, the pulse from the OSC color clock drives the TIA.
	// a pulse from this clock moves the state of the TIA forward one color
	// clock. each of the OSC pulses is fed through a div/3 circuit (ϕ0) the
	// output of which is attached to pin 26 of the TIA and to pin 20 of the
	// CPU. each pulse of ϕ0 drives the CPU forward one CPU cycle.
	//
	// in this emulation meanwhile, the CPU-TIA is reversed. each call to
	// Step() drives the CPU. After each CPU cycle the CPU emulation yields to
	// the cycle() function defined below.
	//
	// the reason for this inside-out arrangement is simply a consequence of
	// the how the CPU emulation is put together. it is easier for the large
	// CPU ExecuteInstruction() function to call out to the cycle()
	// function. if we were to do it the other way around then keeping track of
	// the interim CPU state becomes trickier.
	//
	// we could solve this by using go-channels but early experiments suggested
	// that this was too slow. a better solution would be to build the CPU
	// instructions out of smaller micro-instructions. this should make jumping
	// in and out of the CPU far easier.
	//
	// I don't believe any visual or audible artefacts of the VCS (undocumented
	// or not) rely on the details of the CPU-TIA relationship.
	//
	// at the end of the cycle() function the cycleCallback() function is called
	cycle := func() error {
		if err := vcs.Input.Handle(); err != nil {
			return err
		}

		vcs.TIA.QuickStep(1)
		if err := colorClockCallback(false); err != nil {
			return err
		}

		vcs.TIA.QuickStep(2)
		if err := colorClockCallback(false); err != nil {
			return err
		}

		if reg, ok := vcs.Mem.TIA.ChipHasChanged(); ok {
			vcs.TIA.Step(reg, 3)
		} else {
			vcs.TIA.QuickStep(3)
		}
		if reg, ok := vcs.Mem.RIOT.ChipHasChanged(); ok {
			vcs.RIOT.Step(reg)
		} else {
			vcs.RIOT.QuickStep()
		}

		vcs.Mem.Cart.Step(vcs.Clock)

		if err := colorClockCallback(true); err != nil {
			return err
		}

		return nil
	}

	err := vcs.CPU.ExecuteInstruction(cycle)
	if err != nil {
		return err
	}

	return nil
}
