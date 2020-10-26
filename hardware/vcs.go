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

package hardware

import (
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/riot"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/controllers"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/tia"
)

// VCS struct is the main container for the emulated components of the VCS.
type VCS struct {
	Prefs *preferences.Preferences
	TV    *television.Television

	// references to the different components of the VCS. do not take copies of
	// these pointer values because the rewind feature will change them.
	CPU  *cpu.CPU
	Mem  *memory.Memory
	RIOT *riot.RIOT
	TIA  *tia.TIA
}

// NewVCS creates a new VCS and everything associated with the hardware. It is
// used for all aspects of emulation: debugging sessions, and regular play.
func NewVCS(tv *television.Television) (*VCS, error) {
	// set up preferences
	prefs, err := preferences.NewPreferences()
	if err != nil {
		return nil, err
	}

	// set up hardware
	vcs := &VCS{
		Prefs: prefs,
		TV:    tv,
	}

	vcs.Mem = memory.NewMemory(vcs.Prefs)
	vcs.CPU = cpu.NewCPU(vcs.Prefs, vcs.Mem)
	vcs.RIOT = riot.NewRIOT(vcs.Mem.RIOT, vcs.Mem.TIA)
	vcs.TIA = tia.NewTIA(vcs.TV, vcs.Mem.TIA, vcs.RIOT.Ports)

	err = vcs.RIOT.Ports.AttachPlayer(ports.Player0ID, controllers.NewAuto)
	if err != nil {
		return nil, err
	}

	err = vcs.RIOT.Ports.AttachPlayer(ports.Player1ID, controllers.NewAuto)
	if err != nil {
		return nil, err
	}

	return vcs, nil
}

// AttachCartridge to this VCS. While this function can be called directly it
// is advised that the setup package be used in most circumstances.
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

// Reset emulates the reset switch on the console panel.
func (vcs *VCS) Reset() error {
	err := vcs.TV.Reset()
	if err != nil {
		return err
	}

	vcs.Mem.Reset()
	vcs.CPU.Reset()

	// reset of ports must happen after reset of memory because ports will
	// update memory to the current state of the peripherals
	vcs.RIOT.Ports.Reset()

	// reset PC using reset address in cartridge memory
	err = vcs.CPU.LoadPCIndirect(addresses.Reset)
	if err != nil {
		if !curated.Is(err, cartridge.Ejected) {
			return err
		}
	}

	return nil
}
