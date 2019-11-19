package hardware

func nullVideoCycleCallback() error {
	return nil
}

// Step the emulator state one CPU instruction. we can put this function in a
// loop for an effective debugging loop ths videoCycleCallback function for an
// additional callback point in the debugger.
func (vcs *VCS) Step(videoCycleCallback func() error) error {
	if videoCycleCallback == nil {
		videoCycleCallback = nullVideoCycleCallback
	}

	var err error

	// the videoCycle function defines the order of operation for the rest of
	// the VCS for every CPU cycle. the function block represents the ϕ0 cycle
	//
	// the cpu calls the videoCycle function after every CPU cycle. this is a
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
	// the videoCycle() function defined below.
	//
	// the reason for this inside-out arrangement is simply a consequence of
	// the how the CPU emulation is put together. it is easier for the large
	// CPU ExecuteInstruction() function to call out to the videoCycle()
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
	videoCycle := func() error {
		// ensure controllers have updated their input
		if err := vcs.strobeUserInput(); err != nil {
			return err
		}

		// one
		_, err = vcs.TIA.Step(false)
		if err != nil {
			return err
		}

		err = videoCycleCallback()
		if err != nil {
			return err
		}

		// two
		_, err = vcs.TIA.Step(false)
		if err != nil {
			return err
		}

		err = videoCycleCallback()
		if err != nil {
			return err
		}

		// three
		vcs.CPU.RdyFlg, err = vcs.TIA.Step(true)
		if err != nil {
			return err
		}
		err = videoCycleCallback()
		if err != nil {
			return err
		}

		vcs.RIOT.Step()

		return nil
	}

	err = vcs.CPU.ExecuteInstruction(videoCycle)
	if err != nil {
		return err
	}

	// CPU has been left in the unready state - continue cycling the VCS hardware
	// until the CPU is ready
	for !vcs.CPU.RdyFlg {
		_ = videoCycle()
	}

	return nil
}
