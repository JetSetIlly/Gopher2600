package hardware

import (
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/peripherals"
	"gopher2600/hardware/riot"
	"gopher2600/hardware/tia"
	"gopher2600/television"
)

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	MC   *cpu.CPU
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

	vcs := new(VCS)
	vcs.TV = tv

	vcs.Mem, err = memory.NewVCSMemory()
	if err != nil {
		return nil, err
	}

	vcs.MC, err = cpu.NewCPU(vcs.Mem)
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
	if err := vcs.MC.Reset(); err != nil {
		return err
	}

	// TODO: consider implementing tia.Reset and riot.Reset instead of
	// recreating the two components

	vcs.TIA = tia.NewTIA(vcs.TV, vcs.Mem.TIA)
	if vcs.TIA == nil {
		return errors.NewFormattedError(errors.VCSError, "can't create TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return errors.NewFormattedError(errors.VCSError, "can't create RIOT")
	}

	err := vcs.MC.LoadPCIndirect(memory.AddressReset)
	if err != nil {
		return err
	}

	return nil
}

func (vcs *VCS) strobe() error {
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

// Step the emulator state one CPU instruction
// -- we can put this function in a loop for an effective debugging loop
// ths videoCycleCallback function for an additional callback point in the
// debugger.
func (vcs *VCS) Step(videoCycleCallback func(*result.Instruction) error) (int, *result.Instruction, error) {
	var r *result.Instruction
	var err error

	// the number of CPU cycles that have elapsed.  note this is *not* the same
	// as Instructionresult.ActualCycles because in the event of a WSYNC
	// cpuCycles will continue to accumulate until the WSYNC has been resolved.
	cpuCycles := 0

	// the cpu calls the cycleVCS function after every CPU cycle. the cycleVCS
	// function defines the order of operation for the rest of the VCS for
	// every CPU cycle.
	cycleVCS := func(r *result.Instruction) error {
		cpuCycles++

		// ensure controllers have updated their input
		if err := vcs.strobe(); err != nil {
			return err
		}

		// run riot only once per CPU cycle
		// TODO: not sure when in the video cycle sequence it should be run
		// TODO: is this something that can drift, thereby causing subtly different
		// results / graphical effects? is this what RSYNC is for?

		vcs.RIOT.ReadRIOTMemory()
		vcs.RIOT.Step()

		// read tia memory just once and before we cycle the tia
		vcs.TIA.ReadTIAMemory()

		// three color clocks per CPU cycle so we run video cycle three times
		_, err = vcs.TIA.StepVideoCycle()
		if err != nil {
			return err
		}
		videoCycleCallback(r)

		_, err = vcs.TIA.StepVideoCycle()
		if err != nil {
			return err
		}
		videoCycleCallback(r)

		vcs.MC.RdyFlg, err = vcs.TIA.StepVideoCycle()
		if err != nil {
			return err
		}
		videoCycleCallback(r)

		return nil
	}

	r, err = vcs.MC.ExecuteInstruction(cycleVCS)
	if err != nil {
		return cpuCycles, nil, err
	}

	// CPU has been left in the unready state - continue cycling the VCS hardware
	// until the CPU is ready
	for !vcs.MC.RdyFlg {
		cycleVCS(r)
	}

	return cpuCycles, r, nil
}

// Run sets the emulation running as quickly as possible.  eventHandler()
// should return false when an external event (eg. a GUI event) indicates that
// the emulation should stop.
func (vcs *VCS) Run(continueCheck func() (bool, error)) error {
	var err error

	cycleVCS := func(r *result.Instruction) error {
		// ensure controllers have updated their inpu
		if err := vcs.strobe(); err != nil {
			return err
		}

		vcs.RIOT.ReadRIOTMemory()
		vcs.RIOT.Step()
		vcs.TIA.ReadTIAMemory()
		vcs.TIA.StepVideoCycle()
		vcs.TIA.StepVideoCycle()
		vcs.MC.RdyFlg, err = vcs.TIA.StepVideoCycle()
		return err
	}

	cont := true
	for cont {
		_, err = vcs.MC.ExecuteInstruction(cycleVCS)
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
func (vcs *VCS) RunForFrameCount(numFrames int) error {
	fn, err := vcs.TV.GetState(television.ReqFramenum)
	if err != nil {
		return err
	}

	targetFrame := fn + numFrames

	for fn != targetFrame {
		_, _, err = vcs.Step(func(*result.Instruction) error { return nil })
		fn, err = vcs.TV.GetState(television.ReqFramenum)
		if err != nil {
			return err
		}
	}

	return nil
}
