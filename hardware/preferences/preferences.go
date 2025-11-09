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

// Preferences defines and collates all the preference values used by the emulation.
type Preferences struct {
	dsk *prefs.Disk

	// initialise hardware to unknown state after reset
	RandomState prefs.Bool

	// unused pins when reading TIA/RIOT registers take the value of the last
	// value on the bus. if RandomPins is true then the values of the unusued
	// pins are randomised. this is the equivalent of the Stella option "drive
	// unused pins randomly on a read/peek"
	RandomPins prefs.Bool

	// unwrap ACE binaries when possible and use a more direct emulation
	UnwrapACE prefs.Bool

	// preferences used by the television
	TV *TVPreferences

	// preferences used by the ARM sub-system
	ARM *ARMPreferences

	// preferences used by PlusROM cartridges
	PlusROM *PlusROMPreferences

	// preferences used by the TIA package in order to emulate different
	// revisions of the TIA chip
	Revision *RevisionPreferences

	// preferences for the AtariVox peripheral
	AtariVox *AtariVoxPreferences
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

// NewPreferences is the preferred method of initialisation for the Preferences type.
func NewPreferences() (*Preferences, error) {
	p := &Preferences{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.randState", &p.RandomState)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.randPins", &p.RandomPins)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.unwrapAce", &p.UnwrapACE)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Load()
	if err != nil {
		return nil, err
	}

	p.TV, err = newTVPreferences()
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

	p.Revision, err = newRevisionPreferences()
	if err != nil {
		return nil, err
	}

	p.AtariVox, err = newAtariVoxPreferences()
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *Preferences) SetDefaults() {
	// initialise random number generator
	p.RandomState.Set(false)
	p.RandomPins.Set(false)
	p.UnwrapACE.Set(true)
}

// Load current hardware preference from disk.
func (p *Preferences) Load() error {
	return p.dsk.Load()
}

// Save current hardware preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
