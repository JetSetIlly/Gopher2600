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
	"github.com/jetsetilly/gopher2600/emulation"
)

// A convenient value to use as the checkFreq parameter of the Run() function.
// Can give a modest improvement to performance.
const PerformanceBrake = 100

// Run sets the emulation running as quickly as possible. continuteCheck()
// should return false when an external event (eg. a GUI event) indicates that
// the emulation should stop.
//
// The frequency of the check is given by the checkFreq parameter.
func (vcs *VCS) Run(continueCheck func() (emulation.State, error), checkFreq int) error {
	if continueCheck == nil {
		continueCheck = func() (emulation.State, error) { return emulation.Running, nil }
	}

	// see the equivalient videoCycle() in the VCS.Step() function for an
	// explanation for what's going on here:
	videoCycle := func() error {
		if err := vcs.RIOT.Ports.GetPlayback(); err != nil {
			return err
		}

		vcs.TIA.Step(false)
		vcs.TIA.Step(false)
		vcs.TIA.Step(true)
		vcs.RIOT.Step()
		vcs.Mem.Cart.Step(vcs.Clock)

		return nil
	}

	state := emulation.Running
	checkCt := 0
	for state != emulation.Ending {
		if state == emulation.Running {
			err := vcs.CPU.ExecuteInstruction(videoCycle)
			if err != nil {
				return err
			}
		} else {
			// paused
		}

		// only call continue check every N iterations
		checkCt++
		if checkCt >= checkFreq {
			var err error
			state, err = continueCheck()
			if err != nil {
				return err
			}
			checkCt = 0
		}
	}

	return nil
}

// RunForFrameCount sets emulator running for the specified number of frames.
// Useful for FPS and regression tests. Not used by the debugger because traps
// (and volatile traps) are more flexible.
func (vcs *VCS) RunForFrameCount(numFrames int, continueCheck func(frame int) (emulation.State, error)) error {
	if continueCheck == nil {
		continueCheck = func(frame int) (emulation.State, error) { return emulation.Running, nil }
	}

	frameNum := vcs.TV.GetCoords().Frame
	targetFrame := frameNum + numFrames

	state := emulation.Running
	for frameNum != targetFrame && state != emulation.Ending {
		err := vcs.Step(nil)
		if err != nil {
			return err
		}

		frameNum = vcs.TV.GetCoords().Frame

		state, err = continueCheck(frameNum)
		if err != nil {
			return err
		}
	}

	return nil
}
