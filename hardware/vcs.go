package hardware

import (
	"headlessVCS/hardware/cpu"
	"headlessVCS/hardware/memory"
)

const addressReset = 0xFFFC
const addressIRQ = 0xFFFE

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	MC  *cpu.CPU
	Mem *memory.VCSMemory
}

// NewVCS is the preferred method of initialisation for the VCS structure
func NewVCS() *VCS {
	vcs := new(VCS)
	vcs.Mem = memory.NewVCSMemory()
	vcs.MC = cpu.NewCPU(vcs.Mem)
	return vcs
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
		r, err = vcs.MC.ExecuteInstruction(func() {})
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
