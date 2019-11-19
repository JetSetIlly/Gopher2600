package hardware

import (
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
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
		return nil, errors.New(errors.VCSError, "can't create TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return nil, errors.New(errors.VCSError, "can't create RIOT")
	}

	vcs.Panel = peripherals.NewPanel(vcs.Mem.RIOT)
	if vcs.Panel == nil {
		return nil, errors.New(errors.VCSError, "can't create control panel")
	}

	vcs.Ports = peripherals.NewPorts(vcs.Mem.RIOT, vcs.Mem.TIA, vcs.Panel)
	if vcs.Ports == nil {
		return nil, errors.New(errors.VCSError, "can't create player ports")
	}

	return vcs, nil
}

// AttachCartridge loads a cartridge (given by filename) into the emulators
// memory. While this function can be called directly it is advised that the
// equivalent function call in the setup package is used. that function in turn
// calls this function in this package
func (vcs *VCS) AttachCartridge(cartload cartridgeloader.Loader) error {
	if cartload.Filename == "" {
		vcs.Mem.Cart.Eject()
	} else {
		err := vcs.Mem.Cart.Attach(cartload)
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
		return errors.New(errors.VCSError, "can't create TIA")
	}

	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT)
	if vcs.RIOT == nil {
		return errors.New(errors.VCSError, "can't create RIOT")
	}

	vcs.Mem.Cart.Initialise()

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
