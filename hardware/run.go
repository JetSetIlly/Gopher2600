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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package hardware

import "github.com/jetsetilly/gopher2600/television"

// Run sets the emulation running as quickly as possible. continuteCheck()
// should return false when an external event (eg. a GUI event) indicates that
// the emulation should stop.
func (vcs *VCS) Run(continueCheck func() (bool, error)) error {
	var err error

	continueCheck()

	if continueCheck == nil {
		continueCheck = func() (bool, error) { return true, nil }
	}

	// see the equivalient videoCycle() in the VCS.Step() function for an
	// explanation for what's going on here:
	videoCycle := func() error {
		if err := vcs.checkDeviceInput(); err != nil {
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
			return err
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

	fn, err := vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}

	targetFrame := fn + numFrames

	cont := true
	for fn != targetFrame && cont {
		err = vcs.Step(nil)
		if err != nil {
			return err
		}
		fn, err = vcs.TV.GetState(television.ReqFramenum)
		if err != nil {
			return err
		}

		cont, err = continueCheck(fn)
		if err != nil {
			return err
		}
	}

	return nil
}
