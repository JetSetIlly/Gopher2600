package hardware

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/riot"
	"gopher2600/hardware/riot/input"
	"gopher2600/hardware/tia"
	"gopher2600/television"
)

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	CPU  *cpu.CPU
	Mem  *memory.VCSMemory
	TIA  *tia.TIA
	RIOT *riot.RIOT

	TV television.Television

	Panel   *input.Panel
	Player0 *input.Player
	Player1 *input.Player
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

	vcs.RIOT, err = riot.NewRIOT(vcs.Mem.RIOT, vcs.Mem.TIA)
	if err != nil {
		return nil, errors.New(errors.VCSError, fmt.Sprintf("can't create RIOT: %v", err))
	}

	// for convenience, these should point to the equivalent instances in the
	// RIOT.Input type
	vcs.Panel = vcs.RIOT.Input.Panel
	vcs.Player0 = vcs.RIOT.Input.Player0
	vcs.Player1 = vcs.RIOT.Input.Player1

	return vcs, nil
}

// AttachCartridge loads a cartridge (given by filename) into the emulators
// memory. While this function can be called directly it is advised that the
// setup package is used in most circumstances.
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
func (vcs *VCS) Reset() error {
	// note that there is no reset of the CPU, the TIA or the RIOT. this is
	// because I don't believe it's required. memory is an unknown state and
	// the RIOT/TIA registers are in an unknown state - effectively randomised.
	// we could maybe had a "hard reset" option in the future if we need it

	vcs.Mem.Cart.Initialise()

	err := vcs.CPU.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return err
	}

	return nil
}

// check all devices for pending input
func (vcs *VCS) checkDeviceInput() error {
	err := vcs.Player0.CheckInput()
	if err != nil {
		return err
	}

	err = vcs.Player1.CheckInput()
	if err != nil {
		return err
	}

	return vcs.Panel.CheckInput()
}
