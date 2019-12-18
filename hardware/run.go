package hardware

import "gopher2600/television"

// Run sets the emulation running as quickly as possible.  eventHandler()
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

// RunForFrameCount sets emulator running for the specified number of frames
// - not used by the debugger because traps and steptraps are more flexible
// - useful for fps and regression tests
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
