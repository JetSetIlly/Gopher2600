package hardware

import (
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/peripherals"
	"gopher2600/hardware/riot"
	"gopher2600/hardware/tia"
	"gopher2600/television"
)

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	CPU  *cpu.CPU
	Mem  *memory.VCSMemory
	TIA  *tia.TIA
	RIOT *riot.RIOT

	// tv is not part of the VCS but is attached to it
	TV television.Television

	Panel *peripherals.Panel
	Ports *peripherals.Ports
}

// NewVCS creates a new VCS and everything associated with the hardware. It is
// used for all aspects of emulation: debugging sessions, and regular play
func NewVCS(tv television.Television) (*VCS, error) {
	var err error

	vcs := &VCS{TV: tv}

	vcs.Mem, err = memory.NewVCSMemory()
	if err != nil {
		return nil, err
	}

	vcs.CPU, err = cpu.NewCPU(vcs.Mem)
	if err != nil {
		return nil, err
	}

	vcs.TIA = tia.NewTIA(vcs.TV, vcs.Mem.TIA)
	if vcs.TIA == nil {
		return nil, errors.NewFormattedError(errors.VCSError, "can't create TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return nil, errors.NewFormattedError(errors.VCSError, "can't create RIOT")
	}

	vcs.Panel = peripherals.NewPanel(vcs.Mem.RIOT)
	if vcs.Panel == nil {
		return nil, errors.NewFormattedError(errors.VCSError, "can't create control panel")
	}

	vcs.Ports = peripherals.NewPorts(vcs.Mem.RIOT, vcs.Mem.TIA, vcs.Panel)
	if vcs.Ports == nil {
		return nil, errors.NewFormattedError(errors.VCSError, "can't create player ports")
	}

	return vcs, nil
}

// AttachCartridge loads a cartridge (given by filename) into the emulators memory
func (vcs *VCS) AttachCartridge(filename string) error {
	if filename == "" {
		vcs.Mem.Cart.Eject()
	} else {
		err := vcs.Mem.Cart.Attach(filename)
		if err != nil {
			return err
		}
	}
	err := vcs.Reset()
	if err != nil {
		return err
	}

	return nil
}

// Reset emulates the reset switch on the console panel
//  - reset the CPU
//  - destroy and create the TIA and RIOT
//  - load reset address into the PC
func (vcs *VCS) Reset() error {
	if err := vcs.CPU.Reset(); err != nil {
		return err
	}

	// !!TODO: consider implementing tia.Reset and riot.Reset instead of
	// recreating the two components

	vcs.TIA = tia.NewTIA(vcs.TV, vcs.Mem.TIA)
	if vcs.TIA == nil {
		return errors.NewFormattedError(errors.VCSError, "can't create TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return errors.NewFormattedError(errors.VCSError, "can't create RIOT")
	}

	err := vcs.CPU.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return err
	}

	return nil
}

func (vcs *VCS) strobeUserInput() error {
	var err error
	if vcs.Ports.Player0 != nil {
		err = vcs.Ports.Player0.Strobe()
		if err != nil {
			return err
		}
	}
	if vcs.Ports.Player1 != nil {
		err = vcs.Ports.Player1.Strobe()
		if err != nil {
			return err
		}
	}

	return vcs.Panel.Strobe()
}

// Step the emulator state one CPU instruction. we can put this function in a
// loop for an effective debugging loop ths videoCycleCallback function for an
// additional callback point in the debugger.
func (vcs *VCS) Step(videoCycleCallback func(*result.Instruction) error) (*result.Instruction, error) {
	var r *result.Instruction
	var err error

	// the cpu calls the videoCycle function after every CPU cycle. the
	// videoCycle function defines the order of operation for the rest of the
	// VCS for every CPU cycle.
	//
	// this block represents the Q0 cycle
	//
	// !!TODO: the following would be a good test case for the proposed try()
	// function, coming in a future language version
	videoCycle := func(r *result.Instruction) error {
		// ensure controllers have updated their input
		if err := vcs.strobeUserInput(); err != nil {
			return err
		}
		// update RIOT memory and step
		//
		vcs.RIOT.ReadMemory()
		vcs.RIOT.Step()

		// three color clocks per CPU cycle so we run video cycle three times.
		// step one ...
		vcs.CPU.RdyFlg, err = vcs.TIA.Step()
		if err != nil {
			return err
		}
		_ = videoCycleCallback(r)

		// update TIA from memory. from "TIA 1A" document:
		//
		// "if the read-write line is low, the data [...] will be writted in
		// the addressed write location when the Q2 clock goes from high to
		// low."
		//
		// from my understanding, we can say that this always happens after the
		// first TIA step and before the second.
		vcs.TIA.ReadMemory()

		// ... tia step two ...
		vcs.CPU.RdyFlg, err = vcs.TIA.Step()
		if err != nil {
			return err
		}
		_ = videoCycleCallback(r)

		// ... tia step three
		vcs.CPU.RdyFlg, err = vcs.TIA.Step()
		if err != nil {
			return err
		}
		_ = videoCycleCallback(r)

		// also from the "TIA 1A" document:
		//
		// "If the read-write line is high, the addressed location can be read
		// by the microprocessor..."
		//
		// we don't need to do anything here. any writes that have happened are
		// sitting in memory ready for the CPU.

		return nil
	}

	r, err = vcs.CPU.ExecuteInstruction(videoCycle)
	if err != nil {
		return nil, err
	}

	// CPU has been left in the unready state - continue cycling the VCS hardware
	// until the CPU is ready
	for !vcs.CPU.RdyFlg {
		_ = videoCycle(r)
	}

	return r, nil
}

// Run sets the emulation running as quickly as possible.  eventHandler()
// should return false when an external event (eg. a GUI event) indicates that
// the emulation should stop.
func (vcs *VCS) Run(continueCheck func() (bool, error)) error {
	var err error

	videoCycle := func(r *result.Instruction) error {
		// see videoCycle in Step() function for an explanation for what's
		// going on here
		if err := vcs.strobeUserInput(); err != nil {
			return err
		}
		_, _ = vcs.TIA.Step()
		vcs.TIA.ReadMemory()
		vcs.RIOT.ReadMemory()
		vcs.RIOT.Step()
		_, _ = vcs.TIA.Step()
		vcs.CPU.RdyFlg, err = vcs.TIA.Step()
		return err
	}

	cont := true
	for cont {
		_, err = vcs.CPU.ExecuteInstruction(videoCycle)
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
// - callback is a simple hook called every video step
func (vcs *VCS) RunForFrameCount(numFrames int, callback func()) error {
	fn, err := vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}

	targetFrame := fn + numFrames

	for fn != targetFrame {
		callback()
		_, err = vcs.Step(func(_ *result.Instruction) error { return nil })
		if err != nil {
			return err
		}
		fn, err = vcs.TV.GetState(television.ReqFramenum)
		if err != nil {
			return err
		}
	}

	return nil
}
