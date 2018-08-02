package hardware

import (
	"fmt"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/peripherals"
	"gopher2600/hardware/riot"
	"gopher2600/hardware/tia"
	"gopher2600/television"
)

// AddressReset is the address where the reset address is stored
// - used by VCS.Reset() and Disassembly module
const AddressReset = 0xfffc

// AddressIRQ is the address where the interrupt address is stored
const AddressIRQ = 0xfffe

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	MC   *cpu.CPU
	Mem  *memory.VCSMemory
	TIA  *tia.TIA
	RIOT *riot.RIOT

	// tv is not part of the VCS but is attached to it
	TV television.Television

	panel      *peripherals.Panel
	controller *peripherals.Stick
}

// New is the preferred method of initialisation for the VCS structure
func New(tv television.Television) (*VCS, error) {
	var err error

	vcs := new(VCS)
	vcs.TV = tv

	vcs.Mem, err = memory.New()
	if err != nil {
		return nil, err
	}

	vcs.MC, err = cpu.New(vcs.Mem)
	if err != nil {
		return nil, err
	}

	vcs.TIA = tia.New(vcs.TV, vcs.Mem.TIA)
	if vcs.TIA == nil {
		return nil, fmt.Errorf("can't allocate memory for VCS TIA")
	}

	vcs.RIOT = riot.New(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return nil, fmt.Errorf("can't allocate memory for VCS RIOT")
	}

	vcs.panel = peripherals.NewPanel(vcs.Mem.RIOT)
	if vcs.panel == nil {
		return nil, fmt.Errorf("can't create console control panel")
	}

	// TODO: better contoller support
	vcs.controller = peripherals.NewStick(vcs.Mem.TIA, vcs.Mem.RIOT, vcs.panel)
	if vcs.panel == nil {
		return nil, fmt.Errorf("can't create new stick controller")
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

// NullVideoCycleCallback can be used when calling Step() when no special
// behaviour is required
func NullVideoCycleCallback(*result.Instruction) error {
	return nil
}

// Step the emulator state one CPU instruction
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
	cycleVCS := func(r *result.Instruction) {
		cpuCycles++

		// run riot only once per CPU cycle
		// TODO: not sure when in the video cycle sequence it should be run
		// TODO: is this something that can drift, thereby causing subtly different
		// results / graphical effects? is this what RSYNC is for?

		vcs.RIOT.ReadRIOTMemory()
		vcs.RIOT.Step()

		// three color clocks per CPU cycle so we run video cycle three times

		vcs.TIA.ReadTIAMemory()
		vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		videoCycleCallback(r)

		vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		videoCycleCallback(r)

		vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		videoCycleCallback(r)
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

// Reset emulates the reset switch on the console panel
//  - reset the CPU
//  - reload reset address into the PC
func (vcs *VCS) Reset() error {
	if err := vcs.MC.Reset(); err != nil {
		return err
	}
	err := vcs.MC.LoadPC(AddressReset)
	if _, ok := err.(*memory.MissingCartridgeError); !ok {
		return err
	}
	return nil
}
