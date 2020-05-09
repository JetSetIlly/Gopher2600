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
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/riot/input"
	"github.com/jetsetilly/gopher2600/hardware/tia"
	"github.com/jetsetilly/gopher2600/television"
)

// VCS struct is the main container for the emulated components of the VCS
type VCS struct {
	CPU  *cpu.CPU
	Mem  *memory.VCSMemory
	TIA  *tia.TIA
	RIOT *riot.RIOT

	TV television.Television

	Panel           *input.Panel
	HandController0 *input.HandController
	HandController1 *input.HandController
}

// NewVCS creates a new VCS and everything associated with the hardware. It is
// used for all aspects of emulation: debugging sessions, and regular play
// !!TODO: option for random state on VCS creation
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

	vcs.RIOT, err = riot.NewRIOT(vcs.Mem.RIOT, vcs.Mem.TIA)
	if err != nil {
		return nil, err
	}

	vcs.TIA, err = tia.NewTIA(vcs.TV, vcs.Mem.TIA, &vcs.RIOT.Input.VBlankBits)
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
	err := vcs.TV.Reset()
	if err != nil {
		return err
	}

	vcs.HandController0.Reset()
	vcs.HandController1.Reset()

	// not resetting anything else is effectively leaving the VCS in a random
	// state (if the emulation has moved forward any cycles that is)

	// !!TODO: option for random state on VCS reset

	err = vcs.CPU.LoadPCIndirect(addresses.Reset)
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
