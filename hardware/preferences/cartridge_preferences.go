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

package preferences

import (
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

type Cartridge struct {
	dsk *prefs.Disk

	// unwrap ACE binaries when possible and use a more direct emulation
	UnwrapACE prefs.Bool

	// emulate the cycle limitations of the SARA chip
	// NOTE that this value is not hooked into the live emulate sara value in the atari mapper. it
	// probably should be, although the GUI preferences window handles that for us
	EmulateSARA prefs.Bool

	// preferences used by the ARM sub-system
	ARM *ARMPreferences

	// preferences used by PlusROM cartridges
	PlusROM *PlusROMPreferences
}

func newCartridgePreferences() (*Cartridge, error) {
	p := &Cartridge{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("hardware.cartridge.unwrapAce", &p.UnwrapACE)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.cartridge.emulateSARA", &p.EmulateSARA)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Load()
	if err != nil {
		return nil, err
	}

	p.ARM, err = newARMprefrences()
	if err != nil {
		return nil, err
	}

	p.PlusROM, err = newPlusROMpreferences()
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *Cartridge) SetDefaults() {
	p.UnwrapACE.Set(true)
	p.EmulateSARA.Set(false)
}

// Load current hardware preference from disk.
func (p *Cartridge) Load() error {
	return p.dsk.Load()
}

// Save current hardware preferences to disk.
func (p *Cartridge) Save() error {
	return p.dsk.Save()
}
