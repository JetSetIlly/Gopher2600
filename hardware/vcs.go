// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package hardware

import (
	"gopher2600/cartridgeloader"
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

	// convenient pointers to the panel and hand controller instances in the
	// RIOT.Input type
	Panel           input.Port
	HandController0 input.Port
	HandController1 input.Port
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

	vcs.TIA, err = tia.NewTIA(vcs.TV, vcs.Mem.TIA)
	if err != nil {
		return nil, err
	}

	vcs.RIOT, err = riot.NewRIOT(vcs.Mem.RIOT, vcs.Mem.TIA)
	if err != nil {
		return nil, err
	}

	vcs.Panel = vcs.RIOT.Input.Panel
	vcs.HandController0 = vcs.RIOT.Input.HandController0
	vcs.HandController1 = vcs.RIOT.Input.HandController1

	return vcs, nil
}

// AttachCartridge loads a cartridge (given by filename) into the emulators
// memory. While this function can be called directly it is advised that the
// setup package be used in most circumstances.
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
	err := vcs.HandController0.CheckInput()
	if err != nil {
		return err
	}

	err = vcs.HandController1.CheckInput()
	if err != nil {
		return err
	}

	return vcs.Panel.CheckInput()
}
