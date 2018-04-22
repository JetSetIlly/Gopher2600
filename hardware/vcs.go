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
	vcs.Reset()
	return nil
}

// Step the emulator state one CPU insruction
func (vcs *VCS) Step() (*cpu.InstructionResult, error) {
	return vcs.MC.StepInstruction()
}

// Reset emulates the reset switch on the console panel
func (vcs *VCS) Reset() {
	vcs.MC.Reset()
	vcs.MC.LoadPC(addressReset)
}
