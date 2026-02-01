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

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/debugger/govern"
)

// While the continueCheck() function only runs at the end of a CPU instruction
// (unlike the corresponding function in VCS.Step() which runs every color clock),
// it can still be expensive to do a full continue check every time.
//
// It depends on context whether it is used or not but the PerformanceBrake is
// a standard value that can be used to filter out expensive code paths within
// a continueCheck() implementation. For example:
//
//	performanceFilter++
//	if performanceFilter >= hardware.PerfomrmanceBrake {
//		performanceFilter = 0
//		if end_condition == true {
//			return govern.Ending, nill
//		}
//	}
//	return govern.Running, nill
const PerformanceBrake = 100

// Run sets the emulation running as quickly as possible
func (vcs *VCS) Run(continueCheck func() (govern.State, error)) error {
	if continueCheck == nil {
		continueCheck = func() (govern.State, error) { return govern.Running, nil }
	}

	// see the equivalient colorClock() in the VCS.Step() function for an
	// explanation for what's going on here:
	colorClock := func() error {
		if err := vcs.Input.Handle(); err != nil {
			return err
		}

		if reg, ok := vcs.Mem.TIA.ChipHasChanged(); ok {
			vcs.TIA.QuickStep(1)
			vcs.TIA.QuickStep(2)
			vcs.TIA.Step(reg, 3)
			vcs.RIOT.QuickStep()
		} else {
			vcs.TIA.QuickStep(1)
			vcs.TIA.QuickStep(2)
			vcs.TIA.QuickStep(3)

			if reg, ok := vcs.Mem.RIOT.ChipHasChanged(); ok {
				vcs.RIOT.Step(reg)
			} else {
				vcs.RIOT.QuickStep()
			}
		}

		vcs.Mem.Cart.Step(vcs.Clock)

		return nil
	}

	var err error

	state := govern.Running

	for state != govern.Ending && state != govern.Initialising {
		switch state {
		case govern.Running:
			err := vcs.CPU.ExecuteInstruction(colorClock)
			if err != nil {
				return err
			}
		case govern.Paused:
		default:
			return fmt.Errorf("vcs: unsupported emulation state (%s) in Run() function", state)
		}

		state, err = continueCheck()
		if err != nil {
			return err
		}
	}

	return nil
}

// RunForFrameCount sets emulator running for the specified number of frames.
// Useful for FPS and regression tests. Not used by the debugger because traps
// (and volatile traps) are more flexible.
//
// continueCheck can be used to instruct the emulation to end before the
// specified number of frames has elapsed.
func (vcs *VCS) RunForFrameCount(numFrames int, continueCheck func() (govern.State, error)) error {
	if continueCheck == nil {
		continueCheck = func() (govern.State, error) { return govern.Running, nil }
	}

	targetFrame := vcs.TV.GetCoords().Frame + numFrames

	state := govern.Running
	for state != govern.Ending {
		// check if CPU has been jammed. emulation will run forever if we don't check for this
		if vcs.CPU.Jammed {
			return nil
		}

		if vcs.TV.IsFrameNum(targetFrame) {
			state = govern.Ending
		} else {
			err := vcs.Step(nil)
			if err != nil {
				return err
			}

			state, err = continueCheck()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
