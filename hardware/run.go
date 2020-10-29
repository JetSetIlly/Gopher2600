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

// Run sets the emulation running as quickly as possible. continuteCheck()
// should return false when an external event (eg. a GUI event) indicates that
// the emulation should stop.
func (vcs *VCS) Run(continueCheck func() (bool, error)) error {
	var err error

	if continueCheck == nil {
		continueCheck = func() (bool, error) { return true, nil }
	}

	// see the equivalient videoCycle() in the VCS.Step() function for an
	// explanation for what's going on here:
	videoCycle := func() error {
		if err := vcs.RIOT.Ports.GetPlayback(); err != nil {
			return err
		}

		vcs.CPU.RdyFlg, err = vcs.TIA.Step(false)
		if err != nil {
			return err
		}

		vcs.CPU.RdyFlg, err = vcs.TIA.Step(false)
		if err != nil {
			return err
		}

		vcs.CPU.RdyFlg, err = vcs.TIA.Step(true)
		if err != nil {
			return err
		}

		vcs.RIOT.Step()
		vcs.Mem.Cart.Step()

		return nil
	}

	cont := true
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

		cont, err = continueCheck()
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
