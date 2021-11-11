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
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/emulation"
)

// While the continueCheck() function only runs at the end of a CPU instruction
// (unlike the corresponding function in VCS.Step() which runs every video
// cycle), it can still be expensive to do a full continue check every time.
//
// It depends on context whether it is used or not but the PerformanceBrake is
// a standard value that can be used to filter out expensive code paths within
// a continueCheck() implementation. For example:
//
//		performanceFilter++
//		if performanceFilter >= hardware.PerfomrmanceBrake {
//			performanceFilter = 0
//			if end_condition == true {
//				return emulation.Ending, nill
//			}
//		}
//		return emulation.Running, nill
//
const PerformanceBrake = 100

// Run sets the emulation running as quickly as possible. continuteCheck()
// should return false when an external event (eg. a GUI event) indicates that
// the emulation should stop.
func (vcs *VCS) Run(continueCheck func() (emulation.State, error)) error {
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

	var err error

	state := emulation.Running

	for state != emulation.Ending {
		switch state {
		case emulation.Running:
			err := vcs.CPU.ExecuteInstruction(videoCycle)
			if err != nil {
				return err
			}
		case emulation.Paused:
		default:
			return curated.Errorf("vcs: unsupported emulation state (%d) in Run() function", state)
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
