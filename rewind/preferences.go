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

package rewind

import (
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

type Preferences struct {
	r   *Rewind
	dsk *prefs.Disk

	// whether to apply the high mirror bits to the displayed address
	MaxEntries prefs.Int
	Freq       prefs.Int
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

// the maximum number of entries to store before the earliest steps are forgotten.
const maxEntries = 250

// how often a frame snapshot of the system be taken. the higher the number,
// the more laggy the rewind system will feel, particularly in a GUI.
//
// 5 is probably the maximum you'd want to go for now.
const snapshotFreq = 1

// newPreferences is the preferred method of initialisation for the Preferences type.
func newPreferences(r *Rewind) (*Preferences, error) {
	p := &Preferences{r: r}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("rewind.maxEntries", &p.MaxEntries)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("rewind.snapshotFreq", &p.Freq)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load()
	if err != nil {
		return nil, err
	}

	p.MaxEntries.SetHookPost(func(_ prefs.Value) error {
		r.allocate()
		return nil
	})

	p.Freq.SetHookPost(func(_ prefs.Value) error {
		r.allocate()
		return nil
	})

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *Preferences) SetDefaults() {
	p.MaxEntries.Set(maxEntries)
	p.Freq.Set(snapshotFreq)
}

// Load rewind preferences from disk.
func (p *Preferences) Load() error {
	return p.dsk.Load()
}

// Save current rewind preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
