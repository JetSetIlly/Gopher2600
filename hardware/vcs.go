package hardware

import (
	"headlessVCS/hardware/cpu"
	"headlessVCS/hardware/memory"
)

// VCS is the main container for the emulated components of the VCS
type VCS struct {
	mc  *cpu.CPU
	mem *memory.VCSMemory
}

// NewVCS is the preferred method of initialisation for the VCS structure
func NewVCS() *VCS {
	vcs := new(VCS)
	vcs.mem = memory.NewVCSMemory()
	vcs.mc = cpu.NewCPU(vcs.mem)
	return vcs
}

// AttachCartridge loads a cartridge (a file) into the emulators memory
func (vcs *VCS) AttachCartridge(filename string) error {
	return vcs.mem.Cart.Attach(filename)
}
