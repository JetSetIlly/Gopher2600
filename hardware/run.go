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
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
)

// checking continue condition every Run iteration is too frequent. A modest
// brake on how often it is called improves and smooths out performance.
const continueCheckFreq = 100

// Run sets the emulation running as quickly as possible. continuteCheck()
// should return false when an external event (eg. a GUI event) indicates that
// the emulation should stop.
//
// Not suitable if continueCheck must run very frequently. If you need to check
// every CPU or every video cycle then the Step() function should be preferred.
func (vcs *VCS) Run(continueCheck func() error) error {
	var err error

	if continueCheck == nil {
		continueCheck = func() error { return nil }
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

	cont := true
	contCt := 0
	for cont {
		err = vcs.CPU.ExecuteInstruction(videoCycle)
		if err != nil {
			// see debugger.inputLoop() function for explanation
			if onTapeLoaded, ok := err.(supercharger.FastLoaded); ok {
				vcs.CPU.Interrupted = true
				vcs.CPU.LastResult.Final = true
				err = onTapeLoaded(vcs.CPU, vcs.Mem.RAM, vcs.RIOT.Timer)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// only call continue check every N iterations
		contCt++
		if contCt%continueCheckFreq == 0 {
			err = continueCheck()
			cont = err == nil
			contCt = 0
		}
	}

	return err
}

// RunForFrameCount sets emulator running for the specified number of frames.
// Useful for FPS and regression tests. Not used by the debugger because traps
// and steptraps are more flexible.
func (vcs *VCS) RunForFrameCount(numFrames int, continueCheck func(frame int) (bool, error)) error {
	if continueCheck == nil {
		continueCheck = func(frame int) (bool, error) { return true, nil }
	}

	frameNum := vcs.TV.GetState(signal.ReqFramenum)
	targetFrame := frameNum + numFrames

	cont := true
	for frameNum != targetFrame && cont {
		err := vcs.Step(nil)
		if err != nil {
			return err
		}

		frameNum = vcs.TV.GetState(signal.ReqFramenum)

		cont, err = continueCheck(frameNum)
		if err != nil {
			return err
		}
	}

	return nil
}
