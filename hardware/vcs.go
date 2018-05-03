package hardware

import (
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/tia"
)

const addressReset = 0xfffc
const addressIRQ = 0xfffe

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	MC  *cpu.CPU
	Mem *memory.VCSMemory
	TIA *tia.TIA
}

// NewVCS is the preferred method of initialisation for the VCS structure
func NewVCS() (*VCS, error) {
	var err error

	vcs := new(VCS)

	vcs.Mem, err = memory.NewVCSMemory()
	if err != nil {
		return nil, err
	}

	vcs.MC, err = cpu.NewCPU(vcs.Mem)
	if err != nil {
		return nil, err
	}

	vcs.TIA = tia.NewTIA(vcs.Mem)
	if vcs.TIA == nil {
		return nil, nil
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
func (vcs *VCS) Step() (*cpu.InstructionResult, error) {
	var r *cpu.InstructionResult
	var err error

	for {
		r, err = vcs.MC.ExecuteInstruction(func() {
			// three video cycles per cpu cycle
			vcs.TIA.StepVideoCycle()
			vcs.TIA.StepVideoCycle()
			vcs.TIA.StepVideoCycle()
		})
		if err != nil {
			return nil, err
		}

		// TODO: update rest of VCS

		if r.Final == true {
			return r, err
		}
	}
}

// Reset emulates the reset switch on the console panel
func (vcs *VCS) Reset() error {
	vcs.MC.Reset()
	err := vcs.MC.LoadPC(addressReset)
	if err != nil {
		return err
	}
	return nil
}
