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

// Limits of VSYNC recovery values
const (
	VSYNCrecoveryMin = 0
	VSYNCrecoveryMax = 99
)

type TVPreferences struct {
	dsk *prefs.Disk

	// number of scanlines required for a valid scanline signal
	VSYNCscanlines prefs.Int

	// whether synchronisation should happen immediately
	VSYNCimmedateSync prefs.Bool

	// whether the televsion should be synced on start. the value of this should
	// be ignored if 'immedate sync' is enabled
	VSYNCsyncedOnStart prefs.Bool

	// halt conditions
	HaltChangedVBLANK prefs.Bool
	HaltChangedVSYNC  prefs.Bool
}

func newTVPreferences() (*TVPreferences, error) {
	p := &TVPreferences{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("television.vsync.scanlines", &p.VSYNCscanlines)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("television.vsync.immediatesync", &p.VSYNCimmedateSync)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("television.vsync.syncedonstart", &p.VSYNCsyncedOnStart)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("television.halt.changedvblank", &p.HaltChangedVBLANK)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("television.halt.changedvsync", &p.HaltChangedVSYNC)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load()
	if err != nil {
		return p, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *TVPreferences) SetDefaults() {
	p.VSYNCscanlines.Set(2)
	p.VSYNCimmedateSync.Set(false)
	p.VSYNCsyncedOnStart.Set(true)
	p.HaltChangedVBLANK.Set(false)
	p.HaltChangedVSYNC.Set(false)
}

// Load television preferences from disk.
func (p *TVPreferences) Load() error {
	return p.dsk.Load()
}

// Save current television preferences to disk.
func (p *TVPreferences) Save() error {
	return p.dsk.Save()
}
