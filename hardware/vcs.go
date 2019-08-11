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
	// the videoCycle() function defined below. the only practical effect I can
	// see from this is that it alters the skew between the OSC and ϕ0 - the
	// changes to ϕ0 and OSC still happen at more-or-less the same time, which
	// I think is good enough for accurate emulation.
	//
	// the reason for this inside-out arrangement is simply a consequence of
	// the how the CPU emulation is put together. it is easier for the large
	// CPU ExecuteInstruction() function to call out to the videoCycle()
	// function. if we were to do it the other way around then keeping track of
	// the interim CPU state becomes trickier.
	//
	// we could solve this by using go-channels but early experiments suggested
	// that this was too slow. a better solution would be to build the CPU
	// instructions out of smaller micro-instructions. we sort of do that now
	// but doing so explicitely will make jumping in and out of the CPU far
	// easier. (note that changing how CPU cycles and video cycles interact
	// will also effect how the debugger is structured.)
	//
	// I don't believe any visual or audible artefacts of the VCS (undocumented
	// or not) rely on the details of the CPU-TIA relationship.
	videoCycle := func(r *result.Instruction) error {
		// ensure controllers have updated their input
		if err := vcs.strobeUserInput(); err != nil {
			return err
		}

		// update RIOT memory and step
		vcs.RIOT.ReadMemory()
		vcs.RIOT.Step()

		// in addition to the ϕ0 clock, which is connected from the TIA to the
		// CPU, there is the ϕ2 clock. The ϕ2 clock is connected from the CPU
		// to the TIA. in that sense at least, this emulation is correct.
		//
		// examining the "TIA ϕ0-ϕ2 and LUM timing" diagram in Steve Wright's
		// "Stella Programmer's Guide" and the "TIA 1A" document, we can see
		// that the ϕ2 clock is exactly one OSC pulse behind the ϕ0 clock. that
		// is, a ϕ2 rising edge occurs one tick later than a rising edge of ϕ0.
		//
		// according to the "TIA 1A" document:
		//
		// "if the read-write line is low, the data [...] will be written into
		// the addressed write location when the Q2 clock goes from high to
		// low."
		//
		// to help us understand what's going on, the following diagram
		// replicates the diagram mentioned above.
		//
		// OSC  .-._.-._.-._.-._.-._.-._.-._
		//  ϕ0  ___.-----._____.-----._____.
		//  ϕ2  .____.-----._____.-----.____
		//
		// to reiterate, each pulse of the OSC is a color-clock, or put another
		// way, one tick of the TIA. every third causes the ϕ0 to tick. in this
		// emulation however, we've altered the skew between these two clocks;
		// so the diagram looks more like this:
		//
		// OSC  ._.-._.-._.-._.-._.-._.-._
		//  ϕ0  ___.-----._____.-----._____.
		//  ϕ2  .____.-----._____.-----.____
		//
		// we've already mentioned how memory should be read by the TIA on the
		// lowering edge of ϕ2. according to the ammednded diagram above, this
		// edge conincides with the 2nd step of the OSC clock; or, in the
		// context of this emulation, sometime between the 2nd and 3rd call to
		// vcs.TIA.Step() in this videoCycle function.

		// step one ...
		_, err = vcs.TIA.Step(false)
		if err != nil {
			return err
		}
		_ = videoCycleCallback(r)

		// ... tia step two ...
		_, err = vcs.TIA.Step(false)
		if err != nil {
			return err
		}
		_ = videoCycleCallback(r)

		// ... tia step three
		vcs.CPU.RdyFlg, err = vcs.TIA.Step(true)
		if err != nil {
			return err
		}
		_ = videoCycleCallback(r)

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

		vcs.RIOT.ReadMemory()
		vcs.RIOT.Step()

		_, _ = vcs.TIA.Step(false)
		_, _ = vcs.TIA.Step(false)
		vcs.CPU.RdyFlg, err = vcs.TIA.Step(true)

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
