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

	// the speed at which the screen recovers once a valid VSYNC signal is
	// received. the higher the value the slower the recovery
	VSYNCrecovery prefs.Int

	// whether the televsion should be synced on start
	VSYNCsyncedOnStart prefs.Bool

	// halt conditions
	HaltVSYNCTooShort      prefs.Bool
	HaltVSYNCScanlineStart prefs.Bool
	HaltVSYNCScanlineCount prefs.Bool
	HaltVSYNCabsent        prefs.Bool
	HaltChangedVBLANK      prefs.Bool
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

	err = p.dsk.Add("television.vsync.recovery", &p.VSYNCrecovery)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("television.vsync.syncedonstart", &p.VSYNCsyncedOnStart)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("television.halt.vsynctooshort", &p.HaltVSYNCTooShort)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("television.halt.vsyncscanlinestart", &p.HaltVSYNCScanlineStart)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("television.halt.vsyncscanlinecount", &p.HaltVSYNCScanlineCount)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("television.halt.vsyncabsent", &p.HaltVSYNCabsent)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("television.halt.changedvblank", &p.HaltChangedVBLANK)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load(true)
	if err != nil {
		return p, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *TVPreferences) SetDefaults() {
	p.VSYNCscanlines.Set(2)
	p.VSYNCrecovery.Set(75)
	p.VSYNCsyncedOnStart.Set(true)
	p.HaltVSYNCTooShort.Set(false)
	p.HaltVSYNCScanlineStart.Set(false)
	p.HaltVSYNCScanlineCount.Set(false)
	p.HaltVSYNCabsent.Set(false)
	p.HaltChangedVBLANK.Set(false)
}

// Load television preferences from disk.
func (p *TVPreferences) Load() error {
	return p.dsk.Load(false)
}

// Save current television preferences to disk.
func (p *TVPreferences) Save() error {
	return p.dsk.Save()
}
