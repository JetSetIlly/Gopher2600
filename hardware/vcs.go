package hardware

import (
	"fmt"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/riot"
	"gopher2600/hardware/tia"
	"gopher2600/television"
)

const addressReset = 0xfffc
const addressIRQ = 0xfffe

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	MC   *cpu.CPU
	Mem  *memory.VCSMemory
	TIA  *tia.TIA
	RIOT *riot.RIOT

	// tv is not part of the VCS but is attached to it
	TV television.Television
}

// NewVCS is the preferred method of initialisation for the VCS structure
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
		return nil, fmt.Errorf("can't allocate memory for VCS TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return nil, fmt.Errorf("can't allocate memory for VCS RIOT")
	}

	return vcs, nil
}

// AttachCartridge loads a cartridge (a file) into the emulators memory
func (vcs *VCS) AttachCartridge(filename string) error {
	err := vcs.Mem.Cart.Attach(filename)
	if err != nil {
		return err
	}
	err = vcs.Reset()
	if err != nil {
		return err
	}
	return nil
}

// Step the emulator state one CPU instruction
func (vcs *VCS) Step() (int, *cpu.InstructionResult, error) {
	var r *cpu.InstructionResult
	var err error

	// the number of CPU cycles elapsed before we return control to the calling
	// function. note this is *not* the same as Instructionresult.ActualCycles
	// because in the event of a WSYNC, this function will continue until the
	// WSYNC has completed
	cpuCycles := 0

	// cycle the VCS hardware whenever this function is called -- defined to be
	// run once per CPU cycle
	// TODO: allow debugger to take control after every color clock
	cycleVCS := func() {
		cpuCycles++

		// three color clocks per CPU cycle:

		// two color clocks to begin...
		vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		vcs.RIOT.StepVideoCycle()
		vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		vcs.RIOT.StepVideoCycle()

		// ... check for side effects from the CPU operation ...
		vcs.TIA.ReadTIAMemory()
		vcs.RIOT.ReadRIOTMemory()

		// ... final color clock
		vcs.MC.RdyFlg = vcs.TIA.StepVideoCycle()
		vcs.RIOT.StepVideoCycle()
	}

	// TODO: controllers

	// loop until we have a completed instruction result
	for r == nil || r.Final == false {
		r, err = vcs.MC.ExecuteInstruction(cycleVCS)
		if err != nil {
			return cpuCycles, nil, err
		}
	}

	// CPU has been left in the unready state - continue cycling the VCS hardware
	// until the CPU is ready
	for vcs.MC.RdyFlg == false {
		cycleVCS()
	}

	return cpuCycles, r, nil
}

// Reset emulates the reset switch on the console panel
//  - reset the CPU
//  - reload reset address into the PC
func (vcs *VCS) Reset() error {
	vcs.MC.Reset()
	err := vcs.MC.LoadPC(addressReset)
	if err != nil {
		return err
	}
	return nil
}
