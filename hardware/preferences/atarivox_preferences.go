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

type AtariVoxPreferences struct {
	dsk *prefs.Disk

	FestivalEnabled  prefs.Bool
	FestivalBinary   prefs.String
	SubtitlesEnabled prefs.Bool
}

// NewPreferences is the preferred method of initialisation for the Preferences type.
func newAtariVoxPreferences() (*AtariVoxPreferences, error) {
	p := &AtariVoxPreferences{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("peripherals.atarivox.festival.enabled", &p.FestivalEnabled)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("peripherals.atarivox.festival.binary", &p.FestivalBinary)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("peripherals.atarivox.subtitles.enabled", &p.SubtitlesEnabled)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load()
	if err != nil {
		return nil, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *AtariVoxPreferences) SetDefaults() {
	p.FestivalEnabled.Set(true)
	p.FestivalBinary.Set(p.binary())
	p.SubtitlesEnabled.Set(false)
}

// Load disassembly preferences and apply to the current disassembly.
func (p *AtariVoxPreferences) Load() error {
	return p.dsk.Load()
}

// Save current disassembly preferences to disk.
func (p *AtariVoxPreferences) Save() error {
	return p.dsk.Save()
}
